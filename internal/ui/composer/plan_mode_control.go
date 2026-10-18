package composer

import "github.com/egoist/mygo/ui"

func paintPlanModeControl(c *ui.Context, m Model, ink ui.Color, height float32) {
	p := m.Snapshot.PlanModeControl
	if !p.Present || !p.Effective {
		return
	}
	ui.Column(c).Gap(4).Children(func() {
		label := "计划模式 · 退出 / Exit plan mode"
		if p.Leaving {
			label = "正在退出计划模式… / Leaving plan mode…"
		}
		locked := m.Editing || m.State != nil && (m.State.Sending || m.State.Preparing > 0)
		if ui.Button(c, label).Height(height).TextColor(ink).Disabled(locked || p.Leaving || m.OnExitPlanMode == nil).Clicked() && m.OnExitPlanMode != nil {
			m.OnExitPlanMode()
		}
		if p.Error != "" {
			ui.Text(c, p.Error).TextColor(ink).MaxWidth(300)
		}
	})
}
