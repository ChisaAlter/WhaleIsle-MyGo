package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Only presentation preferences are persisted. Raw logs, selected values and
// source bindings never enter workspace layout persistence.
type SessionInspectorPaneState struct {
	filterOpen                                              bool
	filterActive                                            int
	filterHovered                                           int
	filterQuery                                             string
	filterList                                              ui.ListState
	detailsPercent                                          float32
	detailsDragging                                         bool
	detailsDragBase, detailsDragDelta, detailsDragAvailable float32
	navigation                                              chat.InspectorNavigation
	breadcrumb                                              []string
	tree                                                    inspectorJSONTreeState
	objectTree                                              inspectorObjectTreeState
	objects                                                 map[string]inspectorObject
	dismissedHistoryError                                   string
	Mode                                                    string `json:"mode,omitempty"`
	Filter                                                  string `json:"filter,omitempty"`
	binding, draft, selected                                string
	collapsed                                               map[string]inspectorDisclosure
	list                                                    ui.ListState
	tableX, tableY                                          ui.ScrollState
	scroll                                                  inspectorScrollState
	pulse                                                   inspectorPulseState
	// pendingPick is an inspect request that arrived before this pane rendered
	// its binding, e.g. a tool card clicked from the transcript.
	pendingPick                         *chat.InspectorTarget
	pendingPickBinding                  string
	rendered, chatProjected, pickQueued bool
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
	return func(callID, messageID string, seq float64) {
		m.inspectToolCall(callID, messageID, seq)
	}
}

// inspectToolCall routes the right dock to the session log and asks the inspector
// to select the containing Chat node. CallID retains the particular nested call;
// messageID and seq identify its projected owner in the current loaded window.
func (m Model) inspectToolCall(callID, messageID string, seq float64) {
	if m.RightPaneTab == nil || m.Sizes == nil {
		return
	}
	*m.RightPaneTab = "session-inspector-log"
	if !m.RightPaneOpen && m.OnToggleTools != nil {
		m.OnToggleTools()
	}
	m.RightPaneOpen = true
	prepareWorkspace(m)
	w := m.Sizes.Workspace
	n := w.Root.find(w.Active)
	if n == nil || n.Kind != "session-inspector-log" {
		return
	}
	if n.Inspector == nil {
		n.Inspector = &SessionInspectorPaneState{}
	}
	pane := n.Inspector
	binding, sessionID := "", ""
	if m.Store != nil {
		binding, sessionID = m.Store.InspectorBinding(), m.Store.SelectedSessionID()
	}
	if pane.Mode != "chat-node" {
		pane.changeInspectorMode("chat-node", sessionID)
	}
	w.Root.eachResource(func(other *WorkspacePane) {
		if other.Inspector != nil {
			other.Inspector.cancelInspectorNavigation()
		}
	})
	pane.pendingPick = &chat.InspectorTarget{ID: messageID, Seq: seq, CallID: callID}
	pane.pendingPickBinding = binding
	pane.pickQueued = true
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

// Type filtering retains matches and their ancestor context before disclosure
// choices are applied. A confirmed filter resets those choices, not each frame.
func inspectorTypeRows(rows []session.InspectorRecord, filter string) []session.InspectorRecord {
	query := strings.ToLower(inspectorFilterQuery(filter))
	if query == "" {
		return rows
	}
	byKey := make(map[string]session.InspectorRecord, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}
	keep := map[string]bool{}
	for _, row := range rows {
		if !strings.Contains(strings.ToLower(row.Type), query) {
			continue
		}
		keep[row.Key] = true
		for parent := row.Parent; parent != "" && !keep[parent]; parent = byKey[parent].Parent {
			keep[parent] = true
		}
	}
	branches := map[string]bool{}
	for key := range keep {
		if parent := byKey[key].Parent; parent != "" {
			branches[parent] = true
		}
	}
	var out []session.InspectorRecord
	for _, row := range rows {
		if keep[row.Key] {
			row.Disclosure = ""
			if branches[row.Key] {
				row.Disclosure = "open"
			}
			out = append(out, row)
		}
	}
	return out
}

type inspectorDisclosure struct {
	state     string
	collapsed bool
}

func inspectorRowCollapsed(row session.InspectorRecord, choices map[string]inspectorDisclosure) bool {
	if choice, ok := choices[row.Key]; ok && choice.state == row.Disclosure {
		return choice.collapsed
	}
	return row.Disclosure == "closed"
}

func (p *SessionInspectorPaneState) setInspectorCollapsed(row session.InspectorRecord, collapsed bool) {
	if p.collapsed == nil {
		p.collapsed = map[string]inspectorDisclosure{}
	}
	p.collapsed[row.Key] = inspectorDisclosure{state: row.Disclosure, collapsed: collapsed}
}

func inspectorDisclosedRows(rows []session.InspectorRecord, folded map[string]inspectorDisclosure) []session.InspectorRecord {
	byKey := make(map[string]session.InspectorRecord, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}
	var out []session.InspectorRecord
	for _, row := range rows {
		hidden := false
		for parent := row.Parent; parent != ""; parent = byKey[parent].Parent {
			if inspectorRowCollapsed(byKey[parent], folded) {
				hidden = true
				break
			}
		}
		if !hidden {
			out = append(out, row)
		}
	}
	return out
}

func inspectorVisibleRows(rows []session.InspectorRecord, filter string, folded map[string]inspectorDisclosure) []session.InspectorRecord {
	return inspectorDisclosedRows(inspectorTypeRows(rows, filter), folded)
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
	p.rendered = true
	source := m.Store.InspectorSnapshot()
	p.bindInspectorSource(source.Binding, source.SessionID)
	if source.SessionID == "" {
		ui.Text(c, paneLabel("选择会话以检查日志", "Select a session to inspect its log"))
		return
	}
	inspectorPresentation(c, p, source, renderSnapshot(m), theme.Resolve(m.ThemeID).Active(m.SystemDark), func() { m.Store.LoadEarlierHistory(context.Background()) })
}

func inspectorPresentation(c *ui.Context, p *SessionInspectorPaneState, source session.InspectorSnapshot, state session.ViewState, colors theme.Colors, loadOlder func()) {
	source.Chat = inspectorChatRows(source, &p.navigation)
	objects := inspectorNativeObjects(source, &p.navigation)
	p.objects = objects
	rows := source.Records
	if p.Mode == "chat-node" {
		rows = source.Chat
	}
	root := ui.Column(c.Key("inspector-root")).Fill().MinHeight(0).Font(theme.RenderPreferences().Interface.Family).FontSize(11).LineHeight(0).TextColor(chrome.Color(colors.FG))
	if root.Shortcut(0, ui.KeyEscape) && p.navigation.Picking {
		p.navigation.CancelPicking()
	}
	root.Children(func() {
		inspectorToolbar(c, p, source, rows, state.HistoryHasMore, state.HistoryPageBusy, loadOlder, colors)
		if p.navigation.Picking {
			ui.Text(c, paneLabel("点击对话中的元素；Esc 取消。", "Choose a node in Chat; Esc cancels.")).FontSize(11).Padding(4, 8).TextColor(chrome.Color(colors.Secondary))
		}

		if state.HistoryPageErr != "" && state.HistoryPageErr != p.dismissedHistoryError {
			ui.Row(c).FillWidth().Padding(4, 8).Gap(8).Children(func() {
				ui.Text(c, state.HistoryPageErr).Grow(1).FontSize(11)
				button := inspectorButton(c, paneLabel("关闭错误", "Dismiss error"))
				button.Children(func() { inspectorText(c, paneLabel("关闭", "Dismiss")) })
				if button.Clicked() {
					p.dismissedHistoryError = state.HistoryPageErr
				}
			})
		} else if state.HistoryPageErr == "" {
			p.dismissedHistoryError = ""
		}
		// A pending inspect request (a tool card clicked before this pane had rows)
		// turns into a pick once the chat projection exists, so the selection is the
		// real record rather than an invented one.
		p.applyPendingPick(rows)
		hadPick := p.navigation.Picked != nil
		p.consumeInspectorPick(rows)
		if hadPick {
			c.Services().Invalidate()
		}
		widgets.Notice(c, "session_inspector_pane:294", "", p.navigation.Notice != "", func() string { return p.navigation.Notice })
		body := ui.Column(c).Grow(1).MinHeight(0).FillWidth()
		body.Children(func() {
			available := root.Bounds().H
			if available <= 0 {
				available = 400
				c.Invalidate()
			}
			inspectorTable(c, p, rows, colors)
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
			if p.selected != "" {
				if !ok {
					object = inspectorObject{Key: key, Missing: true}
				}
				inspectorDetails(c, p, object, objects, rows, available, colors)
			}
		})
	})
}
