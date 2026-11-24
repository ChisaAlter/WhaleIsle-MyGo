package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// AgentsPanel includes the Session's background job summaries, independently
// of the header's expanded-output popover. Both read the same native controller.
func workspaceAgentJobs(c *ui.Context, m Model) {
	controller := m.Store.NativeJobs()
	controller.Mount()
	if m.jobsHeaderMounted != nil {
		*m.jobsHeaderMounted = true
	}
	s := controller.Snapshot()
	if len(s.Rows) == 0 && s.Error == "" {
		return
	}
	t := tk(c)
	fg, muted := t.Foreground, t.MutedForeground
	secondary := muted
	ui.Column(c.Key("workspace-agents-jobs")).FillWidth().MinWidth(0).Shrink(0).Margin(design.Sp(2), 0, 0, 0).Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Children(func() {
			ui.Text(c, paneLabel("后台任务", "Background jobs")).FontSize(theme.InterfaceSize(12)).FontWeight(design.WeightMedium).FixedLineHeight(theme.InterfaceSize(18)).TextColor(secondary).Margin(0, 0, design.Sp(1), 0)
			if s.Error != "" {
				ui.Text(c, s.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				if coss.Button(c, paneLabel("重新读取任务", "Retry jobs"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: s.Loading}).AlignSelf(ui.Start).Clicked() {
					controller.Retry(s.Generation)
				}
			}
			for _, job := range s.Rows {
				ui.Row(c.Key("agent-job:"+job.ID)).FillWidth().MinWidth(0).MinHeight(36).Padding(design.Sp(2), design.Sp(2.5)).Radius(design.RadiusSM).Background(frameInsetBg(t)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, job.Label).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg).SingleLine().Ellipsis("…")
					status := jobStatus(job.Status)
					if job.Detail != "" {
						status += " · " + job.Detail
					}
					ui.Text(c, status).Shrink(0).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				})
			}
		})
	})
}
