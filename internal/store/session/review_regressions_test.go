package session

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"whaleisle/internal/dsh"
)

func TestCanceledHistoryPageReleasesOnlyItsBusyState(t *testing.T) {
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	s.Attach(client, nil)
	s.selectedID = "A"
	s.historyPageBusy = true
	s.historyHasMore = true
	s.lastAction = "keep status"
	before := s.Snapshot()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.runHistoryPage(ctx, client, "A", s.followGeneration, 10, 5, s.historyPageRequest)
	got := s.Snapshot()
	before.HistoryPageBusy = false
	if !reflect.DeepEqual(got, before) {
		t.Fatalf("canceled page must release owned busy without changing transcript/error/status: %+v", got)
	}
}

func TestListReplayDoesNotRepeatForegroundEffects(t *testing.T) {
	for _, kind := range []string{"error", "remove-add"} {
		t.Run(kind, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				close(started)
				<-release
				rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "A", "running": true}}})
			}))
			defer srv.Close()
			s := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			s.Attach(client, nil)
			s.selectedID = "A"
			s.items = []Item{{SessionID: "A"}}
			s.listLoading = true
			generation := s.listGeneration
			done := make(chan struct{})
			go func() { defer close(done); s.runRefreshList(context.Background(), client, generation) }()
			wait(t, started)
			if kind == "error" {
				receiveLifecycle(t, s, "api-session/error", `"A"`, `"old A error"`)
			} else {
				receiveLifecycle(t, s, "api-session/removed", `"A"`)
				receiveLifecycle(t, s, "api-session/added", `{"sessionId":"A","running":false,"blank":false,"agentAvailable":true,"updatedAt":2}`)
			}
			s.mu.Lock()
			s.selectSessionLocked("B")
			s.selectSessionLocked("A")
			s.lastAction = "new A status"
			s.messages = []Message{{ID: "new", Body: "new A"}}
			s.planBoard = PlanBoard{PlanMode: true}
			s.mu.Unlock()
			before := s.Snapshot()
			close(release)
			wait(t, done)
			got := s.Snapshot()
			if got.LastAction != before.LastAction || got.SelectedID != before.SelectedID || !reflect.DeepEqual(got.Messages, before.Messages) || !reflect.DeepEqual(got.PlanBoard, before.PlanBoard) {
				t.Fatalf("replayed transient effect changed new A: %+v", got)
			}
		})
	}
}

func TestBufferedCanceledEventsCannotMutateReplacementClient(t *testing.T) {
	firstNotify, releaseNotify, lost := make(chan struct{}), make(chan struct{}), make(chan struct{})
	serverStop := make(chan struct{})
	var notifyOnce, lostOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), c, &open) != nil {
			return
		}
		for _, raw := range []string{`{"type":"emit","event":"api-session/activity","args":["A",2]}`, `{"type":"emit","event":"api-session/status","args":["A",false]}`} {
			if wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: json.RawMessage(raw)}) != nil {
				return
			}
		}
		// Establish the blocked first callback before dropping the carrier.
		// Recv intentionally rejects all queued frames after carrier loss; an
		// immediate close can therefore retire both frames before either emits.
		select {
		case <-firstNotify:
		case <-serverStop:
		}
	}))
	defer func() {
		close(serverStop)
		srv.Close()
	}()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint, OnCarrierLost: func(error) { lostOnce.Do(func() { close(lost) }) }}
	defer mux.Close()
	s := New()
	s.Attach(client, func() { notifyOnce.Do(func() { close(firstNotify); <-releaseNotify }) })
	s.items = []Item{{SessionID: "A", Running: true}}
	s.selectedID = "A"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := s.BeginEvents(ctx, client)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = client.FollowEvents(ctx, mux, dsh.EventsFuncs{Emit: func(emit dsh.EventsEmit) { s.OnEmit(owner, emit) }})
	}()
	wait(t, firstNotify)
	wait(t, lost)
	cancel()
	s.Attach(&dsh.Client{}, nil)
	s.mu.Lock()
	s.items = []Item{{SessionID: "A", Running: true}}
	s.mu.Unlock()
	close(releaseNotify)
	wait(t, done)
	if !s.Snapshot().Items[0].Running {
		t.Fatal("buffered canceled old events changed replacement client")
	}
}

func TestCanceledOldHistoryRequestCannotClearNewRequestBusy(t *testing.T) {
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	s.Attach(client, nil)
	s.selectedID = "A"
	s.historyPageBusy = true
	s.historyPageRequest = 2
	before := s.Snapshot()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.runHistoryPage(ctx, client, "A", s.followGeneration, 10, 5, 1)
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("old canceled request released newer request busy: %+v", got)
	}
}

func TestListReplayDeletedAbsentLocalDoesNotRetainSelection(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(started)
		<-release
		rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{{"sessionId": "A"}}})
	}))
	defer srv.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.Attach(client, nil)
	s.selectedID = "A"
	s.messages = []Message{{Body: "A"}}
	s.listLoading = true
	generation := s.listGeneration
	done := make(chan struct{})
	go func() { defer close(done); s.runRefreshList(context.Background(), client, generation) }()
	wait(t, started)
	receiveLifecycle(t, s, "api-session/deleted", `"A"`)
	close(release)
	wait(t, done)
	if got := s.Snapshot(); got.SelectedID != "" || len(got.Items) != 0 || len(got.Messages) != 0 {
		t.Fatalf("deleted baseline-only target remained selected: %+v", got)
	}
}

func TestEventsOwnersFenceSameClientReconnectAndCanceledReady(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	s.Attach(client, nil)
	s.items = []Item{{SessionID: "A", Running: true}}
	s.selectedID = "A"
	oldOwner := s.BeginEvents(context.Background(), client)
	newOwner := s.BeginEvents(context.Background(), client)
	before := s.Snapshot()
	generation := s.listGeneration
	s.OnEmit(oldOwner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/status", Args: []json.RawMessage{json.RawMessage(`"A"`), json.RawMessage(`false`)}})
	s.OnEventsReady(oldOwner)
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) || s.listGeneration != generation {
		t.Fatalf("same-client old stream bypassed generation fence: %+v", got)
	}
	s.OnEmit(newOwner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/status", Args: []json.RawMessage{json.RawMessage(`"A"`), json.RawMessage(`false`)}})
	if s.Snapshot().Items[0].Running {
		t.Fatal("current stream owner was rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	canceledOwner := s.BeginEvents(ctx, client)
	cancel()
	before = s.Snapshot()
	generation = s.listGeneration
	s.OnEventsReady(canceledOwner)
	s.OnEmit(canceledOwner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/status", Args: []json.RawMessage{json.RawMessage(`"A"`), json.RawMessage(`true`)}})
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) || s.listGeneration != generation {
		t.Fatalf("canceled Ready/Emit disturbed active store: %+v", got)
	}
	if s.BeginEvents(ctx, client) != nil {
		t.Fatal("canceled ctx acquired a new events generation")
	}
	if s.BeginEvents(context.Background(), &dsh.Client{}) != nil {
		t.Fatal("wrong client acquired a new events generation")
	}
}

func TestEventsOwnerCannotCrossStores(t *testing.T) {
	client := &dsh.Client{}
	a, b := New(), New()
	a.Attach(client, nil)
	b.Attach(client, nil)
	owner := a.BeginEvents(context.Background(), client)
	b.BeginEvents(context.Background(), client)
	b.items = []Item{{SessionID: "A", Running: true}}
	before := b.Snapshot()
	generation := b.listGeneration
	b.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/status", Args: []json.RawMessage{json.RawMessage(`"A"`), json.RawMessage(`false`)}})
	b.OnEventsReady(owner)
	if got := b.Snapshot(); !reflect.DeepEqual(got, before) || b.listGeneration != generation {
		t.Fatalf("another store's owner was accepted: %+v", got)
	}
}
