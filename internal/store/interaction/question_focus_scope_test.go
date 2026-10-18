package interaction

import (
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestChildQuestionFocusFrameIsolation(t *testing.T) {
	s := New()
	req := userQuestionsRequest{Questions: []userQuestionItem{{ID: "q", Options: []userQuestionOption{{Label: "A"}}}}}
	for _, id := range []string{"parent", "child"} {
		s.pending = append(s.pending, Pending{EventID: id, AgentID: id, Event: EventUserQuestions})
		s.questionDrafts[id] = newDraftSession(req)
		s.questionDrafts[id].wait = questionWaitState{active: true, deadline: time.Now().Add(time.Minute)}
	}
	s.BeginQuestionFocusFrame()
	s.SyncQuestionFocusForAgent("child", "child")
	s.SyncQuestionFocusForAgent("parent", "")
	s.SyncQuestionFocusForAgent("child", "")
	s.EndQuestionFocusFrame()
	if s.Snapshot().QuestionCards["child"].Wait.State != "focused" || s.Snapshot().QuestionCards["parent"].Wait.State != "counting" {
		t.Fatal("panes clobbered focus")
	}
	s.BeginQuestionFocusFrame()
	s.SyncQuestionFocusForAgent("parent", "")
	s.EndQuestionFocusFrame()
	if s.Snapshot().QuestionCards["child"].Wait.State != "counting" {
		t.Fatal("closed child retained pristine pause")
	}
	a, b := &dsh.Client{}, &dsh.Client{}
	s.AttachClient(a, nil)
	if !s.MatchesClient(a) || s.MatchesClient(b) || s.MatchesClient(nil) {
		t.Fatal("Host scope failed")
	}
}
