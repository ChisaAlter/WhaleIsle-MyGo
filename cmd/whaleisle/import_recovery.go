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
	errMsg := ""
	if err != nil {
		errMsg = i18n.T(i18n.KeyCmdImportRecoveryIsIncompleteAllCopiesWer) + err.Error()
		s.launcherImport = errMsg
	} else {
		s.launcherImport = i18n.T(i18n.KeyCmdAnInterruptedImportWasRecoveredComplet)
	}
	win := s.win
	s.mu.Unlock()
	s.launcherController().SetImportRecoveryStatus(launcher.TabImport, false, err != nil, errMsg)
	if win != nil {
		win.Update(func() {})
	}
	return false
}
