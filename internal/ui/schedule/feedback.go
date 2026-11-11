package schedule

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type scheduleNoticeAction struct {
	Label    string
	Disabled bool
	Run      func()
}

// Notices are owned by their visible source consumer, not by a permanent row in
// the draft. Overlay preserves the panel/composer geometry and retires on unmount.
func scheduleNotice(c *ui.Context, key, message string, colors theme.Colors, bottom float32, actions []scheduleNoticeAction, dismiss func()) {
	width, height := c.Size()
	size := max(float32(0), min(float32(520), width-32))
	ui.Overlay(c.Key(key), func() {
		panel := ui.Column(c.Key("notice")).Absolute().Left((width-size)/2).Bottom(bottom).Width(size).MaxHeight(max(float32(0), height-bottom-24)).Padding(12, 16).Gap(8).Role(ui.RoleStatus).Label(message).Border(.5, chrome.Color(colors.Border)).Radius(12)
		// The notice's blank space belongs to it, not to a composer underneath.
		_ = panel.Clicked()
		panel.Cursor(ui.CursorDefault)
		chrome.GlassSurface(panel, chrome.Color(colors.Field), chrome.MaterialPanel, 12)
		panel.Shadow(0, 6, 24, 0, ui.RGBA(0, 0, 0, .18)).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Text(c, message).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG))
				if dismiss != nil && scheduleIconButton(c, "notice-close", scheduleText("关闭提示", "Dismiss"), "close", chrome.Color(colors.Muted)).Clicked() {
					dismiss()
				}
			})
			if len(actions) != 0 {
				ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
					for index, action := range actions {
						action := action
						if widgets.Button(c.Key(fmt.Sprintf("notice-action-%d", index)), colors, action.Label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: action.Disabled}).Clicked() && action.Run != nil {
							action.Run()
						}
					}
				})
			}
		})
	})
}

func (d *taskDraft) draftKey() string {
	return strings.Join([]string{d.Title, d.Prompt, d.timingKey()}, "\x00")
}

func scheduleSessionNotice(m Model, id string) string {
	if m.SessionLinkState == nil {
		return ""
	}
	switch m.SessionLinkState(id) {
	case "loading":
		return scheduleText("正在加载原会话信息", "Loading original session information.")
	case "archived":
		return scheduleText("原会话已归档", "The original session is archived.")
	case "unavailable":
		return scheduleText("原会话当前不可用", "The original session is unavailable.")
	}
	return ""
}

func (p *Panel) detailFeedback(c *ui.Context, m Model, d *taskDraft, colors theme.Colors, blocked bool) {
	p.mu.Lock()
	if p.statusLocal && p.status != "" {
		if p.statusDraft == "" {
			p.statusDraft = d.draftKey()
		} else if p.statusDraft != d.draftKey() {
			_, err := d.validatedChange()
			if err == nil {
				p.status, p.actionFailed = "", false
			} else {
				p.status = err.Error()
			}
			p.statusDraft = d.draftKey()
		}
	}
	message, owner, dismissed, pending := p.status, p.statusOwner, p.statusDismissed, p.actionPending
	p.mu.Unlock()
	if message == "" || dismissed || owner != "" && owner != taskKey(d.row) {
		return
	}
	actions := []scheduleNoticeAction{}
	if d.dirty() && d.row.Status != "inactive" {
		actions = []scheduleNoticeAction{
			{Label: scheduleText("保存修改", "Save changes"), Disabled: blocked, Run: func() { p.saveDraft(m, d) }},
			{Label: scheduleText("取消", "Cancel"), Disabled: pending, Run: func() { *d = *newTaskDraft(d.row); p.message("") }},
		}
	}
	bottom := float32(24)
	if d.dirty() {
		bottom = 88
	}
	scheduleNotice(c, "schedule-draft-failure", message, colors, bottom, actions, func() {
		p.mu.Lock()
		p.statusDismissed = true
		p.mu.Unlock()
	})
}
