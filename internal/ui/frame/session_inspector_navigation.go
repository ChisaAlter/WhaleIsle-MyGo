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
	w := m.Sizes.Workspace
	binding := ""
	if m.Store != nil {
		binding = m.Store.InspectorBinding()
	}
	var chosen *SessionInspectorPaneState
	rank := -1
	w.Root.eachResource(func(n *WorkspacePane) {
		p := n.Inspector
		if p == nil {
			return
		}
		if m.DisableDeveloperTools || m.Store == nil || !inspectorPaneVisible(w, n, m.RightPaneOpen) {
			p.cancelInspectorNavigation()
			return
		}
		if p.binding != binding {
			pending, owner := p.pendingPick, p.pendingPickBinding
			p.cancelInspectorNavigation()
			if owner == binding {
				p.pendingPick, p.pendingPickBinding = pending, owner
			}
			return
		}
		priority := 0
		if n.ID == w.Active {
			priority = 1
		}
		if p.navigation.Target != nil {
			priority += 2
		}
		if p.navigation.Picking {
			priority += 4
		}
		if priority > rank {
			chosen, rank = p, priority
		}
	})
	if chosen == nil {
		return nil
	}
	// The Chat has one picker owner, including when inspectors are split or float.
	w.Root.eachResource(func(n *WorkspacePane) {
		if p := n.Inspector; p != nil && p != chosen {
			if chosen.navigation.Picking || p.navigation.Picking {
				p.cancelInspectorNavigation()
			} else {
				p.navigation.CancelReveal()
			}
		}
	})
	return &chosen.navigation
}

// Chat projects its current runtime groups once. Every visible inspector sees
// that same projection; only the selected navigation owner handles gestures.
func syncInspectorChatProjection(m Model, source *chat.InspectorNavigation) {
	if m.Store == nil || m.DisableDeveloperTools || m.GroupRoom || m.Sizes == nil || m.Sizes.Workspace == nil {
		return
	}
	w := m.Sizes.Workspace
	binding := m.Store.InspectorBinding()
	w.Root.eachResource(func(n *WorkspacePane) {
		p := n.Inspector
		if p == nil || !inspectorPaneVisible(w, n, m.RightPaneOpen) {
			return
		}
		if p.binding == binding || p.pendingPick != nil && p.pendingPickBinding == binding {
			p.chatProjected = true
		}
		if source != nil && p.binding == binding && p.navigation.SessionID == source.SessionID {
			p.navigation.Nodes, p.navigation.Groups = source.Nodes, source.Groups
		}
	})
}

func inspectorPaneVisible(w *WorkspacePanes, n *WorkspacePane, dockOpen bool) bool {
	if n.Kind != "session-inspector-log" {
		return false
	}
	if n.Float != nil {
		return !n.Float.Minimized
	}
	group := w.Root.dockGroup(n.ID)
	return dockOpen && group != nil && group.dockedLeaf() == n
}

func (p *SessionInspectorPaneState) cancelInspectorNavigation() {
	p.navigation.Picking = false
	p.navigation.Picked, p.navigation.Target = nil, nil
	p.navigation.CancelReveal()
	p.navigation.Notice = ""
	p.pendingPick, p.pendingPickBinding = nil, ""
}

func (p *SessionInspectorPaneState) toggleInspectorPicking() {
	start := !p.navigation.Picking
	p.cancelInspectorNavigation()
	p.navigation.Picking = start
}

func (p *SessionInspectorPaneState) bindInspectorSource(binding, sessionID string) {
	if p.binding == binding {
		return
	}
	pending, owner := p.pendingPick, p.pendingPickBinding
	p.cancelInspectorNavigation()
	if owner == binding {
		p.pendingPick, p.pendingPickBinding = pending, owner
	}
	p.binding = binding
	p.filterOpen, p.detailsDragging = false, false
	p.filterActive = -1
	p.filterList = ui.ListState{}
	p.navigation = chat.InspectorNavigation{SessionID: sessionID}
	p.closeInspectorDetails()
	p.draft = p.Filter
	p.collapsed = map[string]inspectorDisclosure{}
	p.list = ui.ListState{FollowEnd: true}
	p.tableX, p.tableY = ui.ScrollState{}, ui.ScrollState{}
	p.scroll, p.pulse = inspectorScrollState{}, inspectorPulseState{}
}
func inspectorPickRecord(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	for _, r := range rows {
		if r.Key == "chat:"+target.ID {
			return r.Key
		}
	}
	if target.ID != "" {
		for _, r := range rows {
			if r.ChatPickID == target.ID && r.Type == "block-start" {
				return r.Key
			}
		}
		for _, r := range rows {
			if r.ChatPickID == target.ID {
				return r.Key
			}
		}
	}
	for _, r := range rows {
		if r.Seq == target.Seq && r.Seq > 0 && r.Type != "assistant/live" && !strings.Contains(r.Key, "/chunk:") {
			return r.Key
		}
	}
	return ""
}
func (p *SessionInspectorPaneState) selectInspectorRow(row session.InspectorRecord) {
	p.selected = row.Key
	p.breadcrumb = []string{row.Key}
	p.tree = inspectorJSONTreeState{}
	p.locateInspectorChat(row)
}

func (p *SessionInspectorPaneState) locateInspectorChat(row session.InspectorRecord) {
	target := chat.InspectorTarget{ID: row.ChatID, Seq: row.Seq}
	if strings.HasPrefix(row.Key, "chat:") {
		target.ID = strings.TrimPrefix(row.Key, "chat:")
	}
	p.navigation.Target = &target
}

// Reveal keeps a matching filter and unrelated disclosures. The scroll index
// belongs to the resulting visible hierarchy, not the unfiltered source.
func (p *SessionInspectorPaneState) revealInspectorRow(key string, rows []session.InspectorRecord) bool {
	if !inspectorContainsRow(rows, key) {
		return false
	}
	filtered := inspectorTypeRows(rows, p.Filter)
	if !inspectorContainsRow(filtered, key) {
		p.Filter, p.draft = "", ""
		filtered = rows
	}
	byKey := make(map[string]session.InspectorRecord, len(filtered))
	for _, row := range filtered {
		byKey[row.Key] = row
	}
	for parent := byKey[key].Parent; parent != ""; parent = byKey[parent].Parent {
		p.setInspectorCollapsed(byKey[parent], false)
	}
	p.selected = key
	p.scroll.mode, p.scroll.reset, p.scroll.anchor = "scroll", true, nil
	previousOffset := p.tableY.Y
	p.scroll.requestedFrom = &previousOffset
	p.pulse.revealKey, p.pulse.revealGeneration = key, p.pulse.revealGeneration+1
	p.list.FollowEnd = false
	for i, row := range inspectorDisclosedRows(filtered, p.collapsed) {
		if row.Key == key {
			p.list.ScrollTo(i, ui.Center)
			break
		}
	}
	return true
}

func inspectorContainsRow(rows []session.InspectorRecord, key string) bool {
	for _, row := range rows {
		if row.Key == key {
			return true
		}
	}
	return false
}

// applyPendingPick turns a queued inspect request into a pick once the pane has
// rows to resolve it against: immediately when the record is present, or after
// the chat projection loads so the "no matching record" notice is only raised
// once there is something real to search.
func (p *SessionInspectorPaneState) applyPendingPick(rows []session.InspectorRecord) {
	if p.pendingPick == nil {
		return
	}
	if p.pendingPickBinding != p.binding {
		p.pendingPick, p.pendingPickBinding = nil, ""
		return
	}
	if inspectorPickRecord(rows, *p.pendingPick) == "" && len(p.navigation.Nodes) == 0 {
		return
	}
	p.navigation.Picked = p.pendingPick
	p.pendingPick, p.pendingPickBinding = nil, ""
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
	p.revealInspectorRow(key, rows)
	p.breadcrumb = []string{key}
	p.tree = inspectorJSONTreeState{}
	p.navigation.Picking = false
	p.navigation.Target = nil
	p.navigation.Notice = ""
}

func (p *SessionInspectorPaneState) followInspectorReference(key string, rows []session.InspectorRecord) {
	p.breadcrumb = append(p.breadcrumb, key)
	p.revealInspectorReference(key, rows)
}

func (p *SessionInspectorPaneState) backInspectorReference(index int, rows []session.InspectorRecord) {
	p.breadcrumb = p.breadcrumb[:index+1]
	p.revealInspectorReference(p.breadcrumb[index], rows)
}

func (p *SessionInspectorPaneState) revealInspectorReference(key string, rows []session.InspectorRecord) {
	p.tree = inspectorJSONTreeState{}
	p.navigation.Target = nil
	if !p.revealInspectorRow(key, rows) {
		return
	}
	for _, row := range rows {
		if row.Key == key {
			p.locateInspectorChat(row)
			return
		}
	}
}

func (p *SessionInspectorPaneState) closeInspectorDetails() {
	p.selected = ""
	p.breadcrumb = nil
	p.tree = inspectorJSONTreeState{}
	p.detailsPercent = 0
	p.detailsDragging = false
}
