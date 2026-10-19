package main

import (
	"encoding/json"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/shortcuts"
)

type desktopApplicationMenu struct {
	signature string
	pending   []string
}

// Menu gestures execute in the next native frame, using the same live actions
// as page shortcuts. Labels show accepted bindings; no second accelerator
// registry steals keys from native input, IME, terminals or dialogs.
func (s *mainState) syncApplicationMenu(c *ui.Context) {
	if s.applicationMenu == nil {
		s.applicationMenu = &desktopApplicationMenu{}
	}
	state := s.applicationMenu
	bindings := shortcuts.Resolve(s.pageShortcutBindings())
	encoded, _ := json.Marshal(bindings)
	signature := string(encoded) + resumeText("zh", "en")
	if state.signature != signature {
		state.signature = signature
		s.syncWebApplicationShortcuts(bindings)
		mygo.App.SetMenu(buildApplicationMenu(bindings, runtime.GOOS, func(id string) {
			state.pending = append(state.pending, id)
			s.showMainWindow()
			s.mu.Lock()
			w := s.win
			s.mu.Unlock()
			if w != nil {
				w.Update(func() {})
			}
		}))
	}
	pending := state.pending
	state.pending = nil
	actions := s.applicationMenuActions(c)
	for _, id := range pending {
		if run := actions[id]; run != nil {
			run()
		}
	}
}

func (s *mainState) applicationMenuActions(c *ui.Context) map[string]func() {
	actions := s.applicationCommandActions(c)
	settings := func(section string) {
		s.selectMainPanel("")
		s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: section})
	}
	actions["workspace.add"] = func() {
		s.selectMainPanel("")
		s.mu.Lock()
		w, sessions := s.win, s.sessions
		s.settingsOpen = false
		s.launcherOpen = false
		s.mu.Unlock()
		s.openDirectoryPicker(sessions, w, "workspace")
	}
	actions["workspace.openLocal"] = func() {
		s.mu.Lock()
		sessions, cfg := s.sessions, s.cfgStore
		s.mu.Unlock()
		root := ""
		if sessions != nil {
			root = strings.TrimSpace(sessions.WorkspaceCwd())
		}
		if root == "" && cfg != nil {
			root = strings.TrimSpace(cfg.Get().Workspace)
		}
		if root != "" {
			_ = desktopcomponents.OpenInFileManager(root)
		}
	}
	actions["launcher.open"] = func() { s.openDesktopTrayPage(deeplink.ActionLauncher) }
	actions["settings.open"] = func() { settings("") }
	actions["market.open"] = func() { s.openDesktopTrayPage(deeplink.ActionMarket) }
	actions["settings.mcp"] = func() { settings("mcp") }
	actions["settings.skills"] = func() { settings("skills") }
	actions["settings.about"] = func() { settings("about") }
	actions["app.quit"] = s.quitDesktopTray
	actions["window.close"] = func() {
		s.mu.Lock()
		w := s.win
		s.mu.Unlock()
		if w != nil {
			w.Close()
		}
	}
	actions["harness.restart"] = func() {
		s.mu.Lock()
		sup, quitting := s.sup, s.quitting
		s.mu.Unlock()
		if sup != nil && !quitting {
			s.openDesktopTrayPage(deeplink.ActionLauncher)
			go s.restartUnlessImport(sup)
		}
	}
	// Native UI has no web document to reload. Rebuild its current frame while
	// preserving drafts and Host state, rather than invoking a no-op page role.
	actions["window.refresh"] = func() { c.Invalidate() }
	actions["help.web"] = func() { _ = mygo.Shell.OpenExternal("https://deepseek-harness.github.io/deepseek-harness/guide/") }
	actions["help.sdk"] = func() {
		_ = mygo.Shell.OpenExternal("https://deepseek-harness.github.io/deepseek-harness/guide/python-sdk")
	}
	return actions
}

func buildApplicationMenu(bindings map[string]shortcuts.Chord, platform string, dispatch func(string)) *mygo.Menu {
	item := func(id, zh, en string) *mygo.MenuItem {
		label := resumeText(zh, en)
		if chord, ok := bindings[id]; ok {
			label += "    " + chord.String()
		}
		return &mygo.MenuItem{ID: id, Label: label, Click: func(*mygo.MenuItem, *mygo.Window) {
			if dispatch != nil {
				dispatch(id)
			}
		}}
	}
	group := func(id, zh, en string, items ...*mygo.MenuItem) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: resumeText(zh, en), Submenu: items}
	}
	quit := item("app.quit", "退出", "Quit")
	last := quit
	if platform == "darwin" {
		last = item("window.close", "关闭窗口", "Close window")
	}
	menu := []*mygo.MenuItem{}
	if platform == "darwin" {
		menu = append(menu, group("app", "鲸屿", "Whale Isle", item("settings.about", "关于", "About"), mygo.Separator(), quit))
	}
	menu = append(menu,
		group("file", "文件", "File", item("workspace.add", "打开工作区…", "Open workspace…"), item("launcher.open", "打开启动器", "Open launcher"), item("workspace.openLocal", "在文件管理器打开工作区", "Reveal workspace"), mygo.Separator(), item("settings.open", "设置…", "Settings…"), item("market.open", "插件市场…", "Marketplace…"), item("settings.mcp", "MCP…", "MCP…"), item("settings.skills", "技能…", "Skills…"), mygo.Separator(), last),
		&mygo.MenuItem{ID: "edit", Role: mygo.RoleEditMenu},
		group("run", "运行", "Run", item("harness.restart", "重启 Harness", "Restart Harness"), item("window.refresh", "重新绘制界面", "Refresh interface")),
		group("view", "视图", "View", item("window.actualSize", "实际大小", "Actual size"), item("window.zoomIn", "放大", "Zoom in"), item("window.zoomOut", "缩小", "Zoom out"), mygo.Separator(), item("window.fullscreen", "全屏", "Fullscreen"), &mygo.MenuItem{ID: "window.inspector", Label: resumeText("开发者工具", "Developer tools"), Role: mygo.RoleToggleDevTools}, item("shortcuts.open", "快捷键…", "Keyboard shortcuts…")),
		group("help", "帮助", "Help", item("help.web", "Web UI 指南", "Web UI guide"), item("help.sdk", "Python SDK", "Python SDK"), mygo.Separator(), item("settings.about", "关于", "About")),
	)
	return mygo.NewMenu(menu)
}
