package frame

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
)

// WorkspaceFileState is one pane's editor, tab set, selection and viewport.
// FileTabs still owns the existing read/save queues and draft conflict checks.
type WorkspaceFileState struct {
	ResourcePath                                          string                  `json:"resourcePath,omitempty"`
	ResourceTitle                                         string                  `json:"resourceTitle,omitempty"`
	ProjectTarget                                         *projectflow.FileTarget `json:"projectTarget,omitempty"`
	projectAccess                                         *ProjectFileAccess
	projectBinding                                        string
	Tabs                                                  files.Tabs      `json:"tabs"`
	Expanded                                              map[string]bool `json:"expanded,omitempty"`
	Selected, Filter, Preview, Error, Draft, Status, Stat string
	Editing                                               bool
	Mode                                                  files.EditViewMode
	Split, TreeSplit                                      float32
	List                                                  ui.ListState `json:"-"`
	rendered                                              bool
	previewBinding, previewRoot                           string
}

// Explorer selection opens a stable DockKit resource rather than nesting a
// preview beneath the tree. The existing Files reader/editor owns its content,
// async jobs, draft conflict checks and recovery data unchanged.
func openWorkspaceFileResource(m Model, source files.Model, value string) error {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		return fmt.Errorf("%s", paneLabel("文件资源入口不可用", "File resource opener is unavailable"))
	}
	preferred := ""
	if m.workspacePane != nil {
		preferred = m.workspacePane.ID
	}
	handled, err := m.Sizes.Workspace.OpenFileResource(source, value, preferred)
	if err != nil {
		return err
	}
	if !handled {
		return fmt.Errorf("%s", paneLabel("文件在当前工作区不可用", "File is unavailable in the current workspace"))
	}
	return nil
}

// OpenFileResource is the common tree, diff, picker and conversation-reference
// entry point. An unhandled address remains available to the existing Host
// opener; an existing resource keeps its live draft instead of being reread.
func (w *WorkspacePanes) OpenFileResource(source files.Model, value string, preferred ...string) (bool, error) {
	base, line, hasLine := files.PresentedPath(value)
	address := value
	if hasLine {
		if source.ReadRemoteContext != nil || source.ReadTextPageContext != nil {
			address = base
		} else {
			if _, within := files.WorkspaceRelativePath(source.WorkspaceRoot, value); !within {
				address = base
			} else {
				literal := value
				if !filepath.IsAbs(literal) {
					literal = filepath.Join(source.WorkspaceRoot, literal)
				}
				if info, err := os.Stat(literal); err != nil || !info.Mode().IsRegular() {
					address = base
				}
			}
		}
	}
	rel, ok := files.WorkspaceRelativePath(source.WorkspaceRoot, address)
	if !ok || rel == "." {
		return false, nil
	}
	var target *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Kind == "files" && n.Files != nil && n.Files.ResourcePath == rel && n.Files.previewBinding == source.BindingID && n.Files.previewRoot == source.WorkspaceRoot {
			target = n
		}
	})
	if target != nil {
		if hasLine && address == base {
			files.RequestLine(workspaceFileResourceModel(source, target.Files), line)
		}
		w.pending = func() {
			w.activateDockTab(target.ID)
			w.focusPending = target.ID
			if target.Float != nil {
				w.MinimizeWorkspaceFloat(target.ID, false)
			}
		}
		return true, nil
	}
	target = w.leaf("files")
	state := &WorkspaceFileState{ResourcePath: rel, Expanded: map[string]bool{}, Mode: files.EditViewEdit, Split: 200, TreeSplit: 200}
	target.Files = state
	if source.Tabs != nil {
		state.Tabs.Preferences = source.Tabs.Preferences
	}
	state.Tabs.Bind(source.BindingID, source.WorkspaceRoot)
	state.previewBinding, state.previewRoot = source.BindingID, source.WorkspaceRoot
	source = workspaceFileResourceModel(source, state)
	handled, err := files.OpenPresentedPath(source, value)
	if err != nil || !handled {
		return handled, err
	}
	if state.Selected != "" {
		state.ResourcePath = state.Selected
	}
	sourceID := w.Active
	if len(preferred) > 0 && preferred[0] != "" {
		sourceID = preferred[0]
	}
	w.pending = func() {
		seat := w.Root.find(sourceID)
		w.placeDockResource(target, seat, false)
		w.activateDockTab(target.ID)
		w.focusPending = target.ID
		if seat != nil && seat.Kind == "guide" {
			w.disposeWorkspaceResource(seat.ID)
		}
	}
	return true, nil
}

func workspaceFileResourceModel(source files.Model, state *WorkspaceFileState) files.Model {
	source.TreeOnly, source.ResourceOnly, source.OnOpenResource = false, true, nil
	source.OnNewTab = nil
	source.Tabs, source.Expanded = &state.Tabs, &state.Expanded
	source.SelectedFile, source.Filter = &state.Selected, &state.Filter
	source.Preview, source.PreviewErr, source.PreviewList = &state.Preview, &state.Error, &state.List
	source.EditDraft, source.EditStatus, source.StatNote = &state.Draft, &state.Status, &state.Stat
	source.Editing, source.EditViewMode, source.EditSplit = &state.Editing, &state.Mode, &state.Split
	return source
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
		m.FileOpenWithState = nil
		m.OnRefreshFileOpenWith = nil
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
	w.syncPluginRightPanes(m)
	if previous := w.Root.find(w.legacyFileID); previous != nil && previous.Files != nil && previous.Files.ProjectTarget == nil {
		previous.Files.capture(m)
	}
	active := w.Root.find(w.Active)
	if active != nil && active.Files != nil && active.Files.ProjectTarget != nil {
		active = nil
		w.Root.eachResource(func(n *WorkspacePane) {
			if active == nil && (n.Files == nil || n.Files.ProjectTarget == nil) {
				active = n
			}
		})
	}
	if active != nil {
		actual := w.Active
		w.Active = active.ID
		w.beginDiffProjection(m)
		w.Active = actual
	}
	hasFiles := false
	w.Root.eachResource(func(n *WorkspacePane) { hasFiles = hasFiles || n.Files != nil })
	var state *WorkspaceFileState
	if active != nil {
		state = w.fileState(active, m)
	} else {
		state = &WorkspaceFileState{}
		state.capture(m)
	}
	if !hasFiles {
		state.capture(m)
	}
	// First-frame recovery may create this state after BeginContentFrame's
	// binding sweep. Bind before publishing so restored drafts participate in
	// the same-owner preview gate immediately; Bind keeps those drafts intact.
	state.Tabs.Bind(m.FileBindingID, m.WorkspaceRoot)
	state.previewBinding, state.previewRoot = m.FileBindingID, m.WorkspaceRoot
	state.publish(m)
	w.legacyFileID = ""
	if active != nil {
		w.legacyFileID = active.ID
	}

	if trailing := m.WorkspaceTrailing; trailing != nil {
		m.WorkspaceTrailing = func(c *ui.Context, density string) {
			owner := w.Root.find(w.legacyFileID)
			trailing(c, density)
			// The caption's callbacks retain the main Workbench adapter. Collect
			// their changes before a pane renders or the end-of-frame projection
			// would replace a newly opened review with the pane's earlier copy.
			if owner != nil && w.Root.find(owner.ID) == owner {
				w.diffState(owner).capture(m)
			}
		}
	}

	wrapPreview := func(preview func(string) error) func(string) error {
		return func(path string) error {
			previousID, previousLegacy := w.Active, w.legacyFileID
			previousRoute, previousFocus, previousFullscreen, previousPending := w.route, w.focusPending, w.Fullscreen, w.pending
			adapterRoute := derefString(m.RightPaneTab)
			if owner := w.Root.find(previousLegacy); owner != nil && owner.Files != nil {
				owner.Files.capture(m)
			}
			if w.deliveryPreviewHasUnsavedFile(m, path) {
				return fmt.Errorf("%s", paneLabel("文件有未保存的草稿；请先保存后再预览", "The file has an unsaved draft; save it before previewing"))
			}
			var target *WorkspacePane
			relative, inWorkspace := files.WorkspaceRelativePath(m.WorkspaceRoot, path)
			w.Root.eachResource(func(n *WorkspacePane) {
				if inWorkspace && n.Kind == "files" && n.Files != nil && n.Files.ResourcePath == relative && n.Files.previewBinding == m.FileBindingID && n.Files.previewRoot == m.WorkspaceRoot {
					target = n
				}
			})
			// Unqualified Files is the established legacy preview owner. Keep
			// its tabs, drafts and adapter pointers; addressed per-file resources
			// continue to require their exact path and binding above.
			if target == nil {
				w.Root.eachResource(func(n *WorkspacePane) {
					if target == nil && n.Kind == "files" && w.Root.dockGroup(n.ID) == w.Root.dockGroup(previousID) && (n.Files == nil || n.Files.ResourcePath == "" && n.Files.ProjectTarget == nil) {
						target = n
					}
				})
			}
			added := target == nil
			if target == nil {
				target = w.leaf("files")
				if inWorkspace {
					w.fileState(target, m).ResourcePath = relative
				}
				w.placeDockResource(target, w.Root.find(previousID), false)
			}
			w.activateDockTab(target.ID)
			state := w.fileState(target, m)
			// A first-frame preview can arrive before the Files body renders.
			// Publish the intended occurrence with its real binding, so the normal
			// reader validates its existing drafts rather than a fresh adapter.
			state.Tabs.Bind(m.FileBindingID, m.WorkspaceRoot)
			state.previewBinding, state.previewRoot = m.FileBindingID, m.WorkspaceRoot
			if m.Store != nil {
				state.Tabs.SyncPlanSession(m.Store.SelectedSessionID())
			}
			state.publish(m)
			err := preview(path)
			state.capture(m)
			if err != nil {
				if added {
					w.disposeWorkspaceResource(target.ID)
				}
				w.pending, w.focusPending, w.Fullscreen = previousPending, previousFocus, previousFullscreen
				w.activateDockTab(previousID)
				w.route, w.legacyFileID = previousRoute, previousLegacy
				if m.RightPaneTab != nil {
					*m.RightPaneTab = adapterRoute
				}
				if owner := w.Root.find(previousLegacy); owner != nil && owner.Files != nil {
					owner.Files.publish(m)
				}
				return err
			}
			// The callback selects browser only after accepting a document route.
			// Text/image previews remain in the addressed Files occurrence.
			if m.RightPaneTab != nil && *m.RightPaneTab == "browser" {
				state.ResourcePath = ""
				w.selectKind(target, "browser")
			} else {
				w.activateDockTab(target.ID)
				if target.Float != nil && target.Float.Minimized {
					w.MinimizeWorkspaceFloat(target.ID, false)
				}
				if m.RightPaneTab != nil {
					*m.RightPaneTab = "files"
				}
			}
			if guide := w.Root.find(previousID); guide != nil && guide.Kind == "guide" {
				w.disposeWorkspaceResource(guide.ID)
			}
			w.legacyFileID = w.Active
			return nil
		}
	}
	if preview := m.OnPreviewDeliverable; preview != nil {
		m.OnPreviewDeliverable = wrapPreview(preview)
	}
	if presentation := m.OnPreviewDeliverablePresentation; presentation != nil {
		m.OnPreviewDeliverablePresentation = func(path, mode string) error {
			return wrapPreview(func(path string) error { return presentation(path, mode) })(path)
		}
	}
	return m
}

// Only the same saved file under the current binding blocks a delivery preview;
// drafts of other files and foreign owners never become a global veto.
func (w *WorkspacePanes) deliveryPreviewHasUnsavedFile(m Model, path string) bool {
	relative, ok := previewWorkspaceRelative(m.WorkspaceRoot, path)
	if !ok {
		return false
	}
	dirty := false
	w.Root.eachResource(func(n *WorkspacePane) {
		state := n.Files
		if state == nil || state.previewBinding != m.FileBindingID || !samePreviewPath(state.previewRoot, m.WorkspaceRoot) {
			return
		}
		if selected, ok := previewWorkspaceRelative(m.WorkspaceRoot, state.Selected); ok && samePreviewPath(selected, relative) && state.Editing && state.Draft != state.Preview {
			dirty = true
		}
		for root, workspace := range state.Tabs.Workspaces {
			if workspace == nil || !samePreviewPath(root, m.WorkspaceRoot) {
				continue
			}
			for _, tab := range workspace.Open {
				if tab == nil {
					continue
				}
				if candidate, ok := previewWorkspaceRelative(m.WorkspaceRoot, tab.Path); ok && samePreviewPath(candidate, relative) && tab.Dirty() {
					dirty = true
				}
			}
		}
	})
	return dirty
}

func samePreviewPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	return a == b || runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

func previewWorkspaceRelative(root, path string) (string, bool) {
	if !filepath.IsAbs(root) || strings.TrimSpace(path) == "" || strings.Contains(path, "://") || strings.ContainsRune(path, 0) {
		return "", false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || workspacefs.ValidateRelativePath(filepath.ToSlash(rel)) != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func (w *WorkspacePanes) finishFileProjection(m Model) {
	if n := w.Root.find(w.Active); n != nil && (n.Files == nil || n.Files.ProjectTarget == nil) {
		w.finishDiffProjection(m)
		w.fileState(n, m).publish(m)
		w.legacyFileID = n.ID
	}
}

func workspaceContentModel(m Model, n *WorkspacePane) Model {
	w := m.Sizes.Workspace
	state := w.fileState(n, m)
	if m.Store != nil {
		state.Tabs.SyncPlanSession(m.Store.SelectedSessionID())
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
			refreshModel := local
			refreshModel.workspacePane = n
			bindWorkspacePageRefresh(refreshModel, n.browser.RefreshCurrent)
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
		if p := n.Inspector; p != nil {
			p.rendered, p.chatProjected, p.pickQueued = false, false, false
			p.navigation.Nodes, p.navigation.Groups = nil, nil
		}
		if n.Schedule != nil {
			n.Schedule.rendered = false
		}
		if n.Agents != nil {
			n.Agents.rendered = false
			// Hidden child tabs retain their reader only within the same owner binding.
			if !n.holdsAgentReadOccurrence() && (n.Agents.Binding != binding || (n.SubagentChat != nil && n.SubagentChat.Root != root)) {
				n.releaseAgentPane()
			}
		}
		if n.Files != nil {
			n.Files.rendered = false
			if target := n.Files.ProjectTarget; target != nil {
				n.Files.Tabs.Bind(n.Files.projectBinding, target.Root)
				if n.Files.projectAccess == nil || !n.Files.projectAccess.current() {
					n.Files.Tabs.Pause()
				}
			} else {
				n.Files.Tabs.Bind(binding, root)
				n.Files.previewBinding, n.Files.previewRoot = binding, root
			}
		}
		n.remoteFilesRendered = false
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
		if n.RemoteFiles != nil && !n.remoteFilesRendered {
			n.RemoteFiles.Suspend()
		}
		if n.Schedule != nil && !n.Schedule.rendered {
			n.Schedule.deactivate()
		}
		if p := n.Inspector; p != nil {
			if !p.pickQueued && (!p.rendered || !p.chatProjected || !inspectorPaneVisible(w, n, true)) {
				p.cancelInspectorNavigation()
			} else if !p.chatProjected {
				p.navigation.Picking = false
				p.navigation.CancelReveal()
			}
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
		if n.Inspector != nil {
			n.Inspector.cancelInspectorNavigation()
		}
		if retainReads {
			// This path suspends an outgoing Session. OS hide/minimize uses
			// PauseContent(false), which keeps the mounted task's draft.
			n.Schedule.deactivate()
		}
		if n.RemoteFiles != nil {
			n.RemoteFiles.Suspend()
		}
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
	if n.RemoteFiles != nil {
		n.RemoteFiles.Suspend()
	}
	n.Schedule.deactivate()
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
		if !keep[other.ID] && (other.Files.unsaved() || other.RemoteFiles != nil && other.RemoteFiles.Dirty()) {
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
				w.placeDockResource(target, n, true)
			}
			w.Fullscreen = ""
		}
		target = w.selectKind(target, "browser")
		// An explicit document open must reveal its retained container, including
		// a previously minimized browser. Ordinary tab activation keeps its
		// existing presentation policy. Source authorization is still checked by
		// OpenDocumentSource when this deferred route is consumed.
		if target.Float != nil && target.Float.Minimized {
			w.MinimizeWorkspaceFloat(target.ID, false)
		}
		w.browserFor(target, create).OpenDocumentSource(source, floating...)
		w.focusPending = target.ID
	}
}
