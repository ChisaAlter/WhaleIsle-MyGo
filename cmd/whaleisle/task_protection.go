package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/desktop/taskguard"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
)

func (s *mainState) taskProtectionError(win *mygo.Window, err error) {
	if err == nil || err == taskcontrol.ErrCancelled {
		return
	}
	s.mu.Lock()
	s.launcherStartErr = i18n.T(i18n.KeyCmdActionDidNotCompleteTheWindowWasKept) + err.Error()
	s.launcherStartFailed = true
	s.launcherOpen = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) confirmHostTasks(ctx context.Context, win *mygo.Window, operation string, i taskcontrol.Inspection) (bool, error) {
	s.mu.Lock()
	hasWindow := s.win != nil
	s.mu.Unlock()
	if win == nil || !hasWindow {
		return false, errors.New(i18n.T(i18n.KeyCmdTaskConfirmationWindowMissing))
	}
	label := i18n.T(i18n.KeyCmdQuit)
	if operation == "restart" {
		label = i18n.T(i18n.KeyCmdRestart)
	} else if operation == "update" {
		label = i18n.T(i18n.KeyCmdInstallUpdateAndQuit)
	} else if operation == "component" {
		label = i18n.T(i18n.KeyCmdChangeWhalebridgeComponent)
	} else if operation == "stop" {
		label = i18n.T(i18n.KeyCmdStopRuntime)
	}
	lines := []string{}
	if operation == "stop" {
		lines = append(lines, i18n.T(i18n.KeyCmdThisStopsTheHostRuntimeTheDesktopAppl))
	}
	if i.Unknown != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdHostTaskProtectionIsUnavailableBackgro), i.Unknown)
	}
	lines = append(lines, taskInspectionSummary(i))
	missing := []string{}
	for key, value := range i.Coverage {
		if value != "ok" && value != "intentional-disabled" {
			missing = append(missing, key+": "+value)
		}
	}
	if i.Unknown == "" && i.HostGeneration != "" {
		for _, key := range []string{"agents", "jobs", "schedule", "bots"} {
			if _, ok := i.Coverage[key]; !ok {
				missing = append(missing, key+i18n.T(i18n.KeyCmdTaskCoverageUnknown))
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		lines = append(lines, i18n.T(i18n.KeyCmdTheseChecksAreUnknown)+strings.Join(missing, ", "))
	}
	for _, item := range append(append([]taskcontrol.Work(nil), i.ActiveWork...), i.ScheduledWork...) {
		lines = append(lines, taskWorkDescription(item))
	}
	lines = append(lines, i18n.T(i18n.KeyCmdContinuingMayInterruptWorkThisDoesNot)+label+"?")
	// Every caller reaches this through a background goroutine, so the in-app
	// confirmation can be awaited here; a UI-thread caller would deadlock.
	approved := s.askConfirmBlocking(confirmAskRequest{
		Context:      ctx,
		Owner:        win,
		Title:        label,
		Body:         strings.Join(lines, "\n\n"),
		ConfirmLabel: i18n.T(i18n.KeyCmdContinue) + label,
		CancelLabel:  i18n.T(i18n.KeyCmdCancel),
		Danger:       true,
	})
	return approved, nil
}

func taskInspectionSummary(i taskcontrol.Inspection) string {
	if i.Unknown != "" {
		return fmt.Sprintf(i18n.T(i18n.KeyCmdKnownLocalRunningWorkSessionsAndTermin), i.LocalRunning)
	}
	return fmt.Sprintf(lifecycleText(
		"本地运行工作（会话和终端）：%d；Host 活动工作：%d；计划工作：%d。",
		"Known local running work (sessions and terminals): %d; Host active work: %d; scheduled work: %d."),
		i.LocalRunning, len(i.ActiveWork), len(i.ScheduledWork))
}

func taskWorkDescription(item taskcontrol.Work) string {
	parts := []string{item.Kind}
	for _, value := range []string{item.Title, item.ID, item.Owner, item.Detail, item.SessionID, item.ScheduledAt} {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	if item.Count > 0 {
		parts = append(parts, fmt.Sprintf("× %d", item.Count))
	}
	if item.Due {
		parts = append(parts, lifecycleText("已到期", "Due"))
	}
	if item.Recurring {
		parts = append(parts, lifecycleText("重复任务", "Recurring"))
	}
	return strings.Join(parts, " · ")
}

func lifecycleText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

func (s *mainState) withTaskProtection(win *mygo.Window, sup *harness.Supervisor, operation string, commit func() error) bool {
	s.mu.Lock()
	busy := s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		s.taskProtectionError(win, errors.New(i18n.T(i18n.KeyCmdImportIsStillRunningWaitForCompletion)))
		return false
	}
	did, err := s.protectTaskAction(context.Background(), win, sup, operation, commit)
	s.taskProtectionError(win, err)
	return did && err == nil
}

// Include shell-owned PTYs: the Host cannot inspect them, and hidden terminal
// panes continue running until an approved lifecycle commit closes them.
func (s *mainState) localTaskRunningCount() int {
	count := s.sessionRunningCount()
	s.mu.Lock()
	count += s.terminalSessions.RunningCount()
	runner := s.proactivePanel.runner
	s.mu.Unlock()
	if runner != nil && runner.DeliveryRunning() {
		count++
	}
	return count
}

// Shared by approved Stop, Restart and import commits, matching the original
// desktop's PTY cleanup and revoking the old Host's authenticated view.
func (s *mainState) closeTaskProtectedResources() error {
	s.stopPhoneRuntime()
	s.mu.Lock()
	if s.terminalVisible && s.layout.Bottom > terminalMinBottom {
		s.terminalSavedBottom = s.layout.Bottom
	}
	s.terminalVisible = false
	s.layout.Bottom = 0
	s.mu.Unlock()
	s.closeTerminal()
	// Independent browser/document previews retain their own lifetime.
	s.mu.Lock()
	runtimeView := s.webviewHost
	if runtimeView != nil && runtimeView.HasSession() {
		s.lastWebViewURL = runtimeView.URL()
		s.webviewHost = nil
		s.pluginViewResumeAfterRestart = s.webviewOpen && !s.browserPreviewOpen
		s.webviewHostErr = ""
	} else {
		runtimeView = nil
	}
	s.mu.Unlock()
	if runtimeView != nil {
		return runtimeView.Close()
	}
	return nil
}

// Imports already own the import slot. Reuse the same task inspection, drain
// and confirmation without rejecting that owner as a competing operation.
func (s *mainState) protectTaskAction(ctx context.Context, win *mygo.Window, sup *harness.Supervisor, operation string, commit func() error) (bool, error) {
	s.mu.Lock()
	proactiveRunner := s.proactivePanel.runner
	s.mu.Unlock()
	if proactiveRunner != nil {
		defer proactiveRunner.FreezeAdmissions()()
	}
	var did bool
	var err error
	if sup != nil {
		did, err = sup.ProtectTaskAction(ctx, operation, s.localTaskRunningCount, func(i taskcontrol.Inspection) (bool, error) { return s.confirmHostTasks(ctx, win, operation, i) }, commit)
	} else {
		// No owned Host exists. Preserve the original local-session confirmation.
		label := i18n.T(i18n.KeyCmdQuit)
		if operation == "restart" {
			label = i18n.T(i18n.KeyCmdRestart)
		} else if operation == "update" {
			label = i18n.T(i18n.KeyCmdInstallUpdateAndQuit)
		} else if operation == "component" {
			label = i18n.T(i18n.KeyCmdChangeWhalebridgeComponent)
		} else if operation == "stop" {
			label = i18n.T(i18n.KeyCmdStopRuntime)
		}
		running := s.localTaskRunningCount()
		did, err = taskguard.ConfirmRunning(running, func() (bool, error) {
			return s.askConfirmBlocking(confirmAskRequest{Context: ctx, Owner: win, Title: label, Body: fmt.Sprintf(i18n.T(i18n.KeyCmdLocalWorkItemsSessionsAndTerminalsAre), running), ConfirmLabel: label, CancelLabel: i18n.T(i18n.KeyCmdCancel), Danger: true}), nil
		})
		if did && err == nil {
			err = commit()
			did = err == nil
		}
	}
	return did, err
}

func (s *mainState) quitProtected(win *mygo.Window, sup *harness.Supervisor) bool {
	s.mu.Lock()
	busy := s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		s.taskProtectionError(win, errors.New(i18n.T(i18n.KeyCmdImportBeforeQuit)))
		return false
	}
	// Save before acquiring the Host admission lock: a disk-recovery dialog must
	// not leave a lock held indefinitely or let its TTL expire before shutdown.
	if !s.saveResumeBeforeClose(win) {
		return false
	}
	return s.withTaskProtection(win, sup, "quit", func() error {
		s.mu.Lock()
		busy := s.launcherImportBusy
		same := s.sup == sup
		s.mu.Unlock()
		if busy || !same {
			return errors.New(i18n.T(i18n.KeyCmdRuntimeChangedRetryQuit))
		}
		s.finalizeQuit(sup)
		return nil
	})
}

// stopProtected is the launcher's explicit Stop entry. Keep the window and
// running Host intact until the same admission checks used by restart succeed.
func (s *mainState) stopProtected(win *mygo.Window, sup *harness.Supervisor) {
	if sup == nil {
		return
	}
	s.mu.Lock()
	if s.taskRestartPending || s.quitting {
		s.mu.Unlock()
		return
	}
	s.taskRestartPending = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.taskRestartPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		s.withTaskProtection(win, sup, "stop", func() error {
			s.mu.Lock()
			if s.launcherImportBusy || s.quitting || s.sup != sup {
				s.mu.Unlock()
				return errors.New(i18n.T(i18n.KeyCmdRuntimeChangedRetryStop))
			}
			s.mu.Unlock()
			if err := s.closeTaskProtectedResources(); err != nil {
				return err
			}
			sup.Stop()
			s.mu.Lock()
			s.launcherStartBusy = false
			s.launcherStartFailed = false
			s.launcherStartErr = ""
			s.mu.Unlock()
			return nil
		})
	}()
	if win != nil {
		win.Update(func() {})
	}
}

// installStagedUpdate hands off only verified bytes and uses the already-held
// lifecycle lock for final cleanup, without reentering quitProtected.
func (s *mainState) installStagedUpdate(win *mygo.Window, staged desktopupdate.ApplyResult) bool {
	return s.installStagedUpdateWithConsent(win, staged, false)
}

// confirmedByInstallButton is used only by the explicit, verified Versions
// install action. Task interruption and resume-save protections still apply.
func (s *mainState) installStagedUpdateWithConsent(win *mygo.Window, staged desktopupdate.ApplyResult, confirmedByInstallButton bool) bool {
	s.mu.Lock()
	if s.taskRestartPending || s.quitting || s.launcherImportBusy {
		s.mu.Unlock()
		return false
	}
	s.taskRestartPending = true
	sup := s.sup
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
	if !confirmedByInstallButton {
		if !s.askConfirmBlocking(confirmAskRequest{Owner: win, Title: i18n.T(i18n.KeyCmdInstallUpdate), Body: i18n.T(i18n.KeyCmdTheUpdateHasBeenDownloadedAndVerified), ConfirmLabel: i18n.T(i18n.KeyCmdInstallAndQuit), CancelLabel: i18n.T(i18n.KeyCmdLater)}) {
			return false
		}
	}
	if !s.saveResumeBeforeClose(win) {
		return false
	}
	// The install answer is the milestone worth keeping: the journal never
	// records consent that was not given (update-journal.js install-confirmed).
	s.recordUpdateAction(desktopupdate.JournalActionInstallConfirmed)
	did := s.withTaskProtection(win, sup, "update", func() error {
		s.mu.Lock()
		changed := s.launcherImportBusy || s.quitting || s.sup != sup
		s.mu.Unlock()
		if changed {
			return errors.New(i18n.T(i18n.KeyCmdRuntimeChangedRetryUpdate))
		}
		if _, err := desktopupdate.StartStaged(staged); err != nil {
			return err
		}
		s.markNativeInstallerLaunched(staged)
		s.finalizeQuit(sup)
		return nil
	})
	if did {
		s.quitAfterCleanup()
	}
	return did
}
