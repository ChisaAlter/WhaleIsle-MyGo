package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"time"
	"whaleisle/internal/projectflow"
)

func deliveryHistoryView(c *ui.Context, m Model) {
	h := m.Deliveries
	action := func(a, arg string) {
		if m.OnDeliveryAction != nil {
			m.OnDeliveryAction(a, arg)
		}
	}
	ui.Text(c, tr("历次交付", "Delivery history")+" · "+h.Title).FontSize(13)
	ui.Text(c, h.Scope.ProjectID).FontSize(11).Selectable()
	if ui.Button(c, tr("返回 Projects", "Back to Projects")).Clicked() {
		action("close", "")
	}
	if h.Error != "" {
		ui.Text(c, h.Error).FontSize(12).Selectable()
		if h.Report != nil {
			ui.Text(c, tr("读取失败；保留此引用上次读取的报告。", "Read failed; retaining this reference's previously fetched report.")).FontSize(11)
		}
	}
	if h.Selected != nil {
		if h.ReportRef != nil {
			deliveryEvidenceView(c, m)
			return
		}
		if ui.Button(c, tr("返回交付集合", "Back to deliveries")).Clicked() {
			action("back-list", "")
		}
		deliveryRecordView(c, m, *h.Selected)
		return
	}
	if ui.Button(c, tr("刷新此页", "Refresh this page")).Disabled(m.Busy).Clicked() {
		action("refresh", "")
	}
	if h.Loaded && len(h.Page.Items) == 0 {
		ui.Text(c, tr("尚无已保存的交付记录。", "No recorded deliveries yet.")).FontSize(12)
	}
	// Group only the current authoritative page. Pagination never claims that a
	// page contains every revision of a request.
	groups := map[string][]projectflow.HistoricalDelivery{}
	var order []string
	for _, d := range h.Page.Items {
		if _, ok := groups[d.RequestID]; !ok {
			order = append(order, d.RequestID)
		}
		groups[d.RequestID] = append(groups[d.RequestID], d)
	}
	ui.Scroll(c).Height(320).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, id := range order {
				ui.Text(c, tr("要求：", "Request: ")+id).FontSize(12).Selectable()
				versions := groups[id]
				sort.SliceStable(versions, func(i, j int) bool {
					if versions[i].RequestRevision != versions[j].RequestRevision {
						return versions[i].RequestRevision > versions[j].RequestRevision
					}
					return versions[i].At > versions[j].At
				})
				for _, d := range versions {
					d := d
					label := fmt.Sprintf(tr("第%d版 · %s", "Revision %d · %s"), d.RequestRevision, d.Title)
					if ui.Button(c, label).FillWidth().Disabled(m.Busy).Clicked() {
						action("select", d.ID)
					}
					ui.Text(c, d.ID+" · "+d.Status+" · "+d.Outcome+" · "+time.UnixMilli(int64(d.At)).Format(time.RFC3339)).FontSize(11).Selectable()
				}
			}
		})
	})
	ui.Text(c, fmt.Sprintf(tr("第%d页 · 全部%d条交付；同一要求的其他版本可能在更早页。", "Page %d · %d deliveries total; more revisions may appear on earlier pages."), len(h.Cursors), h.Page.Total)).FontSize(11)
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || len(h.Cursors) < 2).Clicked() {
			action("previous", "")
		}
		if ui.Button(c, tr("更早交付", "Older deliveries")).Disabled(m.Busy || h.Page.NextCursor == "").Clicked() {
			action("next", "")
		}
	})
}
func deliveryRecordView(c *ui.Context, m Model, d projectflow.HistoricalDelivery) {
	ui.Text(c, d.Title+fmt.Sprintf(tr(" · 要求第%d版", " · Request revision %d"), d.RequestRevision)).FontSize(13)
	ui.Text(c, d.RequestID+" · "+d.ID+" · "+d.Status+" · "+d.Outcome).FontSize(11).Selectable()
	ui.Text(c, time.UnixMilli(int64(d.At)).Format(time.RFC3339)).FontSize(11)
	if d.Status == "historical" || d.Status == "obsolete" {
		ui.Text(c, tr("历史交付记录；不代表当前要求已经完成。", "Historical delivery; it does not establish that the current requirement is complete.")).FontSize(11)
	} else if !d.Applicable {
		ui.Text(c, tr("此交付的要求或证据已变化，需要重新复核。", "The requirement or evidence has changed since this delivery.")).FontSize(11)
	}
	ui.Scroll(c).Height(380).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, d.Summary).FontSize(12).Selectable()
			for _, v := range d.Evidence {
				ui.Text(c, tr("证据：", "Evidence: ")+v).FontSize(12).Selectable()
			}
			for _, v := range d.RemainingIssues {
				ui.Text(c, tr("未完成条件：", "Remaining issue: ")+v).FontSize(12).Selectable()
			}
			for _, ref := range d.ReportRefs {
				ref := ref
				if ui.Button(c, tr("打开原报告：", "Open recorded report: ")+ref.WorkstreamID+" · "+ref.DelegationRef).FillWidth().Disabled(m.Busy).Clicked() && m.OnDeliveryReport != nil {
					m.OnDeliveryReport(d.ID, ref)
				}
				if ref.ReportID != "" {
					ui.Text(c, "reportId: "+ref.ReportID).FontSize(11).Selectable()
				}
			}
			for _, ref := range d.TerminalRefs {
				ui.Text(c, tr("原结算身份：", "Recorded settlement: ")+ref.WorkstreamID+" · "+ref.DelegationRef+" · "+ref.RunID).FontSize(11).Selectable()
			}
			for _, file := range d.Artifacts {
				ui.Text(c, tr("引用文件：", "Referenced file: ")+file.Path+"\n"+file.WorkstreamID+" · "+file.DelegationRef).FontSize(11).Selectable()
			}
			for _, page := range d.Pages {
				ui.Text(c, tr("页面引用：", "Recorded page reference: ")+page.Title+"\n"+page.SessionID+" · "+page.File).FontSize(11).Selectable()
			}
			if len(d.Error) > 0 && string(d.Error) != "null" {
				ui.Text(c, tr("原记录错误：", "Recorded error: ")+string(d.Error)).FontSize(11).Selectable()
			}
			if len(d.Artifacts) > 0 || len(d.Pages) > 0 {
				ui.Text(c, tr("这里保留文件/页面来源身份；此视图不打开主机路径或执行页面。", "File and page provenance is retained here; this view does not open Host paths or execute pages.")).FontSize(11)
			}
		})
	})
}
func deliveryEvidenceView(c *ui.Context, m Model) {
	h := m.Deliveries
	d := h.Selected
	ref := h.ReportRef
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("返回该次交付", "Back to this delivery")).Clicked() && m.OnDeliveryAction != nil {
			m.OnDeliveryAction("back-report", "")
		}
		if ui.Button(c, tr("重读原报告", "Reload recorded report")).Disabled(m.Busy).Clicked() && m.OnDeliveryReport != nil {
			m.OnDeliveryReport(d.ID, *ref)
		}
	})
	ui.Text(c, fmt.Sprintf(tr("交付 %s · 要求 %s 第%d版", "Delivery %s · Request %s revision %d"), d.ID, d.RequestID, d.RequestRevision)).FontSize(11).Selectable()
	ui.Text(c, ref.WorkstreamID+" · "+ref.DelegationRef+" · "+ref.ReportID).FontSize(11).Selectable()
	if h.Report == nil {
		return
	}
	r := h.Report
	ui.Scroll(c).Height(380).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, r.Outcome+" · "+time.UnixMilli(int64(r.At)).Format(time.RFC3339)).FontSize(11)
			if r.PrerequisitesChanged {
				ui.Text(c, tr("此原报告的依据已经变化。", "This recorded report's prerequisites have changed.")).FontSize(11)
			}
			ui.Text(c, r.Summary).FontSize(12).Selectable()
			for _, v := range r.Evidence {
				ui.Text(c, tr("原报告证据：", "Recorded evidence: ")+v).FontSize(12).Selectable()
			}
			for _, v := range r.RemainingIssues {
				ui.Text(c, tr("原报告未完成项：", "Recorded remaining issue: ")+v).FontSize(12).Selectable()
			}
			if len(r.Request) > 0 && string(r.Request) != "null" {
				var value any
				if json.Unmarshal(r.Request, &value) == nil {
					body, _ := json.MarshalIndent(value, "", "  ")
					ui.Text(c, tr("原报告的要求/版本：", "Report's recorded requirement/version: ")+string(body)).FontSize(11).Selectable()
				}
			}
			for _, file := range r.Artifacts {
				ui.Text(c, tr("原报告文件：", "Recorded artifact: ")+file.Path).FontSize(11).Selectable()
			}
		})
	})
}
