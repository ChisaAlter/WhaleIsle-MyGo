package chat

import (
	"context"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
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
		"hint":                        {"填写详情以帮助我们改进体验，提交内容会包括当前对话的日志", "Add details to help us improve. Your submission will include the current conversation log."},
		"task-result":                 {"任务结果", "Task result"},
		"instruction-following":       {"指令理解与遵循", "Instruction understanding and following"},
		"product-interaction":         {"产品功能与交互", "Product features and interaction"},
		"service-stability":           {"稳定性和速度", "Stability and speed"},
		"resource-cost":               {"资源使用与费用", "Resource usage and cost"},
		"security-privacy-permission": {"安全隐私与权限", "Security, privacy, and permissions"},
		"other":                       {"其他", "Other"},
		"recorded":                    {"感谢你的反馈", "Thanks for your feedback"},
		"conflict":                    {"这条反馈已在别处改动，已显示最新状态", "This feedback changed elsewhere; the latest state is shown"},
		"noteTooLarge":                {"描述太长，请缩短后再提交", "The description is too long; shorten it and submit again"},
		"error":                       {"反馈保存失败", "Could not save feedback"},
		"error.generic":               {"反馈保存失败", "Could not save feedback"},
		"error.conflict":              {"这条反馈已在别处改动，已显示最新状态", "This feedback changed elsewhere; the latest state is shown"},
		"error.noteTooLarge":          {"描述太长，请缩短后再提交", "The description is too long; shorten it and submit again"},
		"load":                        {"反馈状态加载失败", "Could not load feedback"},
		"error.load":                  {"反馈状态加载失败", "Could not load feedback"},
		"like":                        {"好的回答", "Good response"},
		"action.like":                 {"好的回答", "Good response"},
		"likeActive":                  {"取消标记", "Remove rating"},
		"action.likeActive":           {"取消标记", "Remove rating"},
		"dislike":                     {"有问题的回答", "Bad response"},
		"action.dislike":              {"有问题的回答", "Bad response"},
		"dislikeActive":               {"取消标记", "Remove rating"},
		"action.dislikeActive":        {"取消标记", "Remove rating"},
	}
	if i18n.Locale() == i18n.LocaleEN {
		return copy[key][1]
	}
	return copy[key][0]
}

type feedbackDialogNotice struct {
	failure         string
	acknowledgement uint64
	recorded        bool
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
		notice.acknowledgement = v.Acknowledgement
		notice.recorded = true
	}
	key := ""
	if notice.recorded {
		key = strconv.FormatUint(v.Acknowledgement, 10)
	}
	chatAckLineKeyed(c, "message-feedback-recorded", key, feedbackText("recorded"))
	if v.EditorID == "" {
		return
	}
	failure := feedbackFailureText(v.Error)
	feedbackDialogView(c, m.ThemeID, m.SystemDark, v.Note, v.Category, failure, v.BusyID != "" || v.Loading,
		m.FeedbackStore.EditFeedback, func() { m.FeedbackStore.SubmitFeedback(context.Background()) }, m.FeedbackStore.CancelFeedback)
}

// SessionFeedbackDialog shares the exact message-feedback presentation recipe;
// only its owner and record verb differ.
func SessionFeedbackDialog(c *ui.Context, store *session.Store, themeID string, systemDark bool) {
	v := store.SessionFeedback()
	if !v.Open || v.SessionID != store.SelectedSessionID() {
		return
	}
	feedbackDialogView(c.Key("session-feedback"), themeID, systemDark, v.Text, v.Category, feedbackFailureText(v.Error), v.Busy,
		store.EditSessionFeedback, func() { store.SubmitSessionFeedback(context.Background()) }, store.CancelSessionFeedback)
}

func feedbackDialogView(c *ui.Context, _ string, _ bool, note, category, failure string, busy bool, edit func(string, string), submit, dismiss func()) {
	t := design.For(c)
	mask := design.TW.Black.Alpha(0.32)
	fg, muted := t.Foreground, t.MutedForeground
	primary, onPrimary := t.Primary, t.PrimaryForeground
	open := true
	w, h := c.Size()
	ui.DialogBase(c.Key("message-feedback-modal"), &open, func(backdrop, panel ui.Element) {
		progress := float32(1)
		if !c.Preferences().ReduceMotion {
			progress = backdrop.AnimateWith("coss.dialog.mask", 1, design.DurationPanel, design.EaseInOut)
		}
		backdrop.Background(mask.Alpha(progress)).Padding(design.Sp(6))
		chatPopupSurface(c, panel.Label(feedbackText("title")).Width(max(0, min(float32(488), w-48))).MaxHeight(max(0, h-48)).MinHeight(0).Gap(design.Sp(9.5)).Padding(0, 0, design.Sp(6), 0).
			Clip().Opacity(progress), design.RadiusXL3+design.Sp(1)).BorderWidth(0).Children(func() {
			ui.Column(c).FillWidth().MinHeight(0).Shrink(1).Children(func() {
				ui.Row(c).FillWidth().Shrink(0).Padding(design.Sp(5.5), design.Sp(3.5), design.Sp(3), design.Sp(6)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, feedbackText("title")).FontSize(16).FixedLineHeight(24).FontWeight(500).TextColor(fg).Grow(1)
					close := ui.Button(c.Key("feedback-close"), "").Label(feedbackText("close")).Width(28).Height(28).Padding(0).Radius(design.RadiusMD).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Justify(ui.Center).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
					if close.Hovered() {
						close.Background(t.Accent)
					}
					close.Children(func() { ui.Icon(c, messageIconCloseOutline).Width(14).Height(14).PassThrough() })
					if close.Clicked() {
						open = false
					}
				})
				ui.Scroll(c).FillWidth().MinHeight(0).Shrink(1).Margin(design.Sp(5), 0, 0, 0).Padding(0, design.Sp(6)).Children(func() {
					ui.Row(c).Key("feedback-categories").Role(ui.RoleGroup).Label(feedbackText("categories")).FillWidth().Wrap().Gap(design.Sp(2)).Margin(-design.Sp(3.5), 0, 0, 0).Children(func() {
						for _, chipCategory := range feedbackCategories {
							chipCategory := chipCategory
							chip := ui.Button(c.Key("feedback-category-"+chipCategory), feedbackText(chipCategory)).Height(28).Padding(0, design.Sp(3)).FontSize(13).FixedLineHeight(18).Radius(design.RadiusMD).Border(.5, t.Border).Background(ui.Transparent).TextColor(fg).Checked(chipCategory == category).Disabled(busy)
							if !busy {
								chip.Cursor(ui.CursorPointer)
							}
							if chipCategory == category {
								chip.Border(.5, primary).Background(primary).TextColor(onPrimary)
							} else if chip.Hovered() {
								chip.Background(t.Accent)
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
					ui.TextAreaBase(c.Key("feedback-detail"), &draft).Label(feedbackText("detail")).Placeholder(feedbackText("hint")).FillWidth().Lines(3, 12).MinHeight(116).MaxHeight(280).Margin(design.Sp(4.5), 0, 0, 0).Padding(design.Sp(3), design.Sp(3.5)).
						FontSize(14).FixedLineHeight(22).Radius(design.RadiusXL2).Border(.5, t.Input).Background(t.Muted).TextColor(fg).ReadOnly(busy).OnChange(func() { edit(draft, category) })
				})
			})
			if failure != "" {
				ui.Row(c).FillWidth().Shrink(0).Padding(0, design.Sp(6), design.Sp(3), design.Sp(6)).Children(func() {
					coss.Alert(c, failure, "", coss.AlertOptions{Variant: coss.AlertError, Key: "feedback-failure"})
				})
			}
			ui.Row(c).FillWidth().Shrink(0).Padding(0, design.Sp(6)).Children(func() {
				label := feedbackText("submit")
				if busy {
					label = feedbackText("submitting")
				}
				submitBtn := ui.Button(c.Key("feedback-submit"), label).Label(label).FillWidth().Height(44).Radius(design.RadiusXL2).FontSize(14).FontWeight(500).BorderWidth(0).Background(primary).TextColor(onPrimary).Disabled(busy)
				if !busy {
					submitBtn.Cursor(ui.CursorPointer)
				}
				if submitBtn.Clicked() {
					submit()
				}
			})
		})
	})
	if !open {
		dismiss()
	}
}

func feedbackStatusErrorText(v session.FeedbackView) string {
	if !v.Ready && v.Error != "" {
		return feedbackText("load")
	}
	return feedbackFailureText(v.Error)
}

// feedbackFailureText is the sentence a refused feedback write shows in its own
// dialog, where the submit button sits.
func feedbackFailureText(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "version-conflict") {
		return feedbackText("conflict")
	}
	if strings.Contains(raw, "note-too-large") {
		return feedbackText("noteTooLarge")
	}
	return feedbackText("error")
}

func unusedFeedbackDialogTail(dismiss func()) {
	if false {
		dismiss()
	}
}
