package petgrowth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAccountingV2MigrationAndSourceChange(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.Local)
	id := strings.Repeat("a", 64)
	old := int64(10)
	g := State{Points: 30, TokensFed: 20, TokensSeen: 100, Baseline: &old, Food: &Food{Version: 1, Available: 70, Seen: map[string]int64{id: 100}}}
	today := int64(120)
	next, err := RefreshUsage(g, Usage{Samples: map[string]int64{id: 120}, Source: "magpie", TodayUsed: &today}, now)
	if err != nil || next.Points != 30 || next.TokensFed != 0 || next.Food.Available != 0 || next.AccountingVersion != 2 || next.Today.Used != 120 {
		t.Fatalf("migration %+v %v", next, err)
	}
	next, err = RefreshUsage(next, Usage{Samples: map[string]int64{id: 150}, Source: "magpie", TodayUsed: &today}, now)
	if err != nil || next.Food.Available != 30 {
		t.Fatalf("delta %+v %v", next, err)
	}
	next, err = RefreshUsage(next, Usage{Samples: map[string]int64{id: 500}, Source: "sessions"}, now)
	if err != nil || next.Food.Available != 30 || next.Food.Seen[id] != 500 {
		t.Fatalf("source flip %+v %v", next, err)
	}
	next, err = RefreshUsage(next, Usage{Samples: map[string]int64{id: 510}, Source: "sessions"}, now)
	if err != nil || next.Food.Available != 40 {
		t.Fatalf("after flip %+v %v", next, err)
	}
}
func TestGatewayUsageDeduplicatesRequestsAndExcludesCache(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	file := filepath.Join(t.TempDir(), "usage.jsonl")
	raw := `{"agent":"dsh","route_id":"one","in":10,"out":2,"cache_read":800,"t":"2026-11-02T01:00:00Z"}
{"agent":"dsh","route_id":"one","in":11,"out":3,"t":"2026-11-02T01:00:00Z"}
{"agent":"other","route_id":"two","in":999,"out":999,"t":"2026-11-02T01:00:00Z"}
{"agent":"dsh","route_id":"old","in":7,"out":3,"t":"2026-11-01T01:00:00Z"}`
	if err := os.WriteFile(file, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	u, err := ReadMagpieUsage(file, now)
	if err != nil || len(u.Samples) != 2 || *u.TodayUsed != 26 {
		t.Fatalf("usage %+v %v", u, err)
	}
}
func TestScanChoosesHighestCanonicalGenerationAndSkipsInherited(t *testing.T) {
	node := scanTestNode(t)
	resetScanCache()
	defer resetScanCache()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", "one")
	os.MkdirAll(dir, 0700)
	writeLogFrame(t, node, filepath.Join(dir, "session.v4.jsonl.zstd"), usageEvent(1, 1, 1, 900))
	writeLogFrame(t, node, filepath.Join(dir, "session.v05.jsonl.zstd"), usageEvent(2, 2, 1, 800))
	writeLogFrame(t, node, filepath.Join(dir, "session.v5.jsonl.zstd"), `{"type":"session","isSeeded":true,"seedLength":4}`, usageEvent(2, 1, 1, 700), `{"type":"session/end-seed","seq":6,"data":{"inherited":true}}`, usageEvent(5, 2, 1, 600), usageEvent(7, 3, 1, 10))
	samples, err := Scan(context.Background(), node, home)
	total, e := ValidateSamples(samples)
	if err != nil || e != nil || total != 11 || len(samples) != 1 {
		t.Fatalf("canonical samples %v %v %v", samples, err, e)
	}
}
