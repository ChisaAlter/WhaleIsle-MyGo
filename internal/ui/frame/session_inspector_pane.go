package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
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
	filter = strings.ToLower(strings.Trim(filter, "* "))
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
	rows := source.Records
	if p.Mode == "chat-node" {
		rows = source.Chat
	}
	ui.Row(c).Gap(6).Padding(6).Children(func() {
		for _, mode := range []string{"session-log", "chat-node"} {
			label := paneLabel("原始日志", "Raw Log")
			if mode == "chat-node" {
				label = paneLabel("聊天节点", "Chat Nodes")
			}
			if ui.Button(c, label).Disabled(p.Mode == mode || (p.Mode == "" && mode == "session-log")).Clicked() {
				p.Mode = mode
				p.filterOpen = false
				p.filterActive = -1
				p.navigation = chat.InspectorNavigation{SessionID: source.SessionID}
				p.breadcrumb = nil
				p.Filter = ""
				p.draft = ""
				p.selected = ""
				p.collapsed = map[string]bool{}
				p.list = ui.ListState{FollowEnd: true}
			}
		}
		ui.Text(c, fmt.Sprintf("%d", len(rows)))
		if ui.Button(c, paneLabel("拾取聊天节点", "Pick chat node")).Clicked() {
			p.navigation.Picking = !p.navigation.Picking
			p.navigation.Notice = ""
		}
		if p.navigation.Picking {
			ui.Text(c, paneLabel("点击聊天中的选择按钮；再次点击拾取可取消", "Choose a node in chat; click Pick again to cancel"))
		}
	})
	inspectorTypeFilter(c, p, rows)
	state := m.Store.Snapshot()
	ui.Row(c).Gap(4).Padding(6).Children(func() {
		if ui.Button(c, paneLabel("加载更早", "Load earlier")).Disabled(!state.HistoryHasMore || state.HistoryPageBusy).Clicked() {
			p.list.FollowEnd = false
			m.Store.LoadEarlierHistory(context.Background())
		}
		if ui.Button(c, paneLabel("跟随最新", "Follow latest")).Clicked() {
			p.list.FollowEnd = true
			p.list.ScrollToEnd()
		}
	})
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
		available := body.Bounds().H
		if available <= 0 {
			available = 400
		}
		shown := inspectorVisibleRows(rows, p.Filter, p.collapsed)
		if len(shown) == 0 {
			ui.Text(c, paneLabel("当前窗口没有匹配记录", "No matching records in the loaded window"))
		}
		children := map[string]bool{}
		for _, r := range rows {
			children[r.Parent] = true
		}
		p.list.Key = func(i int) any { return shown[i].Key }
		ui.List(c, &p.list, len(shown), func(i int) {
			r := shown[i]
			ui.Row(c).Gap(4).FillWidth().Children(func() {
				if children[r.Key] {
					label := "▾"
					if p.collapsed[r.Key] {
						label = "▸"
					}
					if ui.Button(c, label).Clicked() {
						p.collapsed[r.Key] = !p.collapsed[r.Key]
						p.list.FollowEnd = false
					}
				}
				caption := r.Type
				if r.Parent != "" {
					caption = "  " + caption
				}
				if r.Seq > 0 {
					caption = fmt.Sprintf("%.0f · %s", r.Seq, caption)
				}
				if r.Time > 0 && p.Mode != "chat-node" {
					caption = time.UnixMilli(int64(r.Time)).UTC().Format("15:04:05.000") + " · " + caption
				}
				if ui.Button(c, caption).Clicked() {
					p.selectInspectorRow(r)
				}
			})
		}).Grow(1).MinHeight(0)
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
			ui.Column(c).Height(available * p.inspectorDetailsPercent() / 100).Shrink(0).MinHeight(0).FillWidth().Children(func() {
				inspectorDetailsDivider(c, p, available)
				if p.Mode == "chat-node" {
					ui.Row(c).Gap(4).Children(func() {
						for i, path := range p.breadcrumb {
							label := path
							if o, exists := objects[path]; exists {
								label = o.Label
							}
							if ui.Button(c, label).Clicked() {
								p.breadcrumb = p.breadcrumb[:i+1]
								p.tree = inspectorJSONTreeState{}
							}
						}
					})
					ui.Row(c).Gap(4).Children(func() {
						for _, link := range object.Links {
							if next, exists := objects[link]; exists {
								if ui.Button(c, next.Label+" →").Clicked() {
									p.followInspectorReference(link, rows)
								}
							}
						}
					})
				}
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, object.Label).Grow(1)
					if ui.Button(c, paneLabel("关闭详情", "Close details")).Clicked() {
						p.selected = ""
						p.breadcrumb = nil
						p.tree = inspectorJSONTreeState{}
						p.navigation.Target = nil
					}
					if ui.Button(c, paneLabel("复制原始 JSON", "Copy raw JSON")).Clicked() {
						mygo.Clipboard.WriteText(string(object.Raw))
					}
				})
				ui.Scroll(c.Key("inspector-details:" + p.binding + ":" + key)).Grow(1).MinHeight(0).FillWidth().Children(func() { inspectorJSONTree(c, &p.tree, object.Raw) })
			})
		}
	})
}
