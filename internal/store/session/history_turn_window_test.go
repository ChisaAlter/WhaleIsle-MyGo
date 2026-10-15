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

// One new synthetic fixture for ordinary single-page turn-aligned history.
// No real Host, model, session, or old test suite is started.
func TestNativeOrdinaryHistoryTurnWindowFixture(t *testing.T) {
	var records []api.SessionPageRecordsItem
	records = append(records, api.SessionPageRecordsItem{Type: "event", Event: api.SessionPageRecordsItemEvent{Type: "turn/start", Seq: 120, Data: json.RawMessage(`{"turn":2}`)}})
	for seq := 121; seq <= 180; seq++ {
		records = append(records, api.SessionPageRecordsItem{Type: "event", Event: api.SessionPageRecordsItemEvent{Type: "user/message", Seq: float64(seq), Data: json.RawMessage(`{"turn":2,"source":{"kind":"user"},"content":[{"type":"text","text":"older page"}]}`)}})
	}
	makeStore := func(client *dsh.Client) *Store {
		s := New()
		s.client = client
		s.selectedID = "main"
		s.followGeneration = 8
		s.historyThroughSeq = 500
		s.historyHasMore = true
		s.historyNavigation.baseKnown = true
		s.historyNavigation.baseSeq = 480
		s.messages = []Message{{ID: "latest", Seq: 490, Kind: MessageKindText, Role: "user", Body: "existing"}, {ID: "live", Kind: MessageKindText, Role: "assistant", Body: "live prefix", Streaming: true}}
		return s
	}
	waitDone := func(t *testing.T, ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("synthetic request did not settle")
		}
	}
	waitIdle := func(t *testing.T, s *Store) {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for time.Now().Before(until) {
			if !s.Snapshot().HistoryPageBusy {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("page stayed busy")
	}
	type wire struct {
		RPCID   any `json:"rpcId"`
		Payload struct {
			Args struct {
				Request api.SessionPageRequest `json:"request"`
			} `json:"args"`
		} `json:"payload"`
	}
	t.Run("one click one turn aligned page with captured cut", func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		requests := make(chan api.SessionPageRequest, 2)
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in wire
			_ = json.NewDecoder(r.Body).Decode(&in)
			requests <- in.Payload.Args.Request
			if calls.Add(1) == 1 {
				close(entered)
			}
			<-release
			rpcOK(w, in.RPCID, api.SessionPage{Records: records, HasMore: true})
		}))
		defer srv.Close()
		s := makeStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
		s.LoadEarlierHistory(context.Background())
		waitDone(t, entered)
		// The user clicks again while busy and a new durable event arrives. Neither
		// can alter this request's cut or start a second automatic page.
		s.LoadEarlierHistory(context.Background())
		s.mu.Lock()
		s.bumpHistoryThroughSeq(501)
		s.mu.Unlock()
		close(release)
		waitIdle(t, s)
		q := <-requests
		if string(q.Address) != `{"kind":"session","sessionId":"main"}` || q.ThroughSeq != 500 || q.BeforeSeq == nil || *q.BeforeSeq != 480 || q.MaxMessages == nil || *q.MaxMessages != 500 || q.TurnWindow == nil || q.TurnWindow.MinMessages != 50 || q.TurnWindow.MinTurns != 2 {
			t.Fatalf("wrong original wire: %+v", q)
		}
		if calls.Load() != 1 {
			t.Fatalf("ordinary paging became an automatic loop: %d", calls.Load())
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.messages) != 62 || s.messages[0].Seq != 121 || !s.messages[0].TurnOpening || s.messages[0].TurnSeq != 120 || s.messages[61].ID != "live" || s.messages[61].Body != "live prefix" || s.historyThroughSeq != 501 || !s.historyHasMore || s.historyBeforeSeqLocked() != 120 {
			t.Fatal("page was clipped, reordered or changed the live window")
		}
	})
	t.Run("failure keeps history and explicit retry", func(t *testing.T) {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in wire
			_ = json.NewDecoder(r.Body).Decode(&in)
			if calls.Add(1) == 1 {
				http.Error(w, "synthetic temporary failure", http.StatusServiceUnavailable)
				return
			}
			rpcOK(w, in.RPCID, api.SessionPage{Records: records, HasMore: false})
		}))
		defer srv.Close()
		s := makeStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
		s.LoadEarlierHistory(context.Background())
		waitIdle(t, s)
		v := s.Snapshot()
		if v.HistoryPageErr == "" || !v.HistoryHasMore || len(v.Messages) != 2 {
			t.Fatalf("failure discarded current window: %+v", v)
		}
		s.LoadEarlierHistory(context.Background())
		waitIdle(t, s)
		if s.Snapshot().HistoryPageErr != "" || calls.Load() != 2 {
			t.Fatal("explicit retry did not settle")
		}
	})
	for _, action := range []string{"cancel", "Host replacement", "session replacement", "follow replacement"} {
		t.Run(action, func(t *testing.T) {
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var in wire
				_ = json.NewDecoder(r.Body).Decode(&in)
				calls.Add(1)
				close(entered)
				<-release
				rpcOK(w, in.RPCID, api.SessionPage{Records: records, HasMore: false})
			}))
			defer srv.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			s := makeStore(client)
			s.historyPageBusy = true
			s.historyPageRequest = 4
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { s.runHistoryPage(ctx, client, "main", 8, 500, 480, 4); close(done) }()
			waitDone(t, entered)
			switch action {
			case "cancel":
				cancel()
			case "Host replacement":
				s.Attach(&dsh.Client{}, nil)
			case "session replacement":
				s.mu.Lock()
				s.selectSessionLocked("other")
				s.mu.Unlock()
			case "follow replacement":
				s.mu.Lock()
				s.invalidateFollowLocked()
				s.mu.Unlock()
			}
			close(release)
			waitDone(t, done)
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, m := range s.messages {
				if strings.Contains(m.Body, "older page") {
					t.Fatal("stale/cancelled page polluted history")
				}
			}
			if calls.Load() != 1 {
				t.Fatal("cancelled single-page read retried itself")
			}
		})
	}
}
