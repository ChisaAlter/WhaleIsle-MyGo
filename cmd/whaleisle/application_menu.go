package main

import (
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
	snapshot := prefs.Accepted()
	hasRevision := snapshot.Status == "ready"
	bindings := s.pageShortcutChords()
	encoded := s.pageShortcutEncoded()
	revision := snapshot.Revision
	signature := encoded + resumeText("zh", "en") + revision
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
	revision := s.pageShortcutPreferences().Accepted().Revision
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
	item := func(id, label string) *mygo.MenuItem {
		if chord := spec.keycap(id); chord != "" {
			label += applicationMenuKeycapGap + chord
		}
		return &mygo.MenuItem{ID: id, Label: label, Click: func(*mygo.MenuItem, *mygo.Window) {
			if spec.Dispatch != nil {
				spec.Dispatch(id)
			}
		}}
	}
	plain := func(id, label string) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: label, Click: func(*mygo.MenuItem, *mygo.Window) {
			if spec.Dispatch != nil {
				spec.Dispatch(id)
			}
		}}
	}
	group := func(id, label string, items ...*mygo.MenuItem) *mygo.MenuItem {
		return &mygo.MenuItem{ID: id, Label: label, Submenu: items}
	}
	quit := plain("app.quit", i18n.T(i18n.KeyCmdQuit))
	last := quit
	if spec.Platform == "darwin" {
		last = plain("window.close", i18n.T(i18n.KeyCmdCloseWindow))
	}
	menu := []*mygo.MenuItem{}
	if spec.Platform == "darwin" {
		// The native macOS app menu (menu.js:23-32) names the product, then About
		// and Quit. MyGo has no native About panel: its About page is the command.
		menu = append(menu, group("app", i18n.T(i18n.KeyAppName),
			plain("settings.about", i18n.T(i18n.KeyCmdAbout)), mygo.Separator(), quit))
	}
	menu = append(menu,
		group("file", i18n.T(i18n.KeyCmdFile),
			item("workspace.add", i18n.T(i18n.KeyCmdOpenWorkspace)),
			plain("launcher.open", i18n.T(i18n.KeyCmdOpenLauncher)),
			plain("workspace.reveal", i18n.T(i18n.KeyCmdOpenWorkspaceInExplorer)),
			mygo.Separator(),
			item("settings.open", i18n.T(i18n.KeyCmdSettings)),
			item("market.open", i18n.T(i18n.KeyCmdMarketplace)),
			plain("settings.mcp", "MCP…"),
			plain("settings.skills", i18n.T(i18n.KeyCmdSkills)),
			mygo.Separator(),
			last),
		group("run", i18n.T(i18n.KeyCmdRun),
			item("harness.restart", i18n.T(i18n.KeyCmdRestartHarness)),
			item("window.refresh", i18n.T(i18n.KeyCmdReloadInterface))),
		group("view", i18n.T(i18n.KeyCmdView),
			item("window.actualSize", i18n.T(i18n.KeyCmdActualSize)),
			item("window.zoomIn", i18n.T(i18n.KeyCmdZoomIn)),
			item("window.zoomOut", i18n.T(i18n.KeyCmdZoomOut)),
			mygo.Separator(),
			item("window.fullscreen", i18n.T(i18n.KeyCmdFullscreen)),
			&mygo.MenuItem{ID: "window.inspector", Label: i18n.T(i18n.KeyCmdDeveloperTools), Role: mygo.RoleToggleDevTools, SuppressAccelerator: true}),
		group("help", i18n.T(i18n.KeyCmdHelp),
			plain("help.web", i18n.T(i18n.KeyCmdWebUiGuide)),
			plain("help.sdk", "Python SDK"),
			mygo.Separator(),
			plain("settings.about", i18n.T(i18n.KeyCmdAbout))),
	)
	return mygo.NewMenu(menu)
}
