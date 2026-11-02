package frame

import (
	"bytes"
	"encoding/json"
	"sync"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
)

var questionHistoryRawsMemo struct {
	sync.Mutex
	key             session.ProjectionKey
	raws            []json.RawMessage
	questionRaws    []json.RawMessage
	hasQuestionCall bool
}

func syncQuestionHistory(store *interaction.Store, snap session.ViewState) {
	var raws []json.RawMessage
	var questionRaws []json.RawMessage
	hasQuestionCall := true
	if key := snap.HistoryProjectionKey(); key.Valid() {
		questionHistoryRawsMemo.Lock()
		if questionHistoryRawsMemo.key == key {
			raws = questionHistoryRawsMemo.raws
			questionRaws = questionHistoryRawsMemo.questionRaws
			hasQuestionCall = questionHistoryRawsMemo.hasQuestionCall
			questionHistoryRawsMemo.Unlock()
		} else {
			questionHistoryRawsMemo.Unlock()
			callName := []byte("ask_user_question")
			answersLower := []byte("answers")
			answersUpper := []byte("Answers")
			hasQuestionCall = false
			raws = make([]json.RawMessage, 0, len(snap.Messages))
			questionRaws = make([]json.RawMessage, 0, 8)
			for _, m := range snap.Messages {
				if len(m.Raw) > 0 && !m.Streaming {
					raws = append(raws, m.Raw)
					isCall := (m.EventType == "" || m.EventType == "tool/call" || m.EventType == "tool/ptc-dispatch-start" || m.EventType == "tool/ptc-dispatch") && bytes.Contains(m.Raw, callName)
					if isCall {
						hasQuestionCall = true
						questionRaws = append(questionRaws, m.Raw)
					} else if (m.EventType == "" || m.EventType == "tool/result" || m.EventType == "tool/ptc-dispatch") && (bytes.Contains(m.Raw, answersLower) || bytes.Contains(m.Raw, answersUpper)) {
						questionRaws = append(questionRaws, m.Raw)
					}
				}
			}
			questionHistoryRawsMemo.Lock()
			questionHistoryRawsMemo.key = key
			questionHistoryRawsMemo.raws = raws
			questionHistoryRawsMemo.questionRaws = questionRaws
			questionHistoryRawsMemo.hasQuestionCall = hasQuestionCall
			questionHistoryRawsMemo.Unlock()
		}
	} else {
		raws = make([]json.RawMessage, 0, len(snap.Messages))
		for _, m := range snap.Messages {
			if len(m.Raw) > 0 && !m.Streaming {
				raws = append(raws, m.Raw)
			}
		}
		questionRaws = raws
	}
	store.SyncApprovalCalls(snap.SelectedID, raws)
	q := snap.QuestionProjection
	questionEvents := questionRaws
	if !hasQuestionCall && !store.HasQuestionDrafts(snap.SelectedID) {
		questionEvents = nil
	}
	store.SyncQuestionHistory(snap.SelectedID, questionEvents, q.Questions, q.Inbox, q.QuestionsSeq, q.InboxSeq)
}
