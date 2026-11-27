package main

import (
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/app/startup"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/harness"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/boot"
	"whaleisle/internal/ui/chrome"
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
		cfg = s.ensureLaunchWorkspace(cfg)
		l.FinishStartPreparation(false, "", strings.TrimSpace(cfg.Workspace), gen)
		startHarness(cfg)
	}()
	win.Invalidate()
}

func (s *mainState) workbenchHostStatus(c *ui.Context, win *mygo.Window, st harness.Status, sup *harness.Supervisor, logs []string) {
	if st.State == harness.Ready && !st.Recovery {
		return
	}
	m, _, message, label := s.hostStatusModel(win, st, sup, logs)
	boot.WorkbenchStatus(c, m, message, label)
}

// openedColdPage is the page a cold launch opens on. A declined automatic start
// still opens there: the reason the shell already carries becomes its status
// line, and the page offers the action that resolves it (boot.Model.Declined).
func openedColdPage(m boot.Model, st harness.Status, skipped bool, message string) boot.Model {
	if !skipped || st.State != harness.Stopped {
		return m
	}
	m.Declined = true
	if m.StatusText == "" {
		m.StatusText = message
	}
	return m
}

// coldHandoffTimeout bounds how long the boot page waits after
// WorkbenchBaseline() for Host appearance, wallpaper decoding, resumed session
// history and default model readiness so startup can never stall on boot.View.
const coldHandoffTimeout = 1500 * time.Millisecond

func (s *mainState) coldHandoffActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coldLoading && s.coldHandoffArmed
}

func (s *mainState) finishColdHandoff() {
	s.mu.Lock()
	s.coldLoading = false
	s.coldHandoffArmed = false
	s.coldHandoffSince = time.Time{}
	s.mu.Unlock()
}

func (s *mainState) prewarmColdHandoffWallpaper(c *ui.Context) bool {
	if renderpolicy.NoBackdrop() || s.activeSkin() == theme.SkinCodex {
		return true
	}
	prefs := wallpapers.Defaults()
	if g := s.wallpaperEditor(); g != nil {
		prefs = g.EffectivePreferences()
	}
	path := s.resolvedWallpaperPathFor(s.currentWallpaperConfigPath(), prefs)
	if strings.TrimSpace(path) == "" {
		return true
	}
	prefs = wallpaperBackdropPrefs(path, prefs)
	cw, ch := c.Size()
	w, h := int(math.Round(float64(cw))), int(math.Round(float64(ch)))
	return chrome.PrewarmWallpaper(path, prefs, w, h)
}

// tryCompleteColdHandoff runs on the UI thread inside paintColdLoading. Once
// the session store reaches WorkbenchBaseline (or reports a workspace/list
// error), it prepares Host appearance, wallpaper bitmap decoding, resumed
// session selection/history and default model state before lowering
// coldLoading so the very first painted workbench frame is complete.
func (s *mainState) tryCompleteColdHandoff(c *ui.Context) bool {
	s.mu.Lock()
	win := s.win
	sessions := s.sessions
	setStore := s.settings
	muxReady := s.dshMuxReady
	s.mu.Unlock()

	if sessions == nil || !muxReady {
		return false
	}
	snap := sessions.Published().Value
	hasErr := snap.WorkspaceError != nil || snap.ListErr != nil
	baseline := sessions.WorkbenchBaseline()
	if !baseline && !hasErr {
		return false
	}

	s.syncHostAppearance()
	if hasErr {
		s.finishColdHandoff()
		return true
	}

	s.restoreResumeSelection()
	s.maybeStartupBlankSession(win, sessions)
	snap = sessions.Published().Value

	now := c.Now()
	timedOut := false
	var remaining time.Duration
	s.mu.Lock()
	s.coldHandoffArmed = true
	if !now.IsZero() {
		if s.coldHandoffSince.IsZero() || s.coldHandoffSince.After(now) {
			s.coldHandoffSince = now
		}
		elapsed := now.Sub(s.coldHandoffSince)
		if elapsed >= coldHandoffTimeout {
			timedOut = true
		} else {
			remaining = coldHandoffTimeout - elapsed
		}
	}
	s.mu.Unlock()

	if timedOut {
		s.finishColdHandoff()
		return true
	}

	settingsReady := true
	if setStore != nil {
		setView := setStore.Snapshot().Value
		settingsReady = setView.Described || setView.Err != "" || !setView.ClientReady
	}
	wallpaperReady := false
	if settingsReady {
		wallpaperReady = s.prewarmColdHandoffWallpaper(c)
	}
	historyReady := snap.SelectedID == "" || !snap.HistoryLoading || snap.HistoryReady || snap.HistoryError != nil
	modelReady := !sessions.ModelCatalogPending()

	if settingsReady && wallpaperReady && historyReady && modelReady {
		s.finishColdHandoff()
		return true
	}
	if remaining > 0 {
		c.After(remaining)
	}
	return false
}

func (s *mainState) paintColdLoading(c *ui.Context) bool {
	s.mu.Lock()
	if !s.coldLoading || s.mainRoute != routeConversation || !s.settingsLaunchHandled && slices.Contains(os.Args, "--settings") {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()
	if s.tryCompleteColdHandoff(c) {
		return false
	}
	s.mu.Lock()
	win, sup, st, themeID, skipped := s.win, s.sup, s.bootStatus, s.themeID, s.coldStartSkipped
	connectionIssue := connectionUnavailableMessage(s.dshBootstrapError, s.dshNetworkOffline, s.dshMuxLost)
	logs := s.bootLogSnapshot()
	s.mu.Unlock()
	m, _, message, _ := s.hostStatusModel(win, st, sup, logs)
	m = openedColdPage(m, st, skipped, message)
	systemDark := mygo.App.IsReady() && mygo.Theme.IsDark()
	themed := theme.ResolveWithSystem(themeID, systemDark)
	m.Dark = themed.Scheme == theme.SchemeDark
	if st.State == harness.Ready {
		// The page stays up until the workbench baselines arrive; name the
		// readiness level it is waiting on (internal/app/startup).
		m.StatusText = lifecycleText("正在打开工作区…", "Opening workspace…")
		if s.startupLevel() < startup.EventsReady {
			m.StatusText = lifecycleText("正在连接会话服务…", "Connecting to the session service…")
		}
		if connectionIssue != "" {
			m.Status.State = harness.Failed
			m.StatusText = connectionIssue
			m.OnRetry = s.requestConnectionReconnect
			m.OnConfigure = func() { s.navigateMainRoute(routeStartupDiagnostics) }
		}
	} else if st.State == harness.Stopped && !st.RecoveryCancelled {
		m.Status.State = harness.Starting
	}
	s.mu.Lock()
	s.hadColdLoading = true
	s.lastColdDark = m.Dark
	s.lastColdFailed = m.Status.State == harness.Failed
	s.mu.Unlock()
	colors := themed.Active(systemDark)
	defer s.paintClosingOverlayLayer(c, colors)
	defer s.syncConfirmAsk(c, colors)
	boot.View(c, m)
	return true
}

func (s *mainState) paintBootReveal(c *ui.Context) {
	s.mu.Lock()
	if s.coldLoading {
		s.mu.Unlock()
		return
	}
	if s.hadColdLoading {
		s.hadColdLoading = false
		dark, failed := s.lastColdDark, s.lastColdFailed
		s.mu.Unlock()
		s.bootReveal.Trigger(dark, failed, c.Preferences().ReduceMotion)
	} else {
		s.mu.Unlock()
	}
	s.bootReveal.Paint(c)
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
