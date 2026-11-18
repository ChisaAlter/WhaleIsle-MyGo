package main

import (
	"log"
	"os"
	"slices"
	"strings"
	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/boot"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) openWorkbenchSettings() {
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "desktop"})
}

func (s *mainState) startWorkbenchHost(win *mygo.Window, sup *harness.Supervisor) {
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	if store == nil {
		s.refuseHostAction()
		return
	}
	if sup != nil {
		s.restartUnlessImport(sup)
		return
	}
	l := s.launcherController()
	_, gen, ok := l.TryBeginStart()
	if !ok {
		return
	}
	go func() {
		cfg := store.Get()
		ws := strings.TrimSpace(cfg.Workspace)
		if folder, err := os.Stat(ws); err != nil || !folder.IsDir() {
			// Native folder selection is a user action and belongs to the workbench.
			mygo.RunOnMain(func() {
				picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeySettingsDesktopBrowseWorkspace), Directory: true})
				if err != nil || len(picked) == 0 {
					l.EndPreparationOnly()
					l.SetStartBusy(false)
					if err != nil {
						s.notifyUser(win, errcopy.Sentence(err))
					}
					win.Invalidate()
					return
				}
				go func() {
					if err := store.Update(func(c *config.Config) { c.Workspace = picked[0] }); err != nil {
						l.FinishStartPreparation(true, errcopy.Sentence(err), "", gen)
						win.Invalidate()
						return
					}
					l.FinishStartPreparation(false, "", picked[0], gen)
					startHarness(store.Get())
				}()
			})
			return
		}
		l.FinishStartPreparation(false, "", ws, gen)
		startHarness(cfg)
	}()
	win.Invalidate()
}

func (s *mainState) workbenchHostStatus(c *ui.Context, win *mygo.Window, st harness.Status, sup *harness.Supervisor, logs []string) {
	if st.State == harness.Ready && !st.Recovery {
		return
	}
	m, starting, message, label := s.hostStatusModel(win, st, sup, logs)
	boot.WorkbenchStatus(c, m, starting, message, label)
}

func (s *mainState) paintColdLoading(c *ui.Context) bool {
	s.mu.Lock()
	if !s.coldLoading || s.mainRoute != routeConversation || !s.settingsLaunchHandled && slices.Contains(os.Args, "--settings") {
		s.mu.Unlock()
		return false
	}
	win, sup, st, themeID, skipped := s.win, s.sup, s.bootStatus, s.themeID, s.coldStartSkipped
	connectionIssue := connectionUnavailableMessage(s.dshBootstrapError, s.dshNetworkOffline, s.dshMuxLost)
	logs := append([]string(nil), s.bootLogs...)
	s.mu.Unlock()
	m, _, message, _ := s.hostStatusModel(win, st, sup, logs)
	// A declined automatic start still opens on the page: the reason the shell
	// already carries becomes its status line, and the page offers the retry and
	// diagnostics actions (boot.Model.Declined).
	if skipped && st.State == harness.Stopped {
		m.Declined = true
		if m.StatusText == "" {
			m.StatusText = message
		}
	}
	systemDark := mygo.Theme.IsDark()
	themed := theme.ResolveWithSystem(themeID, systemDark)
	m.Dark = themed.Scheme == theme.SchemeDark
	if st.State == harness.Ready {
		m.StatusText = lifecycleText("正在打开工作区…", "Opening workspace…")
		if connectionIssue != "" {
			m.Status.State = harness.Failed
			m.StatusText = connectionIssue
			m.OnRetry = s.requestConnectionReconnect
			m.OnConfigure = func() { s.navigateMainRoute(routeStartupDiagnostics) }
		}
	} else if st.State == harness.Stopped && !st.RecoveryCancelled {
		m.Status.State = harness.Starting
	}
	colors := themed.Active(systemDark)
	defer s.paintClosingOverlayLayer(c, colors)
	defer s.syncConfirmAsk(c, colors)
	boot.View(c, m)
	return true
}

func (s *mainState) hostStatusModel(win *mygo.Window, st harness.Status, sup *harness.Supervisor, logs []string) (boot.Model, bool, string, string) {
	s.mu.Lock()
	importPending, importBusy, importSummary := s.importRecoveryPending, s.launcherImportBusy, s.launcherImport
	busy := s.taskRestartPending || s.quitting || s.quitRequestPending || importBusy
	s.mu.Unlock()
	activity := s.launcherController().StartupActivity()
	starting := !st.RecoveryCancelled && (st.State == harness.Starting || activity.Busy || activity.Preparing)
	message, _, _ := strings.Cut(activity.Error, "\n")
	if (st.State == harness.Starting && st.Err == nil) || (st.State == harness.Failed && st.Err != nil) {
		// The supervisor speaks for the Host: a start in progress, or a failure
		// boot.StatusLine names with its next step. The launcher's error is the
		// same event or a stale one from the previous attempt, and never the
		// strip's copy — the raw text stays in the logs.
		message = ""
	}
	if importBusy {
		message = lifecycleText("正在导入数据。导入完成后可启动 Host。", "Importing data. You can start Host when the import finishes.")
	} else if importPending {
		message = importSummary
	} else if st.State == harness.Ready && st.Recovery {
		message = lifecycleText("Host 已在恢复模式下启动，暂未加载用户插件。", "Host is running in recovery mode without user plugins.")
	}
	m := boot.Model{Status: st, Logs: logs}
	if sup != nil {
		m.OnCancelRecovery = func() { sup.CancelRecovery() }
	}
	if !starting && !busy && (!st.RecoveryScheduled || st.RecoveryCancelled) && st.State != harness.Ready {
		m.OnRetry = func() { s.startWorkbenchHost(win, sup) }
	}
	if len(logs) != 0 || st.State == harness.Failed {
		m.OnExport = func() {
			if _, err := harness.ExportBootLogsDialog(win, buildVersion, st, logs, i18n.T(i18n.KeyBootExport)); err != nil {
				log.Println("export log:", err)
			}
		}
	}
	var label string
	if importPending || importBusy {
		label = lifecycleText("查看导入", "View import")
		m.OnConfigure = func() { s.navigateMainRoute(routeImport) }
	} else if !starting && (st.State == harness.Failed || st.Recovery || activity.Error != "") {
		label = lifecycleText("启动诊断", "Startup diagnostics")
		m.OnConfigure = func() { s.navigateMainRoute(routeStartupDiagnostics) }
	}
	return m, starting, message, label
}
