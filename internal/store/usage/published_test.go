package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

// setStatsRaw installs a statistics payload the way a completed read does:
// decoded outside the lock, published with the rest of the views.
func (s *Store) setStatsRaw(raw json.RawMessage) {
	parsed := decodeStats(raw)
	s.mu.Lock()
	s.stats.Raw, s.stats.parsed = raw, parsed
	s.unlockPublish()
}

func TestUsageSnapshotSharedUntilWrite(t *testing.T) {
	s := New()
	s.Attach(&dsh.Client{}, nil)
	s.setStatsRaw(json.RawMessage(statsFixture))
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	if got := testing.AllocsPerRun(100, func() { _ = s.Stats(); _ = s.Billing(); _ = s.Repair() }); got != 0 {
		t.Fatalf("view accessors allocate %g per run", got)
	}
	if a, b := s.Stats().Overview, s.Stats().Overview; a == nil || a != b {
		t.Fatal("the overview must be decoded once and shared, not per read")
	}
	// Read-only sections mint no revision.
	views := s.pub.Load()
	s.fire()
	s.CanRepair("bad")
	s.StatisticsOwner()
	if s.pub.Load() != views || s.Snapshot() != first {
		t.Fatal("a section that changed nothing published a new version")
	}
}

func TestUsageWritePublishesAndSignals(t *testing.T) {
	s := New()
	before := s.Snapshot()
	changed := s.Changed()
	s.Attach(&dsh.Client{}, nil)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 || !after.Value.ClientReady {
		t.Fatalf("Attach did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	// A stats-only change republishes Stats but leaves the account view.
	s.SetStatsDays(14)
	if s.Snapshot() != after || s.Stats().Days != 14 {
		t.Fatal("stats-only change must not mint an account-view revision")
	}
}

func TestUsageStaleStatsReadDroppedAndSnapshotUnblocked(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		close(started)
		<-release
		writeRPC(w, req.RPCID, json.RawMessage(statsFixture))
	}))
	defer srv.Close()
	defer close(release)
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.Attach(client, nil)
	s.RefreshStats(context.Background(), false)
	<-started
	read := make(chan bool, 1)
	go func() { read <- s.Stats().Loading }()
	select {
	case loading := <-read:
		if !loading {
			t.Fatal("in-flight read not published as Loading")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stats blocked behind the overview RPC")
	}
	s.Attach(client, nil) // a new attachment retires the read
	held := s.pub.Load()
	release <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	if s.pub.Load() != held || s.Stats().Overview != nil {
		t.Fatal("stale stats completion was published")
	}
}

func TestUsageFaultMapping(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.RefreshStats(context.Background(), false)
	v := waitStats(t, s)
	if v.Err == nil || v.ErrLead == "" || v.Err.SentenceText() == "" {
		t.Fatalf("stats failure not classified: %+v", v)
	}
	if err := s.SetStatsDateRange("bad", "bad"); err == nil {
		t.Fatal("bad range accepted")
	}
	s.ApplyStatsRangeDraft()
	if f := s.Stats().RangeError; f == nil || f.SentenceText() != "Start date must be YYYY-MM-DD (UTC)" {
		t.Fatalf("curated validation copy lost: %+v", f)
	}
}

// Draft edits copy the map: a billing view a reader holds keeps its drafts.
func TestUsageBillingDraftCopyOnWrite(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.billing.Loaded = true
	s.billing.Drafts = map[string]PriceDraft{"p/m": {Peak: [3]string{"1", "2", "3"}}}
	s.unlockPublish()
	held := s.Billing()
	s.EditPrice("p/m", PriceDraft{Peak: [3]string{"9", "9", "9"}})
	if held.Drafts["p/m"].Peak[0] != "1" || s.Billing().Drafts["p/m"].Peak[0] != "9" {
		t.Fatal("price edit reached a published billing view")
	}
}
