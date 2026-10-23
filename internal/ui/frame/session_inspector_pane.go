package frame

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Only presentation preferences are persisted. Raw logs, selected values and
// source bindings never enter workspace layout persistence.
type SessionInspectorPaneState struct {
	filterOpen                                              bool
	filterActive                                            int
	filterQuery                                             string
	filterList                                              ui.ListState
	detailsPercent                                          float32
	detailsDragging                                         bool
	detailsDragBase, detailsDragDelta, detailsDragAvailable float32
	navigation                                              chat.InspectorNavigation
	breadcrumb                                              []string
	tree                                                    inspectorJSONTreeState
	Mode                                                    string `json:"mode,omitempty"`
	Filter                                                  string `json:"filter,omitempty"`
	binding, draft, selected                                string
	collapsed                                               map[string]bool
	list                                                    ui.ListState
	tableX, tableY                                          ui.ScrollState
	// pendingPick is an inspect request that arrived before this pane rendered
	// its binding, e.g. a tool card clicked from the transcript.
	pendingPick *chat.InspectorTarget
}

func copySessionInspectorPane(to, from *WorkspacePane) {
	if to.Kind == "session-inspector-log" && from.Inspector != nil {
		to.Inspector = &SessionInspectorPaneState{Mode: from.Inspector.Mode, Filter: from.Inspector.Filter}
	}
}
func sessionInspectorGuide(c *ui.Context, m Model) {
	if m.DisableDeveloperTools {
		return
	}
	if ui.Button(c, paneLabel("会话日志 · 检查当前会话原始事件", "Session Log · Inspect this session's raw events")).Clicked() && m.RightPaneTab != nil {
		*m.RightPaneTab = "session-inspector-log"
	}
}

// inspectActionFor keeps the card's inspect action available in every layout: the
// trajectory target when the chat view tabs expose it, otherwise the session log
// at the record that carries the call. Both are real destinations; neither
// invents a selection.
func inspectActionFor(trajectory func(callID string), inspector func(callID, messageID string, seq float64)) func(callID, messageID string, seq float64) {
	if trajectory != nil {
		return func(callID string, _ string, _ float64) { trajectory(callID) }
	}
	return inspector
}

// toolCallInspector is the transcript's per-call inspect action. With developer
// tools disabled there is no inspector seat, so the card keeps its inspect action
// hidden instead of offering a button that opens nothing.
func (m Model) toolCallInspector() func(callID, messageID string, seq float64) {
	if m.DisableDeveloperTools || m.RightPaneTab == nil || m.Sizes == nil || m.Store == nil {
		return nil
	}
	return func(_, messageID string, seq float64) {
		m.inspectToolCall(messageID, seq)
	}
}

// inspectToolCall routes the right dock to the session log and asks the inspector
// to select the record that carries the call. The log keys chat rows by message
// ID, so the owning message identifies the row; seq stays as the fallback anchor
// for records without an ID. A call outside the loaded window produces the
// inspector's own "no matching record" notice rather than a fake selection.
func (m Model) inspectToolCall(messageID string, seq float64) {
	if m.RightPaneTab == nil || m.Sizes == nil {
		return
	}
	*m.RightPaneTab = "session-inspector-log"
	m.RightPaneOpen = true
	prepareWorkspace(m)
	target := chat.InspectorTarget{ID: messageID, Seq: seq}
	for _, pane := range inspectorPaneStates(m.Sizes.Workspace) {
		pane.Mode = "chat-node"
		pane.pendingPick = &target
	}
}

// inspectorPaneStates lists every mounted session-log pane, creating its state so
// a request can be queued before the pane first renders.
func inspectorPaneStates(w *WorkspacePanes) []*SessionInspectorPaneState {
	if w == nil || w.Root == nil {
		return nil
	}
	var out []*SessionInspectorPaneState
	var visit func(*WorkspacePane)
	visit = func(n *WorkspacePane) {
		if n == nil {
			return
		}
		if n.Kind == "session-inspector-log" {
			if n.Inspector == nil {
				n.Inspector = &SessionInspectorPaneState{}
			}
			out = append(out, n.Inspector)
		}
		for _, tab := range n.DockTabs {
			visit(tab)
		}
		visit(n.First)
		visit(n.Second)
	}
	visit(w.Root)
	return out
}
func inspectorVisibleRows(rows []session.InspectorRecord, filter string, folded map[string]bool) []session.InspectorRecord {
	byKey := map[string]session.InspectorRecord{}
	keep := map[string]bool{}
	filter = strings.ToLower(inspectorFilterQuery(filter))
	for _, r := range rows {
		byKey[r.Key] = r
	}
	for _, r := range rows {
		if filter == "" || strings.Contains(strings.ToLower(r.Type), filter) {
			keep[r.Key] = true
			for p := r.Parent; p != ""; p = byKey[p].Parent {
				keep[p] = true
				if _, ok := byKey[p]; !ok {
					break
				}
			}
		}
	}
	var out []session.InspectorRecord
	for _, r := range rows {
		if !keep[r.Key] {
			continue
		}
		hidden := false
		for p := r.Parent; p != ""; p = byKey[p].Parent {
			if filter == "" && folded[p] {
				hidden = true
				break
			}
			if _, ok := byKey[p]; !ok {
				break
			}
		}
		if !hidden {
			out = append(out, r)
		}
	}
	return out
}
func sessionInspectorPane(c *ui.Context, m Model) {
	if m.DisableDeveloperTools {
		ui.Text(c, paneLabel("开发工具已关闭", "Developer Tools are disabled"))
		return
	}
	if m.Store == nil || m.workspacePane == nil {
		return
	}
	n := m.workspacePane
	if n.Inspector == nil {
		n.Inspector = &SessionInspectorPaneState{}
	}
	p := n.Inspector
	source := m.Store.InspectorSnapshot()
	if p.binding != source.Binding {
		p.binding = source.Binding
		p.filterOpen = false
		p.detailsDragging = false
		p.filterActive = -1
		p.filterList = ui.ListState{}
		p.navigation = chat.InspectorNavigation{SessionID: source.SessionID}
		p.breadcrumb = nil
		p.tree = inspectorJSONTreeState{}
		p.selected = ""
		p.draft = p.Filter
		p.collapsed = map[string]bool{}
		p.list = ui.ListState{FollowEnd: true}
		p.tableX, p.tableY = ui.ScrollState{}, ui.ScrollState{}
		p.detailsPercent = 0
	}
	if source.SessionID == "" {
		ui.Text(c, paneLabel("选择会话以检查日志", "Select a session to inspect its log"))
		return
	}
	if len(p.navigation.Nodes) > 0 {
		source.Chat = nil
		for _, msg := range p.navigation.Nodes {
			raw, _ := json.Marshal(msg)
			source.Chat = append(source.Chat, session.InspectorRecord{Key: "chat:" + msg.ID, Type: string(msg.Kind), Seq: msg.Seq, Raw: raw})
		}
	}
	inspectorPresentation(c, p, source, m.Store.Snapshot(), theme.Resolve(m.ThemeID).Active(m.SystemDark), func() { m.Store.LoadEarlierHistory(context.Background()) })
}

func inspectorPresentation(c *ui.Context, p *SessionInspectorPaneState, source session.InspectorSnapshot, state session.ViewState, colors theme.Colors, loadOlder func()) {
	rows := source.Records
	if p.Mode == "chat-node" {
		rows = source.Chat
	}
	root := ui.Column(c.Key("inspector-root")).Fill().MinHeight(0).Font(theme.RenderPreferences().Interface.Family).FontSize(11).LineHeight(0).TextColor(chrome.Color(colors.FG))
	if root.Shortcut(0, ui.KeyEscape) && p.navigation.Picking {
		p.navigation.Picking = false
		p.navigation.Notice = ""
	}
	root.Children(func() {
		inspectorToolbar(c, p, source, rows, state.HistoryHasMore, state.HistoryPageBusy, loadOlder, colors)
		if p.navigation.Picking {
			ui.Text(c, paneLabel("点击对话中的选择按钮；Esc 取消。", "Choose a node in Chat; Esc cancels.")).FontSize(11).Padding(4, 8).TextColor(chrome.Color(colors.Secondary))
		}

		if state.HistoryPageErr != "" {
			ui.Text(c, state.HistoryPageErr)
		}
		// A pending inspect request (a tool card clicked before this pane had rows)
		// turns into a pick once the chat projection exists, so the selection is the
		// real record rather than an invented one.
		p.applyPendingPick(rows)
		p.consumeInspectorPick(rows)
		if p.navigation.Notice != "" {
			ui.Text(c, p.navigation.Notice)
		}
		body := ui.Column(c).Grow(1).MinHeight(0).FillWidth()
		body.Children(func() {
			available := root.Bounds().H
			if available <= 0 {
				available = 400
			}
			inspectorTable(c, p, rows, colors)
			objects := inspectorObjects(source)
			for group, members := range p.navigation.Groups {
				key := "chat:" + group
				o, exists := objects[key]
				if !exists {
					continue
				}
				for _, id := range members {
					child := "chat:" + id
					if _, ok := objects[child]; ok {
						o.Links = append(o.Links, child)
						node := objects[child]
						node.Links = append(node.Links, key)
						objects[child] = node
					}
				}
				objects[key] = o
			}
			key := p.selected
			if len(p.breadcrumb) > 0 {
				key = p.breadcrumb[len(p.breadcrumb)-1]
			}
			object, ok := objects[key]
			if !ok {
				for _, row := range rows {
					if row.Key == key {
						object = inspectorObject{Key: key, Label: row.Type, Raw: row.Raw}
						ok = true
						break
					}
				}
			}
			if ok {
				inspectorDetails(c, p, object, objects, rows, available, colors)
			}
		})
	})
}
