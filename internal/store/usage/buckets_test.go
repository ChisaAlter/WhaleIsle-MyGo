package usage

import (
	"testing"
	"time"

	"whaleisle/internal/dsh/api"
)

func TestBuildDayBuckets(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.Local)
	jobs := []api.JobListFrameJobsItem{
		{StartedAt: float64(now.Unix())},
		{StartedAt: float64(now.Add(-24 * time.Hour).Unix())},
		{StartedAt: float64(now.Add(-24 * time.Hour).Unix())},
	}
	buckets := BuildDayBuckets(jobs, 7, now)
	if len(buckets) != 7 {
		t.Fatalf("len = %d", len(buckets))
	}
	if buckets[6].Count != 1 {
		t.Fatalf("today = %v", buckets[6])
	}
	if buckets[5].Count != 2 {
		t.Fatalf("yesterday = %v", buckets[5])
	}
	n, ok := TodayJobCount(buckets)
	if !ok || n != 1 {
		t.Fatalf("TodayJobCount = %d ok=%v", n, ok)
	}
	if _, ok := TodayJobCount(nil); ok {
		t.Fatal("empty buckets should be missing")
	}
}
