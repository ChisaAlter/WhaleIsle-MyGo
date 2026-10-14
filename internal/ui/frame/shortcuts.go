package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
)

// registerPageCommands reuses existing button actions. The command bindings
// follow the Electron feature owners, not the older documentation summary.
// PageShortcut deliberately leaves text/IME, terminal and guest-page input alone.
func registerPageCommands(c *ui.Context, m Model) {
	var toggleSidebar func()
	if m.Sizes != nil {
		toggleSidebar = m.Sizes.ToggleSidebar
	}
	create := m.OnCreateSession
	if create == nil {
		create = func() { m.Store.CreateSession(context.Background()) }
	}
	commands := []struct {
		mods ui.Modifiers
		key  ui.Key
		run  func()
	}{
		{ui.Cmd, ui.KeyB, toggleSidebar},
		{ui.Cmd, ui.KeyN, create},
		{ui.Cmd, ui.KeyO, m.OnBrowseWorkspace},
		{ui.Cmd, ui.KeyComma, m.OnOpenSettings},
		{ui.Cmd, ui.KeyBackslash, m.OnToggleTools},
		{ui.Cmd | ui.Alt, ui.KeyB, m.OnToggleTools},
		{ui.Cmd, ui.KeyBackquote, m.OnToggleTerminal},
	}
	for _, command := range commands {
		if command.run != nil && c.PageShortcut(command.mods, command.key) {
			command.run()
		}
	}
}
