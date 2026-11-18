package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/desktop/importer"
)

func TestDataLocationBlockPaintsBeforeItPlansImportSources(t *testing.T) {
	// The legacy profile only has to look like one: what the walk costs is
	// covered by the importer's own tests, and this one is about where the
	// frame that draws the block is allowed to do it.
	userConfig := t.TempDir()
	t.Setenv("APPDATA", userConfig)
	legacy := importer.ElectronUserDataDir()
	session := filepath.Join(legacy, "dsh-home", "sessions", "ws", "s1", "session.v4.jsonl.zstd")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte("packed"), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &mainState{}
	first := state.dataLocationAccess(nil)
	if first.ImportAvailable {
		t.Fatal("the data-location block offered an import it had not planned yet")
	}
	if !first.ImportChecking {
		t.Fatal("the block neither offered the import nor said it was checking the sources")
	}

	// The offer arrives on a later frame, once the background pass has it.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if !state.dataLocationAccess(nil).ImportChecking {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the background pass never reported its result")
}
