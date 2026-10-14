package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestRankingPaginationFixture(t *testing.T) {
	var mode atomic.Int32
	entered := make(chan struct{}, 8)
	release := make(chan struct{}, 8)
	offsets := make(chan int, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string                     `json:"rpcId"`
			Payload map[string]json.RawMessage `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.URL.Path == "/usage-stats/overview" {
			writeRPC(w, req.RPCID, json.RawMessage(statsFixture))
			return
		}
		if r.URL.Path != "/usage-stats/sessions.more" || r.Header.Get("Cookie") != "fixture=yes" {
			t.Error("wrong paging route/auth")
		}
		if len(req.Payload) != 2 || string(req.Payload["sort"]) != `"tokens"` {
			t.Error("paging must use only offset/sort, never dates")
		}
		var offset int
		json.Unmarshal(req.Payload["offset"], &offset)
		offsets <- offset
		m := mode.Load()
		if m == 1 || m == 4 {
			entered <- struct{}{}
			<-release
		}
		if m == 2 {
			http.Error(w, "retry", 500)
			return
		}
		body := `{"sessions":[{"id":"a","title":null,"totals":{"total":10}}],"hasMore":true}`
		if m == 3 {
			body = `{"sessions":[{"id":"a","totals":{"total":10}},{"id":"b","totals":{"total":5}}],"hasMore":false}`
		}
		if m == 5 {
			body = `{"sessions":[],"hasMore":false}`
		}
		writeRPC(w, req.RPCID, json.RawMessage(body))
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "fixture=yes"}}, nil)
	s.stats.Raw = json.RawMessage(statsFixture)
	wait := func() SessionRanking {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for time.Now().Before(until) {
			v := s.SessionRanking()
			if !v.Loading {
				return v
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("paging timeout")
		return SessionRanking{}
	}
	mode.Store(1)
	s.LoadMoreSessions(context.Background())
	<-entered
	s.LoadMoreSessions(context.Background())
	release <- struct{}{}
	v := wait()
	if len(v.Rows) != 1 || !v.HasMore || <-offsets != 0 {
		t.Fatalf("first page: %+v", v)
	}
	select {
	case <-offsets:
		t.Fatal("duplicate click issued request")
	default:
	}
	mode.Store(2)
	s.LoadMoreSessions(context.Background())
	v = wait()
	if v.Err == "" || len(v.Rows) != 1 || <-offsets != 1 {
		t.Fatal("failed page lost rows/cursor")
	}
	mode.Store(3)
	s.LoadMoreSessions(context.Background())
	v = wait()
	if v.Err != "" || len(v.Rows) != 2 || v.HasMore || <-offsets != 1 {
		t.Fatalf("retry/dedup: %+v", v)
	}
	s.LoadMoreSessions(context.Background())
	select {
	case <-offsets:
		t.Fatal("loaded after terminal page")
	default:
	}
	s.RefreshStats(context.Background(), true)
	waitStats(t, s)
	if s.SessionRanking().Started {
		t.Fatal("refresh retained cursor")
	}
	mode.Store(4)
	s.LoadMoreSessions(context.Background())
	<-entered
	<-offsets
	s.RefreshStats(context.Background(), true)
	waitStats(t, s)
	release <- struct{}{}
	time.Sleep(20 * time.Millisecond)
	if s.SessionRanking().Started {
		t.Fatal("late page crossed refresh")
	}
	mode.Store(5)
	s.LoadMoreSessions(context.Background())
	v = wait()
	if !v.Started || len(v.Rows) != 0 || v.HasMore || <-offsets != 0 {
		t.Fatalf("empty: %+v", v)
	}
	if s.ModelRankLimit() != 10 {
		t.Fatal("model initial limit")
	}
	s.ShowMoreModels()
	if s.ModelRankLimit() != 20 {
		t.Fatal("model next page")
	}
	s.ClearStatsDateRange()
	if s.ModelRankLimit() != 10 {
		t.Fatal("date change retained model page")
	}
	s.Detach()
	if s.SessionRanking().Started {
		t.Fatal("detach retained ranking")
	}
}
