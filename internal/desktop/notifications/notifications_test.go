package notifications

import (
	"os"
	"path/filepath"
	"testing"
)

// TestShortcutPath pins the Start-menu location and name; the empty Programs
// directory means "unknown" and must never produce a path or a write.
func TestShortcutPath(t *testing.T) {
	if got := (Shortcut{}).Path(); got != "" {
		t.Fatalf("empty ProgramsDir produced %q", got)
	}
	if err := (Shortcut{}).Ensure(); err != nil {
		t.Fatalf("unknown location must be a no-op, got %v", err)
	}
	dir := t.TempDir()
	got := Shortcut{ProgramsDir: dir}.Path()
	want := filepath.Join(dir, "Whale Isle.lnk")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if err := (Shortcut{ProgramsDir: dir, Target: ""}).Ensure(); err != nil {
		t.Fatalf("unknown target must be a no-op, got %v", err)
	}
	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Fatalf("no-op must not create the link: %v", err)
	}
}

// TestShortcutStartMenuDir covers the APPDATA shape without touching the real
// profile: the shell seam only reads the environment here.
func TestShortcutStartMenuDir(t *testing.T) {
	t.Setenv("APPDATA", `C:\Users\tester\AppData\Roaming`)
	got := StartMenuProgramsDir()
	want := filepath.Join(`C:\Users\tester\AppData\Roaming`, "Microsoft", "Windows", "Start Menu", "Programs")
	if got != want {
		t.Fatalf("programs dir = %q, want %q", got, want)
	}
	t.Setenv("APPDATA", "")
	if got := StartMenuProgramsDir(); got != "" {
		t.Fatalf("missing APPDATA produced %q", got)
	}
}

// fakeShell records registrations so the command wiring can be tested without
// writing into the real Start menu.
type fakeShell struct {
	calls []struct{ lnk, target, appID string }
	err   error
}

func (f *fakeShell) ensure(lnk, target, appID string) error {
	f.calls = append(f.calls, struct{ lnk, target, appID string }{lnk, target, appID})
	return f.err
}

// TestEnsureUsesShellSeam checks the seam contract: one registration with the
// link path, the executable and the shared app id.
func TestEnsureUsesShellSeam(t *testing.T) {
	dir := t.TempDir()
	shell := &fakeShell{}
	link := Shortcut{ProgramsDir: dir, Target: `C:\Apps\WhaleIsle\whaleisle.exe`}
	if err := link.ensureWith(shell.ensure); err != nil {
		t.Fatal(err)
	}
	if len(shell.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(shell.calls))
	}
	got := shell.calls[0]
	if got.lnk != filepath.Join(dir, "Whale Isle.lnk") || got.target != link.Target || got.appID != AppUserModelID {
		t.Fatalf("unexpected registration: %+v", got)
	}
}
