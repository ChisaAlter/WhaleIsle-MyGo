package widgets

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
)

// AlertOptions describes an Alert.
type AlertOptions struct {
	// ID keys the alert's own state, so a collapsed detail stays collapsed
	// while the same failure is shown again.
	ID string
	// Message is what failed, in the product's words, and what to do next. An
	// empty Message draws nothing.
	Message string
	// Detail is the failure's own text, behind 详情 and never in the sentence.
	Detail string
	// Action labels a button that tries the failed step again; no button shows
	// without one. The caller owns the retry, and the alert clears when the
	// failure does.
	Action   string
	OnAction func()
	// Dismiss labels a button that clears the failure the caller already
	// reported; no button shows without one.
	Dismiss   string
	OnDismiss func()
}

// Alert shows a failure where it happened: a row in the page that carries the
// reason and the next step, not a toast over the content. The caller renders it
// in the row, panel or dialog whose action failed, and clears it by passing an
// empty Message once the failure is gone.
//
// The raw failure text never stands in as the sentence: Detail holds it behind
// a 详情 disclosure, and the log keeps it as well.
func Alert(c *ui.Context, o AlertOptions) {
	if o.Message == "" {
		return
	}
	t := c.Theme()
	box := ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(10, 12).Radius(8).
		Border(0.5, t.Danger.Alpha(0.4)).Background(t.Danger.Alpha(0.07))
	if o.ID != "" {
		box.Key("alert/" + o.ID)
	}
	box.Label(o.Message)
	open := ui.Local(box, "detail", func() bool { return false })
	box.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
			ui.Text(c, o.Message).Grow(1).MinWidth(0).FontSize(13).LineHeight(20.0 / 13).TextColor(t.Text)
			if o.Detail != "" {
				label := i18n.Pick("详情", "Details")
				if *open {
					label = i18n.Pick("收起", "Hide")
				}
				alertButton(c, label, t.TextMuted, func() { *open = !*open })
			}
			if o.Action != "" && o.OnAction != nil {
				alertButton(c, o.Action, t.Text, o.OnAction)
			}
			if o.Dismiss != "" && o.OnDismiss != nil {
				alertButton(c, o.Dismiss, t.TextMuted, o.OnDismiss)
			}
		})
		if *open && o.Detail != "" {
			ui.Text(c, o.Detail).FillWidth().MinWidth(0).FontSize(12).LineHeight(18.0 / 12).
				TextColor(t.TextMuted).Selectable()
		}
	})
}

// alertButton is the flat control inside an alert: the disclosure and the retry.
func alertButton(c *ui.Context, label string, ink ui.Color, run func()) {
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonXS,
		Label:   label,
		Content: func() { ui.Text(c, label).FontSize(12).TextColor(ink) },
	}).Shrink(0)
	if button.Clicked() {
		run()
	}
}
