package launcherctl

import (
	"fmt"
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/i18n"
)

// RecoveryState tracks persistent cold-start recovery state and deduplicated status writes.
type RecoveryState struct {
	Store              *startuprec.Store
	LoadError          string
	SaveError          string
	RecordedGeneration int
	RecordedState      harness.State
	Recorded           bool
}

// OpenStartupRecovery opens startup recovery state and reconciles plugin recovery against buildVersion.
func (c *Controller) OpenStartupRecovery(dir string, cfg *config.Store, buildVersion string) {
	store, err := startuprec.Open(dir)
	if err == nil && cfg != nil {
		err = cfg.ReconcilePluginRecovery(buildVersion)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recovery.Store = store
	if err != nil {
		c.recovery.LoadError = startuprec.RedactLine(err.Error())
	}
}

// ColdStartupAllowed returns whether cold auto-start should proceed without user intervention.
func (c *Controller) ColdStartupAllowed(cfg config.Config, importReady bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := &c.recovery
	failed := r.LoadError != "" || r.Store == nil
	if r.Store != nil {
		failed = failed || r.Store.Last().Failed()
	}
	if failed || cfg.SkipUserPlugins {
		c.startFailed = failed
		c.startErr = i18n.T(i18n.KeyCmdThePreviousStartDidNotFullyRecoverAut)
		if r.LoadError != "" {
			c.startErr = r.LoadError + "\n" + i18n.T(i18n.KeyCmdTheRecoveryFileWasPreservedRepairItAn)
		}
		if cfg.SkipUserPlugins {
			c.note = i18n.T(i18n.KeyCmdSkipUserPluginsRecoveryIsSavedStartIn)
		}
	}
	return startuprec.ShouldAutoStart(cfg.Workspace, cfg.AutoStartDesktop, importReady, failed, cfg.SkipUserPlugins)
}

// RecordStartupStatus persists the settled boot status for the current supervisor
// generation once. Disk I/O executes with c.mu unlocked under recoveryDiskMu.
func (c *Controller) RecordStartupStatus(st harness.Status, logs []string, buildVersion string) bool {
	if st.State != harness.Ready && st.State != harness.Failed {
		c.mu.Lock()
		failed := c.recovery.LoadError != "" || c.recovery.SaveError != ""
		c.mu.Unlock()
		return failed
	}

	c.recoveryDiskMu.Lock()
	defer c.recoveryDiskMu.Unlock()

	c.mu.Lock()
	r := &c.recovery
	if r.Recorded && r.RecordedGeneration == st.Generation && r.RecordedState == st.State {
		failed := r.LoadError != "" || r.SaveError != ""
		c.mu.Unlock()
		return failed
	}
	r.Recorded = true
	r.RecordedGeneration = st.Generation
	r.RecordedState = st.State
	store := r.Store
	c.mu.Unlock()

	reason := ""
	if st.Err != nil {
		reason = st.Err.Error()
	}
	fullReady := st.State == harness.Ready && !st.Recovery
	if st.Recovery && reason == "" {
		reason = i18n.T(i18n.KeyCmdRecoveryModeIsReadyUserPluginsRemainS)
	}
	var err error
	if store == nil {
		err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdStartupRecoveryUnavailable))
	} else {
		err = store.Record(fullReady, reason, logs, buildVersion)
	}

	c.mu.Lock()
	r.SaveError = ""
	if err != nil {
		r.SaveError = startuprec.RedactLine(err.Error())
		c.note = i18n.T(i18n.KeyCmdCouldNotSaveStartupOutcome) + r.SaveError
	}
	if st.Recovery {
		c.note = i18n.T(i18n.KeyCmdRecoveryModeIsReadyUserPluginsAreStil)
	}
	failed := r.LoadError != "" || r.SaveError != ""
	c.mu.Unlock()
	return failed
}

// RecoveryStore returns the active startup recovery store handle, if opened.
func (c *Controller) RecoveryStore() *startuprec.Store {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.recovery.Store
}

// ResetRecoveryRecording resets recorded-once status before starting a fresh supervisor.
func (c *Controller) ResetRecoveryRecording() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.recovery.Recorded = false
	c.mu.Unlock()
}

// RedactRecoveryReason strips sensitive strings from startup failure reasons.
func RedactRecoveryReason(reason string) string {
	return startuprec.RedactLine(strings.TrimSpace(reason))
}
