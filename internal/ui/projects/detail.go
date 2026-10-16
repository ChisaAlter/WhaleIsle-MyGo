package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func detailView(c *ui.Context, m Model) {
	d := m.Detail
	if m.TeamNote != "" {
		ui.Text(c, m.TeamNote).FontSize(fonttheme.InterfaceSize(11))
	}
	if ui.Button(c, tr("返回 Team", "Back to Team")).Clicked() && m.OnCloseDetail != nil {
		m.OnCloseDetail()
	}
	ui.Text(c, d.Workstream.Title+" · "+stateLabel(d.Workstream.Status)).FontSize(fonttheme.InterfaceSize(13))
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("打开 Lead", "Open Lead")).Disabled(m.Busy).Clicked() && m.OnOpen != nil {
			m.OnOpen(m.TeamID)
		}
		if ui.Button(c, tr("实时跟进成员过程", "Follow member live")).Disabled(m.Busy || m.ProcessWatching || d.Worker == nil || !d.Worker.Materialized).Clicked() && m.OnProcess != nil {
			m.OnProcess(false)
		}
		if m.ProcessWatching && ui.Button(c, tr("停止跟进", "Stop following")).Clicked() && m.OnStopProcess != nil {
			m.OnStopProcess()
		}
	})
	if d.Worker != nil && d.Worker.Phase == "failed" {
		ui.Text(c, tr("重试由 Lead 在原工作和成员上执行。这里只准备指令，需你发送。", "Retry is performed by the Lead on the original work and member. This only prepares a request for you to send.")).FontSize(fonttheme.InterfaceSize(11))
		if ui.Button(c, tr("准备 Lead 重试草稿", "Prepare Lead retry draft")).Disabled(m.Busy).Clicked() && m.OnRetryDraft != nil {
			m.OnRetryDraft()
		}
	}
	if d.Workstream.SummaryFailure != nil {
		ui.Text(c, tr("成果已保存，Lead 汇总未完成。点击后复核原记录并向原 Lead 发送重新汇总请求。", "Results are saved. This action verifies the records and sends a summary request to the original Lead.")).FontSize(fonttheme.InterfaceSize(11))
		ui.Text(c, d.Workstream.SummaryFailure.Error.Message).FontSize(fonttheme.InterfaceSize(11))
		disabled := m.Busy || !m.Listing.Available || m.Team == nil || m.Team.Project.Lifecycle != "ready" || m.Team.Project.Archiving
		if ui.Button(c, tr("发送重新汇总请求", "Send summary recovery request")).Disabled(disabled).Clicked() && m.OnSummaryRetryDraft != nil {
			m.OnSummaryRetryDraft()
		}
	}
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, d.Workstream.Brief).FontSize(fonttheme.InterfaceSize(12))
			ui.Text(c, "workstreamId: "+d.Workstream.ID).FontSize(fonttheme.InterfaceSize(11))
			ui.Text(c, "childId: "+d.Workstream.WorkerSessionID).FontSize(fonttheme.InterfaceSize(11))
			if d.Worker != nil {
				ui.Text(c, d.Worker.CWD).FontSize(fonttheme.InterfaceSize(11))
				if d.Worker.Error != "" {
					ui.Text(c, d.Worker.Error).FontSize(fonttheme.InterfaceSize(12))
				}
				if !d.Worker.Materialized {
					ui.Text(c, tr("原成员尚未建立可读取会话，不会创建替代会话。", "The original member has no readable Session yet; no replacement is created.")).FontSize(fonttheme.InterfaceSize(11))
				}
			}
			for _, attempt := range d.Workstream.Delegations {
				ui.Text(c, attempt.Ref+" · "+attempt.Phase+" · "+attempt.RunID).FontSize(fonttheme.InterfaceSize(11))
			}
			if m.Process != nil {
				label := tr("成员过程历史（只读）", "Member process history (read-only)")
				if m.ProcessWatching {
					label = tr("正在持续跟进成员过程（只读）", "Following member process live (read-only)")
				}
				ui.Text(c, label).FontSize(fonttheme.InterfaceSize(11))
				if m.Process.Error != "" {
					ui.Text(c, m.Process.Error).FontSize(fonttheme.InterfaceSize(11))
				}
				if m.Process.Truncated {
					ui.Text(c, tr("只保留最近400条记录；更早内容可通过历史分页读取。", "Showing the latest 400 records; use earlier history for older content.")).FontSize(fonttheme.InterfaceSize(11))
				}
				for _, record := range m.Process.Records {
					if record.Type != "event" {
						ui.Text(c, record.Type).FontSize(fonttheme.InterfaceSize(11))
						continue
					}
					ui.Text(c, fmt.Sprintf("%g · %s", record.Event.Seq, record.Event.Type)).FontSize(fonttheme.InterfaceSize(11))
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
					ui.Text(c, body).FontSize(fonttheme.InterfaceSize(11))
				}
				if m.Process.HasMore && ui.Button(c, tr("查看更早记录（停止实时跟进）", "Earlier history (stop live follow)")).Disabled(m.Busy).Clicked() && m.OnProcess != nil {
					m.OnProcess(true)
				}
			}
		})
	})
}
