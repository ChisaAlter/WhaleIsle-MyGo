package agents

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
	interui "whaleisle/internal/ui/interaction"
)

func childInteractionsView(c *ui.Context, m Model, v session.AgentNavigationView, resident ...ui.Rect) bool {
	if m.Store == nil || m.Interactions == nil {
		return false
	}
	client, q, records, ok := m.Store.AgentInteractionTarget(v.Token, v.Address)
	var cardStore interui.CardStore = m.Interactions
	if m.Store.HasRetainedAgentRead() {
		scope, scopedQuestions, scopedRecords, valid := m.Store.RetainedAgentInteractionViewScope(v.Token, v.Address)
		if !valid {
			return false
		}
		client, q, records, ok = scope.Client(), scopedQuestions, scopedRecords, true
		cardStore = interaction.NewScopedStore(m.Interactions, scope)
	}
	if !ok || !m.Interactions.MatchesClient(client) {
		return false
	}
	id := v.Address.ChildSessionID
	m.Interactions.SyncProjectedQuestionSnapshot(id, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq, q.QueueSeen, m.Store.QuestionQueueObservation)
	m.Interactions.SyncApprovalCalls(id, records)
	m.Interactions.SyncQuestionHistory(id, records, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq)
	snap := m.Interactions.SnapshotForAgent(id)
	// Global auto-allow settings belong to the main settings/chat surface.
	snap.AutoAllow = false
	snap.Banner = ""
	snap, takeover := interui.SelectComposerInteraction(snap)
	interui.View(c.Key("child-interactions:"+id), interui.Model{Snapshot: snap, Store: cardStore, ThemeID: m.ThemeID, SystemDark: m.SystemDark, ScopeAgentID: id, DisablePageShortcuts: true, OnOpenPlanResource: m.OnOpenPlanResource, RenderSlot: m.InteractionRenderSlot}, resident...)
	return takeover
}
