package frame

import (
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// sessionAncestry follows only summaries actually supplied by this Host. Missing
// links and cycles terminate the chain; IDs are never synthesized as ancestors.
func sessionAncestry(items []session.Item, selected string) []session.Item {
	if selected == "" {
		return nil
	}
	var seen map[string]bool
	var chain []session.Item
	for id := selected; id != ""; {
		if seen != nil && seen[id] {
			break
		}
		found := false
		var cur session.Item
		for _, it := range items {
			if it.SessionID == id {
				cur = it
				found = true
				break
			}
		}
		if !found {
			break
		}
		chain = append(chain, cur)
		if cur.ParentID == "" {
			break
		}
		if seen == nil {
			seen = map[string]bool{id: true}
		} else {
			seen[id] = true
		}
		id = cur.ParentID
	}
	slices.Reverse(chain)
	return chain
}

func paintSessionAncestry(c *ui.Context, chain []session.Item, open func(string), blocked ...string) {
	ui.Row(c).FillWidth().MinWidth(0).MaxWidth(280).Clip().AlignItems(ui.Center).Gap(4).Children(func() {
		for i, it := range chain {
			if i > 0 {
				ui.Text(c, "/").FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).Shrink(0)
			}
			title := strings.TrimSpace(it.Title)
			if title == "" {
				// An untitled Session reads as the localized new-session label; the
				// internal id never reaches the title band.
				title = i18n.T(i18n.KeySidebarNewSessionFallback)
			}
			if i == len(chain)-1 || it.Origin == "subagent" || it.ContactError != "" || slices.Contains(blocked, it.SessionID) || open == nil {
				ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).MinWidth(0).Shrink(1).MaxWidth(220).NoWrap().Ellipsis("…").Padding(4, 8)
			} else if ui.Button(c, title).Key("session-ancestor-"+it.SessionID).Tooltip(title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(400).MinWidth(0).Shrink(1).MaxWidth(220).NoWrap().Ellipsis("…").Padding(4, 8).BorderWidth(0).Background(ui.Transparent).Clicked() {
				open(it.SessionID)
			}
		}
	})
}

func sessionAncestryHeader(c *ui.Context, m Model, snap session.ViewState) {
	chain := sessionAncestry(snap.Items, snap.SelectedID)
	if len(chain) == 0 {
		if snap.SelectedID != "" {
			ui.Text(c, i18n.T(i18n.KeySidebarNewSessionFallback)).FontSize(theme.InterfaceSize(14)).
				FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).Padding(4, 8)
		}
		return
	}
	binding := m.Store.InspectorBinding()
	paintSessionAncestry(c, chain, func(id string) {
		if m.Store.InspectorBinding() != binding {
			return
		}
		live := m.Store.Snapshot()
		if live.SelectedID != snap.SelectedID || slices.Contains(live.ArchivedSessionIDs, id) {
			return
		}
		ancestors := sessionAncestry(live.Items, live.SelectedID)
		for i, it := range ancestors {
			if i == len(ancestors)-1 || it.SessionID != id || it.Origin == "subagent" || it.ContactError != "" {
				continue
			}
			if routeRegisteredSessionOpen(m, it) {
				return
			}
			if m.Sizes != nil && m.Sizes.Workspace != nil {
				m.Sizes.Workspace.closeMainChild()
			}
			m.Store.MarkMainConversationViewed(id)
			if m.OnSelectSession != nil {
				m.OnSelectSession(id)
			} else {
				m.Store.Select(id)
			}
			return
		}
	}, snap.ArchivedSessionIDs...)
}
