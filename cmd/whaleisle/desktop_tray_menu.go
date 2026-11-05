package main

import (
	"github.com/egoist/mygo"

	"whaleisle/internal/ui/i18n"
)

type desktopTrayActions struct {
	open, launcher, settings, market, restart, quit, update func()
	updateLabel                                             string
	pet                                                     func(bool)
	canRestart, canTogglePet, petEnabled                    bool
}

// Actions are real application routes supplied by desktopTrayMenu. Keeping the
// template pure makes it testable without creating a native tray or a Host.
func buildDesktopTrayMenu(a desktopTrayActions) *mygo.Menu {
	row := func(id, label string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: label, Disabled: fn == nil, Click: func(*mygo.MenuItem, *mygo.Window) {
			if fn != nil {
				fn()
			}
		}}
	}
	restart := row("restart", i18n.T(i18n.KeyCmdRestartHarness), a.restart)
	restart.Disabled = restart.Disabled || !a.canRestart
	pet := &mygo.MenuItem{ID: "pet", Label: i18n.T(i18n.KeyCmdDesktopPet), Type: mygo.MenuItemCheckbox, Checked: a.petEnabled, Disabled: a.pet == nil || !a.canTogglePet,
		Click: func(item *mygo.MenuItem, _ *mygo.Window) {
			if a.pet != nil && a.canTogglePet {
				a.pet(item.Checked)
			}
		}}
	items := []*mygo.MenuItem{
		row("open", i18n.T(i18n.KeyCmdShowWindow), a.open),
		row("launcher", i18n.T(i18n.KeyCmdOpenLauncher), a.launcher),
		row("settings", i18n.T(i18n.KeyCmdSettings), a.settings),
		row("market", i18n.T(i18n.KeyCmdPluginMarketplace), a.market), restart, pet, mygo.Separator(),
	}
	if a.update != nil {
		items = append(items, row("update", a.updateLabel, a.update), mygo.Separator())
	}
	items = append(items, row("quit", i18n.T(i18n.KeyCmdQuit), a.quit))
	return mygo.NewMenu(items)
}
