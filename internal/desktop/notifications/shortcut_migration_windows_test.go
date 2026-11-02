//go:build windows

package notifications

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestShortcutMigrationOnlySameTargetAndIdentity(t *testing.T) {
	target, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, target, id string
		move             bool
	}{{"owned", target, AppUserModelID, true}, {"other-target", target + ".other.exe", AppUserModelID, false}, {"other-identity", target, "other-app", false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			legacy := filepath.Join(dir, "Electron.lnk")
			if err := writeShortcut(legacy, tc.target, tc.id, false); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(legacy)
			if err != nil {
				t.Fatal(err)
			}
			backup, err := (Shortcut{ProgramsDir: dir, Target: target}).MigrateLegacy(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if tc.move {
				if backup == "" {
					t.Fatal("matching legacy link not migrated")
				}
				raw, err := os.ReadFile(backup)
				if err != nil || !bytes.Equal(raw, original) {
					t.Fatalf("backup differs: %v", err)
				}
				if _, err = os.Stat(legacy); !os.IsNotExist(err) {
					t.Fatal("source remains")
				}
			} else {
				raw, err := os.ReadFile(legacy)
				if err != nil || !bytes.Equal(raw, original) || backup != "" {
					t.Fatal("unrelated link changed")
				}
			}
		})
	}
}
