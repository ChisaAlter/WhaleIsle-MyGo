package interaction

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

// Dropping explicit skip validity or encoding an old selection/custom value
// makes this fixture fail. An empty untouched draft must remain incomplete.
func TestQuestionExplicitSkipContract(t *testing.T) {
	t.Run("draft and waterfall", func(t *testing.T) {
		s := New()
		req := userQuestionsRequest{Questions: []userQuestionItem{
			{ID: "text", Question: "Explain"},
			{ID: "single", Options: []userQuestionOption{{Label: "A"}, {Label: "B"}}},
			{ID: "multi", MultiSelect: true, Options: []userQuestionOption{{Label: "A"}}},
		}}
		s.pending = []Pending{{AgentID: "A", EventID: "live", Event: EventUserQuestions}}
		s.waiters["live"] = make(chan dsh.EventResultOutcome, 1)
		s.questionDrafts["live"] = newDraftSession(req)
		if s.QuestionsCanSubmit("live") {
			t.Fatal("untouched draft accepted")
		}
		s.SetQuestionCustom("live", "text", "old text")
		s.SetQuestionOptionChecked("live", "multi", "A", true)
		for _, q := range req.Questions {
			s.SetQuestionSkipped("live", q.ID, true)
		}
		if !s.QuestionsCanSubmit("live") {
			t.Fatal("explicit skips rejected")
		}
		for _, row := range s.SnapshotForAgent("A").QuestionCards["live"].Rows {
			if !row.Skipped {
				t.Fatal("skip missing from card")
			}
		}
		s.SetQuestionSkipped("live", "text", false)
		if s.QuestionsCanSubmit("live") {
			t.Fatal("undo skip silently answered question")
		}
		s.SetQuestionSkipped("live", "text", true)
		s.SetQuestionCustom("live", "text", "new")
		s.SetQuestionSingleSelect("live", "single", "B")
		s.SetQuestionOptionChecked("live", "multi", "A", true)
		for _, row := range s.SnapshotForAgent("A").QuestionCards["live"].Rows {
			if row.Skipped {
				t.Fatal("answer edit retained skip")
			}
		}
		for _, q := range req.Questions {
			s.SetQuestionSkipped("live", q.ID, true)
		}
		s.HideQuestion("live")
		s.ReopenQuestion("live")
		if !s.QuestionsCanSubmit("live") {
			t.Fatal("hide lost skips")
		}
		waiter := s.waiters["live"]
		s.ResolveQuestions("live")
		out := <-waiter
		expected := `{"answers":[{"id":"text","selected":[]},{"id":"single","selected":[]},{"id":"multi","selected":[]}]}`
		if out.Kind != "result" || string(out.Value) != expected {
			t.Fatalf("skip wire: %+v", out)
		}
		var answer userQuestionsAnswer
		if err := json.Unmarshal(out.Value, &answer); err != nil {
			t.Fatal(err)
		}
		history, ok := pairQuestionHistory("A", "call", req.Questions, answer.Answers)
		if !ok || len(history.Rows) != 3 {
			t.Fatal("skip history lost")
		}
		for _, row := range history.Rows {
			if !row.Skipped || len(row.Selected) != 0 || row.Custom != "" {
				t.Fatal("history restored discarded answer")
			}
		}
	})
	t.Run("continued RPC and discarded queue", func(t *testing.T) {
		received := make(chan json.RawMessage, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				RPCID   string `json:"rpcId"`
				Payload struct {
					Args struct {
						AgentID, CallID string
						Answer          json.RawMessage
					} `json:"args"`
				} `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Payload.Args.AgentID != "A" || body.Payload.Args.CallID != "call-1" {
				t.Error("wrong question route")
			}
			received <- body.Payload.Args.Answer
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": true}})
		}))
		defer srv.Close()
		s := New()
		s.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
		s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
		id := questionCardID("A", "call-1")
		s.SetQuestionSkipped(id, "q", true)
		s.ResolveQuestions(id)
		awaitQuestionFixture(t, func() bool { return s.SnapshotForAgent("A").QuestionStatus == "submitted" })
		if got := string(<-received); got != `{"answers":[{"id":"q","selected":[]}]}` {
			t.Fatalf("late skip wire: %s", got)
		}
		inbox := json.RawMessage(`{"next-step":[],"next-turn":[{"source":{"kind":"user-question-reply","callId":"call-1"},"content":[{"type":"text","text":"{\"kind\":\"answer_to_pending_question\",\"callId\":\"call-1\",\"answers\":[{\"id\":\"q\",\"selected\":[]}]}"}]}]}`)
		s.SyncProjectedQuestions("A", continuedFixture, inbox)
		s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, continuedFixture, inbox, 0, 0)
		view := s.SnapshotForAgent("A")
		if len(view.Pending) != 0 || len(view.QuestionHistory) != 1 || !view.QuestionHistory[0].Queued {
			t.Fatal("queued skip did not become read-only history")
		}
		s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
		view = s.SnapshotForAgent("A")
		if len(view.Pending) != 1 || !view.QuestionCards[id].Rows[0].Skipped {
			t.Fatal("discarded queue lost skip draft")
		}
		s.SetQuestionCustom(id, "q", "supplement")
		if view = s.SnapshotForAgent("A"); view.QuestionCards[id].Rows[0].Skipped || !s.QuestionsCanSubmit(id) {
			t.Fatal("supplemental edit did not replace skip")
		}
	})
}
