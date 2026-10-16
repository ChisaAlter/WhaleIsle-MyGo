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
	navigation               chat.InspectorNavigation
	breadcrumb               []string
	tree                     inspectorJSONTreeState
	Mode                     string `json:"mode,omitempty"`
	Filter                   string `json:"filter,omitempty"`
	binding, draft, selected string
	collapsed                map[string]bool
	list                     ui.ListState
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
			if folded[p] {
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
	ui.Row(c).Gap(4).Padding(6).Children(func() {
		ui.TextInput(c, &p.draft).Grow(1).Placeholder(paneLabel("事件类型筛选", "Filter event type"))
		if ui.Button(c, paneLabel("应用", "Apply")).Clicked() {
			p.Filter = p.draft
			p.selected = ""
			p.collapsed = map[string]bool{}
			p.list = ui.ListState{}
			p.list.ScrollTo(0, ui.Start)
		}
		if ui.Button(c, paneLabel("清除", "Clear")).Clicked() {
			p.Filter = ""
			p.draft = ""
			p.selected = ""
			p.collapsed = map[string]bool{}
		}
	})
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
	p.consumeInspectorPick(rows)
	if p.navigation.Notice != "" {
		ui.Text(c, p.navigation.Notice)
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
			if ui.Button(c, paneLabel("复制原始 JSON", "Copy raw JSON")).Clicked() {
				mygo.Clipboard.WriteText(string(object.Raw))
			}
		})
		ui.Scroll(c.Key("inspector-details:" + p.binding + ":" + key)).Height(180).FillWidth().Children(func() { inspectorJSONTree(c, &p.tree, object.Raw) })
	}
}
