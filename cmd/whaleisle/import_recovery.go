package main

import (
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
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
		s.launcherStartErr = resumeText("导入恢复尚未完成，已保留所有副本，暂不能启动内核：", "Import recovery is incomplete. All copies were preserved and runtime startup is blocked: ") + err.Error()
		s.launcherImport = s.launcherStartErr
	} else {
		s.launcherStartErr = ""
		s.launcherImport = resumeText("已恢复中断的导入。已完成项目保留；未完成目录已回到原状。请检查选择后继续导入，或点击启动内核。", "An interrupted import was recovered. Completed items were kept and incomplete directories were restored. Review your selection to continue importing, or start the runtime.")
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	return false
}
