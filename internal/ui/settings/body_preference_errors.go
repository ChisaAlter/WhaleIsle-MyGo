package settings

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// bodyPreferenceErrorNotice is one field's refusal as it is being shown: the
// sentence on screen, and whether this mount already announced it.
type bodyPreferenceErrorNotice struct {
	failure   string
	stale     string
	announced bool
}

type bodyPreferenceErrorNotices struct {
	rows              map[string]*bodyPreferenceErrorNotice
	announced         map[string]uint64
	frame             uint64
	rendering, closed bool
}

func preferenceErrorNotices(c *ui.Context) *bodyPreferenceErrorNotices {
	return ui.Local(c.Root(), "settings-preference-field-errors", func() bodyPreferenceErrorNotices {
		return bodyPreferenceErrorNotices{rows: map[string]*bodyPreferenceErrorNotice{}, announced: map[string]uint64{}}
	})
}

// preferenceFailureCopy names the setting that was refused, so the sentence a
// person reads tells them which control failed and what to do about it.
func preferenceFailureCopy(title string) string {
	return fmt.Sprintf(providerCopy("「%s」没能保存，请重试。", "“%s” could not be saved. Try again."), title)
}

func preferenceErrorKey(a *BodySettingsAccess, namespace, field string) string {
	return fmt.Sprintf("%d/%s.%s", a.Owner, namespace, field)
}

// A field's alert lives as long as the settings surface that reads it. A new
// mount reads the retained failure as news again; leaving settings retires it.
func beginBodyPreferenceErrorNotices(c *ui.Context) func() {
	notices := preferenceErrorNotices(c)
	if notices.rendering {
		return func() {}
	}
	notices.rendering, notices.closed = true, false
	notices.frame++
	return func() { notices.rendering = false }
}

// ClearPreferenceErrorNotices retires settings-owned field alerts when the
// outer shell leaves both settings and its standalone usage contribution.
func ClearPreferenceErrorNotices(c *ui.Context) {
	notices := preferenceErrorNotices(c)
	notices.closed = true
	clear(notices.rows)
}

// bodyPreferenceFieldError returns the clear a row's own control runs when the
// user retries: the next failure is news even when it repeats the sentence.
func bodyPreferenceFieldError(c *ui.Context, a *BodySettingsAccess, namespace, field, title string) func() {
	return func() {
		// The sentence on screen is about the attempt the user just replaced:
		// it stays where it is, and the next result is the one that is news.
		if row := preferenceErrorNotices(c).rows[preferenceErrorKey(a, namespace, field)]; row != nil {
			row.stale = row.failure
		}
	}
}

// bodyPreferenceFieldAlert shows a row's refusal under that row, where the
// control that was refused sits. The sentence names the setting, so a person
// knows which one failed, and the store's own words sit behind 详情. It
// announces the sentence once, so a screen reader hears a new refusal without
// the page floating a toast over the content.
func bodyPreferenceFieldAlert(c *ui.Context, a *BodySettingsAccess, namespace, field, title string) {
	notices := preferenceErrorNotices(c)
	if notices.closed || !a.Snapshot.ClientReady {
		return
	}
	key := preferenceErrorKey(a, namespace, field)
	failure := a.Snapshot.PreferenceErrors[namespace+"."+field]
	if failure == "" {
		delete(notices.rows, key)
		return
	}
	row := notices.rows[key]
	if row == nil {
		row = &bodyPreferenceErrorNotice{}
		notices.rows[key] = row
	}
	if row.failure != failure {
		row.failure, row.announced = failure, false
	}
	if row.stale == failure {
		row.announced = true
	} else {
		row.stale = ""
	}
	sentence, detail := settingsFailure("settings/"+key, preferenceFailureCopy(title), failure)
	if !row.announced && !a.Snapshot.PreferencePending[namespace+"."+field] {
		row.announced = true
		if notices.announced[failure] != notices.frame {
			notices.announced[failure] = notices.frame
			c.Announce(sentence)
		}
	}
	widgets.Alert(c, widgets.AlertOptions{ID: "settings/" + key, Message: sentence, Detail: detail})
}
