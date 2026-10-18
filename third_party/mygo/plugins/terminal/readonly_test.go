package terminal

import (
	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/ui"
	"testing"
)

// These tests do not call New, Load, Feed, or any native vt function.
func TestHostTerminalReadOnlyQueueDropsWithoutReplay(t *testing.T) {
	term := new(Terminal)
	term.Send([]byte("old"))
	if len(term.in.queue) != 1 || term.ReadOnly() {
		t.Fatal("default writable queue")
	}
	term.SetReadOnly(true)
	term.Send([]byte("typed"))
	term.in.push([]byte("focus/mouse/emulator reply"))
	if len(term.in.queue) != 0 || !term.ReadOnly() {
		t.Fatal("read-only input enqueued")
	}
	term.SetReadOnly(false)
	if len(term.in.queue) != 0 {
		t.Fatal("old input replayed")
	}
	term.Send([]byte("new explicit input"))
	if len(term.in.queue) != 1 || string(term.in.queue[0]) != "new explicit input" {
		t.Fatal(term.in.queue)
	}
}
func TestHostTerminalReadOnlyKeyboardIMEAndPasteAreSuppressed(t *testing.T) {
	term := new(Terminal)
	term.SetReadOnly(true)
	v := &view{t: term, pending: &pendingKey{}, preedit: "uncommitted"}
	for _, ev := range []ui.InputEvent{
		{Kind: ui.InputText, Text: "command\n"}, {Kind: ui.InputCompose, Text: "输入法"},
		{Kind: ui.InputKeyDown, Key: ui.KeyEnter}, {Kind: ui.InputKeyUp, Key: ui.KeyEnter},
		{Kind: ui.InputKeyDown, Key: ui.KeyV, Mods: ui.Ctrl | ui.Shift},
		{Kind: ui.InputCommand, Text: "paste"},
	} {
		v.input(ev)
	}
	v.sendText("direct text")
	v.paste()
	term.Paste("direct paste")
	if v.encodeKey(vt.KeyEvent{Key: vt.KeyEnter}, true) {
		t.Fatal("encoded read-only key")
	}
	if v.report(ui.InputEvent{Kind: ui.InputPointerDown}) {
		t.Fatal("reported read-only pointer")
	}
	if len(term.in.queue) != 0 || v.pending != nil || v.preedit != "" || v.reporting {
		t.Fatal("read-only input/composition leaked")
	}
	// Regaining control must not emit the suppressed composition or key release.
	term.SetReadOnly(false)
	if len(term.in.queue) != 0 || v.pending != nil {
		t.Fatal("suppressed input replayed")
	}
}
func TestHostTerminalReadOnlySelectionRoutesRemainEnabled(t *testing.T) {
	term := new(Terminal)
	term.SetReadOnly(true)
	v := &view{t: term}
	allow := false
	checks := 0
	term.opts.CanCopy = func() bool { checks++; return allow }
	if term.canCopy() {
		t.Fatal("stale copy allowed")
	}
	v.copy()
	if checks != 2 {
		t.Fatal("copy did not revalidate identity")
	}
	allow = true
	if !term.canCopy() {
		t.Fatal("current copy rejected")
	}
	// A nil emulator safely reaches the selection/copy entry points without
	// loading Ghostty; the view itself is no longer disabled for read-only use.
	for _, command := range []string{"copy", "selectAll"} {
		if !v.input(ui.InputEvent{Kind: ui.InputCommand, Text: command}) {
			t.Fatal(command)
		}
	}
	if _, ok := term.SelectionText(); ok {
		t.Fatal("invented a selection without an emulator")
	}
	if len(term.in.queue) != 0 {
		t.Fatal("selection controls produced input")
	}
}
