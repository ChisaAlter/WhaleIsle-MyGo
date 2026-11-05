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
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyComma, "Comma")
	if calls != 0 {
		t.Fatal("old binding still active")
	}
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
	if calls != 1 {
		t.Fatal("new binding not dispatched")
	}
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyS, "KeyS")
	if localS != 1 {
		t.Fatal("existing save scope changed")
	}
	for index, owner := range []string{"text", "custom"} {
		mode = owner
		tt.Frame()
		pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
		if calls != index+2 {
			t.Fatalf("%s native-priority command did not dispatch", owner)
		}
	}
	mode = "text"
	tt.Frame()
	tt.Compose("ni", 2)
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
	tt.Type("你")
	if calls != 3 || text != "你" {
		t.Fatal("IME stolen")
	}
	mode = "page"
	modal = true
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
	if calls != 3 {
		t.Fatal("modal stolen")
	}
	modal = false
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
	if calls != 4 {
		t.Fatal("post-modal command not recovered")
	}
	model.PageShortcutBindings["settings.open"] = ""
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyT, "KeyT")
	if calls != 4 {
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
		binding := chord.PhysicalBinding()
		if binding == nil || binding.Code == "" || len(binding.Modifiers) == 0 {
			t.Fatal("default cannot map to a physical native binding", command.ID, binding)
		}
	}
}
