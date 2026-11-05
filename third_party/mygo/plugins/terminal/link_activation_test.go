package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestNativeLinkActivationUsesPrintedTargetOutsideLockAndReadOnly(t *testing.T) {
	loadLib(t)
	var activated string
	term, err := New(Options{Conn: newPipe(), LinkAt: func(text string, columns []int, column int) string {
		if strings.TrimSpace(text) == "src/main.go:12" && column == 2 && len(columns) > 0 {
			return "src/main.go:12"
		}
		return ""
	}, OnLinkActivate: func(target string) bool {
		activated = target
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("src/main.go:12"))
	term.SetReadOnly(true)
	_ = ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 640, 240)
	// A native callback may take the Terminal lock, e.g. when changing owners.
	// Exercise this on the real emulator without a PTY or an external opener.
	term.opts.OnLinkActivate = func(target string) bool {
		activated = target
		_ = term.Text()
		term.SetReadOnly(true)
		return true
	}
	v := term.v
	done := make(chan bool, 1)
	go func() {
		done <- v.pointerEvent(ui.InputEvent{Kind: ui.InputPointerDown, Button: 0, Mods: ui.Cmd,
			X: float32(v.ox+2*v.cellW+1) / v.scale, Y: float32(v.oy+1) / v.scale})
	}()
	select {
	case handled := <-done:
		if !handled || activated != "src/main.go:12" {
			t.Fatalf("handled=%v target=%q", handled, activated)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("native link callback retained emulator lock")
	}
}
