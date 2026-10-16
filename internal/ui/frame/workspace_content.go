package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/diffsurface"
)

// WorkspaceFileState is one pane's editor, tab set, selection and viewport.
// FileTabs still owns the existing read/save queues and draft conflict checks.
type WorkspaceFileState struct {
	Tabs files.Tabs `json:"tabs"`
	Expanded map[string]bool `json:"expanded,omitempty"`
	Selected, Filter, Preview, Error, Draft, Status, Stat string
	Editing bool
	Mode files.EditViewMode
	Split, TreeSplit float32
	List ui.ListState `json:"-"`
	rendered bool
}

func (s *WorkspaceFileState) capture(m Model) {
	if m.FileTabs != nil { s.Tabs = *m.FileTabs }
	if m.FileExpanded != nil { s.Expanded = *m.FileExpanded }
	if m.SelectedFile != nil { s.Selected = *m.SelectedFile }
	if m.FileFilter != nil { s.Filter = *m.FileFilter }
	if m.FilePreview != nil { s.Preview = *m.FilePreview }
	if m.FilePreviewErr != nil { s.Error = *m.FilePreviewErr }
	if m.FileEditDraft != nil { s.Draft = *m.FileEditDraft }
	if m.FileEditStatus != nil { s.Status = *m.FileEditStatus }
	if m.FileStatNote != nil { s.Stat = *m.FileStatNote }
	if m.FileEditing != nil { s.Editing = *m.FileEditing }
	if m.FileEditViewMode != nil { s.Mode = *m.FileEditViewMode }
	if m.FileEditSplit != nil { s.Split = *m.FileEditSplit }
	if m.FilePreviewList != nil { s.List = *m.FilePreviewList }
}

func (s *WorkspaceFileState) publish(m Model) {
	if m.FileTabs != nil { *m.FileTabs = s.Tabs }
	if m.FileExpanded != nil { *m.FileExpanded = s.Expanded }
	if m.SelectedFile != nil { *m.SelectedFile = s.Selected }
	if m.FileFilter != nil { *m.FileFilter = s.Filter }
	if m.FilePreview != nil { *m.FilePreview = s.Preview }
	if m.FilePreviewErr != nil { *m.FilePreviewErr = s.Error }
	if m.FileEditDraft != nil { *m.FileEditDraft = s.Draft }
	if m.FileEditStatus != nil { *m.FileEditStatus = s.Status }
	if m.FileStatNote != nil { *m.FileStatNote = s.Stat }
	if m.FileEditing != nil { *m.FileEditing = s.Editing }
	if m.FileEditViewMode != nil { *m.FileEditViewMode = s.Mode }
	if m.FileEditSplit != nil { *m.FileEditSplit = s.Split }
	if m.FilePreviewList != nil { *m.FilePreviewList = s.List }
}

func (s *WorkspaceFileState) model(m Model) Model {
	if m.SelectedFile == nil || *m.SelectedFile != s.Selected {
		m.FileOpenWithApps = nil
	}
	m.FileTabs, m.FileExpanded = &s.Tabs, &s.Expanded
	m.SelectedFile, m.FileFilter = &s.Selected, &s.Filter
	m.FilePreview, m.FilePreviewErr = &s.Preview, &s.Error
	m.FileEditDraft, m.FileEditStatus, m.FileStatNote = &s.Draft, &s.Status, &s.Stat
	m.FileEditing, m.FileEditViewMode = &s.Editing, &s.Mode
	m.FileEditSplit, m.FilePreviewList = &s.Split, &s.List
	m.FileTreeSplit = &s.TreeSplit
	return m
}

func (s *WorkspaceFileState) unsaved() bool {
	if s == nil { return false }
	for _, workspace := range s.Tabs.Workspaces {
		if workspace == nil { continue }
		for _, tab := range workspace.Open {
			if tab != nil && tab.Dirty() { return true }
		}
	}
	return s.Editing && s.Draft != s.Preview
}

func (n *WorkspacePane) eachLeaf(visit func(*WorkspacePane)) {
	if n == nil { return }
	if n.First == nil { visit(n); return }
	n.First.eachLeaf(visit)
	n.Second.eachLeaf(visit)
}

func (w *WorkspacePanes) fileState(n *WorkspacePane, m Model) *WorkspaceFileState {
	if n.Files == nil {
		n.Files = &WorkspaceFileState{Expanded: map[string]bool{}, Split: 200, TreeSplit: m.Sizes.RightTree}
		if m.FileTabs != nil { n.Files.Tabs.Preferences = m.FileTabs.Preferences }
	}
	return n.Files
}

// The old main-state fields are an adapter for existing open-file entry points.
// They project only the focused pane; no two rendered panes share those fields.
func (w *WorkspacePanes) beginFileProjection(m Model) Model {
	w.beginDiffProjection(m)
	if previous := w.Root.find(w.legacyFileID); previous != nil && previous.Files != nil {
		previous.Files.capture(m)
	}
	active := w.Root.find(w.Active)
	hasFiles := false
	w.Root.eachLeaf(func(n *WorkspacePane) { hasFiles = hasFiles || n.Files != nil })
	state := w.fileState(active, m)
	if !hasFiles { state.capture(m) }
	state.publish(m)
	w.legacyFileID = active.ID

	// Existing callbacks retain all original session/Host checks. Publish at the
	// gesture, then collect their result into that same occurrence.
	run := func(action func()) {
		target := w.Root.find(w.Active)
		state := w.fileState(target, m)
		state.publish(m)
		action()
		state.capture(m)
		if m.RightPaneTab != nil && *m.RightPaneTab != target.Kind {
			w.selectKind(target, *m.RightPaneTab)
		}
		w.legacyFileID = target.ID
	}
	if open := m.OnOpenDeliverable; open != nil {
		m.OnOpenDeliverable = func(path string) { run(func() { open(path) }) }
	}
	if open := m.OnOpenPlanResource; open != nil {
		m.OnOpenPlanResource = func(doc planpreview.Document) { run(func() { open(doc) }) }
	}
	return m
}

func (w *WorkspacePanes) finishFileProjection(m Model) {
	w.finishDiffProjection(m)
	if n := w.Root.find(w.Active); n != nil {
		w.fileState(n, m).publish(m)
		w.legacyFileID = n.ID
	}
}

func workspaceContentModel(m Model, n *WorkspacePane) Model {
	w := m.Sizes.Workspace
	state := w.fileState(n, m)
	if m.Store != nil { state.Tabs.SyncPlanSession(m.Store.Snapshot().SelectedID) }
	local := w.diffState(n).model(state.model(m))
	if n.Kind == "diff" && !local.GitWorkbench.Loaded && *local.DiffPath != "" && *local.DiffText == "" {
		*local.DiffText, *local.DiffErr = diffsurface.LoadDiff(m.WorkspaceRoot, *local.DiffPath)
	}
	local.FileShortcuts = func() bool { return w.Active == n.ID }
	if open := local.OnOpenDoc; open != nil {
		local.OnOpenDoc = func(path string, kind workspacefs.DocKind, inWebView bool) {
			w.Active = n.ID
			open(path, kind, inWebView)
		}
	}
	if open := local.OnOpenAgentDocument; open != nil {
		local.OnOpenAgentDocument = func(token uint64, address session.AgentSessionAddress, path string) {
			w.Active = n.ID
			open(token, address, path)
		}
	}
	if open := local.OnRenderOfficePDF; open != nil {
		local.OnRenderOfficePDF = func(path string) { w.Active = n.ID; open(path) }
	}
	if m.NewWorkspaceBrowser != nil {
		local.BrowserContent = func(c *ui.Context) {
			w.browserFor(n, m.NewWorkspaceBrowser)
			n.browser.ViewDocked(c, m.ThemeID, m.SystemDark, func() {
				w.pending = func() { w.close(n.ID) }
			})
		}
	}
	return local
}

// BeginContentFrame is called even when an application page replaces frame.View.
func (w *WorkspacePanes) BeginContentFrame(binding, root string) {
	if w == nil { return }
	w.Root.eachLeaf(func(n *WorkspacePane) {
		if n.Files != nil {
			n.Files.rendered = false
			n.Files.Tabs.Bind(binding, root)
		}
		if n.browser != nil {
			n.browser.PrunePDFDocuments()
			n.browser.PruneMediaDocuments()
			n.browser.PruneHTMLDocuments()
			n.browser.BeginFrame()
		}
	})
}

func (w *WorkspacePanes) EndContentFrame() {
	if w == nil { return }
	w.Root.eachLeaf(func(n *WorkspacePane) {
		if n.Files != nil && !n.Files.rendered {
			n.Files.Tabs.CancelPlanRead()
			n.Files.Tabs.Pause()
		}
		if n.browser != nil { n.browser.EndFrame() }
	})
}

func (w *WorkspacePanes) PauseContent() {
	if w == nil { return }
	w.Root.eachLeaf(func(n *WorkspacePane) {
		if n.Files != nil { n.Files.Tabs.CancelPlanRead(); n.Files.Tabs.Pause() }
		if n.browser != nil { n.browser.Hide() }
	})
}

func (n *WorkspacePane) disposeContent() {
	if n.Diff != nil { n.Diff.Workbench.ClosePane() }
	if n.Files != nil { n.Files.Tabs.Dispose() }
	if n.browser != nil { n.browser.Close(); n.browser = nil }
}

func (w *WorkspacePanes) DisposeContent() {
	if w != nil { w.Root.eachLeaf(func(n *WorkspacePane) { n.disposeContent() }) }
}

func (w *WorkspacePanes) captureContent() {
	w.Root.eachLeaf(func(n *WorkspacePane) {
		if n.browser != nil { n.Browser = n.browser.WorkspaceSnapshot() }
	})
}

func (w *WorkspacePanes) restoreContent() error {
	var restoreErr error
	w.Root.eachLeaf(func(n *WorkspacePane) {
		if n.Files == nil || restoreErr != nil { return }
		raw, err := n.Files.Tabs.Export()
		if err == nil { err = n.Files.Tabs.Restore(raw) }
		if err != nil { restoreErr = err }
	})
	return restoreErr
}

func (w *WorkspacePanes) resetTo(n *WorkspacePane) {
	blocked := false
	w.Root.eachLeaf(func(other *WorkspacePane) {
		if other != n && other.Files.unsaved() { blocked = true }
	})
	if blocked {
		w.message = paneLabel("其他面板仍有未保存内容，请先保存或关闭它们。", "Other panes have unsaved drafts. Save or close them first.")
		return
	}
	w.Root.eachLeaf(func(other *WorkspacePane) { if other != n { other.disposeContent() } })
	w.Root, w.Active, w.Fullscreen, w.route = n, n.ID, "", n.Kind
	w.message = ""
}


// browserFor is shared by the renderer and explicit document-open gestures so
// the source occurrence is selected before any asynchronous read starts.
func (w *WorkspacePanes) browserFor(n *WorkspacePane, create func() *browserpreview.Controller) *browserpreview.Controller {
	if n.browser == nil {
		n.browser = create()
		n.browser.RestoreWorkspace(n.Browser)
		id := n.ID
		n.browser.SetOnFocus(func() {
			if pane := w.Root.find(id); pane != nil {
				w.Active, w.focusPending = pane.ID, pane.ID
			}
		})
	}
	return n.browser
}

func (w *WorkspacePanes) OpenDocument(source browserpreview.DocumentSource, create func() *browserpreview.Controller) {
	if w.Root == nil { w.syncRoute("files") }
	n := w.Root.find(w.Active)
	target := n
	var splitSource *WorkspacePane
	if source.AddressKind == "subagent" && n.Kind == "agents" {
		// The existing child reader is owned by the Agents navigation view.
		// Keep that source view mounted while displaying the document beside it.
		target = w.Root.kind("browser")
		if target == nil {
			target = w.leaf("browser")
			splitSource = n
		}
	}
	w.browserFor(target, create).OpenDocumentSource(source)
	id := target.ID
	// Content callbacks may run while the source pane is being painted.
	w.pending = func() {
		if splitSource != nil {
			old := *splitSource
			*splitSource = WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: 0.5, First: &old, Second: target}
		}
		if pane := w.Root.find(id); pane != nil {
			if source.AddressKind == "subagent" { w.Fullscreen = "" }
			w.selectKind(pane, "browser")
			w.focusPending = id
		}
	}
}
