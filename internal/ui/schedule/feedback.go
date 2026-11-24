package schedule

import (
	"strings"
	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

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

func (p *Panel) detailFeedback(c *ui.Context, m Model, d *taskDraft, blocked bool, gutter float32) {
	p.mu.Lock()
	if p.statusLocal && p.status != "" {
		if p.statusDraft == "" {
			p.statusDraft = d.draftKey()
		} else if p.statusDraft != d.draftKey() {
			_, err := d.validatedChange()
			if err == nil {
				p.status, p.actionFailed = "", false
			} else {
				p.status = errcopy.Sentence(err)
			}
			p.statusDraft = d.draftKey()
		}
	}
	message, owner, dismissed, pending := p.status, p.statusOwner, p.statusDismissed, p.actionPending
	p.mu.Unlock()
	if message == "" || dismissed || owner != "" && owner != taskKey(d.row) {
		return
	}
	ui.Column(c.Key("schedule-draft-failure")).FillWidth().MinWidth(0).Padding(design.Sp(3), gutter, 0, gutter).Children(func() {
		coss.Alert(c, "", message, coss.AlertOptions{
			Variant: coss.AlertError,
			Action: func() {
				if d.dirty() && d.row.Status != "inactive" {
					if coss.Button(c, scheduleText("保存修改", "Save changes"), coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Size:     coss.ButtonSM,
						Disabled: blocked,
					}).Clicked() {
						p.saveDraft(m, d)
					}
					if coss.Button(c, scheduleText("取消", "Cancel"), coss.ButtonOptions{
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonSM,
						Disabled: pending,
					}).Clicked() {
						*d = *newTaskDraft(d.row)
						p.message("")
					}
				}
				if coss.Button(c, "", coss.ButtonOptions{
					Key:     "notice-close",
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconX,
					Label:   scheduleText("关闭提示", "Dismiss"),
				}).Clicked() {
					p.mu.Lock()
					p.statusDismissed = true
					p.mu.Unlock()
				}
			},
		})
	})
}
