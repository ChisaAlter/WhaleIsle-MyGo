//go:build windows

package notifications

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestUnregisterIntegrationPreservesForeignAndCustomized(t *testing.T) {
	target, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, command, extra string
		keep                 bool
	}{{"owned", `"` + target + `" ` + ToastServerFlag, "", false}, {"foreign", `"` + target + `.other" ` + ToastServerFlag, "", true}, {"custom-value", `"` + target + `" ` + ToastServerFlag, "user-value", true}} {
		t.Run(tc.name, func(t *testing.T) {
			id := filepath.Base(t.TempDir())
			base := `Software\WhaleIsleFixture\` + id
			key, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\LocalServer32`, registry.SET_VALUE)
			if err != nil {
				t.Fatal(err)
			}
			if err := key.SetStringValue("", tc.command); err != nil {
				t.Fatal(err)
			}
			if tc.extra != "" {
				if err := key.SetStringValue(tc.extra, "preserve"); err != nil {
					t.Fatal(err)
				}
			}
			key.Close()
			t.Cleanup(func() {
				_ = registry.DeleteKey(registry.CURRENT_USER, base+`\LocalServer32`)
				_ = registry.DeleteKey(registry.CURRENT_USER, base)
			})
			s := Shortcut{ProgramsDir: t.TempDir(), Target: target}
			if err := s.unregisterOwnedIntegration(registry.CURRENT_USER, base); err != nil {
				t.Fatal(err)
			}
			k, err := registry.OpenKey(registry.CURRENT_USER, base+`\LocalServer32`, registry.QUERY_VALUE)
			if err == nil {
				k.Close()
			}
			if tc.keep && err != nil {
				t.Fatalf("foreign/custom registration removed: %v", err)
			}
			if !tc.keep && err != registry.ErrNotExist {
				t.Fatalf("owned registration remained: %v", err)
			}
		})
	}
	// A same-target link without the Go activator is a preexisting/custom link.
	dir := t.TempDir()
	s := Shortcut{ProgramsDir: dir, Target: target}
	if err := writeShortcut(s.Path(), target, AppUserModelID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.unregisterOwnedIntegration(registry.CURRENT_USER, `Software\WhaleIsleFixture\absent`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path()); err != nil {
		t.Fatal("same-target foreign activator link removed")
	}
}
