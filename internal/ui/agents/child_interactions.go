package agents

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	interui "whaleisle/internal/ui/interaction"
)

func childInteractionsView(c *ui.Context, m Model, v session.AgentNavigationView) {
	if m.Store == nil || m.Interactions == nil {
		return
	}
	client, q, records, ok := m.Store.AgentInteractionTarget(v.Token, v.Address)
	if !ok || !m.Interactions.MatchesClient(client) {
		return
	}
	id := v.Address.ChildSessionID
	m.Interactions.SyncProjectedQuestions(id, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq)
	m.Interactions.SyncApprovalCalls(id, records)
	m.Interactions.SyncQuestionHistory(id, records, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq)
	snap := m.Interactions.SnapshotForAgent(id)
	// Global auto-allow settings belong to the main settings/chat surface.
	snap.AutoAllow = false
	snap.Banner = ""
	interui.View(c.Key("child-interactions:"+id), interui.Model{Snapshot: snap, Store: m.Interactions, ThemeID: m.ThemeID, SystemDark: m.SystemDark, ScopeAgentID: id, DisablePageShortcuts: true, OnOpenPlanResource: m.OnOpenPlanResource})
}
