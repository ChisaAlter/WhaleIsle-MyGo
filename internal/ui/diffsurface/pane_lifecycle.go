package diffsurface

import "whaleisle/internal/desktop/git"

// CopyPanePresentation intentionally excludes commit/remote drafts, reviewed
// operations, progress pointers and async request ownership.
func (w *Workbench) CopyPanePresentation() Workbench {
	copy := Workbench{Root: w.Root, BaseRef: w.BaseRef, Staged: w.Staged, AgainstHEAD: w.AgainstHEAD, Loaded: w.Loaded, Repository: w.Repository, Changes: append([]git.Change(nil), w.Changes...), Branches: append([]git.Branch(nil), w.Branches...)}
	copy.View.Split, copy.View.Wrap = w.View.Split, w.View.Wrap
	return copy
}
func (w *Workbench) ClosePane() {
	w.cancelBranchList()
	w.cancelRepositoryStatus()
	w.CancelComparison()
	w.cancelCommitPreparation()
	w.cancelRemotePreparation()
}
