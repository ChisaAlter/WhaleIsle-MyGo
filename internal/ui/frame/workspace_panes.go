package frame

import (
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// WorkspacePanes retains native content occurrences and their split geometry.
// Files, browsers and terminals own independent state in every leaf.
type WorkspacePanes struct {
	Root         *WorkspacePane `json:"root"`
	Active       string         `json:"active"`
	Fullscreen   string         `json:"fullscreen,omitempty"`
	Next         int            `json:"next"`
	route        string
	pending      func()
	legacyFileID string
	focusPending string
	closing      string
	message      string
}

type WorkspacePane struct {
	Inspector      *SessionInspectorPaneState        `json:"inspector,omitempty"`
	ChangesReview  *ChangesReviewState               `json:"changesReview,omitempty"`
	SubagentChat   *SubagentChatPaneState            `json:"subagentChat,omitempty"`
	Agents         *WorkspaceAgentState              `json:"-"`
	Diff           *WorkspaceDiffState               `json:"diff,omitempty"`
	Files          *WorkspaceFileState               `json:"files,omitempty"`
	Browser        *browserpreview.WorkspaceSnapshot `json:"browser,omitempty"`
	browser        *browserpreview.Controller
	Schedule       *SchedulePaneState `json:"schedule,omitempty"`
	ID             string             `json:"id"`
	Kind           string             `json:"kind,omitempty"`
	TerminalSource string             `json:"terminalSource,omitempty"`
	Axis           string             `json:"axis,omitempty"`
	Ratio          float32            `json:"ratio,omitempty"`
	First          *WorkspacePane     `json:"first,omitempty"`
	Second         *WorkspacePane     `json:"second,omitempty"`
}

func paneLabel(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}
func (w *WorkspacePanes) leaf(kind string) *WorkspacePane {
	w.Next++
	return &WorkspacePane{ID: fmt.Sprintf("workspace-pane-%d", w.Next), Kind: kind}
}
func (n *WorkspacePane) find(id string) *WorkspacePane {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return n
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
func (w *WorkspacePanes) selectKind(n *WorkspacePane, kind string) {
	if other := w.Root.kind(kind); !independentPaneKind(kind) && other != nil && other != n {
		// Singleton providers are revealed in their existing seat. Moving only a
		// kind would detach an independent file/browser occurrence from its state.
		w.Active, w.route = other.ID, kind
		return
	}
	n.Kind = kind
	w.Active = n.ID
	w.route = kind
}
func (w *WorkspacePanes) syncRoute(kind string) {
	if kind == "" {
		kind = "files"
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
			w.Active = existing.ID
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
	n := w.Root
	if w.Fullscreen != "" {
		n = w.Root.find(w.Fullscreen)
	}
	return n.kind(kind) != nil
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
	if n := w.Root.find(id); n != nil && n.Files.unsaved() {
		w.closing = id
		return false
	}
	w.closeNow(id)
	return true
}
func (w *WorkspacePanes) closeNow(id string) {
	if n := w.Root.find(id); n != nil {
		n.disposeContent()
	}
	if w.legacyFileID == id {
		w.legacyFileID = ""
	}
	w.closing = ""
	w.Root = w.remove(w.Root, id)
	if w.Root == nil {
		w.Root = w.leaf("files")
	}
	if w.Root.find(w.Active) == nil {
		w.Active = w.Root.firstLeaf().ID
	}
	if w.Fullscreen == id {
		w.Fullscreen = ""
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
	if w.Root != nil {
		active := w.Root.find(w.Active)
		if active == nil {
			active = w.Root.firstLeaf()
			w.Active = active.ID
		}
		w.route = active.Kind
	}
	if err := w.restoreContent(); err != nil {
		return nil, err
	}
	return &w, nil
}
func workspaceVisible(m Model, kind string) bool {
	if !m.RightPaneOpen {
		return false
	}
	if m.Sizes != nil && m.Sizes.Workspace != nil {
		return m.Sizes.Workspace.visible(kind)
	}
	current := derefString(m.RightPaneTab)
	if current == "" {
		current = "files"
	}
	return current == kind
}
func prepareWorkspace(m Model) {
	if m.Sizes.Workspace == nil {
		m.Sizes.Workspace = &WorkspacePanes{}
	}
	w := m.Sizes.Workspace
	if pane := w.Root.find(w.focusPending); pane != nil {
		w.Active, w.route = pane.ID, pane.Kind
		if m.RightPaneTab != nil {
			*m.RightPaneTab = pane.Kind
		}
	}
	w.focusPending = ""
	w.syncRoute(derefString(m.RightPaneTab))
}
func workspaceColumns(c *ui.Context, m Model, center func()) {
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
	render := func() { workspaceNode(c, m, w.Root) }
	if m.RightPaneOpen && w.Fullscreen != "" {
		if n := w.Root.find(w.Fullscreen); n != nil {
			ui.Box(c).Grow(1).MinWidth(0).MinHeight(0).Clip().Children(func() { workspaceNode(c, m, n) })
			return
		}
	}
	centerRightColumns(c, m, center, render)
}
func workspaceNode(c *ui.Context, m Model, n *WorkspacePane) {
	box := ui.Box(c).Key(n.ID).Fill().MinWidth(0).MinHeight(0).Clip()
	box.Children(func() {
		if n.First != nil {
			bounds := box.Bounds()
			total := bounds.W
			if n.Axis == "vertical" {
				total = bounds.H
			}
			if total <= 1 {
				total = 400
			}
			if n.Ratio == 0 {
				n.Ratio = 0.5
			}
			size := (total - 1) * n.Ratio
			first := func() { workspaceNode(c, m, n.First) }
			second := func() { workspaceNode(c, m, n.Second) }
			if n.Axis == "vertical" {
				ui.SplitVertical(c, &size, first, second).Fill()
			} else {
				ui.Split(c, &size, first, second).Fill()
			}
			n.Ratio = size / (total - 1)
			return
		}
		if box.FocusWithin() {
			m.Sizes.Workspace.Active = n.ID
			m.Sizes.Workspace.route = n.Kind
		}
		tab := n.Kind
		local := workspaceContentModel(m, n)
		local.RightPaneTab = &tab
		local.workspacePane = n
		width := box.Bounds().W
		local.FilePaneWidth = &width
		local.OnHideBrowser = nil // Visibility belongs to the whole layout, not each seat.
		local.OnOpenBrowserPreview = func() { tab = "browser" }
		rightPane(c, local)
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
	kinds := []string{"files", "diff", "agents"}
	if m.HostTerminalStore != nil {
		kinds = append(kinds, "host-terminal")
	}
	if m.BrowserContent != nil {
		kinds = append(kinds, "browser")
	}
	next := ""
	for _, kind := range kinds {
		if w.Root.kind(kind) == nil {
			next = kind
			break
		}
	}
	if independentPaneKind(n.Kind) {
		next = n.Kind
	}
	split := func(axis string) {
		old := *n
		newPane := w.leaf(next)
		copySchedulePaneTarget(newPane, &old)
		copyDiffPane(newPane, &old)
		copyPlanPane(newPane, &old)
		copyChangesReviewPane(newPane, &old)
		copySessionInspectorPane(newPane, &old)
		copySubagentChatPane(newPane, &old)
		if next == "host-terminal" && n.Kind == "host-terminal" {
			newPane.TerminalSource = old.ID
		}
		*n = WorkspacePane{ID: w.leaf("").ID, Axis: axis, Ratio: 0.5, First: &old, Second: newPane}
		w.Active = newPane.ID
		w.Fullscreen = ""
		w.route = newPane.Kind
	}
	ui.Row(c).FillWidth().Gap(3).Padding(4).Children(func() {
		title := paneLabel("选择面板", "Focus pane")
		if w.Active == n.ID {
			title = paneLabel("当前面板", "Active pane")
		}
		if ui.Button(c, title).FontSize(theme.InterfaceSize(10)).Clicked() {
			w.Active = n.ID
			w.route = n.Kind
		}
		label := paneLabel("面板全屏", "Maximize pane")
		if w.Fullscreen == n.ID {
			label = paneLabel("退出全屏", "Restore pane")
		}
		if ui.Button(c, label).FontSize(theme.InterfaceSize(10)).Clicked() {
			if w.Fullscreen == n.ID {
				w.Fullscreen = ""
			} else {
				w.Fullscreen = n.ID
				w.Active = n.ID
				w.route = n.Kind
			}
		}
		if ui.Button(c, "×").Label(paneLabel("关闭面板", "Close pane")).Clicked() {
			w.pending = func() {
				last := w.Root.First == nil
				closed := w.close(n.ID)
				if closed && last && m.OnToggleTools != nil {
					m.OnToggleTools()
				}
			}
		}
	})
	ui.Row(c).FillWidth().Gap(3).Padding(4).Children(func() {
		if next != "" {
			if ui.Button(c, paneLabel("左右 +", "Right +")).FontSize(theme.InterfaceSize(10)).Clicked() {
				w.pending = func() { split("horizontal") }
			}
			if ui.Button(c, paneLabel("上下 +", "Down +")).FontSize(theme.InterfaceSize(10)).Clicked() {
				w.pending = func() { split("vertical") }
			}
		}
		if ui.Button(c, paneLabel("重置布局", "Reset layout")).FontSize(theme.InterfaceSize(10)).Clicked() {
			w.pending = func() {
				w.resetTo(n)
			}
		}
	})
	if w.closing == n.ID {
		ui.Text(c, paneLabel("此面板有未保存草稿。", "This pane has unsaved drafts."))
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, paneLabel("保留面板", "Keep pane")).Clicked() {
				w.closing = ""
			}
			if ui.Button(c, paneLabel("丢弃并关闭", "Discard and close")).Clicked() {
				w.pending = func() { w.closeNow(n.ID) }
			}
		})
	}
	if w.message != "" {
		ui.Text(c, w.message)
	}
}

func independentPaneKind(kind string) bool {
	return kind == "session-inspector-log" || kind == "changes-review" || kind == "subagentchat" || kind == "plan" || kind == "files" || kind == "browser" || kind == "host-terminal" || kind == "schedule-task" || kind == "diff" || kind == "agents"
}

// SplitWorkspacePane is shared by the pane chrome and editable command registry.
func SplitWorkspacePane(m Model, vertical bool) {
	if !m.RightPaneOpen || m.Sizes == nil {
		return
	}
	prepareWorkspace(m)
	w := m.Sizes.Workspace
	kinds := []string{"files", "diff", "agents"}
	if m.HostTerminalStore != nil {
		kinds = append(kinds, "host-terminal")
	}
	if m.BrowserContent != nil {
		kinds = append(kinds, "browser")
	}
	duplicate := independentPaneKind(w.Root.find(w.Active).Kind)
	if duplicate {
		kinds = []string{w.Root.find(w.Active).Kind}
	}
	for _, kind := range kinds {
		if w.Root.kind(kind) != nil && !duplicate {
			continue
		}
		n := w.Root.find(w.Active)
		old := *n
		added := w.leaf(kind)
		copySchedulePaneTarget(added, &old)
		copyDiffPane(added, &old)
		copyPlanPane(added, &old)
		copyChangesReviewPane(added, &old)
		copySessionInspectorPane(added, &old)
		copySubagentChatPane(added, &old)
		if kind == "host-terminal" && old.Kind == "host-terminal" {
			added.TerminalSource = old.ID
		}
		axis := "horizontal"
		if vertical {
			axis = "vertical"
		}
		*n = WorkspacePane{ID: w.leaf("").ID, Axis: axis, Ratio: 0.5, First: &old, Second: added}
		w.Active = added.ID
		w.Fullscreen = ""
		w.route = kind
		if m.RightPaneTab != nil {
			*m.RightPaneTab = kind
		}
		return
	}
}
func ToggleWorkspacePaneFullscreen(m Model) {
	if !m.RightPaneOpen || m.Sizes == nil {
		return
	}
	prepareWorkspace(m)
	w := m.Sizes.Workspace
	if w.Fullscreen != "" {
		w.Fullscreen = ""
	} else {
		w.Fullscreen = w.Active
	}
}
