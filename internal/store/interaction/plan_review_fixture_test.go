package interaction

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

// One bounded synthetic fixture. No real Host, model, GUI, or tool execution.
func TestPlanReviewContractFixture(t *testing.T) {
	question := json.RawMessage(`{"id":"review-1","question":"Review this plan","detail":"# Migration\n\nKeep the data safe.","intent":{"kind":"plan-review","approve":"Ship this plan","callId":"plan-1"},"options":[{"label":"Discuss changes"},{"label":"Ship this plan","description":"Execute the reviewed plan"}]}`)
	var item userQuestionItem
	if err := json.Unmarshal(question, &item); err != nil {
		t.Fatal(err)
	}
	request := userQuestionsRequest{CallID: "review-call", Questions: []userQuestionItem{item}}
	review := planReviewOf(request.Questions)
	if review == nil || review.Approve.Label != "Ship this plan" || review.CallID != "plan-1" {
		t.Fatalf("intent metadata lost: %+v", review)
	}
	if planReviewOf(append(request.Questions, item)) != nil {
		t.Fatal("multi-question request was taken over")
	}
	malformed := item
	malformed.MultiSelect = true
	if planReviewOf([]userQuestionItem{malformed}) != nil {
		t.Fatal("multi-select request was taken over")
	}
	malformed = item
	malformed.Options = []userQuestionOption{{Label: "Other"}}
	if planReviewOf([]userQuestionItem{malformed}) != nil {
		t.Fatal("missing approve label was invented")
	}

	native := json.RawMessage(`{"type":"tool/call","seq":2,"data":{"name":"exit_plan_mode","callId":"plan-1","arguments":"{\"plan\":\"# Migration\\n\\nKeep the data safe.\"}"}}`)
	ptcStart := json.RawMessage(`{"type":"tool/ptc-dispatch-start","seq":3,"data":{"name":"exit_plan_mode","subCallId":"plan-2","arguments":{"plan":"# PTC plan\n\nOriginal text"}}}`)
	ptcSettle := json.RawMessage(`{"type":"tool/ptc-dispatch","seq":4,"data":{"name":"exit_plan_mode","subCallId":"plan-2","arguments":{"plan":"# PTC plan\n\nChanged settlement text"}}}`)
	messages := []session.Message{{ID: "native", Kind: session.MessageKindTool, Raw: native, Seq: 2, TurnSeq: 1}, {ID: "start", Kind: session.MessageKindTool, Raw: ptcStart, Seq: 3, TurnSeq: 1}, {ID: "settle", Kind: session.MessageKindTool, Raw: ptcSettle, Seq: 4, TurnSeq: 1}}
	cards := session.WithSubmittedPlanCards(messages)
	var plans []session.SubmittedPlan
	for _, message := range cards {
		if message.Kind == session.MessageKindSubmittedPlan {
			p, ok := session.SubmittedPlanFromEvent(message.Raw)
			if !ok {
				t.Fatal("unreadable document card")
			}
			plans = append(plans, p)
		}
	}
	if len(plans) != 2 || plans[0].Markdown != review.Plan || plans[1].Markdown != "# PTC plan\n\nOriginal text" {
		t.Fatalf("plan document dedup/immutability: %+v", plans)
	}
	if len(messages) != 3 || len(session.WithSubmittedPlanCards(cards)) != len(cards) {
		t.Fatal("plan rendering mutated or duplicated transcript")
	}
	if _, ok := session.SubmittedPlanFromEvent(json.RawMessage(`{"type":"tool/call","data":{"name":"exit_plan_mode","callId":"bad","arguments":"{\"plan\":\"no heading\"}"}}`)); ok {
		t.Fatal("malformed plan became document")
	}

	live := New()
	live.questionDrafts["live"] = newDraftSession(request)
	live.pending = []Pending{{EventID: "live", Event: EventUserQuestions, AgentID: "A", CallID: "review-call", questionProjected: true}}
	live.waiters["live"] = make(chan dsh.EventResultOutcome, 2)
	if live.QuestionsCanSubmit("live") {
		t.Fatal("plan approval was selected by default")
	}
	live.ResolveQuestions("live")
	if len(live.waiters["live"]) != 0 {
		t.Fatal("generic continue approved the plan")
	}
	live.DismissPlanReview("live")
	if !live.SnapshotForAgent("A").QuestionCards["live"].PlanReviewHidden || len(live.waiters["live"]) != 0 {
		t.Fatal("discussion settled a named request")
	}
	live.ReopenPlanReview("live")
	if live.SnapshotForAgent("A").QuestionCards["live"].PlanReviewHidden {
		t.Fatal("review could not reopen")
	}
	live.ApprovePlanReview("live")
	live.ApprovePlanReview("live")
	first := <-live.waiters["live"]
	if len(live.waiters["live"]) != 0 || string(first.Value) != `{"answers":[{"id":"review-1","selected":["Ship this plan"]}]}` {
		t.Fatalf("wrong or duplicate approval: %+v", first)
	}
	if !live.OnResult(dsh.EventResult{EventID: "live"}, errors.New("fixture transport failure")) {
		t.Fatal("failed approval was dropped")
	}
	live.Retry("live")
	retry := <-live.waiters["live"]
	if string(retry.Value) != string(first.Value) {
		t.Fatal("retry changed decision")
	}
	live.OnResult(dsh.EventResult{EventID: "live"}, nil)
	if pending := live.SnapshotForAgent("A").Pending; len(pending) != 1 || !pending[0].Sending {
		t.Fatal("HTTP receipt replaced Host plan acceptance")
	}
	live.SyncProjectedQuestions("A", json.RawMessage(`{"active":[],"settled":[{"callId":"review-call"}]}`), emptyInboxFixture)
	if len(live.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("Host-settled live request remained pending")
	}
	unnamed := New()
	unnamed.questionDrafts["unnamed"] = newDraftSession(request)
	unnamed.pending = []Pending{{EventID: "unnamed", Event: EventUserQuestions, AgentID: "A"}}
	unnamed.waiters["unnamed"] = make(chan dsh.EventResultOutcome, 1)
	unnamed.DismissPlanReview("unnamed")
	dismissed := <-unnamed.waiters["unnamed"]
	if dismissed.Kind != "rejected" || dismissed.Error == nil || dismissed.Error.Code != "ASK_CANCELLED" {
		t.Fatalf("unnamed discussion contract: %+v", dismissed)
	}

	var mu sync.Mutex
	var calls []map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		calls = append(calls, body)
		count := len(calls)
		mu.Unlock()
		if count == 1 {
			http.Error(w, "fixture temporary failure", 503)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": true}})
	}))
	defer srv.Close()
	recovered := New()
	recovered.AttachClient(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	face, _ := json.Marshal(map[string]any{"active": []any{map[string]any{"callId": "review-call", "state": "continued", "questions": []json.RawMessage{question}}}, "settled": []any{}})
	inbox := json.RawMessage(`{"next-step":[],"next-turn":[]}`)
	recovered.SyncProjectedQuestions("A", face, inbox, 10, 10)
	snap := recovered.SnapshotForAgent("A")
	if len(snap.Pending) != 1 || len(recovered.SnapshotForAgent("B").Pending) != 0 {
		t.Fatal("recovered review lost session isolation")
	}
	id := snap.Pending[0].EventID
	recovered.DismissPlanReview(id)
	recovered.SyncProjectedQuestions("A", face, inbox, 10, 10)
	if !recovered.SnapshotForAgent("A").QuestionCards[id].PlanReviewHidden {
		t.Fatal("projection reopened dismissed review")
	}
	recovered.ReopenPlanReview(id)
	recovered.ApprovePlanReview(id)
	recovered.ApprovePlanReview(id)
	wait := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !check() {
			if time.Now().After(deadline) {
				t.Fatal("fixture timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	wait(func() bool {
		p := recovered.SnapshotForAgent("A").Pending
		return len(p) == 1 && !p[0].Sending && p[0].Error != ""
	})
	recovered.Retry(id)
	wait(func() bool { return recovered.SnapshotForAgent("A").QuestionStatus == "submitted" })
	recovered.SyncProjectedQuestions("A", face, inbox, 10, 10)
	recovered.ApprovePlanReview(id)
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 2 {
		t.Fatalf("late approval was duplicated: %d", len(calls))
	}
	for _, body := range calls {
		var method string
		_ = json.Unmarshal(body["method"], &method)
		if method != "userQuestions/answer" {
			t.Fatalf("invented endpoint: %s", body["method"])
		}
		var payload struct {
			Args json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal(body["payload"], &payload); err != nil {
			t.Fatal(err)
		}
		var params struct {
			AgentID string              `json:"agentId"`
			CallID  string              `json:"callId"`
			Answer  userQuestionsAnswer `json:"answer"`
		}
		if err := json.Unmarshal(payload.Args, &params); err != nil {
			t.Fatal(err)
		}
		if params.AgentID != "A" || params.CallID != "review-call" || len(params.Answer.Answers) != 1 || params.Answer.Answers[0].Selected[0] != "Ship this plan" {
			t.Fatalf("wrong late contract: %+v", params)
		}
	}
	if len(recovered.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("inbox dedup fence regressed")
	}
	recovered.Detach()
	recovered.AttachClient(&dsh.Client{}, nil)
	recovered.SyncProjectedQuestions("A", face, inbox, 10, 10)
	if p := recovered.SnapshotForAgent("A"); len(p.Pending) != 1 || p.QuestionCards[p.Pending[0].EventID].PlanReviewHidden {
		t.Fatal("review state leaked across Hosts")
	}
}
