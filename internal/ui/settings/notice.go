package settings

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// settingsNotice reports an acknowledgement ("saved", "opened") as a toast
// instead of a line in the page; see widgets.Notice. Failures do not come
// through here: an error is never a toast, so a failed write calls
// settingsAlert where its control was.
func settingsNotice(c *ui.Context, id string, show bool, text func() string) {
	if text == nil {
		widgets.Notice(c, "settings/"+id, show, nil)
		return
	}
	widgets.Notice(c, "settings/"+id, show, func() string { return normalizeSettingsVendorNotice(id, text()) })
}

// settingsFailure splits what a failed setting shows into the sentence a person
// reads and the detail they may open.
//
// sentence is the view's own words: the control that failed and the next step.
// text is whatever the store or the Host said about it. A known vendor message
// becomes the sentence (vendor_locales.go), because it names the cause better
// than any generic line can; anything else — a path, an exit code, an English
// err.Error(), or a message this build has no copy for — goes to detail and
// never stands as the sentence.
func settingsFailure(id, sentence, text string) (message, detail string) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return sentence, ""
	}
	if known := strings.TrimSpace(normalizeSettingsVendorNotice(id, raw)); known != "" && known != raw {
		return known, raw
	}
	return sentence, raw
}

// settingsAlert shows a failed setting where its control was: the sentence and
// the next step sit in the page, and the store's own words stay behind 详情.
func settingsAlert(c *ui.Context, id string, show bool, sentence, text string) {
	if !show {
		return
	}
	message, detail := settingsFailure(id, sentence, text)
	if message == "" {
		return
	}
	widgets.Alert(c, widgets.AlertOptions{ID: "settings/" + id, Message: message, Detail: detail})
}

// settingsFailureLine is settingsAlert for a failure with a line of its own: a
// row, a form footer or a dialog body. The sentence stays on one line and the
// store's own words open underneath it, so a row-level failure does not turn
// into a card in a list.
func settingsFailureLine(c *ui.Context, id string, sentence, text string) {
	message, detail := settingsFailure(id, sentence, text)
	if message == "" {
		return
	}
	t := c.Theme()
	box := ui.Column(c.Key("settings-failure/"+id)).FillWidth().MinWidth(0).Gap(4).Padding(6, 8).Radius(6).
		Border(0.5, t.Danger.Alpha(0.35)).Background(t.Danger.Alpha(0.05))
	box.Label(message)
	open := ui.Local(box, "detail", func() bool { return false })
	box.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Text(c, message).Grow(1).MinWidth(0).FontSize(12).LineHeight(18.0 / 12).TextColor(t.Text)
			if detail != "" {
				label := providerCopy("详情", "Details")
				if *open {
					label = providerCopy("收起", "Hide")
				}
				button := ui.Button(c.Key("settings-failure-detail/"+id), "").Label(label).Height(22).Padding(0, 6).BorderWidth(0).Background(ui.Transparent).TextColor(t.TextMuted).Shrink(0)
				button.Children(func() { ui.Text(c, label).FontSize(12) })
				if button.Clicked() {
					*open = !*open
				}
			}
		})
		if *open && detail != "" {
			ui.Text(c, detail).FillWidth().MinWidth(0).FontSize(12).LineHeight(18.0 / 12).TextColor(t.TextMuted).Selectable()
		}
	})
}

// settingsAuthoredFailure draws a failure whose text the producer already wrote
// for a person: an editor's validation sentence ("this name is taken"), or a
// sentence the store classified with errcopy. It is not the store's raw text, so
// it needs no 详情; a raw failure goes through settingsFailureLine so its words
// land behind the disclosure instead of in the sentence.
func settingsAuthoredFailure(c *ui.Context, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	t := c.Theme()
	ui.Text(c, text).FillWidth().MinWidth(0).FontSize(12).LineHeight(18.0 / 12).TextColor(t.Danger).Selectable()
}
