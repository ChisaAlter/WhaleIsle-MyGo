package frame

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func workflowMemberDialog(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	v := m.Store.WorkflowMemberSnapshot()
	if !v.Open {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	open := true
	_, height := c.Size()
	panelHeight := min(float32(700), max(float32(280), height-64))
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Width(940).MaxWidthPercent(94).Height(panelHeight).Padding(18).Background(chrome.Color(colors.BG)).Radius(14).Children(func() {
			ui.Column(c).Fill().Gap(8).Children(func() {
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, v.Label+" · Workflow").FontSize(15).Bold().TextColor(fg).Grow(1)
					if ui.Button(c, archiveText("关闭", "Close")).Clicked() {
						open = false
					}
				})
				ui.Text(c, v.ParentID+" → "+v.ChildID+" · one-shot").FontSize(10).TextColor(muted).Selectable()
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					status := archiveText("已暂停或结束跟进", "Observation paused or ended")
					if v.Busy {
						status = archiveText("正在读取…", "Loading…")
					} else if v.Process.Live {
						status = archiveText("实时过程", "Live process")
					}
					ui.Text(c, status).FontSize(11).TextColor(muted).Grow(1)
					if ui.Button(c, archiveText("较早记录", "Earlier records")).FontSize(11).Disabled(v.Busy || !v.Process.HasMore).Clicked() {
						m.Store.EarlierWorkflowMember(v.Token)
					}
					if ui.Button(c, archiveText("回到最新 / 重连", "Latest / reconnect")).FontSize(11).Disabled(v.Busy || !m.Store.WorkflowMemberCanReconnect(v.Token)).Clicked() {
						snap := m.Store.Snapshot()
						m.Store.OpenWorkflowMember(snap.PlanHistoryBinding, v.ParentID, v.RunID, v.MemberSeq, v.ChildID, v.Label)
					}
					if ui.Button(c, archiveText("复制记录", "Copy records")).FontSize(11).Disabled(len(v.Process.Records) == 0).Clicked() {
						mygo.Clipboard.WriteText(session.WorkflowProcessText(v))
					}
				})
				if v.Process.Error != "" {
					ui.Text(c, v.Process.Error).FontSize(12).TextColor(fg)
				}
				if v.Process.Truncated {
					ui.Text(c, archiveText("仅保留最近400条；较早记录可分页读取。", "Showing the latest 400 records; earlier history remains pageable.")).FontSize(11).TextColor(muted)
				}
				ui.Scroll(c.Key(fmt.Sprintf("workflow-member:%s:%s:%d", v.ParentID, v.ChildID, v.Token))).FillWidth().Grow(1).MinHeight(0).Children(func() {
					body := session.WorkflowProcessText(v)
					if body == "" {
						body = archiveText("尚无持久过程记录。", "No durable process records yet.")
					}
					ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
				})
			})
		})
	})
	if !open {
		m.Store.CloseWorkflowMember(v.Token)
	}
}
