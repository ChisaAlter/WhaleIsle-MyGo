package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

func paintPlanModeControl(c *ui.Context, m Model, ink ui.Color, height float32) {
	p := m.Snapshot.PlanModeControl
	if !p.Present || !p.Effective {
		return
	}
	t := design.For(c)
	ui.Column(c).Gap(design.Sp(1)).Children(func() {
		label := composerText("计划", "Plan")
		if p.Leaving {
			label = composerText("正在退出计划模式…", "Leaving plan mode…")
		}
		locked := m.Editing || m.State != nil && (m.State.Sending || m.State.Preparing > 0)
		btn := coss.Button(c, label, coss.ButtonOptions{
			Variant:  coss.ButtonSecondary,
			Size:     coss.ButtonSM,
			Disabled: locked || p.Leaving || m.OnExitPlanMode == nil,
		}).Label(composerText("计划模式已开启，按下关闭", "Plan mode on, press to turn off")).
			Tooltip(composerText("计划模式已开启 — 点击关闭（/plan off）", "Plan mode on — click to turn off (/plan off)")).
			Height(height)
		if btn.Clicked() && m.OnExitPlanMode != nil {
			m.OnExitPlanMode()
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground).MaxWidth(300)
		}
	})
}
