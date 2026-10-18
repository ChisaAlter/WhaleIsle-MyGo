package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestPageCommandsRespectInputOwnership(t *testing.T) {
	settings, folders, panels, terminals, customKeys := 0, 0, 0, 0, 0
	text, mode := "", "page"
	modal := false
	model := Model{Store: session.New(),
		OnOpenSettings: func() { settings++ }, OnBrowseWorkspace: func() { folders++ },
		OnToggleTools: func() { panels++ }, OnToggleTerminal: func() { terminals++ },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		registerPageCommands(c, model)
		switch mode {
		case "page":
			ui.Button(c, "Page").Focus()
		case "editor":
			ui.TextInput(c, &text).Label("Editor").Focus()
		case "custom":
			ui.Column(c).Focusable().Focus().HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputKeyDown {
					customKeys++
					return true
				}
				return false
			})
		}
		ui.DialogBase(c, &modal, func(backdrop, panel ui.Element) { panel.Children(func() { ui.Button(c, "Dialog").Focus() }) })
	}, 600, 400)
	for _, chord := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{
		{ui.Cmd, ui.KeyComma}, {ui.Cmd, ui.KeyO}, {ui.Cmd, ui.KeyBackslash},
		{ui.Cmd | ui.Alt, ui.KeyB}, {ui.Cmd, ui.KeyBackquote},
	} {
		tt.Key(chord.mods, chord.key)
	}
	if settings != 1 || folders != 1 || panels != 2 || terminals != 1 {
		t.Fatalf("dispatch: %d %d %d %d", settings, folders, panels, terminals)
	}
	tt.Key(0, ui.KeyO)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyComma)
	if folders != 1 || settings != 1 {
		t.Fatal("unbound chord was captured")
	}
	mode = "editor"
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyComma)
	tt.Compose("ni", 2)
	tt.Key(ui.Cmd, ui.KeyO)
	tt.Type("你")
	if settings != 1 || folders != 1 || text != "你" {
		t.Fatalf("editor/IME: settings=%d folders=%d text=%q", settings, folders, text)
	}
	mode = "custom"
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyBackquote)
	if terminals != 1 || customKeys != 1 {
		t.Fatalf("custom input stolen: terminal=%d keys=%d", terminals, customKeys)
	}
	mode = "page"
	modal = true
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyComma)
	if settings != 1 {
		t.Fatal("modal command escaped")
	}
	modal = false
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyComma)
	if settings != 2 {
		t.Fatal("page command did not recover after modal")
	}
}
