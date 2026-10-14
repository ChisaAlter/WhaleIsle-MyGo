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

func TestProjectRankingFixture(t *testing.T) {
	var mode atomic.Int32
	calls := make(chan string, 12)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct {
				Offset int
				Sort   string
			}
		}
		json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/usage-stats/projects.more" || r.Header.Get("Cookie") != "fixture=yes" {
			t.Error("wrong route/auth")
		}
		calls <- q.Payload.Sort + ":" + string(rune('0'+q.Payload.Offset))
		if mode.Load() == 1 {
			http.Error(w, "retry", 500)
			return
		}
		if mode.Load() == 2 {
			entered <- struct{}{}
			<-release
		}
		body := `{"rows":[{"project":null,"name":"none","totals":{"total":20},"models":[]}],"hasMore":true}`
		if q.Payload.Offset == 1 {
			body = `{"rows":[{"project":null,"name":"none","totals":{"total":20}},{"project":"/work","name":"work","totals":{"total":12},"models":[]}],"hasMore":false}`
		}
		if q.Payload.Sort == "cost" {
			body = `{"rows":[{"project":"/work","name":"work","totals":{"total":12},"models":[]}],"hasMore":false}`
		}
		writeRPC(w, q.RPCID, json.RawMessage(body))
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "fixture=yes"}}, nil)
	wait := func() ProjectRanking {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for time.Now().Before(until) {
			v := s.ProjectRanking()
			if !v.Loading {
				return v
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("timeout")
		return ProjectRanking{}
	}
	ctx := context.Background()
	s.LoadMoreProjects(ctx)
	v := wait()
	if len(v.Rows) != 1 || !v.HasMore || <-calls != "tokens:0" {
		t.Fatal(v)
	}
	mode.Store(1)
	s.LoadMoreProjects(ctx)
	v = wait()
	if v.Err == "" || len(v.Rows) != 1 || <-calls != "tokens:1" {
		t.Fatal("failure lost page", v)
	}
	mode.Store(0)
	s.LoadMoreProjects(ctx)
	v = wait()
	if len(v.Rows) != 2 || v.HasMore || <-calls != "tokens:1" {
		t.Fatal("retry cursor or dedup wrong", v)
	}
	s.ResetProjectRanking(ctx, "cost")
	v = wait()
	if v.Sort != "cost" || len(v.Rows) != 1 || <-calls != "cost:0" {
		t.Fatal("sort did not reset", v)
	}
	mode.Store(2)
	s.ResetProjectRanking(ctx, "tokens")
	<-entered
	s.LoadMoreProjects(ctx)
	s.Detach()
	close(release)
	if v = s.ProjectRanking(); v.Started || len(v.Rows) != 0 || v.Loading {
		t.Fatal("detach retained old ranking", v)
	}
	if <-calls != "tokens:0" {
		t.Fatal("wrong pending offset")
	}
	select {
	case <-calls:
		t.Fatal("duplicate page request")
	default:
	}
	if _, ok := ProjectEstimate(ProjectUsage{}, BillingView{}); ok {
		t.Fatal("unknown cost priced")
	}
}
