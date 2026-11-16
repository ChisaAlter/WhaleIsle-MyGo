package terminal

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestPanelActivateAndCloseDoNotArmFocusPending(t *testing.T) {
	p := &Panel{
		cwd:    "workspace",
		panes:  []*terminalPane{{id: 1}, {id: 2}},
		groups: []*terminalGroup{{id: 1, panes: []uint64{1}}, {id: 2, panes: []uint64{2}}},
		active: 1,
	}
	p.activate(2)
	if p.active != 2 {
		t.Fatalf("active = %d, want 2", p.active)
	}
	if p.focusPending != 0 || p.focused != 0 {
		t.Fatalf("activate armed focus state: focusPending=%d focused=%d", p.focusPending, p.focused)
	}

	p.focusPending = 1
	p.ClearFocus()
	if p.focusPending != 0 || p.focused != 0 {
		t.Fatalf("ClearFocus did not clear pending focus: focusPending=%d focused=%d", p.focusPending, p.focused)
	}

	p.focusPending = 2
	p.closePane(2)
	if p.active != 1 {
		t.Fatalf("active after close = %d, want 1", p.active)
	}
	if p.focusPending != 0 || p.focused != 0 {
		t.Fatalf("closePane left pending focus: focusPending=%d focused=%d", p.focusPending, p.focused)
	}
}

func TestTerminalDrawerHideAndReopenDoesNotStealComposerFocus(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	p := &Panel{
		cwd:    "workspace",
		panes:  []*terminalPane{{id: 1}},
		groups: []*terminalGroup{{id: 1, panes: []uint64{1}}},
		active: 1,
	}
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			composer := ui.Button(c, "Composer").Label("Composer")
			if composer.Clicked() {
				composer.Focus()
			}
			if open {
				View(c, Model{Panel: p})
			} else {
				p.ClearFocus()
			}
		})
	}, 720, 320)

	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.Focused("Composer") {
		t.Fatal("composer lost focus on open terminal drawer frame")
	}

	// Simulate an explicit terminal focus request followed immediately by drawer collapse before paint consumes it.
	p.focusPending = 1
	open = false
	tt.Frame()
	if p.focusPending != 0 {
		t.Fatalf("drawer hide did not clear pending terminal focus: %d", p.focusPending)
	}
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	open = true
	tt.Frame()
	if !tt.Focused("Composer") {
		t.Fatal("reopening terminal drawer stole focus from Composer")
	}

	// Verify explicit tab chip click inside the terminal still activates the chip cleanly when open.
	if err := tt.Click("Terminal 1"); err != nil {
		t.Fatal(err)
	}
	if p.focusPending != 0 {
		t.Fatalf("rendered frame should consume one-shot focusPending, got %d", p.focusPending)
	}
}

func TestTerminalFocusActiveCloseDrawerReturnAndAdjacentPaneTransfer(t *testing.T) {
	returns := 0
	closes := 0
	p := &Panel{
		cwd:    "workspace",
		panes:  []*terminalPane{{id: 1}, {id: 2}, {id: 3}},
		groups: []*terminalGroup{{id: 1, panes: []uint64{1, 2}}, {id: 3, panes: []uint64{3}}},
		active: 2,
		returnFocusFn: func() {
			returns++
		},
	}

	// FocusActive arms pending focus on the active pane.
	p.FocusActive()
	if p.focusPending != 2 {
		t.Fatalf("FocusActive focusPending = %d, want 2", p.focusPending)
	}

	// Simulate active split pane owning focus when it exits/closes: transfers to split sibling 1 (not tab 3).
	p.focused = 2
	p.closePane(2)
	if p.active != 1 || p.focusPending != 1 || p.focused != 1 {
		t.Fatalf("split sibling focus transfer failed: active=%d focusPending=%d focused=%d", p.active, p.focusPending, p.focused)
	}

	// Closing pane 1 transfers adjacent focus to remaining pane 3.
	p.closePane(1)
	if p.active != 3 || p.focusPending != 3 || p.focused != 3 {
		t.Fatalf("tab pane focus transfer failed: active=%d focusPending=%d focused=%d", p.active, p.focusPending, p.focused)
	}

	// Collapsing drawer while terminal holds focus returns focus once; repeated ClearFocus is idempotent.
	p.CloseDrawer()
	if p.focusPending != 0 || p.focused != 0 || returns != 1 {
		t.Fatalf("CloseDrawer did not return focus cleanly once: pending=%d focused=%d returns=%d", p.focusPending, p.focused, returns)
	}
	p.ClearFocus()
	if returns != 1 {
		t.Fatalf("duplicate ClearFocus re-invoked returnFocus: returns=%d", returns)
	}

	// Closing the final pane from view triggers OnReturnFocus and OnClose.
	closePaneFromView(Model{
		Panel:         p,
		OnReturnFocus: func() { returns++ },
		OnClose:       func() { closes++ },
	}, 3)
	if len(p.panes) != 0 || returns != 2 || closes != 1 {
		t.Fatalf("closing final pane failed: panes=%d returns=%d closes=%d", len(p.panes), returns, closes)
	}
}
