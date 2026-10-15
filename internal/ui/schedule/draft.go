package schedule

import (
	"bytes"

	schedstore "whaleisle/internal/store/schedule"
)

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
	if d.Title == base.Title {
		d.Title = latest.Title
	}
	if d.Prompt == base.Prompt {
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
