package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"

	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

// installDir is the directory the running executable lives in. A data
// directory inside it would be removed by an uninstall or replaced by an
// update, so the location chooser refuses it.
func installDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// dataLocationAccess builds the 数据位置 block for the General page from the
// state of this process. The block is rebuilt per frame, so every field is a
// snapshot rather than a draft.
func (s *mainState) dataLocationAccess(win *mygo.Window) *settings.DataLocationAccess {
	s.mu.Lock()
	errText, action := s.dataLocationErr, s.dataLocationAction
	sup := s.sup
	s.mu.Unlock()
	current := paths.UserDataDir()
	access := &settings.DataLocationAccess{
		Current:    current,
		Default:    paths.DefaultUserDataDir(),
		Err:        errText,
		LastAction: action,
		OnChange:   func() { s.chooseDataLocation(win, sup) },
		OnOpen:     func() { s.openDataLocation(current) },
		OnReset:    func() { s.resetDataLocation(win, sup) },
	}
	// A recorded location that is not in use is named, so the fallback to the
	// default is visible instead of silent; a record that cannot be read at
	// all is reported the same way.
	if issue, ok := paths.CachedDataLocationIssue(); ok && !strings.EqualFold(issue.Path, current) {
		access.Recorded = issue.Path
		access.RecordBroken = issue.Unreadable
	}
	// A run whose directory comes from the environment or from isolated mode
	// cannot change it: the override outranks any record. Importing stays
	// available, because it only writes into the fixed directory.
	if !s.dataLocationEditable() {
		access.ReadOnly = true
		access.ReadOnlyReason = i18n.T(i18n.KeyDataLocationReadOnly)
	}
	if plan, ok := s.legacyImportOffer(); ok {
		access.ImportAvailable = true
		access.OnImport = func() { s.importLegacyData(win, plan) }
	}
	if plan, ok := s.previousImportOffer(); ok {
		access.PreviousPath = plan.Source
		access.OnImportPrevious = func() { s.importLegacyData(win, plan) }
	}
	return access
}

// previousImportOffer plans the copy from the location this profile replaced,
// so a location change never strands the sessions it left behind. Cached for
// the same reason as legacyImportOffer: planning walks the source directory.
func (s *mainState) previousImportOffer() (importer.FirstRunPlan, bool) {
	s.mu.Lock()
	if s.previousImportKnown {
		plan, ok := s.previousImport, s.previousImportOK
		s.mu.Unlock()
		return plan, ok
	}
	s.mu.Unlock()
	previous := paths.PreviousDataDir()
	var plan importer.FirstRunPlan
	ok := previous != "" && importer.HasLegacyData(previous)
	if ok {
		plan = importer.PlanFirstRun(paths.UserDataDir(), previous)
		ok = plan.Source != "" && plan.Dest != ""
	}
	s.mu.Lock()
	s.previousImport, s.previousImportOK, s.previousImportKnown = plan, ok, true
	s.mu.Unlock()
	return plan, ok
}

// dataLocationEditable reports whether this run may record a data location.
// WHALEISLE_USER_DATA and isolated mode outrank the record, so writing one
// there would be ignored on the next launch and would edit the installed
// application's record.
func (s *mainState) dataLocationEditable() bool {
	return strings.TrimSpace(os.Getenv(paths.EnvUserData)) == "" && !isolatedDesktopRun()
}

// legacyImportOffer returns the manual legacy copy plan, planned once. The
// legacy profile holds thousands of entries, so this walk may not run per
// frame on the render thread.
func (s *mainState) legacyImportOffer() (importer.FirstRunPlan, bool) {
	s.mu.Lock()
	if s.legacyImportKnown {
		plan, ok := s.legacyImport, s.legacyImportOK
		s.mu.Unlock()
		return plan, ok
	}
	s.mu.Unlock()
	plan, ok := importer.FirstRunOffer(paths.UserDataDir(), importer.ElectronUserDataDir())
	s.mu.Lock()
	s.legacyImport, s.legacyImportOK, s.legacyImportKnown = plan, ok, true
	s.mu.Unlock()
	return plan, ok
}

// invalidateLegacyImport drops the planned copy after the directories it was
// built from changed.
func (s *mainState) invalidateLegacyImport() {
	s.mu.Lock()
	s.legacyImportKnown, s.legacyImport, s.legacyImportOK = false, importer.FirstRunPlan{}, false
	// The previous location source is cached the same way: after the
	// directories it was built from changed, both plans are stale.
	s.previousImportKnown, s.previousImport, s.previousImportOK = false, importer.FirstRunPlan{}, false
	s.mu.Unlock()
}

func (s *mainState) setDataLocationOutcome(errText, action string) {
	s.mu.Lock()
	if errText != s.dataLocationErr || action != s.dataLocationAction {
		s.dataLocationErr, s.dataLocationAction = errText, action
	}
	s.mu.Unlock()
}

// dataLocationMessage maps a refusal to the sentence the settings row shows.
// Every sentence carries its own %s slots, so a path reads as part of the
// sentence instead of being glued on after a full stop.
func dataLocationMessage(err error) string {
	var problem *paths.DataLocationError
	if !errors.As(err, &problem) {
		log.Printf("data location: %v", err)
		return i18n.T(i18n.KeyDataLocationRefusedGeneric)
	}
	// The raw cause of a refusal (an OS error, a probe file name) is for the
	// log; the row shows the reason and the paths a person can act on.
	if problem.Other != "" {
		log.Printf("data location: %q refused (%s): %s", problem.Path, problem.Problem, problem.Other)
	}
	switch problem.Problem {
	case paths.DataLocationEmpty:
		return i18n.T(i18n.KeyDataLocationRefusedEmpty)
	case paths.DataLocationRelative:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedRelative), problem.Path)
	case paths.DataLocationInvalid:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedInvalid), problem.Path)
	case paths.DataLocationNotDirectory:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedNotDir), problem.Path)
	case paths.DataLocationNotWritable:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedNotWritable), problem.Path)
	case paths.DataLocationRemoteVolume:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedRemote), problem.Path)
	case paths.DataLocationNested:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedNested), problem.Path, problem.Other)
	case paths.DataLocationInsideInstall:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedInstall), problem.Path, problem.Other)
	case paths.DataLocationContainsRecord:
		return fmt.Sprintf(i18n.T(i18n.KeyDataLocationRefusedRecord), problem.Path)
	}
	return i18n.T(i18n.KeyDataLocationRefusedGeneric)
}

// refuseDataLocationEdit reports whether this run may record a location at all
// and explains why not; every write below goes through it.
func (s *mainState) refuseDataLocationEdit(win *mygo.Window) bool {
	if s.dataLocationEditable() {
		return false
	}
	s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationReadOnly), "")
	if win != nil {
		win.Update(func() {})
	}
	return true
}

// chooseDataLocation asks for a directory with the native picker, refuses an
// unusable choice with the reason, and otherwise applies it.
func (s *mainState) chooseDataLocation(win *mygo.Window, sup *harness.Supervisor) {
	if s.refuseDataLocationEdit(win) {
		return
	}
	current := paths.UserDataDir()
	picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:      win,
		Title:       i18n.T(i18n.KeyDataLocationTitle),
		DefaultPath: current,
		Directory:   true,
	})
	if err != nil || len(picked) == 0 {
		return
	}
	chosen := strings.TrimSpace(picked[0])
	if err := paths.ValidateDataLocationChoice(chosen, current, installDir()); err != nil {
		s.setDataLocationOutcome(dataLocationMessage(err), "")
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	s.applyDataLocation(win, sup, chosen, false)
}

// resetDataLocation returns to this build's default directory through the same
// protected path: the record is cleared only after the task confirmation.
func (s *mainState) resetDataLocation(win *mygo.Window, sup *harness.Supervisor) {
	if s.refuseDataLocationEdit(win) {
		return
	}
	def := paths.DefaultUserDataDir()
	if err := paths.ValidateDataLocationChoice(def, paths.UserDataDir(), installDir()); err != nil {
		s.setDataLocationOutcome(dataLocationMessage(err), "")
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	s.applyDataLocation(win, sup, def, true)
}

// applyDataLocation records dir and relaunches. The record is written inside
// the task-protected commit, so a running task blocks the change exactly as it
// blocks a restart, and nothing is written when the user declines. restore
// clears the record instead, which returns the next launch to the default.
func (s *mainState) applyDataLocation(win *mygo.Window, sup *harness.Supervisor, dir string, restore bool) {
	if s.refuseDataLocationEdit(win) {
		return
	}
	s.mu.Lock()
	blocked := s.taskRestartPending || s.quitting || s.launcherImportBusy
	if blocked {
		s.mu.Unlock()
		s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationBusy), "")
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	s.taskRestartPending = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer func() {
			s.mu.Lock()
			s.taskRestartPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		if !restore {
			// Name both directories before the restart: the new one starts
			// empty, and the old one is not carried over or deleted.
			if !s.askConfirmBlocking(confirmAskRequest{
				Owner:        win,
				Title:        i18n.T(i18n.KeyDataLocationTitle),
				Body:         fmt.Sprintf(i18n.T(i18n.KeyDataLocationConfirmBody), paths.UserDataDir(), dir),
				ConfirmLabel: i18n.T(i18n.KeyDataLocationConfirmAction),
				CancelLabel:  i18n.T(i18n.KeyCmdCancel),
				Danger:       true,
			}) {
				return
			}
		}
		if !s.saveResumeBeforeClose(win) {
			return
		}
		did := s.withTaskProtection(win, sup, "restart", func() error {
			s.mu.Lock()
			changed := s.launcherImportBusy || s.quitting || s.sup != sup
			s.mu.Unlock()
			if changed {
				return localized(i18n.T(i18n.KeyCmdRuntimeChangedRetryRestart))
			}
			if restore {
				if err := paths.ClearRecordedDataDir(); err != nil {
					log.Printf("data location: clearing the record: %v", err)
					return localized(i18n.T(i18n.KeyDataLocationWriteFailed))
				}
			} else {
				// RecordDataDir creates the directory as well: the record must
				// never name a directory that is not there, because the next
				// launch would fall back to the default instead of opening it.
				if err := paths.RecordDataDir(dir, paths.UserDataDir()); err != nil {
					log.Printf("data location: recording %q: %v", dir, err)
					return localized(i18n.T(i18n.KeyDataLocationWriteFailed))
				}
			}
			s.mu.Lock()
			s.relaunchPending = true
			s.dataLocationErr = ""
			s.dataLocationAction = i18n.T(i18n.KeyDataLocationApplying)
			s.mu.Unlock()
			s.finalizeQuit(sup)
			return nil
		})
		if did {
			s.quitAfterCleanup()
			return
		}
		s.setDataLocationOutcome("", "")
	}()
}

// openDataLocation reveals the directory in the file manager. The directory is
// created first so a fresh profile opens instead of failing.
func (s *mainState) openDataLocation(dir string) {
	if strings.TrimSpace(dir) == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("data location: creating %q to open it: %v", dir, err)
		s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationOpenFailed), "")
		return
	}
	if err := desktopcomponents.OpenInFileManager(dir); err != nil {
		log.Printf("data location: opening %q: %v", dir, err)
		s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationOpenFailed), "")
		return
	}
	s.setDataLocationOutcome("", "")
}

// importLegacyData copies the original Whale Isle profile into this data
// directory through the same migration a first run uses. Nothing is moved or
// deleted, and an existing file is kept.
func (s *mainState) importLegacyData(win *mygo.Window, plan importer.FirstRunPlan) {
	s.mu.Lock()
	busy := s.dataLocationImporting || s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationBusy), "")
		return
	}
	if !s.askConfirmBlocking(confirmAskRequest{
		Owner:        win,
		Title:        i18n.T(i18n.KeyFirstRunImportTitle),
		Body:         fmt.Sprintf(i18n.T(i18n.KeyFirstRunImportBody), plan.Source, plan.Dest),
		ConfirmLabel: i18n.T(i18n.KeyFirstRunImportAccept),
		CancelLabel:  i18n.T(i18n.KeyFirstRunImportSkip),
	}) {
		return
	}
	s.mu.Lock()
	s.dataLocationImporting = true
	s.mu.Unlock()
	s.setDataLocationOutcome("", i18n.T(i18n.KeyDataLocationImportRunning))
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer func() {
			s.mu.Lock()
			s.dataLocationImporting = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		result, err := importer.RunFirstRun(context.Background(), plan)
		for _, line := range result.Journal {
			log.Println("legacy import:", line)
		}
		// The destination changed, so the planned copy is stale.
		s.invalidateLegacyImport()
		if err != nil {
			log.Printf("legacy import: %v", err)
			s.setDataLocationOutcome(i18n.T(i18n.KeyDataLocationImportFailed), "")
			return
		}
		s.setDataLocationOutcome("", fmt.Sprintf(i18n.T(i18n.KeyDataLocationImportSummary), result.Copied, result.Skipped, result.Failed))
	}()
}
