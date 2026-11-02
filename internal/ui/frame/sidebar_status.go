package frame

import (
	"encoding/json"
	"whaleisle/internal/store/interaction"
)

func sidebarPendingStatuses(m Model) map[string]string {
	out := map[string]string{}
	if m.Store != nil {
		m.Store.ForEachSidebarQuestionProjection(func(id string, questions, inbox json.RawMessage) {
			kind := interaction.ProjectedPendingStatus(questions, inbox)
			if m.Interactions != nil {
				kind = m.Interactions.ProjectedSidebarStatus(id, questions, inbox)
			}
			if kind != "" {
				out[id] = kind
			}
		})
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
