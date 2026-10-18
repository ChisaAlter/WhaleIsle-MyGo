package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func materialCollectionView(c *ui.Context, m Model) {
	d := m.MaterialCollection
	action := func(kind string) {
		if m.OnMaterialCollectionAction != nil {
			m.OnMaterialCollectionAction(kind)
		}
	}
	ui.Text(c, d.Title+" · "+tr("材料集合", "Materials")).FontSize(fonttheme.InterfaceSize(13))
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("返回 Projects", "Back to Projects")).Clicked() {
			action("close")
		}
		if ui.Button(c, tr("刷新要求材料", "Refresh requirement materials")).Disabled(m.Busy).Clicked() {
			action("refresh")
		}
	})
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("项目资料", "Documents")).Disabled(m.Busy).Clicked() && m.OnDocuments != nil {
			m.OnDocuments(d.ProjectID)
		}
		if ui.Button(c, tr("历次交付", "Deliveries")).Disabled(m.Busy).Clicked() && m.OnDeliveries != nil {
			m.OnDeliveries(d.ProjectID)
		}
		if ui.Button(c, tr("成员报告", "Member reports")).Disabled(m.Busy).Clicked() && m.OnReports != nil {
			m.OnReports(d.ProjectID)
		}
	})
	ui.Row(c).Gap(4).Children(func() {
		for _, path := range []string{"notes.md", "preferences.md"} {
			if ui.Button(c, path).Disabled(m.Busy).Clicked() && m.OnMaterial != nil {
				m.OnMaterial(d.ProjectID, path)
			}
		}
	})
	ui.Text(c, tr("要求恢复材料", "Requirement recovery materials")).FontSize(fonttheme.InterfaceSize(12))
	ui.Text(c, tr("保留原始要求及其权威记录错误，独立于当前任务清单。", "Recorded requirements and authority errors, separate from the current task list.")).FontSize(fonttheme.InterfaceSize(11))
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(fonttheme.InterfaceSize(12)).Selectable()
	}
	if d.Loaded && len(d.Requests.Items) == 0 {
		ui.Text(c, tr("没有要求恢复材料。", "No requirement recovery materials.")).FontSize(fonttheme.InterfaceSize(12))
	}
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, request := range d.Requests.Items {
				ui.Text(c, fmt.Sprintf("%s · v%d", request.Title, request.Revision)).FontSize(fonttheme.InterfaceSize(13))
				ui.Text(c, request.ID).FontSize(fonttheme.InterfaceSize(11)).Selectable()
				body := request.Goal
				ui.TextArea(c.Key("material-request-"+d.ProjectID+"-"+request.ID+fmt.Sprint(request.Revision)), &body).FillWidth().Height(130).ReadOnly(true)
				if request.AuthorityError != "" {
					ui.Text(c, request.AuthorityError).FontSize(fonttheme.InterfaceSize(12)).Selectable()
				}
			}
		})
	})
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || len(d.Cursors) < 2).Clicked() {
			action("previous")
		}
		ui.Text(c, fmt.Sprintf(tr("第 %d 页 · 共 %d 条", "Page %d · %d total"), len(d.Cursors), d.Requests.Total)).FontSize(fonttheme.InterfaceSize(11))
		if ui.Button(c, tr("更早的要求", "Earlier requirements")).Disabled(m.Busy || d.Requests.NextCursor == "").Clicked() {
			action("next")
		}
	})
}
