package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSavePreservesUnknownTopLevelAndNestedFields(t *testing.T) {
	old := credentialOS
	t.Cleanup(func() { credentialOS = old })
	backend := &memoryKeychain{key: bytes.Repeat([]byte{5}, 32)}
	credentialOS = credentialProvider{
		version:   KeychainEnvelopeVersion,
		backend:   "mock",
		supported: true,
		key:       func(dir string, create bool) ([]byte, error) { return keychainKey(backend, dir, create) },
	}

	dir := t.TempDir()
	initial := map[string]any{
		"apiKey":           "old-key",
		"baseUrl":          "https://old.example.test",
		"remoteDevices":    []any{map[string]any{"id": "d1", "name": "Phone", "nestedMeta": map[string]any{"fw": 2}}},
		"unknownTop":       "keep-top",
		"unknownNestedObj": map[string]any{"a": 1, "deep": map[string]any{"flag": true}},
	}
	rawPlain, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), rawPlain, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded.APIKey = "new-key"
	if err := Save(dir, loaded); err != nil {
		t.Fatal(err)
	}

	raw, creds, err := loadRawUnlocked(dir)
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "new-key" || creds.BaseURL != "https://old.example.test" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
	if string(raw["unknownTop"]) != `"keep-top"` {
		t.Fatalf("lost unknownTop: %s", raw["unknownTop"])
	}
	var nested struct {
		A    int `json:"a"`
		Deep struct {
			Flag bool `json:"flag"`
		} `json:"deep"`
	}
	if err := json.Unmarshal(raw["unknownNestedObj"], &nested); err != nil || nested.A != 1 || !nested.Deep.Flag {
		t.Fatalf("lost unknownNestedObj: %s (%v)", raw["unknownNestedObj"], err)
	}
	if len(creds.RemoteDevices) != 1 {
		t.Fatalf("unexpected remoteDevices: %+v", creds.RemoteDevices)
	}
	meta, _ := creds.RemoteDevices[0]["nestedMeta"].(map[string]any)
	if meta["fw"] != float64(2) {
		t.Fatalf("lost nestedMeta inside remoteDevice: %+v", creds.RemoteDevices[0])
	}
}

func TestStoreConflictDetectionAndReload(t *testing.T) {
	old := credentialOS
	t.Cleanup(func() { credentialOS = old })
	backend := &memoryKeychain{key: bytes.Repeat([]byte{3}, 32)}
	credentialOS = credentialProvider{
		version:   KeychainEnvelopeVersion,
		backend:   "mock",
		supported: true,
		key:       func(dir string, create bool) ([]byte, error) { return keychainKey(backend, dir, create) },
	}

	dir := t.TempDir()
	if err := UpdateNamespace(dir, "extra", func(_ json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"nested":{"v":42}}`), nil
	}); err != nil {
		t.Fatal(err)
	}

	s1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s1.Update(func(c *Credentials) { c.APIKey = "writer-1" }); err != nil {
		t.Fatal(err)
	}
	if err := s2.Update(func(c *Credentials) { c.APIKey = "writer-2" }); !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if got := s2.Get().APIKey; got != "" {
		t.Fatalf("conflicted update mutated s2 state: %q", got)
	}
	if err := s2.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := s2.Update(func(c *Credentials) { c.GithubToken = "ghp_2" }); err != nil {
		t.Fatal(err)
	}

	raw, creds, err := loadRawUnlocked(dir)
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "writer-1" || creds.GithubToken != "ghp_2" {
		t.Fatalf("merged creds lost fields: %+v", creds)
	}
	if string(raw["extra"]) != `{"nested":{"v":42}}` {
		t.Fatalf("Store update dropped namespace extra: %s", raw["extra"])
	}
}

func TestCredentialsLockWaitStaleCleanupAndContention(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// 1. Stale dead-PID lock is cleaned up automatically.
	staleMeta, _ := json.Marshal(lockMetadata{PID: 99999999, Token: "dead", CreatedAt: time.Now().UnixMilli()})
	if err := os.WriteFile(lockPath, staleMeta, 0o600); err != nil {
		t.Fatal(err)
	}
	release, err := acquireCredentialsLock(dir)
	if err != nil {
		t.Fatalf("stale dead-PID lock was not recovered: %v", err)
	}
	release()

	// 2. Stale expired-timestamp lock (even with live PID) is recovered cleanly.
	expiredMeta, _ := json.Marshal(lockMetadata{PID: os.Getpid(), Token: "old", CreatedAt: time.Now().Add(-time.Hour).UnixMilli()})
	if err := os.WriteFile(lockPath, expiredMeta, 0o600); err != nil {
		t.Fatal(err)
	}
	release, err = acquireCredentialsLock(dir)
	if err != nil {
		t.Fatalf("expired lock was not recovered: %v", err)
	}
	release()

	// 3. Held live lock blocks briefly then allows acquisition once released, or times out when still held.
	liveMeta, _ := json.Marshal(lockMetadata{PID: os.Getpid(), Token: "holder", CreatedAt: time.Now().UnixMilli()})
	if err := os.WriteFile(lockPath, liveMeta, 0o600); err != nil {
		t.Fatal(err)
	}
	oldTimeout := lockAcquireTimeout
	lockAcquireTimeout = 120 * time.Millisecond
	t.Cleanup(func() { lockAcquireTimeout = oldTimeout })

	if _, err := acquireCredentialsLock(dir); err == nil || !strings.Contains(err.Error(), "lock busy") {
		t.Fatalf("want lock busy error while held, got %v", err)
	}
	lockAcquireTimeout = 800 * time.Millisecond
	go func() {
		time.Sleep(45 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()
	release, err = acquireCredentialsLock(dir)
	if err != nil {
		t.Fatalf("did not acquire lock after holder released: %v", err)
	}
	release()
}

func TestConcurrentNamespaceWritesPreserveAllKeys(t *testing.T) {
	old := credentialOS
	t.Cleanup(func() { credentialOS = old })
	backend := &memoryKeychain{key: bytes.Repeat([]byte{8}, 32)}
	credentialOS = credentialProvider{
		version:   KeychainEnvelopeVersion,
		backend:   "mock",
		supported: true,
		key:       func(dir string, create bool) ([]byte, error) { return keychainKey(backend, dir, create) },
	}

	dir := t.TempDir()
	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers*2)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("ns-%d", idx)
			if err := UpdateNamespace(dir, key, func(_ json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(fmt.Sprintf(`{"worker":%d}`, idx)), nil
			}); err != nil {
				errs <- err
				return
			}
			var got map[string]int
			ok, err := LoadNamespace(dir, key, &got)
			if err != nil || !ok || got["worker"] != idx {
				errs <- fmt.Errorf("load %s: ok=%v got=%v err=%v", key, ok, got, err)
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := Save(dir, Credentials{APIKey: "concurrent-api"}); err != nil {
			errs <- err
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	creds, err := Load(dir)
	if err != nil || creds.APIKey != "concurrent-api" {
		t.Fatalf("APIKey missing after concurrent writes: %+v err=%v", creds, err)
	}
	for i := 0; i < workers; i++ {
		var got map[string]int
		ok, err := LoadNamespace(dir, fmt.Sprintf("ns-%d", i), &got)
		if err != nil || !ok || got["worker"] != i {
			t.Fatalf("worker %d lost: ok=%v got=%v err=%v", i, ok, got, err)
		}
	}
}

func TestWriteAtomicRetriesTransientRenameFailures(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, FileName)
	attempts := 0
	err := writeAtomic(target, []byte(`{"ok":true}`), func(oldpath, newpath string) error {
		attempts++
		if attempts < 3 {
			return os.ErrPermission
		}
		return os.Rename(oldpath, newpath)
	})
	if err != nil {
		t.Fatalf("writeAtomic failed after retries: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("unexpected target contents: %q err=%v", data, err)
	}
}

func TestSaveAutoInitializesLocalStateOnFreshDirAndLoadStaysReadOnly(t *testing.T) {
	dir := t.TempDir()
	// Reading a missing directory/file must create nothing.
	missingSub := filepath.Join(dir, "missing")
	if _, err := Load(missingSub); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(missingSub); !os.IsNotExist(err) {
		t.Fatalf("Load created missing directory: %v", err)
	}
	if ok, err := LoadNamespace(dir, "ssh", &map[string]any{}); err != nil || ok {
		t.Fatalf("LoadNamespace on empty dir = (%v, %v)", ok, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("LoadNamespace touched empty dir: %v %v", entries, err)
	}

	// Saving on Windows auto-provisions Local State DPAPI key on a fresh dir.
	if !credentialOS.supported {
		return
	}
	want := Credentials{APIKey: "fresh-dpapi-key"}
	if err := Save(dir, want); err != nil {
		t.Fatalf("Save on fresh dir: %v", err)
	}
	got, err := Load(dir)
	if err != nil || got.APIKey != want.APIKey {
		t.Fatalf("Load round-trip: %+v err=%v", got, err)
	}
}
