package storefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicRoundTripAndFailureCleanup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "state.json")

	if err := WriteFileAtomic(target, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteFileAtomic(target, []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatalf("overwrite write: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"v":2}` {
		t.Fatalf("content = %q, want %q", got, `{"v":2}`)
	}

	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatalf("unexpected files left in dir: %+v", entries)
	}

	// Attempting atomic overwrite when target is an existing directory must fail
	// without leaving temp files behind.
	badTarget := filepath.Join(dir, "dir-as-target")
	if err := os.MkdirAll(badTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(badTarget, []byte(`{}`), 0o644); err == nil {
		t.Fatal("expected error writing over directory")
	}
	rootEntries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rootEntries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("leaked temp file on failure: %s", e.Name())
		}
	}
}

func TestReadJSONMissingAndCorruptPreservation(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	var v map[string]any
	ok, err := ReadJSON("teststore", missing, &v, false)
	if err != nil || ok {
		t.Fatalf("missing ReadJSON = (%v, %v), want (false, nil)", ok, err)
	}

	corruptPath := filepath.Join(dir, "broken.json")
	corruptBody := "{not-valid-json"
	if err := os.WriteFile(corruptPath, []byte(corruptBody), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err = ReadJSON("teststore", corruptPath, &v, false)
	if err == nil || ok {
		t.Fatalf("corrupt ReadJSON = (%v, %v), want (false, error)", ok, err)
	}
	if !strings.Contains(err.Error(), "teststore: parse broken.json") {
		t.Fatalf("unexpected error: %v", err)
	}

	orig, err := os.ReadFile(corruptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != corruptBody {
		t.Fatalf("original corrupt file modified: %q", orig)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	backups := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "broken.json.corrupt-") {
			backups++
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != corruptBody {
				t.Fatalf("backup content = %q, want %q", raw, corruptBody)
			}
		}
	}
	if backups != 1 {
		t.Fatalf("corrupt backups = %d, want 1", backups)
	}
}
