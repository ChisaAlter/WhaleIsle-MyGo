package frame

import (
	"github.com/egoist/mygo/ui"
	"runtime"
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
		code string
	}{
		{ui.Cmd, ui.KeyComma, "Comma"}, {ui.Cmd, ui.KeyO, "KeyO"}, {ui.Cmd, ui.KeyBackslash, "Backslash"},
		{ui.Cmd | ui.Alt, ui.KeyB, "KeyB"}, {ui.Cmd, ui.KeyBackquote, "Backquote"},
	} {
		pressFramePhysicalKey(tt, chord.mods, chord.key, chord.code)
	}
	wantTerminals := 1
	if runtime.GOOS == "darwin" {
		wantTerminals = 0 // macOS leaves its system window-switching chord unbound by default.
	}
	if settings != 1 || folders != 1 || panels != 2 || terminals != wantTerminals {
		t.Fatalf("dispatch: %d %d %d %d", settings, folders, panels, terminals)
	}
	pressFramePhysicalKey(tt, 0, ui.KeyO, "KeyO")
	pressFramePhysicalKey(tt, ui.Cmd|ui.Shift, ui.KeyComma, "Comma")
	if folders != 1 || settings != 1 {
		t.Fatal("unbound chord was captured")
	}
	mode = "editor"
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyComma, "Comma")
	tt.Compose("ni", 2)
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyO, "KeyO")
	tt.Type("你")
	if settings != 2 || folders != 1 || text != "你" {
		t.Fatalf("editor/IME: settings=%d folders=%d text=%q", settings, folders, text)
	}
	mode = "custom"
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyBackquote, "Backquote")
	if terminals != wantTerminals+1 || customKeys != 0 {
		t.Fatalf("custom input stolen: terminal=%d keys=%d", terminals, customKeys)
	}
	mode = "page"
	modal = true
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyComma, "Comma")
	if settings != 2 {
		t.Fatal("modal command escaped")
	}
	modal = false
	tt.Frame()
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyComma, "Comma")
	if settings != 3 {
		t.Fatal("page command did not recover after modal")
	}
}

// Production commands consume physical surface codes. Logical-only Tester.Key
// deliberately supplies no code and remains appropriate for native edit commands.
func pressFramePhysicalKey(tt *ui.Tester, mods ui.Modifiers, key ui.Key, code string) {
	tt.PhysicalKeyDown(code, key, mods, false, false, false)
	tt.PhysicalKeyUp(code, key, mods, false)
}
