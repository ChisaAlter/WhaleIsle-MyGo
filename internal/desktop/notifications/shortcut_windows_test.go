//go:build windows

package notifications

import (
	"os"
	"testing"
)

func containsBytes(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// TestCreateShortcutTempProfile writes the real link under a temp directory and
// checks idempotency: the second Ensure succeeds and the link still exists.
func TestCreateShortcutTempProfile(t *testing.T) {
	dir := t.TempDir()
	target, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	link := Shortcut{ProgramsDir: dir, Target: target}
	if err := link.Ensure(); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	stat, err := os.Stat(link.Path())
	if err != nil {
		t.Fatalf("stat link: %v", err)
	}
	if stat.Size() == 0 {
		t.Fatal("link is empty")
	}
	if err := link.Ensure(); err != nil {
		t.Fatalf("second Ensure (idempotent update): %v", err)
	}
	again, err := os.Stat(link.Path())
	if err != nil {
		t.Fatalf("stat link after update: %v", err)
	}
	if also := (Shortcut{ProgramsDir: dir, Target: target}).Ensure(); also != nil {
		t.Fatalf("fresh Shortcut Ensure: %v", also)
	}
	if again.Size() == 0 {
		t.Fatal("link became empty after update")
	}
	// The property store keeps the app id as a UTF-16 value inside the link.
	raw, err := os.ReadFile(link.Path())
	if err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 0, len(AppUserModelID)*2)
	for _, r := range AppUserModelID {
		want = append(want, byte(r), 0)
	}
	if !containsBytes(raw, want) {
		t.Fatalf("AUMID %q not stored in the link", AppUserModelID)
	}
}
