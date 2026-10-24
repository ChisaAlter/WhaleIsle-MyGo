package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/importer"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/launcher"
	"whaleisle/internal/ui/theme"
)

// Launcher window (G04 phase 1): port of reference/electron/src/main/window.js
// createLauncherWindow/showLauncher (:900-952). The launcher is its own native
// window; the 1440x920 main window stays hidden until the desktop start
// succeeds, exactly like the reference shell, which creates the main window
// only when the harness starts.
//
// The in-window launcher panel stays as the fallback: isolated/QA desktops,
// --background, --settings, headless tests (no native window) and a launcher
// window that could not be created all keep the old single-window panel.
const (
	launcherWindowWidth     = 1060
	launcherWindowHeight    = 660
	launcherWindowMinWidth  = 860
	launcherWindowMinHeight = 560
	// launcher.css:21 .caption is a 48 DIP drag strip shared with the desktop
	// chrome, so the drag band and the 32x32 window buttons line up.
	launcherCaptionHeight = 48
)

// launcherFirstWindow reports whether a cold start hands the visible window to
// the launcher and keeps the main window hidden until the desktop start. The
// reference shell only ever shows the launcher at this point (window.js:900-952
// plus index.js:1364 prepareLauncher); isolated/QA desktops keep driving the
// in-window panel, and --background / --settings / login starts keep their own
// main-window visibility.
func launcherFirstWindow(isolated, hidden, background, settings bool) bool {
	return !isolated && !hidden && !background && !settings
}

// launcherWindowOptions mirrors createLauncherWindow's BrowserWindow options
// (window.js:904-921): 1060x660 with an 860x560 minimum, created hidden,
// transparent, with the integrated 48 DIP caption plate (12 top / 8 right,
// 32x32 buttons, radius 8) and no OS menu bar.
func launcherWindowOptions(cfg config.Config) mygo.WindowOptions {
	return mygo.WindowOptions{
		Title:                    i18n.T(i18n.KeyAppName),
		Width:                    launcherWindowWidth,
		Height:                   launcherWindowHeight,
		MinWidth:                 launcherWindowMinWidth,
		MinHeight:                launcherWindowMinHeight,
		StateKey:                 "launcher",
		Hidden:                   true,
		TitleBarStyle:            mygo.TitleBarHidden,
		TitleBarHeight:           launcherCaptionHeight,
		AutoHideMenuBar:          true,
		WindowsCaptionButtons:    &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 12, Right: 8, CornerRadius: 8, GlyphSize: 12},
		WindowsFrameCornerRadius: 20,
		Transparent:              true,
		BackgroundColor:          theme.Resolve(cfg.Theme).BackgroundColor(),
	}
}

// launcherWindow is the UI-thread-owned launcher window handle.
type launcherWindow struct {
	win *mygo.Window
}

func (lw *launcherWindow) destroyed() bool {
	return lw == nil || lw.win == nil || lw.win.IsDestroyed()
}

// show restores, shows and focuses the window in that order, like
// showLauncher (window.js:944-951), so a minimized launcher comes back.
func (lw *launcherWindow) show() {
	if lw.destroyed() {
		return
	}
	if lw.win.IsMinimized() {
		lw.win.Restore()
	}
	lw.win.Show()
	lw.win.Focus()
}

// launcherPanelHost is the startup snapshot the launcher panel paints from.
// mainView passes its own render snapshot; the launcher window reads the same
// values from the shell state, so both surfaces share one model builder.
type launcherPanelHost struct {
	ThemeID    string
	Version    string
	Status     harness.Status
	Logs       []string
	Workspace  string
	Supervisor *harness.Supervisor
	Sessions   bool
}

// launcherPanelHostSnapshot reads the launcher host state for the window
// surface. The workspace comes from the config (a cold start has no selected
// session yet); mainView keeps using its session-aware root.
func (s *mainState) launcherPanelHostSnapshot() launcherPanelHost {
	s.mu.Lock()
	host := launcherPanelHost{
		ThemeID:    s.themeID,
		Version:    buildVersion,
		Status:     s.bootStatus,
		Logs:       append([]string(nil), s.bootLogs...),
		Supervisor: s.sup,
		Sessions:   s.sessions != nil,
	}
	cfgStore := s.cfgStore
	s.mu.Unlock()
	if cfgStore != nil {
		host.Workspace = cfgStore.Get().Workspace
	}
	return host
}

// launcherStage is the window side of the launcher state machine. The shell
// implements it with the real windows; the tests use a fake so one launcher
// maximum, the start hand-off, cancel/error retention and the nil-window
// fallback are asserted without a display.
type launcherStage interface {
	// launcherOpen reports whether one launcher window is currently open.
	launcherOpen() bool
	// showLauncher creates or reuses the single launcher window and shows it.
	// False means this run has no launcher window, so the in-window panel needs
	// the main window.
	showLauncher() bool
	closeLauncher()
	showMain()
	mainVisible() bool
	quitting() bool
}

type launcherStageResult struct {
	LauncherShown  bool
	LauncherClosed bool
	MainShown      bool
}

// applyLauncherStage moves the native windows to match the launcherOpen stage
// flag. Opening reuses an existing launcher window (window.js:901-903 returns
// launcherWindow before creating one); a run without a launcher window falls
// back to the in-window panel on the main window. Closing the stage closes the
// launcher window and reveals the main window.
func applyLauncherStage(stage launcherStage, open bool) launcherStageResult {
	var res launcherStageResult
	if open {
		if stage.launcherOpen() {
			return res
		}
		if stage.showLauncher() {
			res.LauncherShown = true
			return res
		}
		if !stage.quitting() && !stage.mainVisible() {
			stage.showMain()
			res.MainShown = true
		}
		return res
	}
	if stage.launcherOpen() {
		stage.closeLauncher()
		res.LauncherClosed = true
	}
	if !stage.quitting() && !stage.mainVisible() {
		stage.showMain()
		res.MainShown = true
	}
	return res
}

type shellLauncherStage struct{ s *mainState }

func (st shellLauncherStage) launcherOpen() bool {
	st.s.mu.Lock()
	lw := st.s.launcherWin
	st.s.mu.Unlock()
	return !lw.destroyed()
}

func (st shellLauncherStage) showLauncher() bool {
	st.s.mu.Lock()
	allowed := st.s.launcherWindowAllowed
	st.s.mu.Unlock()
	if !allowed {
		return false
	}
	lw := st.s.ensureLauncherWindow()
	if lw == nil {
		return false
	}
	lw.show()
	return true
}

func (st shellLauncherStage) closeLauncher() { st.s.closeLauncherWindow() }

func (st shellLauncherStage) showMain() { st.s.showMainWindow() }

func (st shellLauncherStage) mainVisible() bool {
	st.s.mu.Lock()
	win := st.s.win
	st.s.mu.Unlock()
	return win != nil && !win.IsDestroyed() && win.IsVisible()
}

func (st shellLauncherStage) quitting() bool {
	st.s.mu.Lock()
	defer st.s.mu.Unlock()
	return st.s.quitting
}

// syncLauncherStage makes the native windows follow the launcherOpen stage
// flag. It is the single entry both surfaces use after a route or boot change.
func (s *mainState) syncLauncherStage() launcherStageResult {
	s.mu.Lock()
	open := s.launcherOpen
	s.mu.Unlock()
	return applyLauncherStage(shellLauncherStage{s: s}, open)
}

// launcherStageHasWindow reports whether this run gives the launcher its own
// window, so the main window (and the welcome hand-over) stays hidden until the
// desktop start.
func (s *mainState) launcherStageHasWindow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.launcherWindowAllowed
}

// ensureLauncherWindow creates the one launcher window and returns it, or nil
// when this run has none. An existing window is reused, never duplicated.
func (s *mainState) ensureLauncherWindow() *launcherWindow {
	s.mu.Lock()
	existing := s.launcherWin
	s.mu.Unlock()
	if !existing.destroyed() {
		return existing
	}
	cfgStore := s.cfgStore
	cfg := config.Config{}
	if cfgStore != nil {
		cfg = cfgStore.Get()
	}
	lw := &launcherWindow{}
	opts := launcherWindowOptions(cfg)
	opts.Content = ui.View(func(c *ui.Context) { s.launcherWindowBody(c, lw) })
	win := mygo.NewWindow(opts)
	if win == nil || win.IsDestroyed() {
		return nil
	}
	lw.win = win
	win.OnClose(func(e *mygo.CloseEvent) { s.launcherWindowCloseRequested(e) })
	win.OnClosed(func() { s.launcherWindowClosed(lw) })
	s.mu.Lock()
	if other := s.launcherWin; !other.destroyed() {
		// A concurrent caller won; keep the single window and drop this one.
		s.mu.Unlock()
		win.Destroy()
		return other
	}
	s.launcherWin = lw
	s.mu.Unlock()
	return lw
}

// closeLauncherWindow destroys the launcher window without running the
// close-to-quit rule: only an explicit user close quits the app.
func (s *mainState) closeLauncherWindow() {
	s.mu.Lock()
	lw := s.launcherWin
	s.launcherWin = nil
	s.mu.Unlock()
	if lw == nil {
		return
	}
	win := lw.win
	lw.win = nil
	if win != nil && !win.IsDestroyed() {
		win.Destroy()
	}
}

func (s *mainState) launcherWindowClosed(lw *launcherWindow) {
	s.mu.Lock()
	if s.launcherWin == lw {
		s.launcherWin = nil
	}
	s.mu.Unlock()
}

// launcherWindowCloseRequested is bindLauncherClose (index.js:326-336): with a
// visible main window the launcher simply closes and the desktop keeps running;
// without one, closing the launcher quits the application.
func (s *mainState) launcherWindowCloseRequested(e *mygo.CloseEvent) {
	s.mu.Lock()
	quitting := s.quitting
	win, sup := s.win, s.sup
	s.mu.Unlock()
	if quitting {
		return
	}
	if win != nil && !win.IsDestroyed() && win.IsVisible() {
		return
	}
	e.PreventDefault()
	s.requestQuitProtected(win, sup)
}

// launcherWindowBody paints the launcher window: the launcher owns the whole
// surface (no main chrome), exactly like the in-window panel branch of
// mainView. The window is transparent, so the panel's own 20 radius shell is
// the silhouette.
func (s *mainState) launcherWindowBody(c *ui.Context, lw *launcherWindow) {
	ui.Column(c).Fill().Children(func() {
		s.launcherPanel(c, lw.win, s.launcherPanelHostSnapshot())
	})
}

// launcherPanel paints the one launcher model. Both surfaces call it: the
// launcher window (win = lw.win) and the in-window fallback panel
// (win = main window).
func (s *mainState) launcherPanel(c *ui.Context, win *mygo.Window, host launcherPanelHost) {
	// mainView used to snapshot these per render; both surfaces now read the
	// same state fields here, so the window and the panel never disagree.
	s.mu.Lock()
	launcherTab := s.launcherTab
	if launcherTab == "" {
		launcherTab = launcher.TabHome
	}
	launcherImport := s.launcherImport
	launcherImportJournal := append([]string(nil), s.launcherImportJournal...)
	launcherImportBusy := s.launcherImportBusy
	launcherUpdate := s.launcherUpdate
	launcherUpdateAvailable := s.launcherUpdateAvailable
	launcherNote := s.launcherNote
	launcherStartBusy := s.launcherStartBusy
	launcherRuntimeBusy := s.taskRestartPending || s.quitRequestPending || s.quitting
	launcherStartFailed := s.launcherStartFailed
	launcherStartErr := s.launcherStartErr
	s.mu.Unlock()
	// The pinned native release catalog owns the one background lookup.
	// Full-window launcher shell (Electron parity): no main chrome.
	vendorReady := false
	if info, err := os.Stat(filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")); err == nil && !info.IsDir() {
		vendorReady = true
	}
	disabled := []string(nil)
	skipUserPlugins := false
	pluginRecoveryReason := ""
	cfgStore := state.cfgStore
	harnessSup := host.Supervisor
	if cfgStore != nil {
		cfg := cfgStore.Get()
		disabled = append([]string(nil), cfg.DisabledPlugins...)
		rec := cfgStore.PluginRecoveryEvidence()
		skipUserPlugins = rec.SkipUserPlugins
		pluginRecoveryReason = rec.Reason
	}
	if strings.TrimSpace(state.launcherWorkspaceDraft) == "" {
		seed := host.Workspace
		if seed == "" && cfgStore != nil {
			seed = cfgStore.Get().Workspace
		}
		state.launcherWorkspaceDraft = seed
	}
	wsDraft := &state.launcherWorkspaceDraft
	if !state.launcherPrefsSeeded && cfgStore != nil {
		cfg := cfgStore.Get()
		state.launcherQuitAfterStart = cfg.QuitAfterStart
		state.launcherAutoStartDesktop = cfg.AutoStartDesktop
		state.launcherAskOnUpdate = cfg.AskOnUpdate
		state.launcherDownloadRoute = cfg.DownloadRoute
		state.closeToTray = cfg.CloseToTray
		state.launcherPrefsSeeded = true
	}
	state.mu.Lock()
	componentsFresh := state.launcherComponentsFresh
	state.mu.Unlock()
	if !componentsFresh {
		state.refreshLauncherComponents()
	}
	state.mu.Lock()
	compRows := append([]desktopcomponents.Row(nil), state.launcherComponents...)
	prefsError := state.launcherPrefsError
	state.mu.Unlock()
	componentActivity := state.launcherComponentActivitySnapshot()
	quitPtr := &state.launcherQuitAfterStart
	trayPtr := &state.closeToTray
	autoPtr := &state.launcherAutoStartDesktop
	askPtr := &state.launcherAskOnUpdate
	routePtr := &state.launcherDownloadRoute
	// Retained session stores keep the shell alive after Stop; they do not
	// mean that a stopped or failed runtime can still accept work.
	deskRunning := host.Status.State == harness.Ready || host.Status.State == harness.Starting && host.Sessions
	launcher.View(c, state.withNativeReleases(win, launcher.Model{
		ThemeID:         host.ThemeID,
		SystemDark:      mygo.Theme.IsDark(),
		Version:         host.Version,
		Workspace:       *wsDraft,
		WorkspacePtr:    wsDraft,
		VendorReady:     vendorReady,
		LauncherPackage: isLauncherPackage(),
		DesktopRunning:  deskRunning,
		// Keep the running chrome even while harness is still Starting.
		StartBusy:            launcherStartBusy,
		RuntimeBusy:          launcherRuntimeBusy,
		StartFailed:          launcherStartFailed,
		StartError:           launcherStartErr,
		BootLogs:             host.Logs,
		PluginLoad:           host.Status.PluginLoad,
		ActiveTab:            launcherTab,
		ImportSummary:        launcherImport,
		ImportJournal:        launcherImportJournal,
		UpdateStatus:         launcherUpdate,
		UpdateAvailable:      launcherUpdateAvailable,
		DisabledPlugins:      disabled,
		SkipUserPlugins:      skipUserPlugins,
		PluginRecoveryReason: pluginRecoveryReason,
		StatusNote:           launcherNote,
		QuitAfterStart:       quitPtr,
		CloseToTray:          trayPtr,
		AutoStartDesktop:     autoPtr,
		AskOnUpdate:          askPtr,
		DownloadRoute:        routePtr,
		RoutePopOpen:         &state.launcherRoutePopOpen,
		ImportOverwrite:      &state.launcherImportOverwrite,
		ImportAttachments:    &state.launcherImportAttachments,
		ImportCategories:     state.launcherImportCategories,
		ImportCatalog:        state.launcherImportCatalog,
		Components:           compRows,
		ComponentPending:     componentActivity.Pending,
		ComponentStatus:      componentActivity.Status,
		LauncherPrefsError:   prefsError,
		ComponentBusy:        bridgeChangeActive.Load(),
		HostRuntimeState:     host.Status.State,
		OnRefreshComponents: func() {
			state.refreshLauncherComponents()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnComponentAction: func(id, action string) {
			if bridgeChangeActive.Load() {
				return
			}
			if id == desktopcomponents.HostComponentID {
				state.changeHostBundle(win, action)
				return
			}
			if id == "whalebridge" && (action == "update" || action == "rollback" || action == "uninstall" || action == "stop") {
				state.changeWhaleBridge(win, action)
				return
			}
			if id == "whalebridge" && (action == "start" || action == "open" || action == "manage") {
				release, ok := state.claimLauncherComponentAction(win, id, action)
				if !ok {
					return
				}
				state.mu.Lock()
				state.launcherUpdate = "正在启动鲸桥并验证就绪状态…"
				if action != "start" {
					state.launcherUpdate = resumeText("正在打开鲸桥设置…", "Opening WhaleBridge settings…")
				}
				state.launcherComponentStatus = state.launcherUpdate
				state.mu.Unlock()
				go func() {
					defer release()
					// open/manage start the instance and then own the embedded
					// management window; the shell never hands the page to the
					// system browser (reference whalebridge-window.js).
					run := action
					if action != "start" {
						run = "start"
					}
					err := desktopcomponents.RunAction(paths.UserDataDir(), id, run)
					state.mu.Lock()
					state.launcherUpdate = desktopcomponents.FormatActionError(id, action, err)
					state.launcherComponentStatus = state.launcherUpdate
					state.mu.Unlock()
					state.refreshLauncherComponents()
					if win != nil {
						win.Update(func() {})
						if err == nil && action != "start" {
							win.Update(func() { state.openWhaleBridgeWindow(win, action == "manage") })
						}
					}
				}()
				return
			}
			if id == "whalebridge" && action == "install" {
				release, ok := state.claimLauncherComponentAction(win, id, action)
				if !ok {
					return
				}
				route := state.updateDownloadRoute()
				state.mu.Lock()
				state.launcherUpdate = "正在下载并校验鲸桥组件…"
				state.launcherComponentStatus = state.launcherUpdate
				state.mu.Unlock()
				go func() {
					defer release()
					err := desktopcomponents.InstallWhaleBridge(context.Background(), paths.UserDataDir(), route)
					state.mu.Lock()
					state.launcherUpdate = desktopcomponents.FormatActionError(id, action, err)
					state.launcherComponentStatus = state.launcherUpdate
					if err == nil {
						state.launcherUpdate = "鲸桥已安装并校验，可点击启动；未自动运行"
						state.launcherComponentStatus = state.launcherUpdate
					}
					state.mu.Unlock()
					state.refreshLauncherComponents()
					if win != nil {
						win.Update(func() {})
					}
				}()
				return
			}
			state.runGenericComponentAction(win, id, action)
		},
		OnSaveLauncherPrefs: func() {
			if cfgStore == nil {
				state.refuseHostAction()
				return
			}
			state.mu.Lock()
			quit := state.launcherQuitAfterStart
			tray := state.closeToTray
			auto := state.launcherAutoStartDesktop
			ask := state.launcherAskOnUpdate
			route := state.launcherDownloadRoute
			state.mu.Unlock()
			err := cfgStore.Update(func(c *config.Config) {
				c.QuitAfterStart = quit
				c.CloseToTray = tray
				c.AutoStartDesktop = auto
				c.AskOnUpdate = ask
				c.DownloadRoute = route
			})
			state.mu.Lock()
			state.launcherPrefsError = ""
			if err != nil {
				state.launcherPrefsError = err.Error()
			}
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnTab: func(tab string) {
			state.mu.Lock()
			state.launcherTab = tab
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnClose: func() {
			state.mu.Lock()
			state.launcherOpen = false
			state.mu.Unlock()
			// Leaving the stage closes the launcher window and reveals the desktop;
			// in the panel fallback it simply repaints the main window.
			state.syncLauncherStage()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnStop: func() {
			state.mu.Lock()
			sup := state.sup
			state.mu.Unlock()
			state.stopProtected(win, sup)
		},
		OnStart: func() {
			state.mu.Lock()
			if state.launcherStartPreparing || state.launcherStartBusy || state.launcherImportBusy || state.taskRestartPending || state.quitRequestPending || state.quitting {
				state.mu.Unlock()
				return
			}
			state.launcherStartPreparing = true
			store := state.cfgStore
			ws := strings.TrimSpace(state.launcherWorkspaceDraft)
			state.launcherTriedStart = true
			state.launcherStartBusy = true
			state.launcherStartErr = ""
			state.startupRecovery.updateHold = false
			state.mu.Unlock()
			go func() {
				defer state.finishLauncherPreparation(win)
				if store == nil {
					state.mu.Lock()
					state.launcherStartBusy = false
					state.launcherStartFailed = true
					state.launcherStartErr = "config store is not ready"
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				if ws == "" {
					ws = paths.DefaultWorkspace()
				}
				if err := os.MkdirAll(ws, 0o755); err != nil {
					state.mu.Lock()
					state.launcherStartBusy = false
					state.launcherStartFailed = true
					state.launcherStartErr = err.Error()
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				if err := store.Update(func(c *config.Config) {
					c.Workspace = ws
				}); err != nil {
					state.mu.Lock()
					state.launcherStartBusy = false
					state.launcherStartFailed = true
					state.launcherStartErr = err.Error()
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				state.mu.Lock()
				state.launcherWorkspaceDraft = ws
				state.launcherOpen = true
				state.mu.Unlock()
				startHarness(store.Get())
				if win != nil {
					win.Update(func() {})
				}
			}()
		},
		OnScanImport: func() {
			state.applyImportCatalog(win, "")
		},
		OnPickImportSource: func() {
			picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:    win,
				Title:     i18n.T(i18n.KeyLauncherImportPickSourceTitle),
				Directory: true,
			})
			if err != nil || len(picked) == 0 {
				return
			}
			state.applyImportCatalog(win, picked[0])
		},
		OnAddSkillImportDir: func() {
			picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:    win,
				Title:     i18n.T(i18n.KeyLauncherImportAddSkillTitle),
				Directory: true,
			})
			if err != nil || len(picked) == 0 {
				return
			}
			state.mu.Lock()
			cat := state.launcherImportCatalog
			state.mu.Unlock()
			if cat == nil {
				state.applyImportCatalog(win, "")
				state.mu.Lock()
				cat = state.launcherImportCatalog
				state.mu.Unlock()
			}
			if cat == nil {
				return
			}
			n := importer.AppendSkillDirectory(cat, picked[0])
			state.mu.Lock()
			state.launcherImportCatalog = cat
			state.launcherImport = fmt.Sprintf(i18n.T(i18n.KeyLauncherImportSummarySkillDir),
				cat.SourceHome, cat.DestHome,
				len(cat.Sessions), len(cat.Skills), len(cat.Plugins),
				len(cat.MCP), len(cat.Settings), len(cat.Presets),
				picked[0], n)
			counts := cat.CategoryCounts()
			state.launcherImportCategories = map[string]bool{}
			for id, pair := range counts {
				state.launcherImportCategories[id] = pair[1] > 0
			}
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnCopyImportJournal: func(lines []string) {
			if len(lines) == 0 {
				return
			}
			mygo.Clipboard.WriteText(strings.Join(lines, "\n"))
		},
		OnStartImport: func() { state.startImport(win) },
		ImportBusy:    launcherImportBusy,
		OnCancelImport: func() {
			state.mu.Lock()
			cancel := state.launcherImportCancel
			state.mu.Unlock()
			if cancel != nil {
				cancel()
			}
		},
		OnInstallUpdate: func() {
			// Electron #btn-home-update → check + download/stage (same as settings update).
			state.showUpdateCheck(win)
			res, err := desktopupdate.CheckForUpdatesVersionRoute(context.Background(), host.Version, state.updateDownloadRoute())
			state.mu.Lock()
			if err != nil {
				state.launcherUpdateAvailable = false
			} else {
				state.launcherUpdateAvailable = res.Status == desktopupdate.StatusUpdateAvailable
				if state.launcherUpdateAvailable {
					state.launcherUpdate = fmt.Sprintf(i18n.T(i18n.KeyLauncherUpdateAvailable), res.Version, host.Version)
				}
			}
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnCheckUpdate: func() {
			state.checkLauncherUpdate(win)
		},
		OnOpenSettings: func() {
			state.mu.Lock()
			state.launcherOpen = false
			state.settingsEmbeddedSection = ""
			state.settingsOpen = true
			state.settingsDesktopActive = true
			state.seedDesktopDraft()
			state.mu.Unlock()
			if state.settings != nil {
				state.settings.OpenSettingsDocument(context.Background())
			}
			// Settings live in the main window: close the launcher window first.
			state.syncLauncherStage()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnEnableSkipUserPlugins: func() {
			if cfgStore == nil {
				state.refuseHostAction()
				return
			}
			err := cfgStore.UpdatePluginRecovery(func(m map[string]any) {
				m["skipUserPlugins"] = true
				m["appVersion"] = "" // explicit manual recovery survives upgrades
				if _, ok := m["reason"]; !ok {
					m["reason"] = "launcher"
				}
			})
			state.mu.Lock()
			if err != nil {
				state.launcherNote = err.Error()
			} else {
				state.launcherNote = i18n.T(i18n.KeyLauncherSkipUserPluginsOn)
			}
			state.mu.Unlock()
			if err == nil {
				state.syncHarnessFromConfig()
				if harnessSup != nil {
					state.restartUnlessImport(harnessSup)
				}
			}
			if win != nil {
				win.Update(func() {})
			}
		},
		OnClearPluginRecovery: func() {
			if cfgStore == nil {
				state.refuseHostAction()
				return
			}
			err := cfgStore.ClearPluginRecovery()
			state.mu.Lock()
			if err != nil {
				state.launcherNote = err.Error()
			} else {
				state.launcherNote = i18n.T(i18n.KeyLauncherSkipUserPluginsOff)
			}
			state.mu.Unlock()
			if err == nil {
				state.syncHarnessFromConfig()
				if harnessSup != nil {
					state.restartUnlessImport(harnessSup)
				}
			}
			if win != nil {
				win.Update(func() {})
			}
		},
		OnRestartHarness: func() {
			state.syncHarnessFromConfig()
			if harnessSup != nil {
				state.restartUnlessImport(harnessSup)
			}
			state.mu.Lock()
			state.launcherNote = i18n.T(i18n.KeyLauncherRestartHarness)
			state.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		},
	}))
}
