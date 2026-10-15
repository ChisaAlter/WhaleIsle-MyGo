package main

import "github.com/egoist/mygo"

type desktopTrayActions struct {
	open, launcher, settings, market, restart, quit func()
	pet                                             func(bool)
	canRestart, canTogglePet, petEnabled            bool
}

// Actions are real application routes supplied by desktopTrayMenu. Keeping the
// template pure makes it testable without creating a native tray or a Host.
func buildDesktopTrayMenu(a desktopTrayActions) *mygo.Menu {
	row := func(id, zh, en string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: resumeText(zh, en), Disabled: fn == nil, Click: func(*mygo.MenuItem, *mygo.Window) {
			if fn != nil {
				fn()
			}
		}}
	}
	restart := row("restart", "重启 Harness", "Restart Harness", a.restart)
	restart.Disabled = restart.Disabled || !a.canRestart
	pet := &mygo.MenuItem{ID: "pet", Label: resumeText("桌面宠物", "Desktop pet"), Type: mygo.MenuItemCheckbox, Checked: a.petEnabled, Disabled: a.pet == nil || !a.canTogglePet,
		Click: func(item *mygo.MenuItem, _ *mygo.Window) {
			if a.pet != nil && a.canTogglePet {
				a.pet(item.Checked)
			}
		}}
	return mygo.NewMenu([]*mygo.MenuItem{
		row("open", "显示窗口", "Show window", a.open),
		row("launcher", "打开启动器", "Open launcher", a.launcher),
		row("settings", "设置…", "Settings…", a.settings),
		row("market", "插件市场", "Plugin marketplace", a.market), restart, pet, mygo.Separator(),
		row("quit", "退出", "Quit", a.quit),
	})
}
