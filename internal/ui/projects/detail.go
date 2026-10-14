package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
)

func detailView(c *ui.Context, m Model) {
	d := m.Detail
	if m.TeamNote != "" {
		ui.Text(c, m.TeamNote).FontSize(11)
	}
	if ui.Button(c, tr("返回 Team", "Back to Team")).Clicked() && m.OnCloseDetail != nil {
		m.OnCloseDetail()
	}
	ui.Text(c, d.Workstream.Title+" · "+stateLabel(d.Workstream.Status)).FontSize(13)
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("打开 Lead", "Open Lead")).Disabled(m.Busy).Clicked() && m.OnOpen != nil {
			m.OnOpen(m.TeamID)
		}
		if ui.Button(c, tr("查看成员过程", "View member history")).Disabled(m.Busy || d.Worker == nil || !d.Worker.Materialized).Clicked() && m.OnProcess != nil {
			m.OnProcess(false)
		}
	})
	if d.Worker != nil && d.Worker.Phase == "failed" {
		ui.Text(c, tr("重试由 Lead 在原工作和成员上执行。这里只准备指令，需你发送。", "Retry is performed by the Lead on the original work and member. This only prepares a request for you to send.")).FontSize(11)
		if ui.Button(c, tr("准备 Lead 重试草稿", "Prepare Lead retry draft")).Disabled(m.Busy).Clicked() && m.OnRetryDraft != nil {
			m.OnRetryDraft()
		}
	}
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, d.Workstream.Brief).FontSize(12)
			ui.Text(c, "workstreamId: "+d.Workstream.ID).FontSize(11)
			ui.Text(c, "childId: "+d.Workstream.WorkerSessionID).FontSize(11)
			if d.Worker != nil {
				ui.Text(c, d.Worker.CWD).FontSize(11)
				if d.Worker.Error != "" {
					ui.Text(c, d.Worker.Error).FontSize(12)
				}
				if !d.Worker.Materialized {
					ui.Text(c, tr("原成员尚未建立可读取会话，不会创建替代会话。", "The original member has no readable Session yet; no replacement is created.")).FontSize(11)
				}
			}
			for _, attempt := range d.Workstream.Delegations {
				ui.Text(c, attempt.Ref+" · "+attempt.Phase+" · "+attempt.RunID).FontSize(11)
			}
			if m.Process != nil {
				ui.Text(c, tr("成员过程快照（只读，最近最多30条消息）", "Member history snapshot (read-only, up to 30 recent messages)")).FontSize(11)
				for _, record := range m.Process.Records {
					if record.Type != "event" {
						ui.Text(c, record.Type).FontSize(11)
						continue
					}
					ui.Text(c, fmt.Sprintf("%g · %s", record.Event.Seq, record.Event.Type)).FontSize(11)
					var content any
					var body string
					if json.Unmarshal(record.Event.Data, &content) == nil {
						pretty, _ := json.MarshalIndent(content, "", "  ")
						body = string(pretty)
					} else {
						body = string(record.Event.Data)
					}
					runes := []rune(body)
					if len(runes) > 8000 {
						body = string(runes[:8000]) + tr("\n（此条显示已截断）", "\n(Display truncated for this record)")
					}
					ui.Text(c, body).FontSize(11)
				}
				if m.Process.HasMore && ui.Button(c, tr("更早的过程记录", "Earlier history")).Disabled(m.Busy).Clicked() && m.OnProcess != nil {
					m.OnProcess(true)
				}
			}
		})
	})
}
