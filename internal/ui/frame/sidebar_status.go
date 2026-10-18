package frame

import "whaleisle/internal/store/interaction"

func sidebarPendingStatuses(m Model) map[string]string {
	out := map[string]string{}
	if m.Store != nil {
		for id, p := range m.Store.SidebarQuestionProjections() {
			kind := interaction.ProjectedPendingStatus(p.Questions, p.Inbox)
			if m.Interactions != nil {
				kind = m.Interactions.ProjectedSidebarStatus(id, p.Questions, p.Inbox)
			}
			if kind != "" {
				out[id] = kind
			}
		}
	}
	if m.Interactions != nil {
		for id, kind := range m.Interactions.PendingStatuses() {
			if interaction.PendingStatusRank(kind) > interaction.PendingStatusRank(out[id]) {
				out[id] = kind
			}
		}
	}
	return out
}
