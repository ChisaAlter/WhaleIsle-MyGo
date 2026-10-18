package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/shortcuts"
)

func TestDSK10ReboundCommandsKeepInputOwnership(t *testing.T) {
	calls, localS := 0, 0
	mode := "page"
	text := ""
	modal := false
	model := Model{PageShortcutBindings: map[string]string{"settings.open": "Mod+Shift+T", "tools.toggle": ""}, OnOpenSettings: func() { calls++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		registerPageCommands(c, model)
		if c.PageShortcut(ui.Cmd, ui.KeyS) {
			localS++
		}
		switch mode {
		case "page":
			ui.Button(c, "Page").Focus()
		case "text":
			ui.TextInput(c, &text).Label("Editor").Focus()
		case "custom":
			ui.Column(c).Focusable().Focus().HandleInput(func(ev ui.InputEvent) bool { return true })
		}
		ui.DialogBase(c, &modal, func(backdrop, panel ui.Element) { panel.Children(func() { ui.Button(c, "Modal").Focus() }) })
	}, 600, 400)
	tt.Key(ui.Cmd, ui.KeyComma)
	if calls != 0 {
		t.Fatal("old binding still active")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if calls != 1 {
		t.Fatal("new binding not dispatched")
	}
	tt.Key(ui.Cmd, ui.KeyS)
	if localS != 1 {
		t.Fatal("existing save scope changed")
	}
	for _, owner := range []string{"text", "custom"} {
		mode = owner
		tt.Frame()
		tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
		if calls != 1 {
			t.Fatalf("%s input stolen", owner)
		}
	}
	mode = "text"
	tt.Frame()
	tt.Compose("ni", 2)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
	tt.Type("你")
	if calls != 1 || text != "你" {
		t.Fatal("IME stolen")
	}
	mode = "page"
	modal = true
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if calls != 1 {
		t.Fatal("modal stolen")
	}
	modal = false
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if calls != 2 {
		t.Fatal("post-modal command not recovered")
	}
	model.PageShortcutBindings["settings.open"] = ""
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if calls != 2 {
		t.Fatal("unbound key still active")
	}
	for _, command := range shortcuts.Commands() {
		chord, err := shortcuts.Parse(command.Default)
		if err != nil {
			t.Fatal(err)
		}
		if chord.Key == "" {
			continue
		}
		mods, key := nativePageChord(chord)
		if mods&ui.Cmd == 0 || key == ui.KeyUnknown {
			t.Fatal("default cannot map to native key")
		}
	}
}
