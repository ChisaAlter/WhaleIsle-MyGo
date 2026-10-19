package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/planpreview"
)

// WorkspaceFileState is one pane's editor, tab set, selection and viewport.
// FileTabs still owns the existing read/save queues and draft conflict checks.
type WorkspaceFileState struct {
	Tabs                                                  files.Tabs      `json:"tabs"`
	Expanded                                              map[string]bool `json:"expanded,omitempty"`
	Selected, Filter, Preview, Error, Draft, Status, Stat string
	Editing                                               bool
	Mode                                                  files.EditViewMode
	Split, TreeSplit                                      float32
	List                                                  ui.ListState `json:"-"`
	rendered                                              bool
}

func (s *WorkspaceFileState) capture(m Model) {
	if m.FileTabs != nil {
		s.Tabs = *m.FileTabs
	}
	if m.FileExpanded != nil {
		s.Expanded = *m.FileExpanded
	}
	if m.SelectedFile != nil {
		s.Selected = *m.SelectedFile
	}
	if m.FileFilter != nil {
		s.Filter = *m.FileFilter
	}
	if m.FilePreview != nil {
		s.Preview = *m.FilePreview
	}
	if m.FilePreviewErr != nil {
		s.Error = *m.FilePreviewErr
	}
	if m.FileEditDraft != nil {
		s.Draft = *m.FileEditDraft
	}
	if m.FileEditStatus != nil {
		s.Status = *m.FileEditStatus
	}
	if m.FileStatNote != nil {
		s.Stat = *m.FileStatNote
	}
	if m.FileEditing != nil {
		s.Editing = *m.FileEditing
	}
	if m.FileEditViewMode != nil {
		s.Mode = *m.FileEditViewMode
	}
	if m.FileEditSplit != nil {
		s.Split = *m.FileEditSplit
	}
	if m.FilePreviewList != nil {
		s.List = *m.FilePreviewList
	}
}

func (s *WorkspaceFileState) publish(m Model) {
	if m.FileTabs != nil {
		*m.FileTabs = s.Tabs
	}
	if m.FileExpanded != nil {
		*m.FileExpanded = s.Expanded
	}
	if m.SelectedFile != nil {
		*m.SelectedFile = s.Selected
	}
	if m.FileFilter != nil {
		*m.FileFilter = s.Filter
	}
	if m.FilePreview != nil {
		*m.FilePreview = s.Preview
	}
	if m.FilePreviewErr != nil {
		*m.FilePreviewErr = s.Error
	}
	if m.FileEditDraft != nil {
		*m.FileEditDraft = s.Draft
	}
	if m.FileEditStatus != nil {
		*m.FileEditStatus = s.Status
	}
	if m.FileStatNote != nil {
		*m.FileStatNote = s.Stat
	}
	if m.FileEditing != nil {
		*m.FileEditing = s.Editing
	}
	if m.FileEditViewMode != nil {
		*m.FileEditViewMode = s.Mode
	}
	if m.FileEditSplit != nil {
		*m.FileEditSplit = s.Split
	}
	if m.FilePreviewList != nil {
		*m.FilePreviewList = s.List
	}
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
	if s == nil {
		return false
	}
	for _, workspace := range s.Tabs.Workspaces {
		if workspace == nil {
			continue
		}
		for _, tab := range workspace.Open {
			if tab != nil && tab.Dirty() {
				return true
			}
		}
	}
	return s.Editing && s.Draft != s.Preview
}

func (n *WorkspacePane) eachLeaf(visit func(*WorkspacePane)) {
	if n == nil {
		return
	}
	if n.First == nil {
		visit(n)
		return
	}
	n.First.eachLeaf(visit)
	n.Second.eachLeaf(visit)
}

func (w *WorkspacePanes) fileState(n *WorkspacePane, m Model) *WorkspaceFileState {
	if n.Files == nil {
		n.Files = &WorkspaceFileState{Expanded: map[string]bool{}, Mode: files.EditViewEdit, Split: 200, TreeSplit: m.Sizes.RightTree}
		if m.FileTabs != nil {
			n.Files.Tabs.Preferences = m.FileTabs.Preferences
		}
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
	w.Root.eachResource(func(n *WorkspacePane) { hasFiles = hasFiles || n.Files != nil })
	state := w.fileState(active, m)
	if !hasFiles {
		state.capture(m)
	}
	state.publish(m)
	w.legacyFileID = active.ID

	// Existing callbacks retain all original session/Host checks. Publish at the
	// gesture, then collect their result into that same occurrence.
	run := func(action func()) {
		target := w.selectKind(w.Root.find(w.Active), "files")
		state := w.fileState(target, m)
		state.publish(m)
		action()
		state.capture(m)
		if m.RightPaneTab != nil && *m.RightPaneTab != target.Kind {
			w.selectKind(target, *m.RightPaneTab)
		}
		w.legacyFileID = w.Active
	}
	if open := m.OnOpenDeliverable; open != nil {
		m.OnOpenDeliverable = func(path string) { run(func() { open(path) }) }
	}
	if open := m.OnOpenPlanResource; open != nil {
		m.OnOpenPlanResource = func(doc planpreview.Document) {
			run(func() { open(doc) })
			w.showOpenedPlan(m, doc)
		}
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
	if m.Store != nil {
		state.Tabs.SyncPlanSession(m.Store.Snapshot().SelectedID)
	}
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
			n.browser.SetMiniPlayerActions(func() {
				w.Root.eachResource(func(other *WorkspacePane) {
					if other != n && other.browser != nil {
						other.browser.CloseMiniPlayer()
					}
				})
			}, func() {
				w.pending = func() {
					if n.Float != nil {
						w.RedockWorkspaceTab(n.ID, "")
					}
					w.activateDockTab(n.ID)
					if m.OnOpenBrowserPreview != nil {
						m.OnOpenBrowserPreview()
					}
				}
			})
			n.browser.ViewDocked(c, m.ThemeID, m.SystemDark, func() {
				w.pending = func() { w.close(n.ID) }
			})
		}
	}
	if n.Kind == "agents" || n.Kind == "subagentchat" {
		local = w.agentPaneModel(local, n)
	}
	return local
}

// BeginContentFrame is called even when an application page replaces frame.View.
func (w *WorkspacePanes) BeginContentFrame(binding, root string) {
	if w == nil {
		return
	}
	if route := w.conversation; route != nil {
		if w.currentMainChild(Model{Store: route.Owner}) == nil {
			w.closeMainChild()
		} else if route.Pane.Agents != nil {
			route.Pane.Agents.rendered = false
		}
	}
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Agents != nil {
			n.Agents.rendered = false
			// Hidden child tabs retain their reader only within the same owner binding.
			if !n.holdsAgentReadOccurrence() && (n.Agents.Binding != binding || (n.SubagentChat != nil && n.SubagentChat.Root != root)) {
				n.releaseAgentPane()
			}
		}
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
	if w == nil {
		return
	}
	if route := w.conversation; route != nil && route.Pane.Agents != nil && !route.Pane.Agents.rendered {
		route.Pane.releaseAgentPane()
	}
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Inspector != nil && (n.Float != nil && n.Float.Minimized || n.Float == nil && w.Root.dockGroup(n.ID) != n) {
			n.Inspector.navigation.Picking = false
			n.Inspector.navigation.Target = nil
		}
		if n.Agents != nil && !n.Agents.rendered && (n.Kind != "subagentchat" || n.SubagentChat == nil) {
			n.releaseAgentPane()
		}
		if n.Files != nil && !n.Files.rendered {
			n.Files.Tabs.CancelPlanRead()
			n.Files.Tabs.Pause()
		}
		if n.browser != nil {
			n.browser.EndFrame()
		}
	})
}

func (w *WorkspacePanes) PauseContent() { w.pauseContent(false) }
func (w *WorkspacePanes) pauseContent(retainReads bool) {
	if w == nil {
		return
	}
	if route := w.conversation; route != nil {
		route.Pane.releaseAgentPane()
	}
	w.Root.eachResource(func(n *WorkspacePane) {
		if !retainReads || !n.holdsAgentReadOccurrence() {
			n.releaseAgentPane()
		}
		if n.Files != nil {
			n.Files.Tabs.CancelPlanRead()
			n.Files.Tabs.Pause()
		}
		if n.browser != nil {
			n.browser.Hide()
		}
	})
}

func (n *WorkspacePane) disposeContent() {
	// Generic disposal only releases presentation; process termination belongs
	// exclusively to the explicit close request in WorkspacePanes.close.
	if n.terminalDispose != nil {
		n.terminalDispose()
		n.terminalDispose = nil
	}
	n.terminalClose = nil

	n.releaseAgentPane()
	if n.Diff != nil {
		n.Diff.Workbench.ClosePane()
	}
	if n.Files != nil {
		n.Files.Tabs.Dispose()
	}
	if n.browser != nil {
		n.browser.Close()
		n.browser = nil
	}
}

func (w *WorkspacePanes) DisposeContent() {
	if w != nil {
		w.Root.eachResource(func(n *WorkspacePane) { n.disposeContent() })
	}
}

func (w *WorkspacePanes) captureContent() {
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.browser != nil {
			n.Browser = n.browser.WorkspaceSnapshot()
		}
	})
}

func (w *WorkspacePanes) restoreContent() error {
	var restoreErr error
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Files == nil || restoreErr != nil {
			return
		}
		raw, err := n.Files.Tabs.Export()
		if err == nil {
			err = n.Files.Tabs.Restore(raw)
		}
		if err != nil {
			restoreErr = err
		}
	})
	return restoreErr
}

func (w *WorkspacePanes) resetTo(n *WorkspacePane) {
	group := w.Root.dockGroup(n.ID)
	if group == nil {
		return
	}
	keep := map[string]bool{}
	for _, tab := range group.dockTabs() {
		keep[tab.ID] = true
	}
	blocked := false
	w.Root.eachResource(func(other *WorkspacePane) {
		if !keep[other.ID] && other.Files.unsaved() {
			blocked = true
		}
	})
	if blocked {
		w.message = paneLabel("其他面板仍有未保存内容，请先保存或关闭它们。", "Other panes have unsaved drafts. Save or close them first.")
		return
	}
	w.Root.eachResource(func(other *WorkspacePane) {
		if !keep[other.ID] {
			other.disposeContent()
		}
	})
	w.Root, w.Active, w.Fullscreen, w.route = group, group.ID, "", group.Kind
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

func (w *WorkspacePanes) OpenDocument(source browserpreview.DocumentSource, create func() *browserpreview.Controller, floating ...bool) {
	if w.Root == nil {
		w.syncRoute("files")
	}
	sourceID := w.Active
	// The authenticated caller already resolved the document. Defer only the
	// layout mutation so no tab changes ownership while its body is painted.
	w.pending = func() {
		n := w.Root.find(sourceID)
		if n == nil {
			return
		}
		target := n
		if source.AddressKind == "subagent" && n.Kind == "agents" {
			target = w.Root.kind("browser")
			if target == nil {
				target = w.leaf("browser")
				old := *n
				*n = WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: 0.5, First: &old, Second: target}
			}
			w.Fullscreen = ""
		}
		target = w.selectKind(target, "browser")
		w.browserFor(target, create).OpenDocumentSource(source, floating...)
		w.focusPending = target.ID
	}
}
