package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	ui.Column(c.Key("workspace-agents-jobs")).FillWidth().MinWidth(0).Shrink(0).Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Padding(10, 12).Gap(6).Children(func() {
			ui.Text(c, paneLabel("后台任务", "Background jobs")).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
			if s.Error != "" {
				ui.Text(c, s.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				if ui.Button(c, paneLabel("重新读取任务", "Retry jobs")).AlignSelf(ui.Start).Disabled(s.Loading).Clicked() {
					controller.Retry(s.Generation)
				}
			}
			for _, job := range s.Rows {
				ui.Column(c.Key("agent-job:" + job.ID)).FillWidth().MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, job.Label).FontSize(theme.InterfaceSize(13)).TextColor(fg).MaxLines(2)
					status := jobStatus(job.Status)
					if job.Detail != "" {
						status += " · " + job.Detail
					}
					ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(muted).MaxLines(2)
				})
			}
		})
	})
}
