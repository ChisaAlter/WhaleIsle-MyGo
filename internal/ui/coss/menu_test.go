package coss

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMenuContextMenuViewportClamp(t *testing.T) {
	const (
		winW = 400
		winH = 300
	)
	items := []MenuItem{
		{Kind: MenuAction, ID: "open", Label: "Open Plugin"},
		{Kind: MenuAction, ID: "inspect", Label: "Inspect Details"},
		{Kind: MenuAction, ID: "disable", Label: "Disable Plugin"},
		{Kind: MenuAction, ID: "uninstall", Label: "Uninstall Plugin", Destructive: true},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		target := ui.ButtonBase(c).Fill().Label("Plugin Row")
		ContextMenu(c, target, items, MenuOptions{Label: "Plugin Actions"})
	}, winW, winH)

	// Right-click right next to the bottom-right corner of the window.
	tt.RightClickAt(winW-6, winH-6)
	tt.Frame()

	for _, label := range []string{"Open Plugin", "Inspect Details", "Disable Plugin", "Uninstall Plugin"} {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing menu item %q; texts=%q", label, tt.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > winW || rect.Y+rect.H > winH {
			t.Fatalf("menu item %q clipped outside %dx%d viewport: rect=%+v", label, winW, winH, rect)
		}
	}
}

func TestMenuContextMenuShiftF10KeyboardNav(t *testing.T) {
	chosen := ""
	items := []MenuItem{
		{Kind: MenuAction, ID: "disabled-first", Label: "Disabled First", Disabled: true},
		{Kind: MenuAction, ID: "first", Label: "First Enabled"},
		{Kind: MenuAction, ID: "second", Label: "Second Enabled"},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		target := ui.ButtonBase(c).Size(120, 32).Label("Row Target").AutoFocus()
		if id := ContextMenu(c, target, items, MenuOptions{Label: "Row Menu"}); id != "" {
			chosen = id
		}
	}, 400, 300)

	tt.Key(ui.Shift, ui.KeyF10)
	tt.Frame()
	if !tt.HasText("First Enabled") {
		t.Fatalf("Shift+F10 did not open ContextMenu; texts=%q", tt.Texts())
	}

	// First enabled item should already be focused: Down -> Enter chooses "second".
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if chosen != "second" {
		t.Fatalf("Shift+F10 + Down + Enter chose %q, want %q", chosen, "second")
	}
}

func TestMenuSubClosesOnOutsideClickAndEscape(t *testing.T) {
	items := []MenuItem{
		{Kind: MenuAction, ID: "plain", Label: "Plain Action"},
		{
			Kind:  MenuSub,
			ID:    "more",
			Label: "More Tools",
			Items: []MenuItem{
				{Kind: MenuAction, ID: "sub-a", Label: "Sub Item Alpha"},
				{Kind: MenuAction, ID: "sub-b", Label: "Sub Item Beta"},
			},
		},
	}

	t.Run("outside-click", func(t *testing.T) {
		open := true
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() {
				trigger := Button(c, "Menu Trigger", ButtonOptions{Variant: ButtonOutline})
				Menu(c, trigger, &open, items, MenuOptions{Label: "Main Menu"})
			})
		}, 500, 400)

		if err := tt.Click("More Tools"); err != nil {
			t.Fatalf("click More Tools: %v", err)
		}
		if !tt.HasText("Sub Item Alpha") {
			t.Fatalf("submenu did not open; texts=%q", tt.Texts())
		}

		// Click outside the submenu and menu area.
		tt.ClickAt(480, 380)
		tt.Frame()
		tt.Frame()
		if tt.HasText("Sub Item Alpha") {
			t.Fatalf("submenu remained open after outside click; texts=%q", tt.Texts())
		}
	})

	t.Run("escape-stays-closed-even-while-hovered", func(t *testing.T) {
		open := true
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() {
				trigger := Button(c, "Menu Trigger", ButtonOptions{Variant: ButtonOutline})
				Menu(c, trigger, &open, items, MenuOptions{Label: "Main Menu"})
			})
		}, 500, 400)

		moreRect, ok := tt.Find("More Tools")
		if !ok {
			t.Fatalf("missing More Tools; texts=%q", tt.Texts())
		}
		// Hover to open submenu, keeping pointer on trigger row.
		tt.Move(moreRect.X+moreRect.W/2, moreRect.Y+moreRect.H/2)
		tt.Frame()
		if !tt.HasText("Sub Item Alpha") {
			t.Fatalf("submenu did not open on hover; texts=%q", tt.Texts())
		}

		// Escape closes submenu and subsequent frames do not reopen it.
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		tt.Frame()
		if tt.HasText("Sub Item Alpha") {
			t.Fatalf("submenu reopened after Escape; texts=%q", tt.Texts())
		}
	})
}
