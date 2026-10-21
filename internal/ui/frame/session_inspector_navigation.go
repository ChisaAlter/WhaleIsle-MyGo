package frame

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func sessionInspectorChatNavigation(m Model) *chat.InspectorNavigation {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		return nil
	}
	visible := !m.DisableDeveloperTools && workspaceVisible(m, "session-inspector-log") && m.Store != nil
	var cancelHidden func(*WorkspacePane)
	cancelHidden = func(n *WorkspacePane) {
		if n == nil {
			return
		}
		if n.Inspector != nil && (!visible || n.Kind != "session-inspector-log" || (m.Sizes.Workspace.Fullscreen != "" && m.Sizes.Workspace.Fullscreen != n.ID)) {
			n.Inspector.navigation.Picking = false
			n.Inspector.navigation.Target = nil
		}
		cancelHidden(n.First)
		cancelHidden(n.Second)
	}
	cancelHidden(m.Sizes.Workspace.Root)
	if !visible {
		return nil
	}
	w := m.Sizes.Workspace
	var chosen *SessionInspectorPaneState
	var visit func(*WorkspacePane)
	visit = func(n *WorkspacePane) {
		if n == nil {
			return
		}
		if n.Kind == "session-inspector-log" && n.Inspector != nil {
			p := n.Inspector
			if p.binding == m.Store.InspectorBinding() && (chosen == nil || p.navigation.Picking || p.navigation.Target != nil) {
				chosen = p
			}
		}
		visit(n.First)
		visit(n.Second)
	}
	if w.Fullscreen != "" {
		visit(w.Root.find(w.Fullscreen))
	} else {
		visit(w.Root)
	}
	if chosen == nil {
		return nil
	}
	return &chosen.navigation
}
func inspectorPickRecord(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	for _, r := range rows {
		if r.Key == "chat:"+target.ID {
			return r.Key
		}
	}
	for _, r := range rows {
		if r.Seq == target.Seq && r.Seq > 0 && !strings.Contains(r.Key, "/chunk:") {
			return r.Key
		}
	}
	return ""
}
func (p *SessionInspectorPaneState) selectInspectorRow(row session.InspectorRecord) {
	p.selected = row.Key
	p.breadcrumb = []string{row.Key}
	p.tree = inspectorJSONTreeState{}
	p.list.FollowEnd = false
	p.navigation.Target = &chat.InspectorTarget{ID: strings.TrimPrefix(row.Key, "chat:"), Seq: row.Seq}
	if !strings.HasPrefix(row.Key, "chat:") {
		p.navigation.Target.ID = ""
	}
}
// applyPendingPick turns a queued inspect request into a pick once the pane has
// rows to resolve it against: immediately when the record is present, or after
// the chat projection loads so the "no matching record" notice is only raised
// once there is something real to search.
func (p *SessionInspectorPaneState) applyPendingPick(rows []session.InspectorRecord) {
	if p.pendingPick == nil {
		return
	}
	if inspectorPickRecord(rows, *p.pendingPick) == "" && len(p.navigation.Nodes) == 0 {
		return
	}
	p.navigation.Picked = p.pendingPick
	p.pendingPick = nil
}

func (p *SessionInspectorPaneState) consumeInspectorPick(rows []session.InspectorRecord) {
	target := p.navigation.Picked
	if target == nil {
		return
	}
	p.navigation.Picked = nil
	key := inspectorPickRecord(rows, *target)
	if key == "" {
		p.navigation.Notice = paneLabel("当前加载窗口没有对应记录", "No matching record in this loaded window")
		p.navigation.Picking = true
		return
	}
	p.Filter = ""
	p.draft = ""
	p.collapsed = map[string]bool{}
	for i, row := range rows {
		if row.Key == key {
			p.selectInspectorRow(row)
			p.list.ScrollTo(i, ui.Center)
			return
		}
	}
}

func (p *SessionInspectorPaneState) followInspectorReference(key string, rows []session.InspectorRecord) {
	p.breadcrumb = append(p.breadcrumb, key)
	p.tree = inspectorJSONTreeState{}
	for i, row := range rows {
		if row.Key == key {
			p.selected = key
			p.Filter = ""
			p.draft = ""
			p.collapsed = map[string]bool{}
			p.list.ScrollTo(i, ui.Center)
			target := chat.InspectorTarget{Seq: row.Seq}
			if strings.HasPrefix(key, "chat:") {
				target.ID = strings.TrimPrefix(key, "chat:")
			}
			p.navigation.Target = &target
			return
		}
	}
}
