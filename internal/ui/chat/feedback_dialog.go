package chat

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var feedbackCategories = []string{"task-result", "instruction-following", "product-interaction", "service-stability", "resource-cost", "security-privacy-permission", "other"}

// The copy and presentation order come from ui-message-feedback/locales.ts.
func feedbackText(key string) string {
	copy := map[string][2]string{
		"title":                       {"提交反馈", "Submit feedback"},
		"close":                       {"关闭", "Close"},
		"submit":                      {"提交", "Submit"},
		"submitting":                  {"提交中…", "Submitting…"},
		"categories":                  {"反馈分类", "Feedback category"},
		"detail":                      {"反馈详情", "Feedback details"},
		"hint":                        {"填写详情可以帮助我们改进体验，提交内容会包含当前对话的日志", "Add details to help us improve. Your submission will include the current conversation log."},
		"task-result":                 {"任务结果", "Task result"},
		"instruction-following":       {"指令理解与遵循", "Instruction understanding and following"},
		"product-interaction":         {"产品功能与交互", "Product features and interaction"},
		"service-stability":           {"稳定性和速度", "Stability and speed"},
		"resource-cost":               {"资源使用与成本", "Resource usage and cost"},
		"security-privacy-permission": {"安全隐私与权限", "Security, privacy, and permissions"},
		"other":                       {"其他", "Other"},
		"recorded":                    {"感谢你的反馈", "Thanks for your feedback"},
		"conflict":                    {"这条反馈已在别处改动，已显示最新状态", "This feedback changed elsewhere; the latest state is shown"},
		"noteTooLarge":                {"描述太长，请缩短后再提交", "The description is too long; shorten it and submit again"},
		"error":                       {"反馈提交失败", "Could not save feedback"},
	}
	if i18n.Locale() == i18n.LocaleEN {
		return copy[key][1]
	}
	return copy[key][0]
}

type feedbackDialogNotice struct {
	failure         string
	acknowledgement uint64
}

type feedbackDialogKey struct {
	host    uint64
	session string
}

// FeedbackDialog.tsx wraps Modal's shared header/body/footer tree. The native
// DialogBase supplies the same input isolation, focus trap/return, Escape and
// mask dismissal. FeedbackDialog.module.css overrides the panel to 488 / 28,
// the content/footer gap to 38, chips to 28, and the submit button to 44.
func messageFeedbackDialog(c *ui.Context, m Model) {
	v := m.FeedbackStore.Feedback()
	seat := ui.Box(c.Key(feedbackDialogKey{m.Snapshot.ComposerHostGeneration, m.Snapshot.SelectedID})).Absolute().Width(0).Height(0)
	// A retired overlay must not replay an acknowledgement when remounted.
	notice := ui.Local(seat, "notice", func() feedbackDialogNotice { return feedbackDialogNotice{acknowledgement: v.Acknowledgement} })
	if v.Acknowledgement > notice.acknowledgement {
		c.AddToast(ui.Toast{ID: "message-feedback-recorded", Title: feedbackText("recorded"), Type: "success", Timeout: 3 * time.Second})
		notice.acknowledgement = v.Acknowledgement
	}
	if v.Error == "" && notice.failure != "" {
		c.CloseToast("message-feedback-failure")
		notice.failure = ""
	}
	if v.EditorID == "" {
		if notice.failure != "" {
			c.CloseToast("message-feedback-failure")
		}
		notice.failure = ""
		return
	}
	if v.Error != "" && v.Error != notice.failure {
		text := feedbackText("error")
		if strings.Contains(v.Error, "version-conflict") {
			text = feedbackText("conflict")
		}
		if strings.Contains(v.Error, "note-too-large") {
			text = feedbackText("noteTooLarge")
		}
		c.AddToast(ui.Toast{ID: "message-feedback-failure", Title: text, Type: "error", Timeout: 6 * time.Second})
		notice.failure = v.Error
	} else if v.Error == "" {
		notice.failure = ""
	}
	feedbackDialogView(c, m.ThemeID, m.SystemDark, v.Note, v.Category, v.BusyID != "" || v.Loading,
		m.FeedbackStore.EditFeedback, func() { m.FeedbackStore.SubmitFeedback(context.Background()) }, m.FeedbackStore.CancelFeedback)
}

// SessionFeedbackDialog shares the exact message-feedback presentation recipe;
// only its owner and record verb differ.
func SessionFeedbackDialog(c *ui.Context, store *session.Store, themeID string, systemDark bool) {
	v := store.SessionFeedback()
	if !v.Open || v.SessionID != store.SelectedSessionID() {
		return
	}
	seat := ui.Box(c.Key("session-feedback:"+v.SessionID)).Absolute().Size(0, 0)
	failure := ui.Local(seat, "error", func() string { return "" })
	if v.Error != "" && *failure != v.Error {
		c.AddToast(ui.Toast{ID: "session-feedback-failure", Title: feedbackText("error"), Type: "error", Timeout: 6 * time.Second})
	}
	*failure = v.Error
	feedbackDialogView(c.Key("session-feedback"), themeID, systemDark, v.Text, v.Category, v.Busy,
		store.EditSessionFeedback, func() { store.SubmitSessionFeedback(context.Background()) }, store.CancelSessionFeedback)
}

func feedbackDialogView(c *ui.Context, themeID string, systemDark bool, note, category string, busy bool, edit func(string, string), submit, dismiss func()) {
	colors := theme.Resolve(themeID).Active(systemDark)
	resolved := theme.Resolve(themeID)
	dark := resolved.Scheme == theme.SchemeDark || (resolved.Scheme == theme.SchemeSystem && systemDark)
	mask, stroke, layer := ui.RGBA(0, 0, 0, .24), ui.RGBA(0, 0, 0, .16), ui.RGBA(255, 255, 255, .8)
	if dark {
		mask = ui.RGBA(0, 0, 0, .5)
		stroke = ui.RGBA(255, 255, 255, .2)
		layer = ui.RGBA(44, 44, 46, .8)
	}
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	primary, onPrimary := chrome.Color(colors.Accent), chrome.Color(colors.ButtonFG)
	open := true
	w, h := c.Size()
	ui.DialogBase(c.Key("message-feedback-modal"), &open, func(backdrop, panel ui.Element) {
		progress := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
		chrome.RevealMask(backdrop, mask, progress)
		backdrop.Padding(24)
		panel.Label(feedbackText("title")).Width(max(0, min(float32(488), w-48))).MaxHeight(max(0, h-48)).MinHeight(0).Gap(38).Padding(0, 0, 24, 0).
			Radius(28).Clip().Opacity(progress).Draw(chrome.Glass(layer, chrome.MaterialPanel, 28)).
			Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			ui.Column(c).FillWidth().MinHeight(0).Shrink(1).Children(func() {
				ui.Row(c).FillWidth().Shrink(0).Padding(22, 14, 12, 24).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, feedbackText("title")).FontSize(16).FixedLineHeight(24).FontWeight(500).TextColor(fg).Grow(1)
					close := ui.Button(c.Key("feedback-close"), "").Label(feedbackText("close")).Width(28).Height(28).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Justify(ui.Center).AlignItems(ui.Center).FocusRing(false)
					if close.Hovered() {
						close.Background(chrome.Color(colors.Hover))
					}
					close.Children(func() { ui.Icon(c, messageIconCloseOutline).Width(14).Height(14).PassThrough() })
					if close.Clicked() {
						open = false
					}
				})
				ui.Scroll(c).FillWidth().MinHeight(0).Shrink(1).Margin(20, 0, 0, 0).Padding(0, 24).Children(func() {
					ui.Row(c).Key("feedback-categories").Role(ui.RoleGroup).Label(feedbackText("categories")).FillWidth().Wrap().Gap(8).Margin(-14, 0, 0, 0).Children(func() {
						for _, chipCategory := range feedbackCategories {
							chipCategory := chipCategory
							chip := ui.Button(c.Key("feedback-category-"+chipCategory), feedbackText(chipCategory)).Height(28).Padding(0, 12).FontSize(13).FixedLineHeight(18).Radius(8).Border(.5, stroke).Background(ui.Transparent).TextColor(fg).Checked(chipCategory == category).Disabled(busy)
							if chipCategory == category {
								chip.Border(.5, primary).Background(primary).TextColor(onPrimary)
							} else if chip.Hovered() {
								chip.Background(chrome.Color(colors.Hover))
							}
							if chip.Clicked() {
								next := chipCategory
								if chipCategory == category {
									next = ""
								}
								edit(note, next)
							}
						}
					})
					draft := note
					ui.TextAreaBase(c.Key("feedback-detail"), &draft).Label(feedbackText("detail")).Placeholder(feedbackText("hint")).FillWidth().Lines(3, 12).MinHeight(116).MaxHeight(280).Margin(18, 0, 0, 0).Padding(12, 14).
						FontSize(14).FixedLineHeight(22).Radius(16).Border(.5, stroke).Background(chrome.Color(colors.Field)).TextColor(fg).ReadOnly(busy).OnChange(func() { edit(draft, category) })
				})
			})
			ui.Row(c).FillWidth().Shrink(0).Padding(0, 24).Children(func() {
				label := feedbackText("submit")
				if busy {
					label = feedbackText("submitting")
				}
				if ui.Button(c.Key("feedback-submit"), label).Label(label).FillWidth().Height(44).Radius(16).FontSize(14).FontWeight(500).BorderWidth(0).Background(primary).TextColor(onPrimary).Disabled(busy).Clicked() {
					submit()
				}
			})
		})
	})
	if !open {
		dismiss()
	}
}
