package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
)

type lifecycleReceiver interface {
	OnEmit(*EventsOwner, dsh.EventsEmit)
	OnEventsReady(*EventsOwner)
}

// Pure reducer fixtures intentionally construct a current owner without
// resetting their in-flight baseline. Connected-stream tests use BeginEvents.
func testEventsOwner(s *Store, ctx context.Context) *EventsOwner {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &EventsOwner{store: s, client: s.client, ctx: ctx, generation: s.eventsGeneration}
}

func receiveLifecycle(t *testing.T, s *Store, event string, args ...string) {
	t.Helper()
	receiver, ok := any(s).(lifecycleReceiver)
	if !ok {
		t.Fatal("Store must fold api-session events and rebaseline each ready generation")
	}
	emit := dsh.EventsEmit{Type: dsh.EventEmit, Event: event}
	for _, arg := range args {
		emit.Args = append(emit.Args, json.RawMessage(arg))
	}
	receiver.OnEmit(testEventsOwner(s, context.Background()), emit)
}

func TestLifecycleStatusUpdatesOnlyTarget(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A", Running: true}, {SessionID: "B", Running: true}}
	s.selectedID = "B"
	s.messages = []Message{{ID: "B-message", Body: "keep"}}
	receiveLifecycle(t, s, "api-session/status", `"A"`, `false`)
	snap := s.Snapshot()
	if snap.Items[0].Running || !snap.Items[1].Running || s.RunningCount() != 1 {
		t.Fatalf("status target or running count wrong: %+v", snap.Items)
	}
	if snap.SelectedID != "B" || len(snap.Messages) != 1 || snap.Messages[0].Body != "keep" {
		t.Fatalf("status changed selected conversation: %+v", snap)
	}
}

func TestLifecycleRosterTargetsAndSubagentRemoval(t *testing.T) {
	s := New()
	s.selectedID = "B"
	s.messages = []Message{{ID: "B-message", Body: "keep"}}
	s.items = []Item{{SessionID: "B", Title: "Beta"}}
	receiveLifecycle(t, s, "api-session/added", `{"sessionId":"A","running":true,"blank":true,"agentAvailable":true,"updatedAt":1,"cwd":"/a","projections":{"kind":"sequenced","asOfSeq":1,"values":{"title":"Alpha"}}}`)
	receiveLifecycle(t, s, "api-session/added", `{"sessionId":"A","running":false,"blank":false,"agentAvailable":false,"updatedAt":2,"cwd":"/a"}`)
	receiveLifecycle(t, s, "api-session/activity", `"A"`, `20`)
	if got := s.Snapshot(); len(got.Items) != 2 || got.Items[1].Running || got.Items[1].UpdatedAt != 20 || got.Items[1].Blank {
		t.Fatalf("upsert/activity failed: %+v", got.Items)
	}
	receiveLifecycle(t, s, "api-session/removed", `"A"`)
	if got := s.Snapshot(); len(got.Items) != 1 || got.Items[0].SessionID != "B" || got.Messages[0].Body != "keep" {
		t.Fatalf("removed wrong target: %+v", got)
	}
	receiveLifecycle(t, s, "api-session/added", `{"sessionId":"child","parentSessionId":"B","origin":"subagent","running":true,"blank":false,"agentAvailable":true,"updatedAt":2}`)
	receiveLifecycle(t, s, "api-session/removed", `"child"`)
	if got := s.Snapshot(); len(got.Items) != 2 || got.Items[1].Running {
		t.Fatalf("durable subagent must remain stopped: %+v", got.Items)
	}
	receiveLifecycle(t, s, "api-session/deleted", `"child"`)
	if got := s.Snapshot(); len(got.Items) != 1 || got.SelectedID != "B" || got.Messages[0].Body != "keep" {
		t.Fatalf("deleted wrong target: %+v", got)
	}
}

func TestLifecycleMalformedAndUnknownAreIgnored(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A", Running: true, UpdatedAt: 2}}
	s.selectedID = "A"
	before := s.Snapshot()
	cases := []struct {
		event string
		args  []string
	}{
		{"api-session/status", nil}, {"api-session/status", []string{`"A"`}},
		{"api-session/status", []string{`"A"`, `null`}}, {"api-session/status", []string{`"A"`, `"false"`}},
		{"api-session/status", []string{`"missing"`, `false`}}, {"api-session/status", []string{`{`, `false`}},
		{"api-session/activity", []string{`"A"`, `null`}}, {"api-session/activity", []string{`"A"`, `"bad"`}},
		{"api-session/added", []string{`null`}}, {"api-session/added", []string{`{"sessionId":"A"}`}},
		{"api-session/added", []string{`{"sessionId":"A","running":null,"blank":false,"agentAvailable":true,"updatedAt":3}`}},
		{"api-session/deleted", []string{`null`}}, {"api-session/error", []string{`"missing"`, `"bad"`}},
		{"future/event", []string{`null`}},
	}
	for _, tc := range cases {
		receiveLifecycle(t, s, tc.event, tc.args...)
	}
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("malformed/unknown event mutated store: %+v", got)
	}
}

func TestLifecycleErrorIsSelectedSessionScoped(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A"}, {SessionID: "B"}}
	s.selectedID = "B"
	receiveLifecycle(t, s, "api-session/error", `"A"`, `"A failed"`)
	if s.Snapshot().LastAction != "" {
		t.Fatal("background session error replaced foreground status")
	}
	receiveLifecycle(t, s, "api-session/error", `"B"`, `"B failed"`)
	if s.Snapshot().LastAction != "B failed" {
		t.Fatal("selected session error not reported")
	}
}

func TestLifecycleReconnectRebaselinesAndReplaysInflightEvents(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path == "/api/session/list" {
			close(started)
			<-release
			rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "A", "running": true, "blank": false, "agentAvailable": true, "updatedAt": 1}}})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()
	s := New()
	s.items = []Item{{SessionID: "stale", Running: true}, {SessionID: "A", Running: true}}
	s.selectedID = "A"
	done := make(chan struct{})
	var once sync.Once
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if !s.Snapshot().ListLoading {
			once.Do(func() { close(done) })
		}
	})
	receiver, ok := any(s).(lifecycleReceiver)
	if !ok {
		close(release)
		t.Fatal("Store must handle reconnect-ready")
	}
	receiver.OnEventsReady(s.BeginEvents(context.Background(), s.client))
	wait(t, started)
	receiveLifecycle(t, s, "api-session/status", `"A"`, `false`)
	receiveLifecycle(t, s, "api-session/added", `{"sessionId":"B","running":true,"blank":false,"agentAvailable":true,"updatedAt":2}`)
	receiveLifecycle(t, s, "api-session/activity", `"A"`, `20`)
	close(release)
	wait(t, done)
	got := s.Snapshot()
	if len(got.Items) != 2 || got.Items[0].SessionID != "A" || got.Items[0].Running || got.Items[0].UpdatedAt != 20 || got.Items[1].SessionID != "B" {
		t.Fatalf("baseline did not replace stale state/replay events: %+v", got.Items)
	}
}

func TestSelectClearsPriorConversationBeforeNewFollow(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.messages = []Message{{Body: "A"}}
	s.planBoard = PlanBoard{PlanMode: true, Todos: []PlanTodo{{Content: "A"}}}
	s.historyThroughSeq = 90
	s.historyHasMore = true
	s.Select("B")
	got := s.Snapshot()
	if len(got.Messages) != 0 || got.PlanBoard.PlanMode || len(got.PlanBoard.Todos) != 0 || got.HistoryHasMore || s.historyThroughSeq != 0 {
		t.Fatalf("B inherited A conversation: %+v", got)
	}
}

func TestWorkspaceFramesCannotSupplySessionRosterOrRunning(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A", Running: true, Title: "Alpha"}}
	s.selectedID = "A"
	before := s.Snapshot()
	for _, raw := range []string{
		`{"type":"baseline","value":{"items":[{"id":"ws","path":"/a","name":"A","sessionIds":["A"]}],"archivedSessionIds":[],"pinnedSessionIds":[],"scratchCwd":"/scratch"}}`,
		`{"type":"upsert","workspace":{"id":"ws","path":"/a","name":"A","sessionIds":["A"]}}`,
		`{"type":"remove","workspaceId":"ws"}`, `{"type":"order","workspaceIds":[]}`,
		`{"type":"snapshot","sessions":[{"sessionId":"invented","running":false}]}`,
		`{"type":"upsert","sessionId":"A","running":false,"title":"wrong"}`,
		`{"type":"future","items":[{"sessionId":"invented","running":false}]}`,
	} {
		s.applyWorkspaceFollowFrame(json.RawMessage(raw))
	}
	if got := s.Snapshot(); !reflect.DeepEqual(got.Items, before.Items) || got.SelectedID != before.SelectedID {
		t.Fatalf("workspace frame polluted sessions: %+v", got)
	}
}

func TestWorkspaceBaselineAndArchiveEmptySetsReplacePriorState(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A"}}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[],"archivedSessionIds":["old"],"pinnedSessionIds":["A"],"scratchCwd":"/scratch"}}`))
	if got := s.Snapshot(); len(got.ArchivedSessionIDs) != 1 || len(got.PinnedSessionIDs) != 1 || !got.Items[0].Pinned {
		t.Fatalf("baseline sets not folded: %+v", got)
	}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"baseline","value":{"items":[],"archivedSessionIds":[],"pinnedSessionIds":[],"scratchCwd":"/scratch"}}`))
	if got := s.Snapshot(); len(got.ArchivedSessionIDs) != 0 || len(got.PinnedSessionIDs) != 0 || got.Items[0].Pinned {
		t.Fatalf("empty baseline did not clear: %+v", got)
	}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"archived","archivedSessionIds":["A"]}`))
	if got := s.Snapshot(); len(got.ArchivedSessionIDs) != 1 {
		t.Fatalf("archive delta not folded: %+v", got)
	}
	s.applyWorkspaceFollowFrame(json.RawMessage(`{"type":"archived","archivedSessionIds":[]}`))
	if got := s.Snapshot(); len(got.ArchivedSessionIDs) != 0 {
		t.Fatalf("empty archive delta not folded: %+v", got)
	}
}

func TestLifecycleSelectedRemovalInvalidatesOnlyItsConversation(t *testing.T) {
	for _, event := range []string{"api-session/removed", "api-session/deleted"} {
		t.Run(event, func(t *testing.T) {
			s := New()
			s.items = []Item{{SessionID: "A", Running: true}, {SessionID: "B", Running: true}}
			s.Select("B")
			generation := s.followGeneration
			s.mu.Lock()
			s.messages = []Message{{Body: "B"}}
			s.planBoard = PlanBoard{PlanMode: true}
			s.historyHasMore = true
			s.historyThroughSeq = 10
			s.mu.Unlock()
			receiveLifecycle(t, s, event, `"B"`)
			got := s.Snapshot()
			if len(got.Items) != 1 || got.Items[0].SessionID != "A" || !got.Items[0].Running || got.SelectedID != "" || len(got.Messages) != 0 || got.PlanBoard.PlanMode || got.HistoryHasMore || s.historyThroughSeq != 0 {
				t.Fatalf("selected removal touched wrong state: %+v", got)
			}
			if s.applyFollowFrameForTarget(context.Background(), "B", generation, json.RawMessage(`{"type":"snapshot","records":[]}`)) {
				t.Fatal("removed selection retained follow ownership")
			}
		})
	}
}

func TestLifecyclePromptAndCancelAcceptanceDoNotEndRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path == "/api/session/cancel" {
			rpcOK(w, body["rpcId"], map[string]any{"accepted": true})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()
	s := New()
	s.items = []Item{{SessionID: "A", Running: true}}
	s.selectedID = "A"
	promptDone := make(chan struct{})
	cancelDone := make(chan struct{})
	var promptOnce, cancelOnce sync.Once
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		got := s.Snapshot()
		if !got.PromptBusy && got.LastAction == "prompt sent" {
			promptOnce.Do(func() { close(promptDone) })
		}
		if got.LastAction == "cancel accepted" {
			cancelOnce.Do(func() { close(cancelDone) })
		}
	})
	s.SendPrompt(context.Background(), "hello")
	wait(t, promptDone)
	if got := s.Snapshot(); got.PromptBusy || !got.Items[0].Running {
		t.Fatalf("prompt acceptance ended true running state: %+v", got)
	}
	s.CancelRunning(context.Background())
	wait(t, cancelDone)
	if !s.Snapshot().Items[0].Running {
		t.Fatal("cancel acceptance ended running before status event")
	}
	receiveLifecycle(t, s, "api-session/status", `"A"`, `false`)
	if s.Snapshot().Items[0].Running {
		t.Fatal("real status=false not folded")
	}
}

func TestLifecycleNewReadyDiscardsOldListAndReplaysMutationsInOrder(t *testing.T) {
	for _, oldFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "old-success", true: "old-error"}[oldFailure], func(t *testing.T) {
			oldStarted := make(chan struct{})
			newStarted := make(chan struct{})
			oldRelease := make(chan struct{})
			newRelease := make(chan struct{})
			var calls int
			var mu sync.Mutex
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				mu.Lock()
				calls++
				call := calls
				mu.Unlock()
				if call == 1 {
					close(oldStarted)
					<-oldRelease
					if oldFailure {
						http.Error(w, "old list failed", http.StatusBadGateway)
						return
					}
					rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "stale", "running": true}}})
					return
				}
				close(newStarted)
				<-newRelease
				rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "A", "running": true, "agentAvailable": true, "blank": false, "updatedAt": 1}, {"sessionId": "gone", "running": true}}})
			}))
			defer srv.Close()
			s := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			s.Attach(client, nil)
			s.selectedID = "A"
			s.listGeneration = 1
			s.listLoading = true
			oldFinished := make(chan struct{})
			go func() { defer close(oldFinished); s.runRefreshList(context.Background(), client, 1) }()
			wait(t, oldStarted)
			done := make(chan struct{})
			var once sync.Once
			s.Attach(client, func() {
				if !s.Snapshot().ListLoading {
					once.Do(func() { close(done) })
				}
			})
			s.OnEventsReady(s.BeginEvents(context.Background(), s.client))
			wait(t, newStarted)
			receiveLifecycle(t, s, "api-session/status", `"A"`, `false`)
			receiveLifecycle(t, s, "api-session/deleted", `"gone"`)
			receiveLifecycle(t, s, "api-session/removed", `"A"`)
			receiveLifecycle(t, s, "api-session/added", `{"sessionId":"A","running":true,"agentAvailable":true,"blank":false,"updatedAt":3}`)
			receiveLifecycle(t, s, "api-session/status", `"A"`, `false`)
			receiveLifecycle(t, s, "api-session/activity", `"A"`, `20`)
			close(newRelease)
			wait(t, done)
			before := s.Snapshot()
			if len(before.Items) != 1 || before.Items[0].SessionID != "A" || before.Items[0].Running || before.Items[0].UpdatedAt != 20 || before.ListErr != "" {
				t.Fatalf("ordered replay wrong: %+v", before)
			}
			close(oldRelease)
			wait(t, oldFinished)
			if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
				t.Fatalf("old list superseded reconnect baseline: %+v", got)
			}
		})
	}
}

func TestLifecycleReadyEmptyRosterClearsOldRunningRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcOK(w, body["rpcId"], map[string]any{"items": []any{}})
	}))
	defer srv.Close()
	s := New()
	s.items = []Item{{SessionID: "A", Running: true}}
	s.selectedID = "A"
	done := make(chan struct{})
	var once sync.Once
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if !s.Snapshot().ListLoading {
			once.Do(func() { close(done) })
		}
	})
	s.OnEventsReady(s.BeginEvents(context.Background(), s.client))
	wait(t, done)
	if got := s.Snapshot(); len(got.Items) != 0 || s.RunningCount() != 0 {
		t.Fatalf("empty baseline retained stale running: %+v", got)
	}
}

func TestLifecycleAvailabilityUpsertKeepsPreviouslyKnownTitle(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "A", Title: "Known title", Running: true}}
	receiveLifecycle(t, s, "api-session/added", `{"sessionId":"A","running":false,"agentAvailable":false,"blank":false,"updatedAt":1}`)
	if got := s.Snapshot(); got.Items[0].Title != "Known title" || got.Items[0].Running || got.Items[0].AgentAvailable {
		t.Fatalf("availability refresh erased title projection: %+v", got.Items[0])
	}
}
