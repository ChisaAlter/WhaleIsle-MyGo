package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// One bounded synthetic fixture. It never starts a Host, model, or real session.
func TestNativeHistoryTurnNavigationFixture(t *testing.T) {
	record := func(seq float64, typ string, turn int, body string) api.SessionPageRecordsItem {
		data, _ := json.Marshal(map[string]any{"turn": turn, "source": map[string]any{"kind": "user"}, "content": []any{map[string]any{"type": "text", "text": body}}})
		return api.SessionPageRecordsItem{Type: "event", Event: api.SessionPageRecordsItemEvent{Type: typ, Seq: seq, Data: data}}
	}
	makeStore := func(client *dsh.Client) *Store {
		s := New()
		s.client = client
		s.selectedID = "main"
		s.followGeneration = 8
		s.applyFollowFrameLocked(json.RawMessage(`{"type":"snapshot","cursor":200,"hasMore":true,"records":[{"type":"event","event":{"type":"turn/start","seq":180,"data":{"turn":3}}},{"type":"event","event":{"type":"user/message","seq":181,"data":{"turn":3,"source":{"kind":"user"},"content":[{"type":"text","text":"latest"}]}}}],"projections":{"asOfSeq":200,"values":{"turnOutline":[{"turn":1,"seq":1,"prompt":"first","response":"one"},{"turn":2,"seq":80,"prompt":"middle","response":"two"},{"turn":3,"seq":180,"prompt":"latest","response":""}]}}}`))
		return s
	}
	settle := func(t *testing.T, s *Store) HistoryNavigationView {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for time.Now().Before(until) {
			v := s.Snapshot().HistoryNavigation
			if !v.Busy {
				return v
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("history jump did not settle")
		return HistoryNavigationView{}
	}
	t.Run("projection and malformed entries", func(t *testing.T) {
		s := makeStore(&dsh.Client{})
		v := s.Snapshot().HistoryNavigation
		if len(v.Turns) != 3 || v.Turns[0].Loaded || !v.Turns[2].Loaded {
			t.Fatalf("outline: %+v", v)
		}
		s.mu.Lock()
		s.applyHistoryOutlineControlLocked(json.RawMessage(`{"type":"projection","sessionId":"other","key":"turnOutline","seq":201,"value":[]}`))
		s.applyHistoryOutlineControlLocked(json.RawMessage(`{"type":"projection","sessionId":"main","key":"turnOutline","seq":199,"value":[]}`))
		s.mu.Unlock()
		if len(s.Snapshot().HistoryNavigation.Turns) != 3 {
			t.Fatal("wrong session/stale outline replaced current")
		}
		s.mu.Lock()
		s.applyHistoryOutlineValuesLocked(json.RawMessage(`{"turnOutline":[{"turn":1,"seq":1,"prompt":42},{"seq":5},{"turn":-1,"seq":5},{"turn":2,"seq":2.5}]}`), 202)
		s.mu.Unlock()
		v = s.Snapshot().HistoryNavigation
		if len(v.Turns) != 2 || v.Turns[0].Prompt != "" {
			t.Fatalf("malformed outline or loaded fallback: %+v", v)
		}
	})
	t.Run("newer projection survives older snapshot and stale UI cannot jump", func(t *testing.T) {
		s := makeStore(&dsh.Client{})
		s.mu.Lock()
		s.applyHistoryOutlineControlLocked(json.RawMessage(`{"type":"projection","sessionId":"main","key":"turnOutline","seq":201,"value":[{"turn":4,"seq":201,"prompt":"new"}]}`))
		oldRequest := s.historyNavigation.view.Request
		s.resetHistoryPagingLocked()
		s.applyHistoryOutlineValuesLocked(json.RawMessage(`{"turnOutline":[{"turn":1,"seq":1,"prompt":"old"}]}`), 200)
		s.mu.Unlock()
		v := s.Snapshot().HistoryNavigation
		if len(v.Turns) != 1 || v.Turns[0].Turn != 4 {
			t.Fatalf("snapshot rewound newer control projection: %+v", v)
		}
		s.JumpToHistoryTurn(context.Background(), "main", oldRequest, 201)
		if s.Snapshot().HistoryNavigation.HasTarget {
			t.Fatal("stale UI state accepted")
		}
	})
	t.Run("multi page wire and atomic landing", func(t *testing.T) {
		var calls atomic.Int32
		var wireErr atomic.Value
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				RPCID   any `json:"rpcId"`
				Payload struct {
					Args struct {
						Request api.SessionPageRequest `json:"request"`
					} `json:"args"`
				} `json:"payload"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			q := in.Payload.Args.Request
			n := calls.Add(1)
			if !strings.HasSuffix(r.URL.Path, "/session/page") || string(q.Address) != `{"kind":"session","sessionId":"main"}` || q.ThroughSeq != 200 || q.MaxMessages == nil || *q.MaxMessages != 500 || q.TurnWindow == nil || q.TurnWindow.MinMessages != 200 || q.TurnWindow.MinTurns != 2 {
				wireErr.Store("incorrect canonical page request")
			}
			want := 180.
			if n == 2 {
				want = 80
			}
			if q.BeforeSeq == nil || *q.BeforeSeq != want {
				wireErr.Store("page cursor did not advance through raw turn boundary")
			}
			records := []api.SessionPageRecordsItem{record(80, "turn/start", 2, ""), record(81, "user/message", 2, "middle")}
			if n == 2 {
				records = []api.SessionPageRecordsItem{record(1, "turn/start", 1, ""), record(2, "user/message", 1, "first")}
			}
			records = denseHistoryFixture(records, want)
			rpcOK(w, in.RPCID, api.SessionPage{Records: records, HasMore: n == 1})
		}))
		defer srv.Close()
		s := makeStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
		s.messages = append(s.messages, Message{ID: "pending", Role: "assistant", Kind: MessageKindText, Body: "live prefix", Streaming: true})
		s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1)
		v := settle(t, s)
		if v.Err != "" || !v.Ready || v.TargetSeq != 1 || calls.Load() != 2 {
			t.Fatalf("jump: %+v calls=%d", v, calls.Load())
		}
		if x := wireErr.Load(); x != nil {
			t.Fatal(x)
		}
		s.mu.Lock()
		if s.selectedID != "main" || s.historyThroughSeq != 200 || s.messages[len(s.messages)-1].Body != "live prefix" || s.messages[0].Body != "first" {
			t.Fatal("jump changed live scope or order")
		}
		s.mu.Unlock()
		if !s.ConsumeHistoryTurnLanding("main", v.Request) || s.ConsumeHistoryTurnLanding("main", v.Request) {
			t.Fatal("landing must be consumed once")
		}
		s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 80)
		v = settle(t, s)
		if !v.Ready || calls.Load() != 2 {
			t.Fatal("loaded turn should scroll without RPC")
		}
	})
	t.Run("no progress retry and ordinary page exclusion", func(t *testing.T) {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			n := calls.Add(1)
			records := []api.SessionPageRecordsItem{record(180, "turn/start", 3, "")}
			if n == 2 {
				records = []api.SessionPageRecordsItem{record(1, "turn/start", 1, ""), record(2, "user/message", 1, "first")}
				records = denseHistoryFixture(records, 180)
			}
			rpcOK(w, in["rpcId"], api.SessionPage{Records: records, HasMore: true})
		}))
		defer srv.Close()
		s := makeStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
		s.historyPageBusy = true
		s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1)
		if calls.Load() != 0 || s.Snapshot().HistoryNavigation.Busy {
			t.Fatal("jump stole ordinary page")
		}
		s.historyPageBusy = false
		s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1)
		v := settle(t, s)
		if v.Err == "" || v.Ready || v.TargetSeq != 1 || calls.Load() != 1 {
			t.Fatalf("no progress: %+v", v)
		}
		s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1)
		v = settle(t, s)
		if !v.Ready || v.Err != "" || calls.Load() != 2 {
			t.Fatalf("retry: %+v", v)
		}
	})
	for _, kind := range []string{"cancel", "new selection", "host replacement", "follow replacement", "new target"} {
		t.Run(kind, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var in map[string]any
				_ = json.NewDecoder(r.Body).Decode(&in)
				calls.Add(1)
				close(started)
				<-release
				rpcOK(w, in["rpcId"], api.SessionPage{Records: []api.SessionPageRecordsItem{record(1, "turn/start", 1, ""), record(2, "user/message", 1, "first")}})
			}))
			defer srv.Close()
			s := makeStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
			s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1)
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("page did not start")
			}
			s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 1) // duplicate must not dispatch
			switch kind {
			case "cancel":
				s.CancelHistoryTurnJump("main", s.Snapshot().HistoryNavigation.Request)
			case "new selection":
				s.mu.Lock()
				s.selectSessionLocked("other")
				s.mu.Unlock()
			case "host replacement":
				s.Attach(&dsh.Client{}, nil)
			case "follow replacement":
				s.mu.Lock()
				s.invalidateFollowLocked()
				s.mu.Unlock()
			case "new target":
				s.JumpToHistoryTurn(context.Background(), "main", s.Snapshot().HistoryNavigation.Request, 180)
			}
			close(release)
			v := settle(t, s)
			if v.Busy || calls.Load() != 1 {
				t.Fatalf("cancellation: %+v", v)
			}
			if kind == "new target" {
				if !v.Ready || v.TargetSeq != 180 {
					t.Fatal("replacement target lost")
				}
			} else if v.Ready {
				t.Fatal("cancelled work offered a landing")
			}
			s.mu.Lock()
			for _, m := range s.messages {
				if m.Body == "first" {
					t.Fatal("stale page changed history")
				}
			}
			s.mu.Unlock()
		})
	}
}

// Control is independent of follow generation, but a replacement HTTP client
// must not accept projections from an older Host even when the mux is reused.
func TestNativeHistoryTurnControlOwnerFixture(t *testing.T) {
	s := New()
	client, mux := &dsh.Client{}, &dsh.Mux{}
	s.client = client
	s.mux = mux
	s.selectedID = "main"
	ctx, cancel := context.WithCancel(context.Background())
	if !s.controlWatchCurrentLocked(ctx, client, mux, "main") {
		t.Fatal("current control owner rejected")
	}
	s.client = &dsh.Client{}
	if s.controlWatchCurrentLocked(ctx, client, mux, "main") {
		t.Fatal("old Host accepted")
	}
	s.client = client
	if s.controlWatchCurrentLocked(ctx, client, &dsh.Mux{}, "main") {
		t.Fatal("old mux accepted")
	}
	if s.controlWatchCurrentLocked(ctx, client, mux, "other") {
		t.Fatal("other session accepted")
	}
	cancel()
	if s.controlWatchCurrentLocked(ctx, client, mux, "main") {
		t.Fatal("cancelled watcher accepted")
	}
}
