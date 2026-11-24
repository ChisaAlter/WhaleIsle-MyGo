package settings

import (
	"strings"

	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/widgets"
)

// settingsNotice reports an acknowledgement ("saved", "opened") as a muted
// status line where the action happened; see widgets.AckLine. Failures do not
// come through here: an error is never a toast, so a failed write calls
// settingsAlert where its control was.
func settingsNotice(c *ui.Context, id string, show bool, text func() string) {
	message := ""
	if show && text != nil {
		message = normalizeSettingsVendorNotice(id, text())
	}
	if widgets.AckLive(c, "settings/"+id, message, widgets.StatusDuration) {
		t := tk(c)
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center).Role(ui.RoleStatus).Label(message).Children(func() {
			ui.Icon(c, coss.IconCircleCheck.SVG()).Size(design.IconXS, design.IconXS).Shrink(0).TextColor(t.Success).PassThrough()
			kText(c, message, design.TextXS, design.WeightNormal, t.MutedForeground).Grow(1).MinWidth(0)
		})
	}
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
	settingsErrorAlert(c, "settings/"+id, message, detail)
}

// settingsErrorAlert is coss's error Alert carrying a failure sentence, with
// the store's own words behind a 详情 disclosure (a diagnostics detail, never
// the sentence).
func settingsErrorAlert(c *ui.Context, id, message, detail string) {
	anchor := ui.Column(c.Key("settings-alert/" + id)).FillWidth().MinWidth(0).Gap(design.Sp(1.5))
	open := ui.Local(anchor, "detail", func() bool { return false })
	anchor.Children(func() {
		var action func()
		if detail != "" {
			action = func() {
				label := providerCopy("详情", "Details")
				if *open {
					label = providerCopy("收起", "Hide")
				}
				if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Key: "settings-failure-detail/" + id}).Clicked() {
					*open = !*open
				}
			}
		}
		coss.Alert(c, "", message, coss.AlertOptions{Variant: coss.AlertError, Action: action}).Label(message)
		if *open && detail != "" {
			kText(c, detail, design.TextXS, design.WeightNormal, tk(c).MutedForeground).FillWidth().MinWidth(0).Selectable() // verbatim-diagnostics
		}
	})
}

// settingsRefusal is one refused write as a surface keeps it: the sentence a
// person reads, and the store's own words for the 详情 disclosure. A surface
// that keeps only the sentence throws the cause away, which is why every failed
// write goes through refuse().
type settingsRefusal struct {
	sentence string
	detail   string
}

// refuse records err: errcopy names the cause in the interface language, and
// err.Error() stays behind 详情.
func (r *settingsRefusal) refuse(err error) {
	if err == nil {
		*r = settingsRefusal{}
		return
	}
	r.sentence, r.detail = errcopy.Sentence(err), err.Error()
}

// say records a sentence the view wrote itself, with nothing to disclose.
func (r *settingsRefusal) say(sentence string) {
	r.sentence, r.detail = sentence, ""
}

func (r settingsRefusal) failed() bool { return r.sentence != "" }

// settingsFailureLine is settingsAlert for a failure with a line of its own: a
// row, a form footer or a dialog body. The sentence stays on one line and the
// store's own words open underneath it, so a row-level failure does not turn
// into a card in a list — and fits a reserved single-line slot.
func settingsFailureLine(c *ui.Context, id string, sentence, text string) {
	message, detail := settingsFailure(id, sentence, text)
	if message == "" {
		return
	}
	t := tk(c)
	box := ui.Column(c.Key("settings-failure/"+id)).FillWidth().MinWidth(0).Gap(design.Sp(1)).Padding(design.Sp(1), design.Sp(2)).Radius(design.RadiusLG).
		Border(design.BorderWidth, t.Destructive.Alpha(0.32)).Background(t.Destructive.Alpha(0.04))
	box.Label(message)
	open := ui.Local(box, "detail", func() bool { return false })
	box.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			kText(c, message, design.TextXS, design.WeightNormal, t.DestructiveForeground).Grow(1).MinWidth(0)
			if detail != "" {
				label := providerCopy("详情", "Details")
				if *open {
					label = providerCopy("收起", "Hide")
				}
				if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Key: "settings-failure-detail/" + id}).Clicked() {
					*open = !*open
				}
			}
		})
		if *open && detail != "" {
			kText(c, detail, design.TextXS, design.WeightNormal, t.MutedForeground).FillWidth().MinWidth(0).Selectable() // verbatim-diagnostics
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
	kError(c, text).MinWidth(0).Selectable()
}
