package schedule

import (
	"slices"
	"testing"
	"time"
)

// TestNextRunFindsTheFollowingOccurrence covers the form's preview: a daily 09:00
// task seen at 10:00 reports tomorrow, not today.
func TestNextRunFindsTheFollowingOccurrence(t *testing.T) {
	parsed, err := parseTaskCron("0 9 * * *")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	at, ok := parsed.nextAfter(base, time.UTC)
	if !ok {
		t.Fatal("no next run inside the window")
	}
	if at.Day() != 21 || at.Hour() != 9 || at.Minute() != 0 {
		t.Fatalf("next run = %s", at)
	}
}

// TestNextRunHandlesMinuteSteps keeps the interval shapes honest.
func TestNextRunHandlesMinuteSteps(t *testing.T) {
	parsed, err := parseTaskCron("*/15 * * * *")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	base := time.Date(2026, 10, 20, 10, 7, 0, 0, time.UTC)
	at, ok := parsed.nextAfter(base, time.UTC)
	if !ok || at.Minute() != 15 || at.Hour() != 10 {
		t.Fatalf("next run = %s ok=%v", at, ok)
	}
}

// TestMonthLimitsStayOutsideTheWindow keeps a shape that cannot fire inside the
// six-week horizon from inventing a time.
func TestMonthLimitsStayOutsideTheWindow(t *testing.T) {
	parsed, err := parseTaskCron("0 0 1 1 *")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	if at, ok := parsed.nextAfter(base, time.UTC); ok {
		t.Fatalf("January-only task reported %s", at)
	}
}

// TestMinuteMarksAreQuarterHours keeps the picker on quarter hours without moving
// a task that already runs off the mark.
func TestMinuteMarksAreQuarterHours(t *testing.T) {
	if got := minuteMarks("15"); len(got) != 4 {
		t.Fatalf("quarter marks = %v", got)
	}
	off := minuteMarks("07")
	if len(off) != 5 || !slices.Contains(off, "07") {
		t.Fatalf("off-mark minute lost: %v", off)
	}
}
