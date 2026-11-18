//go:build !windows

package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedProfileUnavailablePreservesFileAndPrivateStoreContract(t *testing.T) {
	installEditTestProvider(t)
	dir := t.TempDir()
	file := filepath.Join(dir, FileName)
	original := []byte(`{"apiKey":"original","unknown":{"preserve":true}}`)
	if err := os.WriteFile(file, original, 0600); err != nil {
		t.Fatal(err)
	}
	shared, err := OpenSharedProfile(dir)
	if err != nil || shared.Get().APIKey != "original" {
		t.Fatalf("shared read rejected: %v", err)
	}
	if !errors.Is(shared.WriteAccessError(), ErrSharedProfileWriteUnavailable) {
		t.Fatal("unavailable shared writer was presented as writable")
	}
	namespaceCalled := false
	for name, write := range map[string]func() error{
		"direct save":  func() error { return SaveSharedProfile(dir, Credentials{APIKey: "new"}) },
		"store save":   func() error { return shared.Save(Credentials{APIKey: "new"}) },
		"store update": func() error { return shared.Update(func(c *Credentials) { c.APIKey = "new" }) },
		"field edit":   func() error { _, err := shared.CommitFields(Fields{APIKey: "new"}); return err },
		"namespace": func() error {
			return UpdateSharedProfileNamespace(dir, "fixture", func(json.RawMessage) (json.RawMessage, error) {
				namespaceCalled = true
				return json.RawMessage(`{"new":true}`), nil
			})
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := write(); !errors.Is(err, ErrSharedProfileWriteUnavailable) {
				t.Fatalf("write did not require a common lease: %v", err)
			}
			data, err := os.ReadFile(file)
			if err != nil || !bytes.Equal(data, original) || shared.Get().APIKey != "original" {
				t.Fatal("rejected write changed the file or committed snapshot")
			}
		})
	}
	if namespaceCalled {
		t.Fatal("shared namespace callback ran without write access")
	}
	if err := shared.Reload(); err != nil {
		t.Fatal("write refusal prevented reload", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != FileName {
		t.Fatal("refused writes created a key, migration backup, or temporary file")
	}
	missing := filepath.Join(dir, "not-created")
	if err := SaveSharedProfile(missing, Credentials{}); !errors.Is(err, ErrSharedProfileWriteUnavailable) {
		t.Fatal("new shared profile bypassed write refusal", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused write created the profile directory")
	}
	private, err := Open(dir)
	if err != nil || private.WriteAccessError() != nil {
		t.Fatal("private Store inherited the desktop write requirement", err)
	}
	if _, err := private.CommitFields(Fields{APIKey: "private-update"}); err != nil {
		t.Fatal("private Store contract changed", err)
	}
	got, err := Load(dir)
	if err != nil || got.APIKey != "private-update" {
		t.Fatal("private write did not round trip", err)
	}
}
