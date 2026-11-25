package settings

import (
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/ui/coss"

	"github.com/egoist/mygo/ui"
)

// ProjectPreferenceAccess exposes an accepted lifecycle snapshot, not a local
// form draft. The setter owns leaving a Lead, protected restart and rollback.
type ProjectPreferenceAccess struct {
	State projectpreference.View
	OnSet func(bool)
}

func interfaceProjectPreference(c *ui.Context, a *ProjectPreferenceAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	state := a.State
	title := "Project"
	bodyPreferenceRow(c, "interface-project", title, providerCopy("让一个主对话带领智能体团队，按需求清单持续完成一个目录里的开发。", "Let one lead conversation guide an agent team through a running list of requirements in a directory."), fg, muted, func() {
		value := state.Enabled
		if ui.Switch(c.Key("interface-project-toggle"), &value).Label(title).Disabled(state.Busy || a.OnSet == nil).Changed() && a.OnSet != nil {
			a.OnSet(value)
		}
	})
	kHint(c, providerCopy("切换后会自动重启 Harness。", "Toggling restarts the Harness automatically."))
	helpSeat := ui.Box(c.Key("interface-project-help-seat")).Absolute().Size(0, 0).PassThrough()
	helpOpen := ui.Local(helpSeat, "open", func() bool { return false })
	if coss.Button(c, providerCopy("Project 是什么", "What is Project?"), coss.ButtonOptions{
		Variant: coss.ButtonLink,
		Size:    coss.ButtonSM,
		Key:     "interface-project-help",
	}).AlignSelf(ui.Start).Padding(0).Clicked() {
		*helpOpen = !*helpOpen
	}
	if *helpOpen {
		kHint(c, providerCopy("Project 把一个本地目录变成长期项目。主对话协调需求清单与团队；开启后可在新会话首屏或工作区目录菜单创建 Project。关闭只是隐藏入口，已有 Project 的数据会保留。", "Project turns a local directory into a long-running project. Its lead coordinates requests and a team. When enabled, create one from New Session or a workspace menu. Turning it off only hides the entries; existing Project data is kept."))
	}
	if state.Error != "" {
		message := state.Error
		if state.RestartCancelled {
			message = providerCopy("已取消重启 Harness，Project 设置保持不变。", "Harness restart was cancelled; Project setting was kept")
		}
		kError(c, message)
	}
}
