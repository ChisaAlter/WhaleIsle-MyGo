package main

import (
	"fmt"
	"strings"
	"whaleisle/internal/ui/i18n"

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
		s.launcherStartErr = i18n.T(i18n.KeyCmdThePreviousStartDidNotFullyRecoverAut)
		if r.loadError != "" {
			s.launcherStartErr = r.loadError + "\n" + i18n.T(i18n.KeyCmdTheRecoveryFileWasPreservedRepairItAn)
		}
		if cfg.SkipUserPlugins {
			s.launcherNote = i18n.T(i18n.KeyCmdSkipUserPluginsRecoveryIsSavedStartIn)
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
		reason = i18n.T(i18n.KeyCmdRecoveryModeIsReadyUserPluginsRemainS)
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
		s.launcherNote = i18n.T(i18n.KeyCmdCouldNotSaveStartupOutcome) + r.saveError
	}
	if st.Recovery {
		s.launcherNote = i18n.T(i18n.KeyCmdRecoveryModeIsReadyUserPluginsAreStil)
	}
	return r.loadError != "" || r.saveError != ""
}

func recoveryReason(reason string) string { return startuprec.RedactLine(strings.TrimSpace(reason)) }
