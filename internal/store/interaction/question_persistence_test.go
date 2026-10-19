package interaction

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
)

func TestQuestionDraftPersistenceMatchesProfileBatchAndRetainsReconnect(t *testing.T) {
	const owner = "local-profile:C:/fixture/profiles/web"
	s := New()
	s.SetQuestionDraftPersistence(owner, nil)
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "B")
	s.SetQuestionCustom(id, "q", "saved text")
	raw := s.ExportQuestionDrafts(owner)
	if len(raw) == 0 || s.ExportQuestionDrafts("another profile") != nil {
		t.Fatal("unbound draft export")
	}
	restored := New()
	restored.SetQuestionDraftPersistence(owner, nil)
	restored.RestoreQuestionDrafts(owner, raw)
	if len(restored.Snapshot().Pending) != 0 {
		t.Fatal("disk created a pending question")
	}
	restored.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id = restored.Snapshot().Pending[0].EventID
	if got := *restored.QuestionCustomPtr(id, "q"); got != "saved text" {
		t.Fatalf("lost draft: %q", got)
	}
	restored.SetQuestionCustom(id, "q", "newer edit")
	restored.RestoreQuestionDrafts(owner, raw)
	if got := *restored.QuestionCustomPtr(id, "q"); got != "newer edit" {
		t.Fatal("repeat restore overwrote live edit")
	}
	restored.Detach()
	restored.AttachClient(&dsh.Client{}, nil)
	restored.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if got := *restored.QuestionCustomPtr(restored.Snapshot().Pending[0].EventID, "q"); got != "newer edit" {
		t.Fatal("same-profile reconnect lost retained draft")
	}
	foreign := New()
	foreign.SetQuestionDraftPersistence("other profile", nil)
	foreign.RestoreQuestionDrafts("other profile", raw)
	foreign.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	if got := *foreign.QuestionCustomPtr(foreign.Snapshot().Pending[0].EventID, "q"); got != "" {
		t.Fatal("cross-profile draft")
	}
	changed := New()
	changed.RestoreQuestionDrafts(owner, raw)
	changed.SyncProjectedQuestions("A", json.RawMessage(`{"active":[{"callId":"call-1","state":"continued","questions":[{"id":"q","question":"Different batch"}]}],"settled":[]}`), emptyInboxFixture)
	if got := *changed.QuestionCustomPtr(changed.Snapshot().Pending[0].EventID, "q"); got != "" {
		t.Fatal("stale batch restored")
	}
}

func TestQuestionDraftWritesSerializeLatestAndExposeFailure(t *testing.T) {
	s := New()
	const owner = "profile fixture"
	var mu sync.Mutex
	var saved json.RawMessage
	failing := true
	s.SetQuestionDraftPersistence(owner, func(raw json.RawMessage) error {
		mu.Lock()
		defer mu.Unlock()
		if failing {
			return errors.New("fixture disk failure")
		}
		saved = append(json.RawMessage(nil), raw...)
		return nil
	})
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.Snapshot().Pending[0].EventID
	s.SetQuestionCustom(id, "q", "first")
	if s.FlushQuestionDraftPersistence() == nil || s.Snapshot().QuestionDraftError == "" {
		t.Fatal("save failure hidden")
	}
	mu.Lock()
	failing = false
	mu.Unlock()
	s.SetQuestionCustom(id, "q", "last")
	if err := s.FlushQuestionDraftPersistence(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	raw := append(json.RawMessage(nil), saved...)
	mu.Unlock()
	var envelope questionDraftEnvelope
	if json.Unmarshal(raw, &envelope) != nil || envelope.Drafts[questionCardID("A", "call-1")].Answers[0].Custom != "last" {
		t.Fatal("flush saved an older snapshot")
	}
	if s.Snapshot().QuestionDraftError != "" {
		t.Fatal("successful edit did not clear persistence error")
	}
}

func TestQuestionPagerAndExplicitWaitPersistWithoutCountdown(t *testing.T) {
	s := New()
	s.SetQuestionDraftPersistence("profile", nil)
	req := userQuestionsRequest{CallID: "batch", Questions: []userQuestionItem{{ID: "one", Question: "First"}, {ID: "two", Question: "Second"}}}
	draft := newDraftSession(req)
	draft.wait.active = true
	s.questionDrafts["live"] = draft
	s.pending = append(s.pending, Pending{EventID: "live", AgentID: "A", CallID: "batch", Event: EventUserQuestions})
	s.SetQuestionCustom("live", "one", "answer")
	if !s.NavigateQuestion("A", "live", 1) {
		t.Fatal("pager did not move")
	}
	s.TakeQuestionTime("A", "live")
	var envelope questionDraftEnvelope
	if json.Unmarshal(s.ExportQuestionDrafts("profile"), &envelope) != nil {
		t.Fatal("invalid persisted draft")
	}
	saved := envelope.Drafts[questionCardID("A", "batch")]
	if saved.Index != 1 || saved.Wait != "waiting" || saved.Answers[0].Custom != "answer" {
		t.Fatalf("pager/wait not durable: %+v", saved)
	}
}

func TestQuestionFlushCannotBeOverwrittenByPreviouslyQueuedSnapshot(t *testing.T) {
	s := New()
	started, release := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	var saved json.RawMessage
	first := true
	s.SetQuestionDraftPersistence("profile", func(raw json.RawMessage) error {
		if first {
			first = false
			close(started)
			<-release
		}
		mu.Lock()
		saved = append(json.RawMessage(nil), raw...)
		mu.Unlock()
		return nil
	})
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.Snapshot().Pending[0].EventID
	s.SetQuestionCustom(id, "q", "old")
	<-started
	s.SetQuestionCustom(id, "q", "latest")
	done := make(chan error, 1)
	go func() { done <- s.FlushQuestionDraftPersistence() }()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	raw := append(json.RawMessage(nil), saved...)
	mu.Unlock()
	var envelope questionDraftEnvelope
	if json.Unmarshal(raw, &envelope) != nil || envelope.Drafts[questionCardID("A", "call-1")].Answers[0].Custom != "latest" {
		t.Fatal("older pending callback overwrote flush")
	}
}
