package main

import (
	"encoding/json"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

type desktopApplicationMenu struct {
	signature string
	pending   []desktopMenuGesture
}
type desktopMenuGesture struct{ id, revision string }

// applicationMenuKeycapGap separates a label from its keycap. MyGo shows the
// accepted chord in the label instead of installing a second accelerator owner
// (mygo.MenuItem.SuppressAccelerator documents this pattern), so the keycap
// always names the chord that actually fires.
const applicationMenuKeycapGap = "    "

// applicationMenuSpec is what the application menu needs from the shell.
type applicationMenuSpec struct {
	// Bindings are the accepted page-shortcut chords.
	Bindings map[string]shortcuts.Chord
	// HasRevision reports that a configuration snapshot was accepted. From then
	// on an unbound command shows no keycap; before the first snapshot the
	// shipped default keeps the menu readable, exactly as the native
	// commandItem does (reference/electron/src/main/menu.js:9-19).
	HasRevision bool
	// Fallbacks are the shipped defaults, used only before the first snapshot.
	Fallbacks map[string]shortcuts.Chord
	Platform  string
	Dispatch  func(string)
}

// keycap is the native commandItem resolution: the accepted binding once a
// revision exists (none when unbound), the shipped default before it.
func (spec applicationMenuSpec) keycap(id string) string {
	if spec.HasRevision {
		if chord, ok := spec.Bindings[id]; ok {
			return chord.String()
		}
		return ""
	}
	if chord, ok := spec.Fallbacks[id]; ok {
		return chord.String()
	}
	return ""
}

// Menu gestures execute in the next native frame, using the same live actions
// as page shortcuts. Labels show accepted bindings; no second accelerator
// registry steals keys from native input, IME, terminals or dialogs.
func (s *mainState) syncApplicationMenu(c *ui.Context) {
	if s.applicationMenu == nil {
		s.applicationMenu = &desktopApplicationMenu{}
	}
	state := s.applicationMenu
	prefs := s.pageShortcutPreferences()
	// "ready" is the accepted configuration revision; before it the menu shows
	// the shipped defaults instead of blank keycaps.
	hasRevision := prefs.Get().Status == "ready"
	bindings := shortcuts.Resolve(s.pageShortcutBindings())
	encoded, _ := json.Marshal(bindings)
	revision := prefs.Get().Revision
	signature := string(encoded) + resumeText("zh", "en") + revision
	if hasRevision {
		signature += "|ready"
	}
	if state.signature != signature {
		state.signature = signature
		s.syncWebApplicationShortcuts(bindings)
		mygo.App.SetMenu(buildApplicationMenu(applicationMenuSpec{
			Bindings:    bindings,
			HasRevision: hasRevision,
			Fallbacks:   shortcuts.Resolve(shortcuts.SharedDefaults()),
			Platform:    runtime.GOOS,
			Dispatch: func(id string) {
				state.pending = append(state.pending, desktopMenuGesture{id, revision})
				s.mu.Lock()
				w := s.win
				s.mu.Unlock()
				if w != nil {
					w.Update(func() {})
				}
			},
		}))
	}
}

// Process after the frame has built its overlays, so a newly opened modal
// retires a menu gesture before any application action can run.
func (s *mainState) processApplicationMenu(c *ui.Context) {
	state := s.applicationMenu
	if state == nil {
		return
	}
	revision := s.pageShortcutPreferences().Get().Revision
	pending := state.pending
	state.pending = nil
	actions := s.applicationMenuActions(c)
	for _, gesture := range pending {
		if gesture.revision != revision || !c.MenuCommandsAllowed() || s.shortcutEditor != nil && (s.shortcutEditor.Recording != "" || s.shortcutEditor.ResetPending) {
			continue
		}
		if run := actions[gesture.id]; run != nil {
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
	// The File menu's third row, native menu.js:44-52: reveal the configured
	// workspace, and do nothing when none is configured.
	actions["workspace.reveal"] = func() {
		s.mu.Lock()
		cfgStore := s.cfgStore
		s.mu.Unlock()
		if cfgStore == nil {
			return
		}
		if root := strings.TrimSpace(cfgStore.Get().Workspace); root != "" {
			_ = components.OpenInFileManager(root)
		}
	}
	actions["workspace.openLocal"] = func() {
		s.mu.Lock()
		sessions := s.sessions
		blocked := s.marketOpen || s.scheduleOpen || s.assistantOpen || s.launcherOpen || s.settingsOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen
		s.mu.Unlock()
		if !blocked {
			s.observeWorkspaceOpenInApp().OpenCurrent(sessions)
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

// buildApplicationMenu is the item-by-item port of the native template
// (reference/electron/src/main/menu.js:22-121): File / Run / View / Help on every
// platform, plus the macOS application menu. No item without a native
// counterpart is added, and every row keeps the native order and separators.
func buildApplicationMenu(spec applicationMenuSpec) *mygo.Menu {
	// item carries the command keycap; plain is a row the native template gives
	// no accelerator to (the launcher, reveal, MCP, skills, help links, quit).
	item := func(id, zh, en string) *mygo.MenuItem {
		label := resumeText(zh, en)
		if chord := spec.keycap(id); chord != "" {
			label += applicationMenuKeycapGap + chord
		}
		return &mygo.MenuItem{ID: id, Label: label, Click: func(*mygo.MenuItem, *mygo.Window) {
			if spec.Dispatch != nil {
				spec.Dispatch(id)
			}
		}}
	}
	plain := func(id, zh, en string) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: resumeText(zh, en), Click: func(*mygo.MenuItem, *mygo.Window) {
			if spec.Dispatch != nil {
				spec.Dispatch(id)
			}
		}}
	}
	group := func(id, zh, en string, items ...*mygo.MenuItem) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: resumeText(zh, en), Submenu: items}
	}
	quit := plain("app.quit", "退出", "Quit")
	last := quit
	if spec.Platform == "darwin" {
		last = plain("window.close", "关闭窗口", "Close window")
	}
	menu := []*mygo.MenuItem{}
	if spec.Platform == "darwin" {
		// The native macOS app menu (menu.js:23-32) names the product, then About
		// and Quit. MyGo has no native About panel: its About page is the command.
		menu = append(menu, group("app", i18n.T(i18n.KeyAppName), i18n.T(i18n.KeyAppName),
			plain("settings.about", "关于", "About"), mygo.Separator(), quit))
	}
	menu = append(menu,
		group("file", "文件", "File",
			item("workspace.add", "打开工作区…", "Open workspace…"),
			plain("launcher.open", "打开启动器", "Open launcher"),
			plain("workspace.reveal", "在资源管理器中打开工作区", "Open workspace in explorer"),
			mygo.Separator(),
			item("settings.open", "设置…", "Settings…"),
			item("market.open", "插件市场…", "Marketplace…"),
			plain("settings.mcp", "MCP…", "MCP…"),
			plain("settings.skills", "技能…", "Skills…"),
			mygo.Separator(),
			last),
		group("run", "运行", "Run",
			item("harness.restart", "重启 Harness", "Restart Harness"),
			item("window.refresh", "重新加载界面", "Reload interface")),
		group("view", "视图", "View",
			item("window.actualSize", "实际大小", "Actual size"),
			item("window.zoomIn", "放大", "Zoom in"),
			item("window.zoomOut", "缩小", "Zoom out"),
			mygo.Separator(),
			item("window.fullscreen", "全屏", "Fullscreen"),
			&mygo.MenuItem{ID: "window.inspector", Label: resumeText("开发者工具", "Developer tools"), Role: mygo.RoleToggleDevTools, SuppressAccelerator: true}),
		group("help", "帮助", "Help",
			plain("help.web", "Web UI 指南", "Web UI guide"),
			plain("help.sdk", "Python SDK", "Python SDK"),
			mygo.Separator(),
			plain("settings.about", "关于", "About")),
	)
	return mygo.NewMenu(menu)
}
