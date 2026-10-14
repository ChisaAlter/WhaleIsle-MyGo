package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

const statsFixture = `{"days":[{"date":"2026-10-14","total":12,"models":{"model":{"input":7,"output":5,"total":12}}}],"totals":{"input":7,"output":5,"total":12},"sessionCount":1,"byModel":[],"allTime":{"totals":{"input":7,"output":5,"total":12},"sessionCount":1,"byModel":[{"model":"=BAD()","provider":"p","input":7,"output":5,"total":12}]},"coverage":{"mode":"projection","timezone":"UTC","sessionsTotal":2,"sessionsOk":1,"sessionsFailed":1,"failedSessionIds":["bad"]},"topSessions":[],"updatedAt":1791970000000,"billingExtra":{"preserved":true}}`

func waitStats(t *testing.T, s *Store) StatsView {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Stats()
		if !v.Loading {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out")
	return StatsView{}
}
func TestStatsRPCAndExports(t *testing.T) {
	calls := make(chan bool, 5)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage-stats/overview" || r.Header.Get("Cookie") != "fixture=yes" {
			t.Error("wrong endpoint or authentication")
		}
		var req struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Force bool `json:"force"`
			} `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		calls <- req.Payload.Force
		writeRPC(w, req.RPCID, json.RawMessage(statsFixture))
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "fixture=yes"}}, nil)
	for range 2 {
		s.RefreshStats(context.Background(), true)
		v := waitStats(t, s)
		if v.Err != "" || v.Overview.AllTime.Totals.Total != 12 {
			t.Fatalf("%+v", v)
		}
		if !<-calls {
			t.Fatal("force missing")
		}
	}
	raw, _ := s.ExportStats("json")
	if !strings.Contains(string(raw), "billingExtra") {
		t.Fatal("JSON must retain complete wire payload")
	}
	csv, err := s.ExportStats("models")
	if err != nil || !strings.Contains(string(csv), "'=BAD()") {
		t.Fatalf("unsafe CSV: %s, %v", csv, err)
	}
	s.SetStatsDays(14)
	if s.Stats().Days != 14 {
		t.Fatal("range not retained")
	}
	s.Detach()
	if s.Stats().Overview != nil {
		t.Fatal("leaked stats after detach")
	}
}
func TestStatsFailurePreservesLastSuccess(t *testing.T) {
	s := New()
	s.stats.Raw = json.RawMessage(statsFixture)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.RefreshStats(context.Background(), false)
	v := waitStats(t, s)
	if v.Err == "" || v.Overview == nil {
		t.Fatalf("lost prior data/error: %+v", v)
	}
}
func TestStatsRejectsMissingContract(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"updatedAt":1}`} {
		if decodeOverview(json.RawMessage(raw)) == nil {
			t.Fatal("accepted incomplete overview")
		}
	}
}
func TestStatsDetachDiscardsLateResponse(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		writeRPC(w, "", json.RawMessage(statsFixture))
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.RefreshStats(context.Background(), false)
	<-started
	s.Detach()
	close(release)
	time.Sleep(20 * time.Millisecond)
	v := s.Stats()
	if v.Overview != nil || v.Loading || v.Err != "" {
		t.Fatalf("late response leaked: %+v", v)
	}
}
