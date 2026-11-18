package credentials

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func installEditTestProvider(t *testing.T) []byte {
	t.Helper()
	old := credentialOS
	t.Cleanup(func() { credentialOS = old })
	key := bytes.Repeat([]byte{8}, 32)
	credentialOS = credentialProvider{version: KeychainEnvelopeVersion, backend: "fixture", supported: true, key: func(string, bool) ([]byte, error) { return bytes.Clone(key), nil }}
	return key
}

func TestCredentialEditorMergesOnlyChangedFields(t *testing.T) {
	installEditTestProvider(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"apiKey":"old","githubToken":"old-github","future":{"kept":true},"remoteDevices":[{"id":"phone","future":42}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	editor, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	edited := editor.Fields()
	edited.APIKey = "new"
	peer, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.Update(func(c *Credentials) { c.GithubToken = "peer-github"; c.RemoteDevices[0]["name"] = "changed-phone" }); err != nil {
		t.Fatal(err)
	}
	got, err := editor.CommitFields(edited)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "new" || got.GithubToken != "peer-github" || got.RemoteDevices[0]["name"] != "changed-phone" {
		t.Fatal("editor overwrote independent peer changes")
	}
	raw, _, err := loadRawUnlocked(dir)
	if err != nil || string(raw["future"]) != `{"kept":true}` {
		t.Fatal("unknown namespace lost", err)
	}
	if editor.Fields().GithubToken != "peer-github" {
		t.Fatal("committed baseline did not advance")
	}
}

func TestCredentialEditorConflictPreservesFileAndDraftBaseline(t *testing.T) {
	installEditTestProvider(t)
	dir := t.TempDir()
	if err := Save(dir, Credentials{APIKey: "old"}); err != nil {
		t.Fatal(err)
	}
	editor, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	edited := editor.Fields()
	edited.APIKey = "local-edit"
	if err := Save(dir, Credentials{APIKey: "peer-edit"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, FileName))
	if _, err := editor.CommitFields(edited); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, FileName))
	if !bytes.Equal(before, after) || editor.Fields().APIKey != "old" {
		t.Fatal("conflict modified file or baseline")
	}
	if err := editor.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := editor.CommitFields(edited); err != nil {
		t.Fatal("explicit reload should allow reviewed retry", err)
	}
}

func TestNativeEnvelopeMigrationPreservesEncryptedSource(t *testing.T) {
	installEditTestProvider(t)
	dir := t.TempDir()
	if err := Save(dir, Credentials{APIKey: "source"}); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(dir, FileName))
	var createObserved bool
	password := []byte("fixture-safe-storage-password")
	credentialOS.seal = func(_ string, plain []byte, create bool) (string, []byte, error) {
		createObserved = create
		payload, err := sealLegacyMac(plain, password)
		return EnvelopeVersion, payload, err
	}
	if err := Save(dir, Credentials{APIKey: "updated"}); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(filepath.Join(dir, FileName) + ".mygo-keychain-backup")
	if err != nil || !bytes.Equal(original, backup) {
		t.Fatal("migration source backup not exact", err)
	}
	if !createObserved {
		t.Fatal("native migration cannot create missing shared key")
	}
	current, _ := os.ReadFile(filepath.Join(dir, FileName))
	var env envelope
	if err := json.Unmarshal(current, &env); err != nil || env.Version != EnvelopeVersion {
		t.Fatal("did not publish shared envelope", err)
	}
	payload, _ := base64.StdEncoding.DecodeString(env.Payload)
	plain, err := openLegacyMac(payload, password)
	if err != nil || !bytes.Contains(plain, []byte(`"apiKey":"updated"`)) {
		t.Fatal("shared format cannot read saved data", err)
	}
}
