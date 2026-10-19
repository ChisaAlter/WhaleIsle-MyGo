package interaction

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh"
)

func questionWizardFixture() *Store {
	s := New()
	s.pending = []Pending{{AgentID: "A", EventID: "e", CallID: "call", Event: EventUserQuestions}}
	s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
	s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{CallID: "call", Questions: []userQuestionItem{{ID: "first", Question: "First", Options: []userQuestionOption{{Label: "A"}, {Label: "B"}}}, {ID: "last", Question: "Last"}}})
	return s
}

func TestQuestionWizardPagerScopeAndBusyRetention(t *testing.T) {
	s := questionWizardFixture()
	if !s.NavigateQuestion("A", "e", 1) {
		t.Fatal("pager required an answer")
	}
	for _, target := range []struct {
		agent, event string
		index        int
	}{{"B", "e", 0}, {"A", "stale", 0}, {"A", "e", -1}, {"A", "e", 2}} {
		if s.NavigateQuestion(target.agent, target.event, target.index) {
			t.Fatal("invalid navigation accepted")
		}
	}
	s.SetQuestionCustom("e", "last", "kept\nanswer")
	s.HideQuestion("e")
	s.ReopenQuestion("e")
	if view := s.Snapshot().QuestionCards["e"]; view.Index != 1 || view.Rows[1].Custom != "kept\nanswer" {
		t.Fatal("hide/reopen reset progress")
	}
	s.ResolveDeny("e")
	if s.NavigateQuestion("A", "e", 0) {
		t.Fatal("busy request navigated")
	}
	if view := s.Snapshot().QuestionCards["e"]; view.Index != 1 || view.Rows[1].Custom != "kept\nanswer" {
		t.Fatal("busy request changed")
	}
}

func TestQuestionWizardChoiceAdvanceDoesNotSubmit(t *testing.T) {
	s := questionWizardFixture()
	s.AdvanceQuestionChoice("A", "e", "first")
	if s.Snapshot().QuestionCards["e"].Index != 0 {
		t.Fatal("untouched choice advanced")
	}
	s.SetQuestionOptionChecked("e", "first", "A", true)
	s.AdvanceQuestionChoice("B", "e", "first")
	if s.Snapshot().QuestionCards["e"].Index != 0 {
		t.Fatal("foreign session advanced")
	}
	s.AdvanceQuestionChoice("A", "e", "first")
	if s.Snapshot().QuestionCards["e"].Index != 1 || len(s.waiters["e"]) != 0 {
		t.Fatal("single choice did not advance without sending")
	}
	s.NavigateQuestion("A", "e", 0)
	s.questionDrafts["e"].req.Questions[0].MultiSelect = true
	s.AdvanceQuestionChoice("A", "e", "first")
	if s.Snapshot().QuestionCards["e"].Index != 0 {
		t.Fatal("multi-select advanced")
	}
}

func TestQuestionWizardFinalSkipFindsMissingThenSubmits(t *testing.T) {
	s := questionWizardFixture()
	s.NavigateQuestion("A", "e", 1)
	if focus, status := s.SkipQuestionAndContinue("A", "e", "last"); focus != "first" || status != "incomplete" {
		t.Fatalf("missing earlier answer not targeted: %s %s", focus, status)
	}
	if len(s.waiters["e"]) != 0 || !s.Snapshot().QuestionCards["e"].Rows[1].Skipped {
		t.Fatal("final skip submitted incomplete batch or lost skip")
	}
	s.SetQuestionCustom("e", "first", "custom")
	s.ContinueQuestion("A", "e", "first")
	s.SkipQuestionAndContinue("A", "e", "last")
	if len(s.waiters["e"]) != 1 {
		t.Fatal("final skip did not submit completed batch")
	}
	var answer userQuestionsAnswer
	if json.Unmarshal((<-s.waiters["e"]).Value, &answer) != nil || len(answer.Answers) != 2 || answer.Answers[1].Custom != nil || len(answer.Answers[1].Selected) != 0 {
		t.Fatalf("wrong skip payload: %+v", answer)
	}
}

func TestQuestionWizardIndexSurvivesContinuation(t *testing.T) {
	s := questionWizardFixture()
	projection := json.RawMessage(`{"active":[{"callId":"call","state":"continued","questions":[{"id":"first","question":"First","options":[{"label":"A"},{"label":"B"}]},{"id":"last","question":"Last"}]}],"settled":[]}`)
	s.SetQuestionCustom("e", "first", "draft")
	s.NavigateQuestion("A", "e", 1)
	s.resolve("e", dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Code: "ASK_TIMED_OUT"}})
	s.OnResult(dsh.EventResult{EventID: "e"}, nil)
	if pending := s.SnapshotForAgent("A").Pending; len(pending) != 1 || !pending[0].Sending || pending[0].Continued {
		t.Fatal("HTTP receipt settled the named live question")
	}
	s.SyncProjectedQuestions("A", projection, emptyInboxFixture)
	view := s.SnapshotForAgent("A")
	if len(view.Pending) != 1 || !view.Pending[0].Continued {
		t.Fatal("continued request missing")
	}
	card := view.QuestionCards[view.Pending[0].EventID]
	if card.Index != 1 || card.Rows[0].Custom != "draft" {
		t.Fatalf("continued progress lost: %+v", card)
	}
}
