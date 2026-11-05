package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestMenuLiveRefreshKeepsIdentityAndRejectsRetiredChoice(t *testing.T) {
	rows := []string{"first"}
	chosen := ""
	tt := NewTester(func(c *Context) {
		Button(c, "Resources").Menu(func(m *Menu) {
			m.Live()
			for _, key := range rows {
				if m.Item(key).Key(key).Chosen() {
					chosen = key
				}
			}
		})
	}, 400, 300)
	if err := tt.Click("Resources"); err != nil {
		t.Fatal(err)
	}
	rows = []string{"second", "first"}
	tt.Frame()
	if !slices.Equal(tt.Menu(), rows) || tt.rt.menu.index != 1 {
		t.Fatalf("live inventory or focused identity changed: %v index=%d", tt.Menu(), tt.rt.menu.index)
	}
	// Capture a displayed choice, then withdraw it before its next-frame dispatch.
	callback := tt.h.chosen
	tt.h.menu, tt.h.chosen = nil, nil
	callback(2)
	rows = []string{"second"}
	tt.Frame()
	if chosen != "" {
		t.Fatal("a retired item consumed its delayed choice")
	}
	if err := tt.Click("Resources"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("second"); err != nil || chosen != "second" {
		t.Fatal("current item could not be chosen", err, chosen)
	}
}

func TestDialogOptionsBlocksBusyDismissalAndPeelsOneLayer(t *testing.T) {
	open, busy, nested := true, true, true
	behind, dismissed := 0, 0
	tt := NewTester(func(c *Context) {
		if Button(c, "Behind").Size(80, 30).Clicked() {
			behind++
		}
		DialogBaseWithOptions(c, &open, DialogOptions{DismissDisabled: busy, OnDismiss: func() {
			dismissed++
			if nested {
				nested = false
			} else {
				open = false
			}
		}}, func(backdrop, panel Element) {
			panel.Size(160, 100).Children(func() { Button(c, "Inside").AutoFocus() })
		})
	}, 400, 300)
	tt.ClickAt(10, 10)
	tt.Key(0, KeyEscape)
	if !open || dismissed != 0 || behind != 0 {
		t.Fatal("busy dismissal escaped to its page", open, dismissed, behind)
	}
	busy = false
	tt.Frame()
	tt.Key(0, KeyEscape)
	if !open || nested || dismissed != 1 || behind != 0 {
		t.Fatal("first dismissal did not peel only the nested decision")
	}
	tt.Key(0, KeyEscape)
	if open || dismissed != 2 || behind != 0 {
		t.Fatal("second dismissal did not close the owner")
	}
}

func TestPhysicalNativePriorityHonorsNamedAndNestedModal(t *testing.T) {
	open, nested := false, false
	priority, settings := 0, 0
	draft := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &draft).Label("Editor").AutoFocus()
		if c.PhysicalShortcut(Ctrl, "KeyR", "", "priority", "native-priority") {
			priority++
		}
		if c.PhysicalShortcut(Ctrl, "Comma", "", "settings", "settings-open") {
			settings++
		}
		DialogBase(c, &open, func(backdrop, panel Element) {
			ShortcutModalScope(backdrop, "settings")
			panel.Size(160, 100).Children(func() { Button(c, "Settings").AutoFocus() })
		})
		DialogBase(c, &nested, func(backdrop, panel Element) {
			panel.Size(100, 60).Children(func() { Button(c, "Decision").AutoFocus() })
		})
	}, 400, 300)
	key := func(code string, mods Modifiers, blocked bool) {
		tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Code: code, Mods: platform.Modifiers(mods), ShortcutBlocked: blocked})
		tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Code: code, Mods: platform.Modifiers(mods)})
	}
	key("KeyR", Ctrl, false)
	key("KeyR", Ctrl, true)
	if priority != 1 {
		t.Fatal("editable native priority or IME guard failed", priority)
	}
	open = true
	tt.Frame()
	key("KeyR", Ctrl, false)
	key("Comma", Ctrl, false)
	if priority != 1 || settings != 1 {
		t.Fatal("named modal admitted the wrong command", priority, settings)
	}
	nested = true
	tt.Frame()
	key("Comma", Ctrl, false)
	if settings != 1 {
		t.Fatal("unmarked nested decision inherited settings permission")
	}
}

func TestPhysicalPriorityScopesWaitForImeComposition(t *testing.T) {
	for _, region := range []string{"native-priority", "native-no-modal", "settings-open", "shortcuts-open"} {
		t.Run(region, func(t *testing.T) {
			draft, count := "", 0
			tt := NewTester(func(c *Context) {
				TextInput(c, &draft).Label("IME editor").AutoFocus()
				if c.PhysicalShortcut(Ctrl, "KeyO", "", "open", region) {
					count++
				}
			}, 300, 100)
			tt.Compose("ni", 2)
			tt.PhysicalKeyDown("KeyO", KeyO, Ctrl, false, false, false)
			tt.PhysicalKeyUp("KeyO", KeyO, Ctrl, false)
			if count != 0 || draft != "" {
				t.Fatal("shortcut escaped active composition", count, draft)
			}
			tt.Type("你")
			tt.PhysicalKeyDown("KeyO", KeyO, Ctrl, false, false, false)
			tt.PhysicalKeyDown("KeyO", KeyO, Ctrl, true, false, false)
			tt.PhysicalKeyUp("KeyO", KeyO, Ctrl, false)
			if count != 1 || draft != "你" {
				t.Fatal("ordinary editable shortcut or repeat guard failed", count, draft)
			}
		})
	}
}

func TestShortcutModalNameSeesCurrentNestedDecisionAndPopover(t *testing.T) {
	open, nested, popup := true, false, false
	name := ""
	tt := NewTester(func(c *Context) {
		anchor := Button(c, "Anchor")
		DialogBase(c, &open, func(backdrop, panel Element) {
			ShortcutModalScope(backdrop, "settings")
			panel.Size(160, 100).Children(func() { Button(c, "Settings").AutoFocus() })
		})
		DialogBase(c, &nested, func(backdrop, panel Element) {
			panel.Size(100, 60).Children(func() { Button(c, "Decision").AutoFocus() })
		})
		PopoverBase(c, anchor, &popup, func(panel Element) { panel.Size(80, 60) })
		name = c.ShortcutModalName()
	}, 400, 300)
	if name != "settings" {
		t.Fatal("named settings owner missing", name)
	}
	nested = true
	tt.Frame()
	if name != "other" {
		t.Fatal("current nested decision inherited committed settings", name)
	}
	nested, popup = false, true
	tt.Frame()
	if name != "other" {
		t.Fatal("popover allowed a named-modal command", name)
	}
	popup = false
	tt.Frame()
	tt.SetFocused(false)
	tt.Frame()
	if name != "" {
		t.Fatal("unfocused modal allowed a command", name)
	}
}
