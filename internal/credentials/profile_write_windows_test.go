//go:build windows

package credentials

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCredentialWriteRespectsDesktopProfileLease(t *testing.T) {
	installEditTestProvider(t)
	dir := t.TempDir()
	file := filepath.Join(dir, FileName)
	if err := os.WriteFile(file, []byte(`{"apiKey":"keep"}`), 0600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(filepath.Join(dir, "lockfile"))
	peer, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err != nil {
		t.Fatal("peer lease prevented reading", err)
	}
	if err := Save(dir, Credentials{APIKey: "blocked"}); !errors.Is(err, ErrSharedProfileBusy) {
		t.Fatalf("write bypassed peer lease: %v", err)
	}
	shared, err := OpenSharedProfile(dir)
	if err != nil || shared.Get().APIKey != "keep" || shared.WriteAccessError() != nil {
		t.Fatalf("shared profile read/support changed: %v", err)
	}
	if _, err := shared.CommitFields(Fields{APIKey: "blocked"}); !errors.Is(err, ErrSharedProfileBusy) {
		t.Fatalf("shared editor bypassed peer lease: %v", err)
	}
	if err := SaveSharedProfile(dir, Credentials{APIKey: "blocked"}); !errors.Is(err, ErrSharedProfileBusy) {
		t.Fatalf("shared save bypassed peer lease: %v", err)
	}
	if err := UpdateSharedProfileNamespace(dir, "fixture", func(json.RawMessage) (json.RawMessage, error) {
		t.Fatal("shared namespace callback ran while peer owned the profile")
		return nil, nil
	}); !errors.Is(err, ErrSharedProfileBusy) {
		t.Fatalf("shared namespace bypassed peer lease: %v", err)
	}
	_ = windows.CloseHandle(peer)
	if data, _ := os.ReadFile(file); string(data) != `{"apiKey":"keep"}` {
		t.Fatal("rejected write changed credentials")
	}
	release, err := ClaimDesktopProfileWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := SaveSharedProfile(dir, Credentials{APIKey: "owned"}); err != nil {
		t.Fatal("own lease blocked save", err)
	}
	leaseEnd, err := acquireProfileWrite(dir)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if peer, err = windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0); err == nil {
		windows.CloseHandle(peer)
		t.Fatal("commit lost lease when launcher released")
	}
	leaseEnd()
	peer, err = windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal("finished commit leaked profile lease", err)
	}
	windows.CloseHandle(peer)
}
