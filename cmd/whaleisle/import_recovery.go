package main

import (
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/launcher"
)

// Cold startup checks recovery even when auto-start is disabled. A recovered
// import pauses on its tab once, so the user can inspect/rerun the selection;
// explicit subsequent Start is allowed after reconciliation has settled.
func (s *mainState) recoverImportBeforeStart() bool {
	recovered, err := importer.RecoverInterrupted(paths.DshHome())
	if err == nil && !recovered {
		return true
	}
	s.mu.Lock()
	s.launcherOpen = true
	s.launcherTab = launcher.TabImport
	s.launcherStartBusy = false
	s.launcherStartFailed = err != nil
	if err != nil {
		s.launcherStartErr = i18n.T(i18n.KeyCmdImportRecoveryIsIncompleteAllCopiesWer) + err.Error()
		s.launcherImport = s.launcherStartErr
	} else {
		s.launcherStartErr = ""
		s.launcherImport = i18n.T(i18n.KeyCmdAnInterruptedImportWasRecoveredComplet)
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	return false
}
