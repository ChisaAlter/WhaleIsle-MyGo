package projects

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func assetDiffView(c *ui.Context, m Model) {
	back := tr("返回交付详情", "Back to delivery")
	if m.RequestProjectID == "" && m.TeamID != "" && m.Detail != nil {
		back = tr("返回工作详情", "Back to work details")
	}
	if ui.Button(c, back).Clicked() && m.OnCloseDiff != nil {
		m.OnCloseDiff()
	}
	value := m.AssetDiff
	ui.Text(c, tr("当前工作区差异；这不是历史版本快照或回滚。", "Current workspace diff; this is not a historical version snapshot or rollback.")).FontSize(fonttheme.InterfaceSize(13))
	ui.Text(c, value.Worker.CWD+" · "+value.Worker.Branch).FontSize(fonttheme.InterfaceSize(12))
	if value.Snapshot.Truncated {
		ui.Text(c, tr("差异已截断，请在工作目录中查看完整内容。", "Diff is truncated; inspect the working directory for complete contents.")).FontSize(fonttheme.InterfaceSize(13))
	}
	ui.Column(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			for _, file := range value.Snapshot.Files {
				ui.Text(c, file.Path).FontSize(fonttheme.InterfaceSize(12))
			}
			if value.Snapshot.IsGit != nil && !*value.Snapshot.IsGit {
				ui.Text(c, tr("此工作区不是Git仓库。", "This workspace is not a Git repository.")).FontSize(fonttheme.InterfaceSize(13))
			} else if value.Snapshot.Diff != "" {
				ui.Text(c, value.Snapshot.Diff).FontSize(fonttheme.InterfaceSize(12))
			} else if len(value.Snapshot.Files) == 0 {
				ui.Text(c, tr("没有报告差异。", "No reported changes.")).FontSize(fonttheme.InterfaceSize(13))
			}
		})
	})
}
