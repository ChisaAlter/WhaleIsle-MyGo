package frame

import "whaleisle/internal/ui/diffsurface"

// Persist the selection only. Patch text, commit drafts, review tickets and
// ongoing repository operations are runtime state of this occurrence.
type WorkspaceDiffState struct {
	Path      string                `json:"path,omitempty"`
	Root      string                `json:"root,omitempty"`
	Text      string                `json:"-"`
	Error     string                `json:"-"`
	Workbench diffsurface.Workbench `json:"-"`
}

func (w *WorkspacePanes) diffState(n *WorkspacePane) *WorkspaceDiffState {
	if n.Diff == nil {
		n.Diff = &WorkspaceDiffState{}
	}
	return n.Diff
}
func (d *WorkspaceDiffState) capture(m Model) {
	if d.Root != "" && d.Root != m.WorkspaceRoot {
		d.Workbench.ClosePane()
		*d = WorkspaceDiffState{Root: m.WorkspaceRoot}
		return
	}
	d.Path, d.Text, d.Error = derefString(m.DiffPath), derefString(m.DiffText), derefString(m.DiffErr)
	d.Root = m.WorkspaceRoot
	if m.GitWorkbench != nil {
		d.Workbench = *m.GitWorkbench
	}
}
func (d *WorkspaceDiffState) publish(m Model) {
	if m.DiffPath != nil {
		*m.DiffPath = d.Path
	}
	if m.DiffText != nil {
		*m.DiffText = d.Text
	}
	if m.DiffErr != nil {
		*m.DiffErr = d.Error
	}
	if m.GitWorkbench != nil {
		*m.GitWorkbench = d.Workbench
	}
}
func (d *WorkspaceDiffState) model(m Model) Model {
	if d.Root != "" && d.Root != m.WorkspaceRoot {
		d.Workbench.ClosePane()
		*d = WorkspaceDiffState{}
	}
	d.Root = m.WorkspaceRoot
	m.DiffPath, m.DiffText, m.DiffErr = &d.Path, &d.Text, &d.Error
	m.GitWorkbench = &d.Workbench
	return m
}
func (w *WorkspacePanes) beginDiffProjection(m Model) {
	if previous := w.Root.find(w.legacyFileID); previous != nil && previous.Diff != nil {
		previous.Diff.capture(m)
	}
	hasDiff := false
	w.Root.eachLeaf(func(n *WorkspacePane) { hasDiff = hasDiff || n.Diff != nil })
	current := w.diffState(w.Root.find(w.Active))
	if !hasDiff {
		current.capture(m)
	}
	current.publish(m)
}
func (w *WorkspacePanes) finishDiffProjection(m Model) {
	if n := w.Root.find(w.Active); n != nil {
		w.diffState(n).publish(m)
	}
}
func copyDiffPane(target, source *WorkspacePane) {
	if target.Kind != "diff" || source.Kind != "diff" || source.Diff == nil {
		return
	}
	d := source.Diff
	target.Diff = &WorkspaceDiffState{Path: d.Path, Root: d.Root, Text: d.Text, Error: d.Error, Workbench: d.Workbench.CopyPanePresentation()}
}
