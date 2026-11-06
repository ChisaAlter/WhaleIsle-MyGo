package frame

import (
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	remoteui "whaleisle/internal/ui/remote"
)

// WorkspacePanes retains native content occurrences and their split geometry.
// Files, browsers and terminals own independent state in every leaf.
type WorkspacePanes struct {
	focusedResource    string
	conversation       *mainChildConversation
	Root               *WorkspacePane `json:"root"`
	Active             string         `json:"active"`
	Fullscreen         string         `json:"fullscreen,omitempty"` // nonempty expands the whole resource dock; ID retains the initiating pane
	Next               int            `json:"next"`
	route              string
	pending            func()
	legacyFileID       string
	focusPending       string
	closing            string
	closeSaving        string
	closeError         string
	message            string
	conversationBounds mygo.Rectangle
}

type WorkspacePane struct {
	refreshResource                    func()
	ProjectTeam                        *WorkspaceProjectTeamState  `json:"projectTeam,omitempty"`
	OrdinaryTeam                       *WorkspaceOrdinaryTeamState `json:"ordinaryTeam,omitempty"`
	PluginRightPaneHandle              string                      `json:"pluginRightPaneHandle,omitempty"`
	PluginRightPaneLabel               string                      `json:"pluginRightPaneLabel,omitempty"`
	pluginClose                        func()
	pluginCloseRequested               bool
	terminalFresh                      bool
	terminalFreshBound                 bool
	terminalFreshGeneration            uint64
	terminalFreshSession               string
	remoteFilesRendered                bool
	paneWidth, stripWidth, tabRunWidth float32
	tabDragging                        bool
	terminalClose                      func() *hostterminal.OccurrenceClose
	terminalCloseRequest               *hostterminal.OccurrenceClose
	terminalDispose                    func()
	terminalRecovery                   func() *hostterminal.RecoveryTarget
	ProjectObserver                    *projects.ObserverState           `json:"projectObserver,omitempty"`
	Float                              *WorkspaceFloat                   `json:"float,omitempty"`
	DockTabs                           []*WorkspacePane                  `json:"dockTabs,omitempty"`
	DockIndex                          int                               `json:"dockIndex,omitempty"`
	Inspector                          *SessionInspectorPaneState        `json:"inspector,omitempty"`
	ChangesReview                      *ChangesReviewState               `json:"changesReview,omitempty"`
	SubagentChat                       *SubagentChatPaneState            `json:"subagentChat,omitempty"`
	Agents                             *WorkspaceAgentState              `json:"-"`
	Diff                               *WorkspaceDiffState               `json:"diff,omitempty"`
	Files                              *WorkspaceFileState               `json:"files,omitempty"`
	RemoteFiles                        *remoteui.FileWorkspaceState      `json:"-"`
	Browser                            *browserpreview.WorkspaceSnapshot `json:"browser,omitempty"`
	browser                            *browserpreview.Controller
	Schedule                           *SchedulePaneState           `json:"schedule,omitempty"`
	ID                                 string                       `json:"id"`
	Kind                               string                       `json:"kind,omitempty"`
	TerminalSource                     string                       `json:"terminalSource,omitempty"`
	TerminalRecovery                   *hostterminal.RecoveryTarget `json:"terminalRecovery,omitempty"`
	Axis                               string                       `json:"axis,omitempty"`
	Ratio                              float32                      `json:"ratio,omitempty"`
	First                              *WorkspacePane               `json:"first,omitempty"`
	Second                             *WorkspacePane               `json:"second,omitempty"`
}

func paneLabel(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}
func (w *WorkspacePanes) leaf(kind string) *WorkspacePane {
	w.Next++
	return &WorkspacePane{ID: fmt.Sprintf("workspace-pane-%d", w.Next), Kind: kind, terminalFresh: kind == "host-terminal"}
}
func (n *WorkspacePane) find(id string) *WorkspacePane {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return n
	}
	for _, tab := range n.DockTabs {
		if tab.ID == id {
			return tab
		}
	}
	if p := n.First.find(id); p != nil {
		return p
	}
	return n.Second.find(id)
}
func (n *WorkspacePane) kind(kind string) *WorkspacePane {
	if n == nil {
		return nil
	}
	if n.Kind == kind {
		return n
	}
	for _, tab := range n.DockTabs {
		if tab.Kind == kind {
			return tab
		}
	}
	if p := n.First.kind(kind); p != nil {
		return p
	}
	return n.Second.kind(kind)
}
func (n *WorkspacePane) firstLeaf() *WorkspacePane {
	if n.First == nil {
		return n
	}
	return n.First.firstLeaf()
}
func (w *WorkspacePanes) selectKind(n *WorkspacePane, kind string) *WorkspacePane {
	if n == nil {
		return nil
	}
	group := w.Root.dockGroup(n.ID)
	if group == nil {
		return n
	}
	tabs := group.dockTabs()
	for _, tab := range tabs {
		if tab.Kind == kind {
			if kind == "files" && tab.Files != nil && (tab.Files.ProjectTarget != nil || tab.Files.ResourcePath != "") {
				continue
			}
			if kind == "changes-review" {
				transferDockResource(tab, n, kind)
			}
			w.activateDockTab(tab.ID)
			if n.Kind == "guide" && tab != n {
				w.disposeWorkspaceResource(n.ID)
			}
			return tab
		}
	}
	if !independentPaneKind(kind) {
		var existing *WorkspacePane
		w.Root.eachResource(func(tab *WorkspacePane) {
			if tab.Kind == kind {
				existing = tab
			}
		})
		if existing != nil {
			w.activateDockTab(existing.ID)
			if n.Kind == "guide" && existing != n {
				w.disposeWorkspaceResource(n.ID)
			}
			return existing
		}
	}
	added := w.leaf(kind)
	transferDockResource(added, n, kind)
	if n.Kind == "guide" {
		for i, tab := range tabs {
			if tab == n {
				tabs[i] = added
			}
		}
		n.DockTabs, n.DockIndex = nil, 0
		setDockTabs(added, tabs)
		w.Root = replaceWorkspaceLeaf(w.Root, n, added)
		if w.Fullscreen == n.ID {
			w.Fullscreen = added.ID
		}
	} else {
		tabs = append(tabs, added)
		setDockTabs(group, tabs)
	}
	w.activateDockTab(added.ID)
	return added
}
func (w *WorkspacePanes) syncRoute(kind string) {
	if kind == "" {
		kind = "guide"
	}
	if w.Root == nil {
		w.Root = w.leaf(kind)
		w.Active = w.Root.ID
	}
	if w.Root.find(w.Active) == nil {
		w.Active = w.Root.firstLeaf().ID
	}
	if kind != w.route {
		if existing := w.Root.kind(kind); !independentPaneKind(kind) && existing != nil {
			w.activateDockTab(existing.ID)
		} else {
			w.selectKind(w.Root.find(w.Active), kind)
		}
		if w.Fullscreen != "" {
			w.Fullscreen = w.Active
		}
	}
	w.route = kind
}
func (w *WorkspacePanes) visible(kind string) bool {
	if w == nil || w.Root == nil {
		return false
	}
	// Fullscreen expands the whole dock, not only the initiating pane. Every
	// sibling remains visible, including native browser/terminal occurrences.
	visible := false
	w.Root.eachLeaf(func(group *WorkspacePane) {
		if docked := group.dockedLeaf(); docked != nil && docked.Kind == kind {
			visible = true
		}
		for _, tab := range group.dockTabs() {
			if tab.Float != nil && !tab.Float.Minimized && tab.Kind == kind {
				visible = true
			}
		}
	})
	return visible
}
func (w *WorkspacePanes) remove(n *WorkspacePane, id string) *WorkspacePane {
	if n == nil || n.ID == id {
		return nil
	}
	if n.First == nil {
		return n
	}
	n.First = w.remove(n.First, id)
	n.Second = w.remove(n.Second, id)
	if n.First == nil {
		return n.Second
	}
	if n.Second == nil {
		return n.First
	}
	return n
}
func (w *WorkspacePanes) close(id string) bool {
	if !w.canCloseResource(w.Root.find(id)) {
		return false
	}
	if n := w.Root.find(id); n != nil && n.PluginRightPaneHandle != "" && n.pluginClose != nil {
		n.pluginCloseRequested = true
		n.pluginClose()
		w.message = paneLabel("已请求关闭插件面板；等待原面板确认。", "Plugin close requested; waiting for its owner to confirm.")
		return false
	}
	if n := w.Root.find(id); n != nil && n.Kind == "host-terminal" && n.terminalClose != nil {
		if n.terminalCloseRequest == nil {
			n.terminalCloseRequest = n.terminalClose()
		}
		done, err := n.terminalCloseRequest.Result()
		if !done {
			w.message = paneLabel("正在关闭 Host 终端…", "Closing Host terminal…")
			return false
		}
		n.terminalCloseRequest = nil
		if err != nil {
			w.message = paneLabel("终端未确认关闭，请重试：", "Terminal close was not confirmed; retry: ") + err.Error()
			return false
		}
		w.message = ""
	}

	if n := w.Root.find(id); n != nil && n.Kind == "project-observer" {
		return false
	}
	if n := w.Root.find(id); n != nil && (n.Files.unsaved() || n.RemoteFiles != nil && n.RemoteFiles.Dirty()) {
		w.closing = id
		return false
	}
	w.closeNow(id)
	return true
}
func (w *WorkspacePanes) closeNow(id string) {
	if n := w.Root.find(id); n != nil && n.Kind == "project-observer" {
		return
	}
	w.disposeWorkspaceResource(id)
}

// disposeWorkspaceResource is internal lifecycle cleanup, not a user close
// command. Host/reset/disposal paths may remove non-closeable resource types.
func (w *WorkspacePanes) disposeWorkspaceResource(id string) {
	group := w.Root.dockGroup(id)
	if group == nil {
		return
	}
	tabs := group.dockTabs()
	remaining := make([]*WorkspacePane, 0, len(tabs)-1)
	for _, tab := range tabs {
		if tab.ID == id {
			tab.disposeContent()
		} else {
			remaining = append(remaining, tab)
		}
	}
	group.DockTabs, group.DockIndex = nil, 0
	if len(remaining) == 0 {
		w.Root = w.remove(w.Root, group.ID)
	} else {
		selected := group
		if group.ID == id {
			selected = remaining[0]
		}
		setDockTabs(selected, remaining)
		w.Root = replaceWorkspaceLeaf(w.Root, group, selected)
	}
	if w.legacyFileID == id {
		w.legacyFileID = ""
	}
	w.closing = ""
	if w.Root == nil {
		w.Root = w.leaf("guide")
	}
	if w.Root.find(w.Active) == nil {
		w.Active = w.Root.firstLeaf().ID
	}
	w.activateDockTab(w.Active)
	if w.Fullscreen == id {
		w.Fullscreen = w.Active
	}
	w.route = w.Root.find(w.Active).Kind
}

// CloseKind also serves the existing browser's close control.
func (w *WorkspacePanes) CloseKind(kind string) string {
	if w == nil || w.Root == nil {
		return "files"
	}
	n := w.Root.find(w.Active)
	if n == nil || n.Kind != kind {
		n = w.Root.kind(kind)
	}
	if n != nil {
		w.close(n.ID)
	}
	return w.Root.find(w.Active).Kind
}
func (w *WorkspacePanes) Export() json.RawMessage {
	if w == nil {
		return nil
	}
	w.captureContent()
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.terminalRecovery != nil {
			if saved := n.terminalRecovery(); saved != nil {
				n.TerminalRecovery = saved
			}
		}
	})
	raw, _ := json.Marshal(w)
	return raw
}
func RestoreWorkspacePanes(raw json.RawMessage) (*WorkspacePanes, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var w WorkspacePanes
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, err
	}
	if err := validateWorkspaceRestore(&w); err != nil {
		return nil, err
	}
	if w.Root != nil {
		active := w.Root.find(w.Active)
		if active == nil || active.First != nil {
			active = w.Root.firstLeaf()
			w.Active = active.ID
		}
		w.activateDockTab(active.ID)
	}
	if err := w.restoreContent(); err != nil {
		return nil, err
	}
	return &w, nil
}
func workspaceVisible(m Model, kind string) bool {
	if !m.RightPaneOpen {
		if m.Sizes != nil && m.Sizes.Workspace != nil {
			visible := false
			m.Sizes.Workspace.Root.eachResource(func(n *WorkspacePane) {
				if n.Float != nil && !n.Float.Minimized && n.Kind == kind {
					visible = true
				}
			})
			return visible
		}
		return false
	}
	if m.Sizes != nil && m.Sizes.Workspace != nil {
		return m.Sizes.Workspace.visible(kind)
	}
	current := derefString(m.RightPaneTab)
	if current == "" {
		current = "guide"
	}
	return current == kind
}
func prepareWorkspace(m Model) {
	if m.Sizes.Workspace == nil {
		m.Sizes.Workspace = &WorkspacePanes{}
	}
	w := m.Sizes.Workspace
	if pane := w.Root.find(w.focusPending); pane != nil {
		w.activateDockTab(pane.ID)
		if m.RightPaneTab != nil {
			*m.RightPaneTab = pane.Kind
		}
	}
	w.focusPending = ""
	w.syncRoute(derefString(m.RightPaneTab))
}

// The installed rightbar derives phone fullscreen from frame width. It is
// presentation only: no serialized mode or pane identity changes on resize.
const workspaceAutoFullscreenWidth = float32(768)

func workspaceAutoFullscreen(m Model) bool {
	width := m.ViewportW
	if width <= 0 && m.Sizes != nil {
		width = m.Sizes.viewportW
	}
	return !m.WorkspacePage && m.RightPaneOpen && width > 0 && width < workspaceAutoFullscreenWidth
}

func workspaceFullscreen(m Model) bool {
	return m.Sizes != nil && m.Sizes.Workspace != nil && (workspaceAutoFullscreen(m) || m.Sizes.Workspace.Fullscreen != "")
}

func workspaceColumns(c *ui.Context, m Model, center func(), header ...func()) {
	if m.Sizes.Workspace == nil {
		prepareWorkspace(m)
	}
	w := m.Sizes.Workspace
	w.syncRoute(derefString(m.RightPaneTab))
	defer func() {
		if w.pending != nil {
			action := w.pending
			w.pending = nil
			action()
		}
		if m.RightPaneTab != nil {
			*m.RightPaneTab = w.Root.find(w.Active).Kind
		}
	}()
	defer workspaceFloating(c, m)
	defer workspaceMiniPlayers(c, m)
	w.conversationBounds = mygo.Rectangle{}
	render := func() {
		dock := ui.Box(c).Key("workspace-dock-drop-surface").Label("Workspace resource dock").Fill().MinWidth(0).MinHeight(0)
		workspaceDockDropBarrier(dock)
		dock.Children(func() { workspaceDockNode(c, m, w.Root) })
	}
	if m.RightPaneOpen && workspaceFullscreen(m) && w.Root.dockedLeaf() != nil {
		// SidebarRight keeps one DockLayout tree in both push and fullscreen
		// modes. Retain every split and resource occurrence, and reuse the
		// existing native-safe main area rather than stacking a canvas overlay
		// above live WebView/terminal children.
		ui.Column(c).Key("workspace-fullscreen-rows").Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			if len(header) > 0 && header[0] != nil {
				workspaceHeaderBand(c, header[0])
			}
			ui.Box(c).Grow(1).MinWidth(0).MinHeight(0).Padding(0, max(float32(0), m.RightOverlayReserve), 0, 0).Clip().Children(render)
		})
		return
	}
	if w.Root.dockedLeaf() == nil {
		m.RightPaneOpen = false
	}
	centerRightColumns(c, m, center, render, header...)
}
func workspaceNode(c *ui.Context, m Model, n *WorkspacePane) {
	box := ui.Box(c).Key(n.ID).Fill().MinWidth(0).MinHeight(0).Clip()
	if n.Float == nil && n.First == nil {
		box.FocusBind(&m.Sizes.Workspace.focusedResource, n.ID)
	}
	box.Children(func() {
		if n.First != nil {
			if n.Ratio == 0 {
				n.Ratio = 0.5
			}
			first := func() { workspaceNode(c, m, n.First) }
			second := func() { workspaceNode(c, m, n.Second) }
			if n.Axis == "vertical" {
				ui.SplitVerticalFraction(c, &n.Ratio, .2, first, second).Fill()
			} else {
				ui.SplitFraction(c, &n.Ratio, .2, first, second).Fill()
			}
			return
		}
		if box.FocusWithin() {
			m.Sizes.Workspace.Active = n.ID
			m.Sizes.Workspace.route = n.Kind
		}
		tab := n.Kind
		n.refreshResource = nil
		local := workspaceContentModel(m, n)
		local.RightPaneTab = &tab
		local.workspacePane = n
		width := box.Bounds().W
		n.paneWidth = width
		local.FilePaneWidth = &width
		local.OnHideBrowser = nil // Visibility belongs to the whole layout, not each seat.
		local.OnOpenBrowserPreview = func() { tab = "browser" }
		if !paintAuthorizedProjectFile(c, local, n) {
			rightPane(c, local)
		}
		if n.Float == nil {
			workspaceDockDropTargets(c, local, box)
		}
		if tab != n.Kind {
			m.Sizes.Workspace.selectKind(n, tab)
		}
		if active := m.Sizes.Workspace.Root.find(m.Sizes.Workspace.Active); active != nil && m.RightPaneTab != nil {
			*m.RightPaneTab = active.Kind
		}
	})
}
func workspacePaneControls(c *ui.Context, m Model) {
	n := m.workspacePane
	if n == nil {
		return
	}
	w := m.Sizes.Workspace
	if n.Float != nil {
		if w.closing == n.ID {
			workspacePaneClosePrompt(c, m)
		}
		return
	}
	workspaceDockTabStrip(c, m)
	if w.closing == n.ID {
		workspacePaneClosePrompt(c, m)
	}
	if w.message != "" {
		ui.Text(c, w.message)
	}
}

func workspacePaneClose(m Model, n *WorkspacePane) {
	w := m.Sizes.Workspace
	w.pending = func() {
		if !w.close(n.ID) {
			return
		}
		onlyGuide := true
		w.Root.eachResource(func(tab *WorkspacePane) { onlyGuide = onlyGuide && tab.Kind == "guide" })
		if onlyGuide && m.RightPaneOpen && m.OnToggleTools != nil {
			m.OnToggleTools()
		}
	}
}

func workspacePaneMaximize(m Model, n *WorkspacePane) {
	if n == nil || n.Float != nil || m.Sizes == nil || m.Sizes.Workspace == nil {
		return
	}
	w := m.Sizes.Workspace
	if workspaceAutoFullscreen(m) {
		w.Fullscreen = ""
		if m.OnToggleTools != nil {
			m.OnToggleTools()
		}
		return
	}
	if w.Fullscreen != "" {
		w.Fullscreen = ""
	} else {
		w.Fullscreen, w.Active, w.route = n.ID, n.ID, n.Kind
	}
}

// DockKit TabMenu hosts secondary operations above clipping ancestors.
// Native menus preserve keyboard navigation, dismissal and tab identity.
func workspacePaneMenu(m Model, n *WorkspacePane) func(*ui.Menu) {
	w := m.Sizes.Workspace
	return func(menu *ui.Menu) {
		if n.Kind == "host-terminal" {
			workspaceTerminalResourceMenu(menu, m, n)
			menu.Separator()
		}
		if menu.Item(paneLabel("选中面板", "Focus pane")).Chosen() {
			w.pending = func() { w.activateDockTab(n.ID) }
		}
		if menu.Item(paneLabel("浮动", "Float")).Disabled(n.Kind == "project-observer").Chosen() {
			w.pending = func() { w.FloatWorkspaceTab(n.ID) }
		}
		label := paneLabel("面板全屏", "Maximize pane")
		if workspaceFullscreen(m) {
			label = paneLabel("退出全屏", "Restore pane")
		}
		if menu.Item(label).Chosen() {
			workspacePaneMaximize(m, n)
		}
		if menu.Item(paneLabel("关闭标签", "Close tab")).Disabled(!w.canCloseResource(n)).Chosen() {
			workspacePaneClose(m, n)
		}
		menu.Separator()
		for _, vertical := range []bool{false} {
			label := paneLabel("向右拆分", "Split right")
			if vertical {
				label = paneLabel("向下拆分", "Split down")
			}
			if menu.Item(label).Disabled(!w.canSplit(n)).Chosen() {
				w.pending = func() {
					w.activateDockTab(n.ID)
					local := m
					tab := n.Kind
					local.RightPaneTab = &tab
					splitWorkspaceDock(local, vertical)
				}
			}
		}
		if menu.Item(paneLabel("重置布局", "Reset layout")).Chosen() {
			w.pending = func() { w.resetTo(n) }
		}
	}
}

func (w *WorkspacePanes) canCloseResource(n *WorkspacePane) bool {
	if n == nil || n.Kind == "project-observer" {
		return false
	}
	if n.Kind != "guide" || n.Float != nil {
		return true
	}
	docked := 0
	w.Root.eachResource(func(tab *WorkspacePane) {
		if tab.Float == nil {
			docked++
		}
	})
	return docked > 1
}
func independentPaneKind(kind string) bool {
	return kind == "session-inspector-log" || kind == "changes-review" || kind == "subagentchat" || kind == "plan" || kind == "files" || kind == "browser" || kind == "host-terminal" || kind == "schedule-task" || kind == "diff" || kind == "agents"
}

// SplitWorkspacePane preserves the standalone API's independent-copy contract.
// Native DockKit controls use splitWorkspaceDock, which creates a Start seat.
func SplitWorkspacePane(m Model, vertical bool) {
	splitWorkspacePane(m, vertical, false)
}

func splitWorkspaceDock(m Model, vertical bool) {
	splitWorkspacePane(m, vertical, true)
}

func splitWorkspacePane(m Model, vertical, guide bool) {
	if vertical || !m.RightPaneOpen || m.Sizes == nil {
		return
	}
	prepareWorkspace(m)
	w := m.Sizes.Workspace
	if !w.canSplit(w.Root.find(w.Active)) {
		return
	}
	// DockKit seeds the new seat with Start. Splitting a pane does not clone
	// its resource, reader, terminal or draft into a second occurrence.
	n := w.Root.dockGroup(w.Active)
	old := *n
	kind := "guide"
	if !guide && independentPaneKind(old.Kind) {
		kind = old.Kind
	}
	added := w.leaf(kind)
	if !guide {
		copySchedulePaneTarget(added, &old)
		copyDiffPane(added, &old)
		copyPlanPane(added, &old)
		copyChangesReviewPane(added, &old)
		copySessionInspectorPane(added, &old)
		copySubagentChatPane(added, &old)
		if kind == "host-terminal" && old.Kind == kind {
			added.TerminalSource, added.terminalFresh = old.ID, false
		}
	}
	*n = WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: 0.5, First: &old, Second: added}
	w.Active = added.ID
	w.Fullscreen = ""
	w.route = added.Kind
	if m.RightPaneTab != nil {
		*m.RightPaneTab = added.Kind
	}
}
func ToggleWorkspacePaneFullscreen(m Model) {
	if !m.RightPaneOpen || m.Sizes == nil {
		return
	}
	prepareWorkspace(m)
	w := m.Sizes.Workspace
	workspacePaneMaximize(m, w.Root.find(w.Active))
}
