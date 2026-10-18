package terminal

import (
	"errors"
	"strings"
	"testing"
)

func TestTerminalSelectionTarget(t *testing.T) {
	// Only synthetic strings and injected callbacks. No PTY, file or browser calls.
	tests := []struct {
		selection, cwd, route, value string
		line                         int
	}{
		{"ready at (http://localhost:3000/hello).", "/work", "browser", "http://localhost:3000/hello", 0},
		{"http://127.0.0.1:8080", "/work", "browser", "http://127.0.0.1:8080", 0},
		{"https://[::1]:8443/a", "/work", "browser", "https://[::1]:8443/a", 0},
		{"http://0.0.0.0:9000/a", "/work", "browser", "http://0.0.0.0:9000/a", 0},
		{"http://LOCALHOST/a", "/work", "browser", "http://LOCALHOST/a", 0},
		{"http://localhost.evil/a", "/work", "external", "http://localhost.evil/a", 0},
		{"http://127.0.0.1.evil/a", "/work", "external", "http://127.0.0.1.evil/a", 0},
		{"https://example.test/a_(b)).", "/work", "external", "https://example.test/a_(b)", 0},
		{"https://example.test/src/main.go:12", "/work", "external", "https://example.test/src/main.go:12", 0},
		{"error src/main.go:42:7 then http://localhost:3000", "/terminal-owner", "path", "/terminal-owner/src/main.go", 42},
		{"./src/main.go:8", "/bound-workspace", "path", "/bound-workspace/./src/main.go", 8},
		{"/work/src/main.go:0", "/work", "path", "/work/src/main.go", 1},
		{`C:\work\src\main.go:21:4`, `C:\work`, "path", `C:\work\src\main.go`, 21},
		{"src/main.go:2", `C:\work`, "path", `C:\work\src\main.go`, 2},
		{`\\server\share\file.go:9`, `C:\work`, "path", `\\server\share\file.go`, 9},
		{"~/src/main.go:12", "/home/demo/work", "path", "/home/demo/src/main.go", 12},
		{"~/src/main.go", `C:\Users\demo\work`, "path", `C:\Users\demo\src\main.go`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.selection, func(t *testing.T) {
			route, value, line, calls := "", "", 0, 0
			capture := func(kind string) func(string) error {
				return func(raw string) error { route, value, calls = kind, raw, calls+1; return nil }
			}
			actions := SelectionTargetActions{Valid: func() bool { return true }, OpenBrowser: capture("browser"), OpenExternal: capture("external"), OpenPath: func(path string, n int) error { route, value, line, calls = "path", path, n, calls+1; return nil }}
			if err := OpenSelectionTarget(tt.selection, tt.cwd, actions); err != nil {
				t.Fatal(err)
			}
			if route != tt.route || value != tt.value || line != tt.line || calls != 1 {
				t.Fatalf("got %s %q line=%d calls=%d", route, value, line, calls)
			}
		})
	}
	for _, selection := range []string{"plain output", "file:///etc/passwd", "ftp://example.test/a", "https://user:secret@localhost/", "http://", "http:/tmp/file", "https:tmp/file", "file:/tmp/file", "custom://", "https://localhost:bad/", "./a\x00b", strings.Repeat("x", maxTargetSelectionBytes+1)} {
		calls := 0
		open := func(string) error { calls++; return nil }
		err := OpenSelectionTarget(selection, "/work", SelectionTargetActions{Valid: func() bool { return true }, OpenBrowser: open, OpenExternal: open, OpenPath: func(string, int) error { calls++; return nil }})
		if err == nil || calls != 0 {
			t.Errorf("unsafe/unsupported selection %q: err=%v calls=%d", selection[:min(len(selection), 80)], err, calls)
		}
	}
	if err := OpenSelectionTarget("src/main.go", "", SelectionTargetActions{Valid: func() bool { return true }, OpenPath: func(string, int) error { t.Fatal("missing cwd dispatched"); return nil }}); err == nil {
		t.Fatal("missing cwd accepted")
	}
	for _, changeAt := range []int{1, 2} {
		checks, calls := 0, 0
		err := OpenSelectionTarget("http://localhost:3000", "/work", SelectionTargetActions{Valid: func() bool { checks++; return checks < changeAt }, OpenBrowser: func(string) error { calls++; return nil }})
		if err == nil || calls != 0 {
			t.Fatalf("stale owner dispatched: check=%d calls=%d", changeAt, calls)
		}
	}
	expected := errors.New("opener unavailable")
	err := OpenSelectionTarget("https://example.test", "/work", SelectionTargetActions{Valid: func() bool { return true }, OpenExternal: func(string) error { return expected }})
	if !errors.Is(err, expected) {
		t.Fatalf("lost opener error: %v", err)
	}
	// Resolving traversal is lexical; the Files callback remains the authority.
	denied := errors.New("outside terminal workspace")
	err = OpenSelectionTarget("../outside.txt", "/work", SelectionTargetActions{Valid: func() bool { return true }, OpenPath: func(path string, line int) error {
		if path != "/work/../outside.txt" {
			t.Fatal(path)
		}
		return denied
	}})
	if !errors.Is(err, denied) {
		t.Fatalf("lost path boundary denial: %v", err)
	}
}
