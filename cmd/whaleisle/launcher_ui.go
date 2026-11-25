package main

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"whaleisle/internal/errcopy"

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

type launcherPanelHost struct {
	ThemeID    string
	Version    string
	Status     harness.Status
	Workspace  string
	Supervisor *harness.Supervisor
}

func (s *mainState) launcherPanelHostSnapshot() launcherPanelHost {
	s.mu.Lock()
	host := launcherPanelHost{
		ThemeID:    s.themeID,
		Version:    buildVersion,
		Status:     s.bootStatus,
		Supervisor: s.sup,
	}
	cfgStore := s.cfgStore
	s.mu.Unlock()
	if cfgStore != nil {
		host.Workspace = cfgStore.Get().Workspace
	}
	return host
}

func (s *mainState) launcherSurfaceInvalidator(win *mygo.Window) func() {
	return func() {
		if win != nil {
			win.Invalidate()
		} else {
			s.invalidateMainWindow()
		}
	}
}

func (s *mainState) maintenancePanel(c *ui.Context, win *mygo.Window, route mainRoute) {
	host := s.launcherPanelHostSnapshot()
	l := s.launcherController()
	s.mu.Lock()
	cfgStore := s.cfgStore
	launcherImport := s.launcherImport
	launcherImportJournal := append([]string(nil), s.launcherImportJournal...)
	launcherImportBusy := s.launcherImportBusy
	launcherImportScanBusy := s.launcherImportScan.busy

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
	if route == routeComponents && !panelSnap.ComponentsFresh {
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

	m := launcher.Model{
		ThemeID:        host.ThemeID,
		SystemDark:     mygo.App.IsReady() && mygo.Theme.IsDark(),
		Version:        host.Version,
		VendorReady:    vendorReady,
		StartBusy:      panelSnap.StartBusy,
		StartError:     panelSnap.StartErr,
		ImportSummary:  launcherImport,
		ImportScanBusy: launcherImportScanBusy,
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
		AskOnUpdate:          &askVal,
		DownloadRoute:        &routeVal,
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
				c.AskOnUpdate = saved.AskOnUpdate
				c.DownloadRoute = saved.DownloadRoute
			})
			if err != nil {
				l.SetPrefsError(errcopy.Sentence(err))
			} else {
				l.SetPrefsError("")
			}
		},
		OnClose:               func() { s.navigateMainRoute(routeConversation) },
		OnStop:                func() { s.stopProtected(win, host.Supervisor) },
		OnStart:               func() { s.startWorkbenchHost(win, host.Supervisor) },
		OnChooseRepairRelease: func() { s.openNativeUpdateCenter(win) },
		OnScanImport:          func() { s.rescanImportCatalog(win, "") },
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
		OnEnableSkipUserPlugins: func() {
			if cfgStore == nil {
				s.refuseHostAction()
				return
			}
			err := cfgStore.UpdatePluginRecovery(func(m map[string]any) {
				m["skipUserPlugins"] = true
				m["appVersion"] = ""
				if _, ok := m["reason"]; !ok {
					m["reason"] = "startup-diagnostics"
				}
			})
			if err != nil {
				l.SetNote(errcopy.Sentence(err))
			} else {
				l.SetNote(i18n.T(i18n.KeyLauncherSkipUserPluginsOn))
				s.syncHarnessFromConfig()
				if host.Supervisor != nil {
					s.restartUnlessImport(host.Supervisor)
				}
			}
		},
		OnClearPluginRecovery: func() {
			if cfgStore == nil {
				s.refuseHostAction()
				return
			}
			err := cfgStore.ClearPluginRecovery()
			if err != nil {
				l.SetNote(errcopy.Sentence(err))
			} else {
				l.SetNote(i18n.T(i18n.KeyLauncherSkipUserPluginsOff))
				s.syncHarnessFromConfig()
				if host.Supervisor != nil {
					s.restartUnlessImport(host.Supervisor)
				}
			}
		},
		OnRestartHarness: func() {
			s.syncHarnessFromConfig()
			s.startWorkbenchHost(win, host.Supervisor)
			l.SetNote(i18n.T(i18n.KeyLauncherRestartHarness))
		},
	}
	var page launcher.MaintenancePage
	switch route {
	case routeComponents:
		page = launcher.MaintenanceComponents
		m = s.withWhaleBridgeAlerts(m)
	case routeImport:
		page = launcher.MaintenanceImport
	case routeStartupDiagnostics:
		page = launcher.MaintenanceDiagnostics
		m = s.withStartupDiagnostics(m)
	case routeUpdatePreferences:
		page = launcher.MaintenancePreferences
		m.RouteUnavailable = s.updaterCenter().RouteAvailability()
		m = s.withApplicationUninstall(win, m)
	}
	launcher.Maintenance(c, page, m)
	commitLauncherDrafts()
}

var launcherVendorProbe backgroundProbe[bool]
