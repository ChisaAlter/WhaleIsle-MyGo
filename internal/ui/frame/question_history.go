package frame

import (
	"encoding/json"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
)

func syncQuestionHistory(store *interaction.Store, snap session.ViewState) {
	raws := make([]json.RawMessage, 0, len(snap.Messages))
	for _, m := range snap.Messages {
		if len(m.Raw) > 0 && !m.Streaming {
			raws = append(raws, m.Raw)
		}
	}
	store.SyncApprovalCalls(snap.SelectedID, raws)
	q := snap.QuestionProjection
	store.SyncQuestionHistory(snap.SelectedID, raws, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq)
}
