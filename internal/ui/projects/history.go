package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	fonttheme "whaleisle/internal/ui/theme"
)

func reportsView(c *ui.Context, m Model) {
	h := m.Reports
	action := func(a, v string) {
		if m.OnReportAction != nil {
			m.OnReportAction(a, v)
		}
	}
	ui.Text(c, tr("历史报告", "Recorded reports")+" · "+h.ProjectID).FontSize(fonttheme.InterfaceSize(13))
	if ui.Button(c, tr("返回 Projects", "Back to Projects")).Disabled(m.Busy).Clicked() && m.OnCloseReports != nil {
		m.OnCloseReports()
	}
	if h.Notice != "" {
		ui.Text(c, h.Notice).FontSize(fonttheme.InterfaceSize(11))
	}
	if h.Selected != nil {
		r := h.Selected
		if ui.Button(c, tr("返回报告列表", "Back to reports")).Disabled(m.Busy).Clicked() {
			action("back-detail", "")
		}
		ui.Text(c, h.Title+" · "+h.WorkID).FontSize(fonttheme.InterfaceSize(12))
		ui.Text(c, r.DelegationRef+" · "+r.CallID).FontSize(fonttheme.InterfaceSize(11))
		if h.Work != nil {
			label := tr("历史委派报告", "Historical delegation report")
			if h.Work.Workstream.CurrentDelegationRef == r.DelegationRef {
				label = tr("当前委派报告", "Current delegation report")
			}
			ui.Text(c, label).FontSize(fonttheme.InterfaceSize(11))
		}
		historicalSourceButton(c, m, "reports")
		ui.Text(c, r.Outcome+" · "+time.UnixMilli(int64(r.At)).Format(time.RFC3339)).FontSize(fonttheme.InterfaceSize(11))
		if r.PrerequisitesChanged {
			ui.Text(c, tr("此报告依赖的依据已变化。", "Prerequisite evidence has changed since this report.")).FontSize(fonttheme.InterfaceSize(11))
		}
		ui.Scroll(c).Height(300).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				if len(r.Request) > 0 && string(r.Request) != "null" {
					var v any
					if json.Unmarshal(r.Request, &v) == nil {
						b, _ := json.MarshalIndent(v, "", "  ")
						ui.Text(c, tr("原要求 / 版本：", "Recorded request / revision:")+string(b)).FontSize(fonttheme.InterfaceSize(11))
					}
				}
				ui.Text(c, r.Summary).FontSize(fonttheme.InterfaceSize(12))
				for _, v := range r.Evidence {
					ui.Text(c, tr("验证：", "Evidence:")+v).FontSize(fonttheme.InterfaceSize(11))
				}
				for _, v := range r.RemainingIssues {
					ui.Text(c, tr("剩余问题：", "Remaining issue:")+v).FontSize(fonttheme.InterfaceSize(11))
				}
				for _, file := range r.Artifacts {
					file := file
					if ui.Button(c, tr("打开引用文件：", "Open referenced file:")+file.Path).FillWidth().Disabled(m.Busy || m.OnResolvedFile == nil).Clicked() && m.OnReportFile != nil {
						m.OnReportFile(file.Path, m.OnResolvedFile)
					}
				}
				ui.Text(c, tr("报告内容为历史记录；打开文件读取当前文件，不是历史快照。", "Reports are historical records; opening a file reads its current contents, not a historical snapshot.")).FontSize(fonttheme.InterfaceSize(11))
			})
		})
		return
	}
	if h.WorkID != "" {
		if ui.Button(c, tr("返回全部工作报告", "Back to all work reports")).Disabled(m.Busy).Clicked() {
			action("back-work", "")
		}
		ui.Text(c, h.Title+" · "+h.WorkID).FontSize(fonttheme.InterfaceSize(12))
		ui.Text(c, fmt.Sprintf("%d reports", h.History.Total)).FontSize(fonttheme.InterfaceSize(11))
		ui.Scroll(c).Height(260).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				for _, r := range h.History.Items {
					r := r
					if ui.Button(c, r.Outcome+" · "+r.DelegationRef).FillWidth().Disabled(m.Busy).Clicked() {
						action("detail", r.DelegationRef)
					}
					ui.Text(c, time.UnixMilli(int64(r.At)).Format(time.RFC3339)).FontSize(fonttheme.InterfaceSize(11))
				}
			})
		})
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || len(h.WorkCursors) < 2).Clicked() {
				action("work-prev", "")
			}
			if ui.Button(c, tr("更早报告", "Older reports")).Disabled(m.Busy || h.History.NextCursor == "").Clicked() {
				action("work-next", "")
			}
		})
		return
	}
	ui.Text(c, fmt.Sprintf("%d workstreams with reports", h.Listing.Total)).FontSize(fonttheme.InterfaceSize(11))
	ui.Scroll(c).Height(260).Children(func() {
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			for _, e := range h.Listing.Results {
				e := e
				if ui.Button(c, e.Title+fmt.Sprintf(" (%d)", e.ReportCount)).FillWidth().Disabled(m.Busy).Clicked() {
					action("work", e.WorkstreamID)
				}
				ui.Text(c, e.WorkstreamID+" · "+e.Report.DelegationRef+" · "+e.Report.Outcome).FontSize(fonttheme.InterfaceSize(11))
			}
		})
	})
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || len(h.ListCursors) < 2).Clicked() {
			action("list-prev", "")
		}
		if ui.Button(c, tr("下一页", "Next")).Disabled(m.Busy || h.Listing.NextCursor == "").Clicked() {
			action("list-next", "")
		}
	})
}
