package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	fonttheme "whaleisle/internal/ui/theme"
)

func historicalSourceButton(c *ui.Context, m Model, source string) {
	if ui.Button(c, tr("定位此报告的原派工", "Locate this report's original assignment")).Disabled(m.Busy || m.OnHistoricalSource == nil).Clicked() && m.OnHistoricalSource != nil {
		m.OnHistoricalSource(source)
	}
}
func historicalProcessView(c *ui.Context, m Model) {
	h := m.Historical
	action := func(s string) {
		if m.OnHistoricalAction != nil {
			m.OnHistoricalAction(s)
		}
	}
	ui.Text(c, tr("历史派工 / 原过程", "Historical assignment / original process")).FontSize(fonttheme.InterfaceSize(14))
	if ui.Button(c, tr("返回原报告（保留页码）", "Back to source report (keep page)")).Clicked() {
		action("close")
	}
	t := h.Target
	ui.Text(c, t.ProjectID+"\n"+t.WorkstreamID+" · "+t.DelegationRef+" · "+t.ReportID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	if h.Error != "" {
		ui.Text(c, h.Error).FontSize(fonttheme.InterfaceSize(13)).Selectable()
		if h.Window != nil {
			ui.Text(c, tr("保留上次成功定位的完整记录。", "Keeping the previously located records.")).FontSize(fonttheme.InterfaceSize(12))
		}
	}
	if ui.Button(c, tr("重读原派工记录", "Reload original assignment receipt")).Disabled(m.Busy).Clicked() {
		action("record")
	}
	if b := h.Binding; b != nil {
		d := b.Delegation
		ui.Text(c, fmt.Sprintf("delegation: %s\nmessage: %s\nrun: %s\nrequest: %s · revision %d\nscope: %s · phase: %s\n%s\nLead: %s\nmember: %s", d.Ref, d.MessageID, d.RunID, d.RequestID, d.RequestRevision, d.Scope, d.Phase, time.UnixMilli(int64(d.CreatedAt)).Format(time.RFC3339), b.CoordinatorSessionID, b.WorkerSessionID)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
		_, err := b.Address()
		if err != nil {
			ui.Text(c, err.Error()).FontSize(fonttheme.InterfaceSize(12)).Selectable()
		}
		ui.Text(c, tr("打开过程只观察原成员 Session；不会激活会话、发送输入或重启旧 run。", "Opening process only observes the original member Session; it does not activate it, send input or restart an old run.")).FontSize(fonttheme.InterfaceSize(12))
		if ui.Button(c, tr("打开并定位原过程", "Open and locate original process")).Disabled(m.Busy || err != nil).Clicked() {
			action("process")
		}
	}
	if h.Window == nil {
		ui.Text(c, tr("所选原报告正文", "Selected original report body")).FontSize(fonttheme.InterfaceSize(13))
		ui.Column(c).FillWidth().MinWidth(0).Children(func() {
			ui.Column(c).FillWidth().Gap(5).Children(func() {
				ui.Text(c, t.Report.Summary).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				for _, s := range t.Report.Evidence {
					ui.Text(c, tr("证据：", "Evidence: ")+s).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				for _, s := range t.Report.RemainingIssues {
					ui.Text(c, tr("剩余问题：", "Remaining issue: ")+s).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				if len(t.Report.Request) > 0 {
					ui.Text(c, string(t.Report.Request)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
			})
		})
		return
	}
	w := h.Window
	start := h.Page * 50
	end := start + 50
	if end > len(w.Records) {
		end = len(w.Records)
	}
	ui.Text(c, fmt.Sprintf(tr("已定位原派工 · 固定游标 %g · 记录 %d–%d / %d", "Original assignment located · frozen cursor %g · records %d–%d / %d"), w.Cursor, start+1, end, len(w.Records))).FontSize(fonttheme.InterfaceSize(12))
	if !w.NextAssignmentBoundary {
		ui.Text(c, tr("记录截至此固定快照；尚未观察到下一次 Lead 派工边界。", "Records end at this frozen snapshot; no later Lead assignment boundary was observed.")).FontSize(fonttheme.InterfaceSize(12))
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || start == 0).Clicked() {
			action("previous")
		}
		if ui.Button(c, tr("下一页", "Next")).Disabled(m.Busy || end == len(w.Records)).Clicked() {
			action("next")
		}
	})
	ui.Column(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, r := range w.Records[start:end] {
				ui.Text(c, fmt.Sprintf("%g · %s", r.Event.Seq, r.Event.Type)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				body := string(r.Event.Data)
				var value any
				if json.Unmarshal(r.Event.Data, &value) == nil {
					raw, _ := json.MarshalIndent(value, "", "  ")
					body = string(raw)
				}
				ui.Text(c, body).FontSize(fonttheme.InterfaceSize(13)).Selectable()
			}
		})
	})
}
