package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// PreviewNoticeVersion is the exact preview-notice copy version a profile has to
// acknowledge before the notice stops appearing (native
// ui-settings-models/src/onboarding-copy.ts WELCOME_NOTICE_VERSION). The
// acknowledgement is compared for equality, so a new notice reuses the same
// stored field with a new version string.
const PreviewNoticeVersion = "2026-09-28.1"

// Notice geometry, taken from the installed OnboardingModal/WelcomeNotice rules
// and from the rendered dialog at 1440x920:
//
//	dialog  600x344 at (420,288), radius 28
//	content padding 28
//	title   20/28 weight 500
//	body    margin-top 20, 12 between paragraphs, copy 14/24 secondary
//	action  margin-top 24, button 120 min / 36 tall, radius 12, 14/22
const (
	PreviewNoticeWidth        = 600
	PreviewNoticeRadius       = 28
	PreviewNoticePadding      = 28
	PreviewNoticeTitleSize    = 20
	PreviewNoticeTitleLine    = 28
	PreviewNoticeBodyGap      = 20
	PreviewNoticeParagraphGap = 12
	PreviewNoticeCopySize     = 14
	PreviewNoticeCopyLine     = 24
	PreviewNoticeActionGap    = 24
	PreviewNoticeButtonMin    = 120
)

// PreviewNotice is the product-wide first-run notice ("预览版说明") the installed
// harness shows once per copy version until the user presses 继续. MyGo paints
// it as a blocking layer above the whole window, so the shell behind it stays
// inert while it is open.
type PreviewNotice struct {
	Open bool
	// Saving disables the button while the acknowledgement is written.
	Saving bool
	// Failed shows the native "could not save" line under the copy.
	Failed bool
	Colors theme.Colors
	// OnContinue acknowledges this copy version and closes the layer.
	OnContinue func()
	// Observe receives the painted panel box; the headless geometry test
	// measures the reference metrics through it. Production callers leave it nil.
	Observe func(panel ui.Rect)
}

// PreviewNoticeRequested reports whether a profile still has to acknowledge the
// current notice. An empty or older value shows it again, exactly like the
// native equality check.
func PreviewNoticeRequested(acknowledged string) bool {
	return strings.TrimSpace(acknowledged) != PreviewNoticeVersion
}

// PreviewNoticeParagraphs splits the notice copy on its blank line, as the
// installed WelcomeNotice renders one <p> per paragraph.
func PreviewNoticeParagraphs(body string) []string {
	var out []string
	for _, paragraph := range strings.Split(body, "\n\n") {
		if trimmed := strings.TrimSpace(paragraph); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		if trimmed := strings.TrimSpace(body); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// PreviewNoticeView paints the notice layer. Call it last in the window's view
// so it covers chat, sidebar and dialogs.
func PreviewNoticeView(c *ui.Context, m PreviewNotice) {
	if !m.Open {
		return
	}
	t := design.For(c)
	ink, muted := t.Foreground, t.MutedForeground
	open := true
	ui.DialogBase(c.Key("preview-notice"), &open, func(backdrop, panel ui.Element) {
		progress := float32(1)
		if !c.Preferences().ReduceMotion {
			progress = backdrop.AnimateWith("coss.dialog.mask", 1, design.DurationPanel, design.EaseInOut)
		}
		backdrop.Background(design.TW.Black.Alpha(0.32 * progress))
		chatPopupSurface(c, panel.Width(PreviewNoticeWidth).MaxWidthPercent(100).MaxHeightPercent(100).
			Clip().Opacity(progress), PreviewNoticeRadius).BorderWidth(0).
			Children(func() {
				ui.Column(c).FillWidth().Padding(PreviewNoticePadding).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyChatPreviewNoticeTitle)).
						FillWidth().
						FontSize(theme.InterfaceSize(PreviewNoticeTitleSize)).
						FixedLineHeight(theme.InterfaceSize(PreviewNoticeTitleLine)).
						FontWeight(500).TextColor(ink)
					ui.Column(c).FillWidth().Margin(PreviewNoticeBodyGap, 0, 0, 0).Gap(PreviewNoticeParagraphGap).Children(func() {
						for _, paragraph := range PreviewNoticeParagraphs(i18n.T(i18n.KeyChatPreviewNoticeBody)) {
							ui.Text(c, paragraph).FillWidth().
								FontSize(theme.InterfaceSize(PreviewNoticeCopySize)).
								FixedLineHeight(theme.InterfaceSize(PreviewNoticeCopyLine)).
								TextColor(muted)
						}
						if m.Failed {
							ui.Text(c, i18n.T(i18n.KeyChatPreviewNoticeError)).FillWidth().
								Margin(PreviewNoticeParagraphGap, 0, 0, 0).
								FontSize(theme.InterfaceSize(PreviewNoticeCopySize)).
								FixedLineHeight(theme.InterfaceSize(22)).
								TextColor(t.DestructiveForeground)
						}
					})
					ui.Row(c).FillWidth().Justify(ui.End).Margin(PreviewNoticeActionGap, 0, 0, 0).Children(func() {
						continueLabel := i18n.T(i18n.KeyChatPreviewNoticeContinue)
						button := coss.Button(c, continueLabel, coss.ButtonOptions{
							Variant:  coss.ButtonDefault,
							Size:     coss.ButtonLG,
							Disabled: m.Saving,
						}).Label(continueLabel).MinWidth(PreviewNoticeButtonMin)
						if button.Clicked() && m.OnContinue != nil {
							m.OnContinue()
						}
					})
				})
			})
		if m.Observe != nil {
			panel.DrawOver(func(_ *ui.Painter, r ui.Rect) { m.Observe(r) })
		}
	})
}
