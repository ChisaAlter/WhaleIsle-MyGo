package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	launcherctl "whaleisle/internal/app/launcher"
	"whaleisle/internal/config"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/launcher"
	"whaleisle/internal/ui/theme"
)

func isolatedDesktopRun() bool    { return launcherctl.IsolatedDesktopRun() }
func isolatedDesktopHidden() bool { return launcherctl.IsolatedDesktopHidden() }
func prepareIsolatedDesktop() error {
	return launcherctl.PrepareIsolatedDesktop(mygoNativePathSetter{})
}
func preparePortableDesktop() error { return launcherctl.PreparePortableDesktop() }
func claimDesktopProfile() (func(), error) {
	return launcherctl.ClaimDesktopProfile()
}
func desktopSystemIntegration(apply func() error) error {
	return launcherctl.DesktopSystemIntegration(apply)
}
func desktopComponentRoots() []string { return launcherctl.DesktopComponentRoots() }
func launcherFirstWindow(isolated, hidden, background, settings bool) bool {
	return launcherctl.FirstWindow(isolated, hidden, background, settings)
}

func (s *mainState) openStartupRecovery(dir string, cfg *config.Store) {
	s.launcherController().OpenStartupRecovery(dir, cfg, buildVersion)
}
func (s *mainState) coldStartupAllowed(cfg config.Config, importReady bool) bool {
	return s.launcherController().ColdStartupAllowed(cfg, importReady)
}
func (s *mainState) refreshLauncherComponents() { s.launcherController().RefreshComponents() }
func (s *mainState) harnessVersionSnapshot() string {
	return s.launcherController().HarnessVersionSnapshot()
}
func launcherWindowRef(w *mygo.Window) launcherctl.WindowRef {
	if w == nil {
		return nil
	}
	return w
}

func (s *mainState) checkLauncherUpdate(win *mygo.Window) {
	s.launcherController().CheckUpdate(launcherWindowRef(win), buildVersion)
}
func (s *mainState) installLauncherUpdate(win *mygo.Window, version string) {
	s.launcherController().InstallUpdate(launcherWindowRef(win), version)
}
func (s *mainState) claimStartupSelection(win *mygo.Window, sessions *session.Store, receipt session.InitialWorkspaceSession) bool {
	return s.launcherController().ClaimStartupSelection(launcherWindowRef(win), sessions, receipt)
}
func (s *mainState) maybeStartupBlankSession(win *mygo.Window, sessions *session.Store) {
	s.launcherController().MaybeStartupBlankSession(launcherWindowRef(win), sessions)
}
func (s *mainState) startupDiagnostics() (startuprec.Diagnostics, bool) {
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	supID := ""
	if sup != nil {
		supID = "sup"
	}
	return s.launcherController().StartupDiagnostics(true, supID, nil)
}

func (s *mainState) withStartupDiagnostics(m launcher.Model) launcher.Model {
	l := s.launcherController()
	s.mu.Lock()
	sup := s.sup
	win := s.win
	s.mu.Unlock()
	supID := ""
	if sup != nil {
		supID = "sup"
	}
	report, busy := l.StartupDiagnostics(false, supID, s.launcherSurfaceInvalidator(win))
	m.StartupDiagnostics = report
	m.StartupRepairBusy = busy
	m.OnRefreshStartupDiagnostics = func() { l.StartupRepairNote("") }
	m.OnRepairStartupSuspects = func() { l.RepairSuspects(report.Receipt, slices.Clone(report.RepairNames), supID) }
	m.OnSetStartupPluginEnabled = func(name string, enabled bool) {
		l.ChangeStartupPlugins(report.Receipt, []string{name}, enabled, false, false, supID)
	}
	m.OnRemoveStartupPlugin = func(name string) {
		l.ChangeStartupPlugins(report.Receipt, []string{name}, false, true, false, supID)
	}
	return m
}

func launcherWindowOptions(cfg config.Config) mygo.WindowOptions {
	return mygo.WindowOptions{
		Title:                    i18n.T(i18n.KeyAppName),
		Width:                    launcherctl.WindowWidth,
		Height:                   launcherctl.WindowHeight,
		MinWidth:                 launcherctl.WindowMinWidth,
		MinHeight:                launcherctl.WindowMinHeight,
		StateKey:                 "launcher",
		Hidden:                   true,
		TitleBarStyle:            mygo.TitleBarHidden,
		TitleBarHeight:           launcherctl.CaptionHeight,
		AutoHideMenuBar:          true,
		WindowsCaptionButtons:    &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 12, Right: 8, CornerRadius: 8, GlyphSize: 12},
		WindowsFrameCornerRadius: 20,
		Transparent:              true,
		BackgroundColor:          theme.Resolve(cfg.Theme).BackgroundColor(),
	}
}

type launcherWindow struct{ win *mygo.Window }

func (lw *launcherWindow) destroyed() bool {
	return lw == nil || lw.win == nil || lw.win.IsDestroyed()
}

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

type launcherPanelHost struct {
	ThemeID    string
	Version    string
	Status     harness.Status
	Logs       []string
	Workspace  string
	Supervisor *harness.Supervisor
	Sessions   bool
}

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

type shellLauncherStage struct{ s *mainState }

func (st shellLauncherStage) LauncherOpen() bool {
	st.s.mu.Lock()
	lw := st.s.launcherWin
	st.s.mu.Unlock()
	return !lw.destroyed()
}

func (st shellLauncherStage) ShowLauncher() bool {
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

func (st shellLauncherStage) CloseLauncher() { st.s.closeLauncherWindow() }
func (st shellLauncherStage) ShowMain()      { st.s.showMainWindow() }

func (st shellLauncherStage) MainVisible() bool {
	st.s.mu.Lock()
	win := st.s.win
	st.s.mu.Unlock()
	return win != nil && !win.IsDestroyed() && win.IsVisible()
}

func (st shellLauncherStage) Quitting() bool {
	st.s.mu.Lock()
	defer st.s.mu.Unlock()
	return st.s.quitting
}

func (s *mainState) syncLauncherStage() launcherctl.StageResult {
	s.mu.Lock()
	open := s.launcherOpen
	s.mu.Unlock()
	return launcherctl.ApplyStage(shellLauncherStage{s: s}, open)
}

func (s *mainState) launcherStageHasWindow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.launcherWindowAllowed
}

func (s *mainState) ensureLauncherWindow() *launcherWindow {
	s.mu.Lock()
	existing := s.launcherWin
	cfgStore := s.cfgStore
	s.mu.Unlock()
	if !existing.destroyed() {
		return existing
	}
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
	applyWindowIcon(win)
	win.OnClose(func(e *mygo.CloseEvent) { s.launcherWindowCloseRequested(e) })
	win.OnClosed(func() { s.launcherWindowClosed(lw) })
	s.mu.Lock()
	if other := s.launcherWin; !other.destroyed() {
		s.mu.Unlock()
		win.Destroy()
		return other
	}
	s.launcherWin = lw
	s.mu.Unlock()
	return lw
}

func (s *mainState) closeLauncherWindow() {
	s.mu.Lock()
	lw := s.launcherWin
	s.launcherWin = nil
	s.mu.Unlock()
	if lw == nil {
		return
	}
	win := lw.win
	s.cancelImportCatalogScanOwner(win)
	lw.win = nil
	if win != nil && !win.IsDestroyed() {
		win.Destroy()
	}
}

func (s *mainState) launcherWindowClosed(lw *launcherWindow) {
	s.cancelImportCatalogScanOwner(lw.win)
	s.mu.Lock()
	if s.launcherWin == lw {
		s.launcherWin = nil
	}
	s.mu.Unlock()
}

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

func (s *mainState) launcherWindowBody(c *ui.Context, lw *launcherWindow) {
	defer s.processApplicationMenuFor(c, lw.win)
	defer s.syncConfirmAskFor(c, lw.win)
	defer s.syncNativeUpdateModal(c, lw.win)
	s.registerApplicationCommands(c)
	s.syncApplicationMenu(c)
	ui.Column(c).Fill().Children(func() {
		s.launcherPanel(c, lw.win, s.launcherPanelHostSnapshot())
	})
}

func (s *mainState) launcherSurfaceInvalidator(self *mygo.Window) func() {
	s.mu.Lock()
	lw := s.launcherWin
	s.mu.Unlock()
	var ln *mygo.Window
	if !lw.destroyed() {
		ln = lw.win
	}
	return func() {
		if self != nil {
			self.Invalidate()
		}
		if ln != nil && ln != self {
			ln.Invalidate()
		}
		s.invalidateMainWindow()
	}
}

func (s *mainState) launcherPanel(c *ui.Context, win *mygo.Window, host launcherPanelHost) {
	l := s.launcherController()
	s.mu.Lock()
	cfgStore := s.cfgStore
	launcherImport := s.launcherImport
	launcherImportJournal := append([]string(nil), s.launcherImportJournal...)
	launcherImportBusy := s.launcherImportBusy
	launcherImportScanBusy := s.launcherImportScan.busy
	launcherRuntimeBusy := s.taskRestartPending || s.quitRequestPending || s.quitting
	importCategories := s.launcherImportCategories
	importCatalog := s.launcherImportCatalog
	s.mu.Unlock()

	var cfg config.Config
	var cfgPtr *config.Config
	var disabled []string
	var skipUserPlugins bool
	var pluginRecoveryReason string
	if cfgStore != nil {
		cfg = cfgStore.Get()
		cfgPtr = &cfg
		disabled = append([]string(nil), cfg.DisabledPlugins...)
		rec := cfgStore.PluginRecoveryEvidence()
		skipUserPlugins = rec.SkipUserPlugins
		pluginRecoveryReason = rec.Reason
	}
	binJS := filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")
	vendorReady := l.VendorReady(binJS, s.launcherSurfaceInvalidator(win))

	panelSnap := l.PanelState(cfgPtr, host.Workspace)
	if !panelSnap.ComponentsFresh {
		l.RefreshComponents()
	}
	componentActivity := s.launcherComponentActivitySnapshot()
	draftsBefore := panelSnap.Drafts
	wsDraft := draftsBefore.Workspace
	quitVal := draftsBefore.QuitAfterStart
	trayVal := draftsBefore.CloseToTray
	autoVal := draftsBefore.AutoStartDesktop
	askVal := draftsBefore.AskOnUpdate
	routeVal := draftsBefore.DownloadRoute
	routePopVal := draftsBefore.RoutePopOpen
	importOverwriteVal := draftsBefore.ImportOverwrite
	importAttachVal := draftsBefore.ImportAttachments

	commitLauncherDrafts := func() {
		l.CommitDrafts(&draftsBefore, launcherctl.Drafts{
			Workspace:         wsDraft,
			WorkspaceGen:      draftsBefore.WorkspaceGen,
			QuitAfterStart:    quitVal,
			CloseToTray:       trayVal,
			AutoStartDesktop:  autoVal,
			AskOnUpdate:       askVal,
			DownloadRoute:     routeVal,
			RoutePopOpen:      routePopVal,
			ImportOverwrite:   importOverwriteVal,
			ImportAttachments: importAttachVal,
		})
	}

	deskRunning := host.Status.State == harness.Ready || (host.Status.State == harness.Starting && host.Sessions)
	launcher.View(c, s.withNativeReleases(win, launcher.Model{
		ThemeID:         host.ThemeID,
		SystemDark:      mygo.App.IsReady() && mygo.Theme.IsDark(),
		Version:         host.Version,
		Workspace:       wsDraft,
		WorkspacePtr:    &wsDraft,
		VendorReady:     vendorReady,
		LauncherPackage: isLauncherPackage(),
		DesktopRunning:  deskRunning,
		StartBusy:       panelSnap.StartBusy,
		RuntimeBusy:     launcherRuntimeBusy,
		StartFailed:     panelSnap.StartFailed,
		StartError:      panelSnap.StartErr,
		BootLogs:        host.Logs,
		PluginLoad:      host.Status.PluginLoad,
		ActiveTab:       panelSnap.Tab,
		ImportSummary:   launcherImport,
		ImportScanBusy:  launcherImportScanBusy,
		OnCancelImportScan: func() {
			s.cancelImportCatalogScan()
			s.launcherSurfaceInvalidator(win)()
		},
		ImportJournal:        launcherImportJournal,
		UpdateStatus:         panelSnap.UpdateStatus,
		UpdateAvailable:      panelSnap.UpdateAvailable,
		DisabledPlugins:      disabled,
		SkipUserPlugins:      skipUserPlugins,
		PluginRecoveryReason: pluginRecoveryReason,
		StatusNote:           panelSnap.Note,
		QuitAfterStart:       &quitVal,
		CloseToTray:          &trayVal,
		AutoStartDesktop:     &autoVal,
		AskOnUpdate:          &askVal,
		DownloadRoute:        &routeVal,
		RoutePopOpen:         &routePopVal,
		ImportOverwrite:      &importOverwriteVal,
		ImportAttachments:    &importAttachVal,
		ImportCategories:     importCategories,
		ImportCatalog:        importCatalog,
		Components:           panelSnap.Components,
		ComponentPending:     componentActivity.Pending,
		ComponentStatus:      componentActivity.Status,
		LauncherPrefsError:   panelSnap.PrefsError,
		ComponentBusy:        bridgeChangeActive.Load(),
		HostRuntimeState:     host.Status.State,
		OnRefreshComponents: func() {
			l.RefreshComponents()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnComponentAction: func(id, action string) {
			if bridgeChangeActive.Load() {
				return
			}
			if id == desktopcomponents.HostComponentID {
				s.changeHostBundle(win, action)
				return
			}
			if id == "whalebridge" && (action == "update" || action == "rollback" || action == "uninstall" || action == "stop") {
				s.changeWhaleBridge(win, action)
				return
			}
			if id == "whalebridge" && (action == "start" || action == "open" || action == "manage") {
				release, ok := s.claimLauncherComponentAction(win, id, action)
				if !ok {
					return
				}
				status := i18n.T(i18n.KeyCmdBridgeStarting)
				if action != "start" {
					status = i18n.T(i18n.KeyCmdOpeningWhalebridgeSettings)
				}
				l.SetComponentActivity(status, status, false)
				go func() {
					defer release()
					run := action
					if action != "start" {
						run = "start"
					}
					err := desktopcomponents.RunAction(paths.UserDataDir(), id, run)
					msg := desktopcomponents.FormatActionError(id, action, err)
					l.SetComponentActivity(msg, msg, false)
					l.RefreshComponents()
					if win != nil {
						win.Update(func() {})
						if err == nil && action != "start" {
							win.Update(func() { s.openWhaleBridgeWindow(win, action == "manage") })
						}
					}
				}()
				return
			}
			if id == "whalebridge" && action == "install" {
				release, ok := s.claimLauncherComponentAction(win, id, action)
				if !ok {
					return
				}
				route := s.updateDownloadRoute()
				l.SetComponentActivity(i18n.T(i18n.KeyCmdBridgeDownloading), i18n.T(i18n.KeyCmdBridgeDownloading), false)
				go func() {
					defer release()
					err := desktopcomponents.InstallWhaleBridge(context.Background(), paths.UserDataDir(), route)
					msg := desktopcomponents.FormatActionError(id, action, err)
					if err == nil {
						msg = i18n.T(i18n.KeyCmdBridgeInstalled)
					}
					l.SetComponentActivity(msg, msg, false)
					l.RefreshComponents()
					if win != nil {
						win.Update(func() {})
					}
				}()
				return
			}
			s.runGenericComponentAction(win, id, action)
		},
		OnSaveLauncherPrefs: func() {
			commitLauncherDrafts()
			if cfgStore == nil {
				s.refuseHostAction()
				return
			}
			saved := l.PanelState(nil, "").Drafts
			err := cfgStore.Update(func(c *config.Config) {
				c.QuitAfterStart = saved.QuitAfterStart
				c.CloseToTray = saved.CloseToTray
				c.AutoStartDesktop = saved.AutoStartDesktop
				c.AskOnUpdate = saved.AskOnUpdate
				c.DownloadRoute = saved.DownloadRoute
			})
			if err != nil {
				l.SetPrefsError(err.Error())
			} else {
				l.SetPrefsError("")
				s.mu.Lock()
				s.closeToTray = saved.CloseToTray
				s.mu.Unlock()
			}
			if win != nil {
				win.Update(func() {})
			}
		},
		OnTab: func(tab string) {
			l.SetTab(tab)
			if win != nil {
				win.Update(func() {})
			}
		},
		OnClose: func() {
			s.mu.Lock()
			s.launcherOpen = false
			s.mu.Unlock()
			s.syncLauncherStage()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnStop: func() {
			s.mu.Lock()
			sup := s.sup
			s.mu.Unlock()
			s.stopProtected(win, sup)
		},
		OnStart: func() {
			commitLauncherDrafts()
			ws, startDraftGen, ok := l.TryBeginStart()
			if !ok {
				return
			}
			s.updaterCenter().SetHold(false)
			go func() {
				if cfgStore == nil {
					l.FinishStartPreparation(true, i18n.T(i18n.KeyCmdConfigStoreNotReady), "", startDraftGen)
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				if ws == "" {
					ws = paths.DefaultWorkspace()
				}
				if err := os.MkdirAll(ws, 0o755); err != nil {
					l.FinishStartPreparation(true, err.Error(), "", startDraftGen)
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				if err := cfgStore.Update(func(c *config.Config) { c.Workspace = ws }); err != nil {
					l.FinishStartPreparation(true, err.Error(), "", startDraftGen)
					if win != nil {
						win.Update(func() {})
					}
					return
				}
				l.FinishStartPreparation(false, "", ws, startDraftGen)
				s.mu.Lock()
				s.launcherOpen = true
				s.mu.Unlock()
				startHarness(cfgStore.Get())
				if win != nil {
					win.Update(func() {})
				}
			}()
		},
		OnScanImport: func() { s.rescanImportCatalog(win, "") },
		OnPickImportSource: func() {
			picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyLauncherImportPickSourceTitle), Directory: true})
			if err == nil && len(picked) > 0 {
				s.rescanImportCatalog(win, picked[0])
			}
		},
		OnAddSkillImportDir: func() {
			picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyLauncherImportAddSkillTitle), Directory: true})
			if err == nil && len(picked) > 0 {
				s.requestImportCatalog(win, "", picked[0])
			}
		},
		OnCopyImportJournal: func(lines []string) {
			if len(lines) > 0 {
				mygo.Clipboard.WriteText(strings.Join(lines, "\n"))
			}
		},
		OnStartImport: func() {
			commitLauncherDrafts()
			s.startImport(win)
		},
		ImportBusy: launcherImportBusy,
		OnCancelImport: func() {
			s.mu.Lock()
			cancel := s.launcherImportCancel
			s.mu.Unlock()
			if cancel != nil {
				cancel()
			}
		},
		OnInstallUpdate: func() { s.installLauncherUpdate(win, host.Version) },
		OnCheckUpdate:   func() { s.checkLauncherUpdate(win) },
		OnOpenSettings: func() {
			s.mu.Lock()
			s.launcherOpen = false
			s.settingsEmbeddedSection = ""
			s.settingsOpen = true
			s.settingsDesktopActive = true
			s.seedDesktopDraft()
			st := s.settings
			s.mu.Unlock()
			if st != nil {
				st.RefreshDescribe(context.Background())
			}
			s.syncLauncherStage()
			if win != nil {
				win.Update(func() {})
			}
		},
		OnEnableSkipUserPlugins: func() {
			if cfgStore == nil {
				s.refuseHostAction()
				return
			}
			err := cfgStore.UpdatePluginRecovery(func(m map[string]any) {
				m["skipUserPlugins"] = true
				m["appVersion"] = ""
				if _, ok := m["reason"]; !ok {
					m["reason"] = "launcher"
				}
			})
			if err != nil {
				l.SetNote(err.Error())
			} else {
				l.SetNote(i18n.T(i18n.KeyLauncherSkipUserPluginsOn))
				s.syncHarnessFromConfig()
				if host.Supervisor != nil {
					s.restartUnlessImport(host.Supervisor)
				}
			}
			if win != nil {
				win.Update(func() {})
			}
		},
		OnClearPluginRecovery: func() {
			if cfgStore == nil {
				s.refuseHostAction()
				return
			}
			err := cfgStore.ClearPluginRecovery()
			if err != nil {
				l.SetNote(err.Error())
			} else {
				l.SetNote(i18n.T(i18n.KeyLauncherSkipUserPluginsOff))
				s.syncHarnessFromConfig()
				if host.Supervisor != nil {
					s.restartUnlessImport(host.Supervisor)
				}
			}
			if win != nil {
				win.Update(func() {})
			}
		},
		OnRestartHarness: func() {
			s.syncHarnessFromConfig()
			if host.Supervisor != nil {
				s.restartUnlessImport(host.Supervisor)
			}
			l.SetNote(i18n.T(i18n.KeyLauncherRestartHarness))
			if win != nil {
				win.Update(func() {})
			}
		},
	}))
	commitLauncherDrafts()
}

var launcherVendorProbe backgroundProbe[bool]

var _ = time.Second
