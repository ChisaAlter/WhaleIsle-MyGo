package schedule

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	schedstore "whaleisle/internal/store/schedule"
)

// chooseKind restores the committed rule when returning to its kind. Clock
// choices carry the shown time/zone; weekday choices survive those detours.
func (d *taskDraft) chooseKind(kind string) {
	if kind == d.Kind {
		return
	}
	base := newTaskDraft(d.row)
	date, clock := base.wallClock()
	zone, days := d.Zone, d.Weekdays
	if d.Kind == "daily" || d.Kind == "weekly" {
		clock = d.Clock
	}
	if kind == base.Kind {
		d.At, d.Seconds, d.Clock, d.Expression = base.At, base.Seconds, base.Clock, base.Expression
		d.Zone = base.Zone
	} else {
		d.At, d.Seconds, d.Clock = base.At, "3600", clock
		d.Zone = zone
		hour, _ := strconv.Atoi(clock[:2])
		minute, _ := strconv.Atoi(clock[3:5])
		d.Expression = cronShape{kind: "daily", hour: hour, minute: minute}.expression()
	}
	if kind == "daily" || kind == "weekly" {
		if kind != base.Kind && (d.Kind == "daily" || d.Kind == "weekly") {
			d.Clock, d.Zone = clock, zone
		}
	}
	if days == "" {
		if base.Weekdays != "" {
			days = base.Weekdays
		} else {
			instant, _ := time.Parse("2006-01-02", date)
			day := int(instant.Weekday())
			if day == 0 {
				day = 7
			}
			days = strconv.Itoa(day)
		}
	}
	d.Weekdays = strings.TrimSpace(days)
	d.Kind, d.picker = kind, timingPickerState{}
}

// rebase adopts a refreshed CAS record while preserving fields edited locally.
// A staged rule-kind switch keeps its full timing; untouched timing follows the
// Host. An ended task becomes read-only and cannot retain a misleading draft.
func (d *taskDraft) rebase(row schedstore.Entry) {
	if bytes.Equal(d.row.Raw, row.Raw) && d.row.Status == row.Status {
		return
	}
	latest := newTaskDraft(row)
	if row.Status == "inactive" {
		*d = *latest
		return
	}
	base := newTaskDraft(d.row)
	if strings.TrimSpace(d.Title) == strings.TrimSpace(base.Title) {
		d.Title = latest.Title
	}
	if strings.TrimSpace(d.Prompt) == strings.TrimSpace(base.Prompt) {
		d.Prompt = latest.Prompt
	}
	dirtyTiming := d.timingKey() != d.initialTiming
	if !dirtyTiming {
		d.Kind, d.At, d.Seconds = latest.Kind, latest.At, latest.Seconds
		d.Clock, d.Zone, d.Weekdays, d.Expression = latest.Clock, latest.Zone, latest.Weekdays, latest.Expression
	} else if d.Kind == base.Kind && latest.Kind == base.Kind {
		for _, fields := range []struct {
			shown      *string
			old, fresh string
		}{
			{&d.At, base.At, latest.At},
			{&d.Seconds, base.Seconds, latest.Seconds},
			{&d.Clock, base.Clock, latest.Clock},
			{&d.Zone, base.Zone, latest.Zone},
			{&d.Weekdays, base.Weekdays, latest.Weekdays},
			{&d.Expression, base.Expression, latest.Expression},
		} {
			if *fields.shown == fields.old {
				*fields.shown = fields.fresh
			}
		}
	}
	d.row = row
	d.initialTiming = latest.timingKey()
}

func taskKey(row schedstore.Entry) string {
	return row.Origin + "/" + row.SessionID + "/" + row.ID
}
