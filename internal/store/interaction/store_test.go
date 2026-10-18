package interaction

import (
	"encoding/json"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestDefaultQuestionAnswer(t *testing.T) {
	req := userQuestionsRequest{
		Questions: []userQuestionItem{{
			ID:       "mode",
			Question: "Pick",
			Options:  []userQuestionOption{{Label: "Fast"}},
		}},
	}
	ans := defaultQuestionAnswer(req)
	if len(ans.Answers) != 1 || ans.Answers[0].ID != "mode" || len(ans.Answers[0].Selected) != 1 || ans.Answers[0].Selected[0] != "Fast" {
		t.Fatalf("answer %#v", ans)
	}
	raw, err := json.Marshal(ans)
	if err != nil {
		t.Fatal(err)
	}
	var round userQuestionsAnswer
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalOutcomeWire(t *testing.T) {
	out := approvalOutcome(outcomeAllowedOnce)
	if out.Kind != "result" {
		t.Fatalf("kind %q", out.Kind)
	}
	var s string
	if err := json.Unmarshal(out.Value, &s); err != nil || s != outcomeAllowedOnce {
		t.Fatalf("value %q err %v", out.Value, err)
	}
	_ = dsh.EventResultOutcome{}
}

func TestApprovalArgsPreview(t *testing.T) {
	prev := approvalArgsPreview(approvalRequest{
		Input: json.RawMessage(`{"cmd":"ls","path":"/tmp"}`),
	})
	if prev == "" || !strings.Contains(prev, "cmd") || !strings.Contains(prev, "ls") {
		t.Fatalf("preview %q", prev)
	}
	if approvalArgsPreview(approvalRequest{Command: "echo hi"}) != "echo hi" {
		t.Fatal("command fallback")
	}
}

func TestResolveAllowAlways(t *testing.T) {
	s := New()
	s.SetAutoAllow(false)
	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	s.waiters["ev-a"] = ch
	s.pending = []Pending{{EventID: "ev-a", Event: EventApproval, Title: "bash"}}
	s.mu.Unlock()
	go s.ResolveAllowAlways("ev-a")
	out := <-ch
	var decision string
	if err := json.Unmarshal(out.Value, &decision); err != nil || decision != outcomeAllowedOnce {
		t.Fatalf("decision %q err %v", out.Value, err)
	}
	if s.Snapshot().AutoAllow {
		t.Fatal("one request must not enable automatic approval")
	}
}

func TestQuestionDraftMultiSelect(t *testing.T) {
	s := New()
	req := userQuestionsRequest{
		Questions: []userQuestionItem{
			{
				ID:          "tags",
				Question:    "Pick tags",
				MultiSelect: true,
				Options: []userQuestionOption{
					{Label: "A"},
					{Label: "B"},
				},
			},
			{
				ID:       "mode",
				Question: "Mode",
				Options:  []userQuestionOption{{Label: "Fast"}, {Label: "Slow"}},
			},
		},
	}
	s.mu.Lock()
	s.questionDrafts["ev1"] = newDraftSession(req)
	s.pending = []Pending{{EventID: "ev1", Event: EventUserQuestions, Title: "Q"}}
	s.waiters["ev1"] = make(chan dsh.EventResultOutcome, 1)
	s.mu.Unlock()

	if s.QuestionsCanSubmit("ev1") {
		t.Fatal("multi-select starts empty; should not submit yet")
	}
	s.SetQuestionOptionChecked("ev1", "tags", "A", true)
	s.SetQuestionOptionChecked("ev1", "tags", "B", true)
	s.SetQuestionSingleSelect("ev1", "mode", "Slow")
	if !s.QuestionsCanSubmit("ev1") {
		t.Fatal("expected valid draft")
	}

	ch := s.waiters["ev1"]
	go s.ResolveQuestions("ev1")
	out := <-ch
	if out.Kind != "result" {
		t.Fatalf("kind %q", out.Kind)
	}
	var ans userQuestionsAnswer
	if err := json.Unmarshal(out.Value, &ans); err != nil {
		t.Fatal(err)
	}
	if len(ans.Answers) != 2 {
		t.Fatalf("answers %#v", ans)
	}
	byID := map[string]userQuestionsAnswerItem{}
	for _, a := range ans.Answers {
		byID[a.ID] = a
	}
	if len(byID["tags"].Selected) != 2 || byID["mode"].Selected[0] != "Slow" {
		t.Fatalf("payload %#v", ans)
	}
}

func TestQuestionDraftFreeText(t *testing.T) {
	s := New()
	req := userQuestionsRequest{
		Questions: []userQuestionItem{{
			ID:       "note",
			Question: "Details",
		}},
	}
	s.mu.Lock()
	s.questionDrafts["ev2"] = newDraftSession(req)
	s.pending = []Pending{{EventID: "ev2", Event: EventUserQuestions}}
	s.waiters["ev2"] = make(chan dsh.EventResultOutcome, 1)
	s.mu.Unlock()

	s.SetQuestionCustom("ev2", "note", "hello")
	if !s.QuestionsCanSubmit("ev2") {
		t.Fatal("custom text should satisfy question")
	}
	ch := s.waiters["ev2"]
	go s.ResolveQuestions("ev2")
	out := <-ch
	var ans userQuestionsAnswer
	if err := json.Unmarshal(out.Value, &ans); err != nil {
		t.Fatal(err)
	}
	if ans.Answers[0].Custom == nil || *ans.Answers[0].Custom != "hello" {
		t.Fatalf("custom %#v", ans)
	}
}

func TestSnapshotQuestionCards(t *testing.T) {
	s := New()
	req := userQuestionsRequest{
		Questions: []userQuestionItem{{
			ID: "a", Question: "One", MultiSelect: true,
			Options: []userQuestionOption{{Label: "x"}},
		}},
	}
	s.mu.Lock()
	s.questionDrafts["id"] = newDraftSession(req)
	s.mu.Unlock()
	snap := s.Snapshot()
	card, ok := snap.QuestionCards["id"]
	if !ok || len(card.Rows) != 1 || !card.Rows[0].MultiSelect {
		t.Fatalf("snapshot %#v", snap.QuestionCards)
	}
}
