package widgets

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// AckDuration is how long a confirmation stays on its control or status seat
// before reverting — the original 1.5s a success toast lived.
const AckDuration = 1500 * time.Millisecond

// StatusDuration is how long an outcome that has no single control to carry it
// stays in its surface's status line before it is treated as stale.
const StatusDuration = 6 * time.Second

// ackSeen is the Local record behind AckLive: the last outcome text the seat
// armed on and when that arming expires.
type ackSeen struct {
	text  string
	until time.Time
}

// AckLive reports whether text names an outcome that landed within the last d.
// It arms when text changes to a new non-empty value and schedules the repaint
// that ends the window, so a control or line fed by AckLive reverts on its own
// even though the model flag stays set — the same one-shot behaviour a toast
// had. The seat is the invisible zero-size element the acknowledgement is
// anchored to; id makes it unique inside its parent.
func AckLive(c *ui.Context, id, text string, d time.Duration) bool {
	seat := ui.Box(c.Key("ack/"+id)).Absolute().Size(0, 0).PassThrough()
	st := ui.Local(seat, "state", func() ackSeen { return ackSeen{} })
	now := time.Now()
	if st.text != text {
		st.text = text
		if text != "" {
			st.until = now.Add(d)
		}
	}
	if text == "" || !now.Before(st.until) {
		return false
	}
	c.After(st.until.Sub(now))
	return true
}

// AckLine draws an outcome ("已保存"…) as a muted status line for
// StatusDuration after text appears. It is the no-floating-layer replacement
// for a success toast where the action has no single button to carry the
// confirmation: the surface keeps the acknowledgement in the same place it
// already reports failures, and the line reverts by itself.
func AckLine(c *ui.Context, id, text string) {
	if AckLive(c, id, text, StatusDuration) {
		ui.Text(c, text).FillWidth().FontSize(12).TextColor(c.Theme().TextMuted)
	}
}

// AckLineKeyed is AckLine where the outcome's identity differs from what the
// person reads: a revision, counter, or token decides freshness so a repeated
// "已保存" still re-arms while the label stays fixed.
func AckLineKeyed(c *ui.Context, id, key, label string) {
	if AckLive(c, id, key, StatusDuration) {
		ui.Text(c, label).FillWidth().FontSize(12).TextColor(c.Theme().TextMuted)
	}
}

// StatusLine draws a live condition ("处理中 2 / 5", a read-only host) as a
// muted status line for as long as text is non-empty. One-shot outcomes use
// AckLine instead so a stale notice cannot sit in the page forever.
func StatusLine(c *ui.Context, text string) {
	if text == "" {
		return
	}
	ui.Text(c, text).FillWidth().FontSize(12).TextColor(c.Theme().TextMuted)
}
