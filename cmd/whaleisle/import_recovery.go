package main

import (
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// Cold startup checks recovery even when auto-start is disabled. A recovered
// import pauses automatic Host start, so the user can inspect/rerun the selection;
// explicit subsequent Start is allowed after reconciliation has settled.
func (s *mainState) recoverImportBeforeStart() bool {
	recovered, err := importer.RecoverInterrupted(paths.DshHome())
	if err == nil && !recovered {
		s.mu.Lock()
		s.importRecoveryPending = false
		s.mu.Unlock()
		return true
	}
	s.mu.Lock()
	errMsg := ""
	if err != nil {
		errMsg = i18n.T(i18n.KeyCmdImportRecoveryIsIncompleteAllCopiesWer) + err.Error()
		s.launcherImport = errMsg
	} else {
		s.launcherImport = i18n.T(i18n.KeyCmdAnInterruptedImportWasRecoveredComplet)
	}
	s.importRecoveryPending = true
	win := s.win
	s.mu.Unlock()
	s.launcherController().SetStartError(false, err != nil, errMsg)
	if win != nil {
		win.Update(func() {})
	}
	return false
}
