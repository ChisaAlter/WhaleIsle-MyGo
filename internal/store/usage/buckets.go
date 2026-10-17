package usage

import (
	"time"

	"whaleisle/internal/dsh/api"
)

// DayBucket is one day of job activity for the native chart.
type DayBucket struct {
	Label string
	Count float64
}

// BuildDayBuckets aggregates job StartedAt into the last `days` local days
// (oldest → newest). StartedAt is treated as unix seconds when < 1e12, else ms.
func BuildDayBuckets(jobs []api.JobListFrameJobsItem, days int, now time.Time) []DayBucket {
	if days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -(days - 1))
	counts := make([]float64, days)
	for _, j := range jobs {
		t := jobStartTime(j.StartedAt, loc)
		if t.IsZero() || t.Before(start) || t.After(today.Add(24*time.Hour)) {
			continue
		}
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		idx := int(day.Sub(start) / (24 * time.Hour))
		if idx >= 0 && idx < days {
			counts[idx]++
		}
	}
	out := make([]DayBucket, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		out[i] = DayBucket{
			Label: d.Format("1/2"),
			Count: counts[i],
		}
	}
	return out
}

func jobStartTime(startedAt float64, loc *time.Location) time.Time {
	if startedAt <= 0 {
		return time.Time{}
	}
	sec := startedAt
	if startedAt >= 1e12 {
		sec = startedAt / 1000
	}
	return time.Unix(int64(sec), 0).In(loc)
}

// TodayJobCount returns the newest day bucket count when buckets exist.
func TodayJobCount(buckets []DayBucket) (int, bool) {
	if len(buckets) == 0 {
		return 0, false
	}
	return int(buckets[len(buckets)-1].Count), true
}
