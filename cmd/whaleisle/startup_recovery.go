package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/startuprec"
)

type startupRecoveryState struct {
	store              *startuprec.Store
	loadError          string
	saveError          string
	recordedGeneration int
	recordedState      harness.State
	recorded           bool
	updateHold         bool
	updateAttention    startuprec.UpdateAttention
	updateVendorReady  bool
	updateNotice       *mygo.Notification
	updateNoticeWindow *mygo.Window
	updateNoticeTicket startuprec.UpdateAttentionTicket
	updateWatchWindow  *mygo.Window
	updateWatchOff     []func()
}

func (s *mainState) openStartupRecovery(dir string, cfg *config.Store) {
	store, err := startuprec.Open(dir)
	if err == nil {
		err = cfg.ReconcilePluginRecovery(buildVersion)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startupRecovery.store = store
	if err != nil {
		s.startupRecovery.loadError = startuprec.RedactLine(err.Error())
	}
}

func (s *mainState) coldStartupAllowed(cfg config.Config, importReady bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &s.startupRecovery
	failed := r.loadError != "" || r.store == nil
	if r.store != nil {
		failed = failed || r.store.Last().Failed()
	}
	if failed || cfg.SkipUserPlugins {
		s.launcherStartFailed = failed
		s.launcherStartErr = resumeText("上次启动未完整恢复，已暂停自动启动。请查看诊断后重试。", "The previous start did not fully recover. Auto-start is paused; review diagnostics and retry.")
		if r.loadError != "" {
			s.launcherStartErr = r.loadError + "\n" + resumeText("恢复文件已保留，请修复后重新打开应用。", "The recovery file was preserved; repair it and reopen the app.")
		}
		if cfg.SkipUserPlugins {
			s.launcherNote = resumeText("已保留跳过用户插件的恢复模式。可继续以恢复模式启动，或清除恢复标记后重试全部插件。", "Skip-user-plugins recovery is saved. Start in recovery mode, or clear recovery and retry all plugins.")
		}
	}
	return startuprec.ShouldAutoStart(cfg.Workspace, cfg.AutoStartDesktop, importReady, failed, cfg.SkipUserPlugins)
}

// Called with the main state lock after accepting the current supervisor's
// generation/state. A recovery Ready is deliberately not recorded as healthy.
func (s *mainState) recordStartupStatusLocked(st harness.Status) bool {
	r := &s.startupRecovery
	if st.State != harness.Ready && st.State != harness.Failed {
		return r.loadError != "" || r.saveError != ""
	}
	if r.recorded && r.recordedGeneration == st.Generation && r.recordedState == st.State {
		return r.loadError != "" || r.saveError != ""
	}
	r.recorded = true
	r.recordedGeneration = st.Generation
	r.recordedState = st.State
	logs := []string(nil)
	if s.sup != nil {
		logs = s.sup.CurrentStartLogs()
	}
	reason := ""
	if st.Err != nil {
		reason = st.Err.Error()
	}
	fullReady := st.State == harness.Ready && !st.Recovery
	if st.Recovery && reason == "" {
		reason = resumeText("仅恢复模式就绪；用户插件仍被跳过。", "Recovery mode is ready; user plugins remain skipped.")
	}
	var err error
	if r.store == nil {
		err = fmt.Errorf("startup recovery storage is unavailable")
	} else {
		err = r.store.Record(fullReady, reason, logs, buildVersion)
	}
	r.saveError = ""
	if err != nil {
		r.saveError = startuprec.RedactLine(err.Error())
		s.launcherNote = resumeText("启动结果未能保存：", "Could not save startup outcome: ") + r.saveError
	}
	if st.Recovery {
		s.launcherNote = resumeText("恢复模式已就绪；用户插件仍被跳过。请在插件页检查原因并明确恢复。", "Recovery mode is ready; user plugins are still skipped. Review the cause on the Plugins page before restoring them.")
	}
	return r.loadError != "" || r.saveError != ""
}

func recoveryReason(reason string) string { return startuprec.RedactLine(strings.TrimSpace(reason)) }
