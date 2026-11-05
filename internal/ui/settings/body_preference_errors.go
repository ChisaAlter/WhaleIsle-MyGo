package settings

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

type bodyPreferenceErrorNotice struct {
	id, failure string
	visible     bool
}

type bodyPreferenceErrorNotices struct {
	rows              map[string]*bodyPreferenceErrorNotice
	rendering, closed bool
}

func preferenceErrorNotices(c *ui.Context) *bodyPreferenceErrorNotices {
	return ui.Local(c.Root(), "settings-preference-field-errors", func() bodyPreferenceErrorNotices {
		return bodyPreferenceErrorNotices{rows: map[string]*bodyPreferenceErrorNotice{}}
	})
}

// A field's toast has the same lifetime as its current settings consumer.
// Search, section switches and Host replacement retire notices whose rows left.
func beginBodyPreferenceErrorNotices(c *ui.Context) func() {
	notices := preferenceErrorNotices(c)
	if notices.rendering {
		return func() {}
	}
	notices.rendering, notices.closed = true, false
	for _, row := range notices.rows {
		row.visible = false
	}
	return func() {
		for key, row := range notices.rows {
			if !row.visible || notices.closed {
				c.CloseToast(row.id)
				delete(notices.rows, key)
			}
		}
		notices.rendering = false
	}
}

// ClearPreferenceErrorNotices retires settings-owned field toasts when the
// outer shell leaves both settings and its standalone usage contribution.
func ClearPreferenceErrorNotices(c *ui.Context) {
	notices := preferenceErrorNotices(c)
	notices.closed = true
	for key, row := range notices.rows {
		c.CloseToast(row.id)
		delete(notices.rows, key)
	}
}

// Only a mounted row reads its exact namespace.field failure. Retrying clears
// its prior presentation even if the same refusal returns before another frame.
func bodyPreferenceFieldError(c *ui.Context, a *BodySettingsAccess, namespace, field, title string) func() {
	notices := preferenceErrorNotices(c)
	key := fmt.Sprintf("%d/%s.%s", a.Owner, namespace, field)
	clear := func() {
		if row := notices.rows[key]; row != nil {
			c.CloseToast(row.id)
			delete(notices.rows, key)
		}
	}
	failure := a.Snapshot.PreferenceErrors[namespace+"."+field]
	if notices.closed || !a.Snapshot.ClientReady || failure == "" {
		clear()
		return clear
	}
	row := notices.rows[key]
	if row == nil {
		row = &bodyPreferenceErrorNotice{id: "settings-preference-field/" + key}
		notices.rows[key] = row
	}
	row.visible = true
	if row.failure != failure {
		row.failure = failure
		c.AddToast(ui.Toast{ID: row.id, Title: title, Description: failure, Type: "error"})
	}
	return clear
}
