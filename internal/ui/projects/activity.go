package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/projectflow"
	fonttheme "whaleisle/internal/ui/theme"
)

func activityKindLabel(kind string) string {
	switch kind {
	case "all":
		return tr("全部", "All")
	case "version":
		return tr("要求版本", "Requirement revision")
	case "assignment":
		return tr("派工", "Assignment")
	case "report":
		return tr("成员报告", "Member report")
	case "terminal":
		return tr("运行结算", "Run settlement")
	case "delivery":
		return tr("交付", "Delivery")
	}
	return kind
}
func requestActivityView(c *ui.Context, m Model) {
	a := m.Activity
	action := func(kind, arg string) {
		if m.OnActivityAction != nil {
			m.OnActivityAction(kind, arg)
		}
	}
	ui.Text(c, tr("要求活动时间线", "Request activity timeline")+" · "+a.Title).FontSize(fonttheme.InterfaceSize(14))
	ui.Text(c, fmt.Sprintf("%s · %s · revision %d", a.Scope.ProjectID, a.Scope.RequestID, a.Revision)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	if ui.Button(c, tr("返回原要求详情", "Back to request details")).Clicked() {
		action("close", "")
	}
	ui.Text(c, tr("来自 requests/get 的要求快照；包含版本、所关联派工及当前交付，不是全部跨派工运行日志。", "Request snapshot: revisions, associated assignments and its current delivery. This is not the complete cross-assignment run log.")).FontSize(fonttheme.InterfaceSize(12))
	if a.Error != "" {
		ui.Text(c, a.Error).FontSize(fonttheme.InterfaceSize(13)).Selectable()
		if a.Report != nil {
			ui.Text(c, tr("仍保留此来源上次读取的报告正文。", "Retaining this source's previously fetched report body.")).FontSize(fonttheme.InterfaceSize(12))
		} else if a.Loaded {
			ui.Text(c, tr("仍保留上次读取的完整活动快照。", "Retaining the previously fetched activity snapshot.")).FontSize(fonttheme.InterfaceSize(12))
		}
	}
	if a.Selected != nil {
		if a.ReportRef != nil {
			activityReportView(c, m)
			return
		}
		if ui.Button(c, tr("返回时间线（保留筛选和页码）", "Back to timeline (keep filter/page)")).Clicked() {
			action("back-list", "")
		}
		activityEventView(c, m, *a.Selected)
		return
	}
	if ui.Button(c, tr("刷新要求快照", "Refresh request snapshot")).Disabled(m.Busy).Clicked() {
		action("refresh", "")
	}
	if a.AuthorityError != "" {
		ui.Text(c, tr("要求授权问题：", "Requirement authority issue: ")+a.AuthorityError).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	}
	if a.HandoverError != "" {
		ui.Text(c, tr("交付问题：", "Handover issue: ")+a.HandoverError).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	}
	ui.Row(c).FillWidth().Wrap().Gap(4).Children(func() {
		for _, kind := range []string{"all", "version", "assignment", "report", "terminal", "delivery"} {
			kind := kind
			label := activityKindLabel(kind)
			if a.Filter == kind {
				label = "● " + label
			}
			if ui.Button(c, label).Disabled(m.Busy).Clicked() {
				action("filter", kind)
			}
		}
	})
	if a.Loaded && len(a.Visible) == 0 {
		ui.Text(c, tr("该快照没有此类带时间的活动记录。", "This snapshot contains no timestamped entries of this type.")).FontSize(fonttheme.InterfaceSize(13))
	}
	ui.Column(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, event := range a.Visible {
				event := event
				ui.Text(c, time.UnixMilli(int64(event.At)).Format(time.RFC3339)+" · "+activityKindLabel(event.Kind)).FontSize(fonttheme.InterfaceSize(12))
				if ui.Button(c, event.Title).FillWidth().Disabled(m.Busy).Clicked() {
					action("select", event.Key)
				}
				if event.Version != nil {
					ui.Text(c, fmt.Sprintf(tr("要求第%d版 · 打开完整目标正文", "Requirement revision %d · open full goal"), event.Revision)).FontSize(fonttheme.InterfaceSize(12))
				}
				if event.Work != nil {
					ui.Text(c, event.Work.WorkstreamID+" · "+event.Work.DelegationRef+" · "+event.Work.RunID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				if event.Delivery != nil {
					ui.Text(c, event.Delivery.ID+" · "+event.Delivery.Outcome).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
			}
		})
	})
	ui.Text(c, fmt.Sprintf(tr("本地第%d页 · 此筛选共%d个事件；翻页不调用新的 Host 游标。", "Local page %d · %d matching events; paging uses this captured snapshot."), a.Page+1, a.FilteredTotal)).FontSize(fonttheme.InterfaceSize(12))
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || a.Page == 0).Clicked() {
			action("previous", "")
		}
		if ui.Button(c, tr("下一页", "Next")).Disabled(m.Busy || (a.Page+1)*projectflow.ActivityPageSize >= a.FilteredTotal).Clicked() {
			action("next", "")
		}
	})
}
func activityEventView(c *ui.Context, m Model, e projectflow.ActivityEvent) {
	ui.Text(c, activityKindLabel(e.Kind)+" · "+e.Title).FontSize(fonttheme.InterfaceSize(14))
	ui.Text(c, time.UnixMilli(int64(e.At)).Format(time.RFC3339)).FontSize(fonttheme.InterfaceSize(12))
	ui.Column(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			if v := e.Version; v != nil {
				ui.Text(c, fmt.Sprintf(tr("要求第%d版", "Requirement revision %d"), v.Revision)).FontSize(fonttheme.InterfaceSize(13))
				ui.Text(c, v.Goal).FontSize(fonttheme.InterfaceSize(13)).Selectable()
			}
			if w := e.Work; w != nil {
				ui.Text(c, "workstream: "+w.WorkstreamID+"\ndelegation: "+w.DelegationRef+"\nrun: "+w.RunID+"\nmember: "+w.MemberName+" · "+w.WorkerSessionID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				ui.Text(c, w.Status+" · "+w.Phase).FontSize(fonttheme.InterfaceSize(13))
				if w.Current {
					ui.Text(c, tr("这是此快照的当前派工。", "This is the current assignment in this snapshot.")).FontSize(fonttheme.InterfaceSize(12))
				} else {
					ui.Text(c, tr("保留此关联派工的原身份。", "Original associated assignment identity is retained.")).FontSize(fonttheme.InterfaceSize(12))
				}
				if e.Kind == "report" {
					ui.Text(c, w.Summary).FontSize(fonttheme.InterfaceSize(13)).Selectable()
					ui.Text(c, "reportId: "+w.ReportID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				if e.Kind == "terminal" {
					ui.Text(c, tr("原运行结束原因：", "Recorded run stop reason: ")+w.TerminalStopReason).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				}
				if w.BlockedReason != "" {
					ui.Text(c, w.BlockedReason).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				if len(w.WaitingReason) > 0 && string(w.WaitingReason) != "null" {
					ui.Text(c, string(w.WaitingReason)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
			}
			if d := e.Delivery; d != nil {
				ui.Text(c, d.ID+" · "+d.Outcome).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				if !d.Applicable {
					ui.Text(c, tr("此交付的要求或证据已变化。", "The requirement or evidence has changed since this delivery.")).FontSize(fonttheme.InterfaceSize(12))
				}
				ui.Text(c, d.Summary).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				for _, v := range d.Evidence {
					ui.Text(c, tr("证据：", "Evidence: ")+v).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				}
				for _, v := range d.RemainingIssues {
					ui.Text(c, tr("未完成项：", "Remaining issue: ")+v).FontSize(fonttheme.InterfaceSize(13)).Selectable()
				}
				for _, r := range d.TerminalRefs {
					ui.Text(c, tr("结算来源：", "Settlement source: ")+r.WorkstreamID+" · "+r.DelegationRef+" · "+r.RunID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				for _, f := range d.Artifacts {
					ui.Text(c, tr("文件引用：", "Artifact reference: ")+f.Path+" · "+f.WorkstreamID+" · "+f.DelegationRef).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
				renderVisualPages(c, m, "activity:"+e.Key, d.Pages)
			}
			for _, ref := range projectflow.ActivityEventReports(e) {
				ref := ref
				if ui.Button(c, tr("查看精确来源报告：", "Read exact source report: ")+ref.WorkstreamID+" · "+ref.DelegationRef).FillWidth().Disabled(m.Busy).Clicked() && m.OnActivityReport != nil {
					m.OnActivityReport(e.Key, ref)
				}
			}
		})
	})
}
func activityReportView(c *ui.Context, m Model) {
	a := m.Activity
	ref := a.ReportRef
	e := a.Selected
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		if ui.Button(c, tr("返回原事件", "Back to original event")).Clicked() && m.OnActivityAction != nil {
			m.OnActivityAction("back-report", "")
		}
		if ui.Button(c, tr("重读此报告", "Reload this report")).Disabled(m.Busy).Clicked() && m.OnActivityReport != nil {
			m.OnActivityReport(e.Key, *ref)
		}
	})
	ui.Text(c, ref.WorkstreamID+" · "+ref.DelegationRef+" · "+ref.ReportID).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	if a.Report == nil {
		return
	}
	r := a.Report
	historicalSourceButton(c, m, "activity")
	ui.Column(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, r.Outcome+" · "+time.UnixMilli(int64(r.At)).Format(time.RFC3339)).FontSize(fonttheme.InterfaceSize(12))
			if r.PrerequisitesChanged {
				ui.Text(c, tr("此报告的原依据已经变化。", "This report's original prerequisites have changed.")).FontSize(fonttheme.InterfaceSize(12))
			}
			ui.Text(c, r.Summary).FontSize(fonttheme.InterfaceSize(13)).Selectable()
			for _, v := range r.Evidence {
				ui.Text(c, tr("报告证据：", "Report evidence: ")+v).FontSize(fonttheme.InterfaceSize(13)).Selectable()
			}
			for _, v := range r.RemainingIssues {
				ui.Text(c, tr("未完成项：", "Remaining issue: ")+v).FontSize(fonttheme.InterfaceSize(13)).Selectable()
			}
			if len(r.Request) > 0 && string(r.Request) != "null" {
				var v any
				if json.Unmarshal(r.Request, &v) == nil {
					raw, _ := json.MarshalIndent(v, "", "  ")
					ui.Text(c, tr("报告的原要求/版本：", "Report's recorded requirement/version: ")+string(raw)).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
			}
			for _, f := range r.Artifacts {
				ui.Text(c, tr("原文件引用：", "Recorded artifact reference: ")+f.Path).FontSize(fonttheme.InterfaceSize(12)).Selectable()
			}
		})
	})
}
