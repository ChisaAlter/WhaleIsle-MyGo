package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// FeaturesAccess is the 内置功能 page: the switches that start or stop a whole
// Harness feature. Each one restarts the Harness, so the setter owns the
// protected restart and the rollback.
type FeaturesAccess struct {
	Project *ProjectPreferenceAccess
	// AssistantEnabled/OnSetAssistant drive 鲸鱼娘助理.
	AssistantEnabled *bool
	OnSetAssistant   func(bool)
	// RemoteWorkspaceEnabled/OnSetRemoteWorkspace drive 远程工作区（SSH）.
	RemoteWorkspaceEnabled *bool
	OnSetRemoteWorkspace   func(bool)
}

// featureRow is one switch row of the page: title (with an optional trailing
// accessory such as the Project help button), description, an optional 11 DIP
// hint line, and the 36x20 switch. The last row carries no bottom rule.
type featureRow struct {
	key, title, badge, desc, hint string
	value                         *bool
	disabled                      bool
	changed                       func(bool)
	accessory                     func()
	last                          bool
	// compact is the pet page's 14/32 DIP item padding instead of 16/48.
	compact bool
}

func featureToggleRow(c *ui.Context, row featureRow, fg, muted ui.Color) {
	if row.value == nil {
		return
	}
	outer := ui.Column(c.Key(row.key)).FillWidth().MinWidth(0).Gap(0)
	outer.Draw(func(p *ui.Painter, bounds ui.Rect) {
		if !row.last {
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y + bounds.H - .667, W: bounds.W, H: .667}, c.Theme().Border, 0)
		}
	})
	outer.Children(func() {
		padY, padRight := float32(16), float32(48)
		if row.compact {
			padY, padRight = 14, 32
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(padY, 0).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Padding(0, padRight, 0, 0).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(2).AlignItems(ui.Center).Children(func() {
					ui.Text(c, row.title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
					if row.accessory != nil {
						row.accessory()
					}
					if row.badge != "" {
						ui.Text(c, row.badge).Padding(1, 6).Radius(4).FontSize(theme.InterfaceSize(10)).
							Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .35)).TextColor(muted)
					}
				})
				if row.desc != "" {
					ui.Text(c, row.desc).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				}
				if row.hint != "" {
					ui.Text(c, row.hint).FillWidth().FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted)
				}
			})
			if settingsPreferenceSwitch(c.Key(row.key+"-switch"), row.value, settingsSwitchLabelled).Label(row.title).Disabled(row.disabled).Changed() && row.changed != nil {
				row.changed(*row.value)
			}
		})
	})
}

func featuresPane(c *ui.Context, m Model, fg, muted ui.Color) {
	a := m.Features
	if a == nil {
		return
	}
	restartHint := providerCopy("切换后会自动重启 Harness。", "Toggling restarts the Harness automatically.")
	featureToggleRow(c, featureRow{
		key:      "features-assistant",
		title:    providerCopy("鲸鱼娘助理", "Whale assistant"),
		desc:     providerCopy("开启她的常驻对话与侧栏入口；切换后会重启 Harness。", "Turn on her resident conversation and sidebar entry; switching restarts the Harness."),
		value:    a.AssistantEnabled,
		disabled: a.OnSetAssistant == nil,
		changed:  a.OnSetAssistant,
	}, fg, muted)
	if p := a.Project; p != nil {
		state := p.State
		value := state.Enabled
		help := providerCopy("Project 是什么", "What is Project?")
		featureToggleRow(c, featureRow{
			key:      "features-project",
			title:    "Project",
			desc:     providerCopy("让一个主对话带领智能体团队，按需求清单持续完成一个目录里的开发。", "Let one lead conversation guide an agent team through a running list of requirements in a directory."),
			hint:     restartHint,
			value:    &value,
			disabled: state.Busy || p.OnSet == nil,
			changed:  func(v bool) { p.OnSet(v) },
			accessory: func() {
				button := ui.Button(c, "").Key("features-project-help").Label(help).Size(24, 24).Shrink(0).Padding(4).Radius(8).BorderWidth(0).Background(ui.Transparent).Tooltip(help)
				button.Children(func() { ui.Icon(c, bodyIcon("help")).Size(16, 16).TextColor(muted) })
				if button.Hovered() {
					button.Background(c.Theme().SurfaceHover)
				}
				if button.Clicked() {
					c.AddToast(ui.Toast{ID: "interface-project-help", Title: providerCopy("Project 把一个本地目录变成长期项目。主对话协调需求清单与团队；开启后可在新会话首屏或工作区目录菜单创建 Project。关闭只是隐藏入口，已有 Project 的数据会保留。", "Project turns a local directory into a long-running project. Its lead coordinates requests and a team. When enabled, create one from New Session or a workspace menu. Turning it off only hides the entries; existing Project data is kept.")})
				}
			},
		}, fg, muted)
		if state.Busy {
			ui.Text(c, providerCopy("正在应用 Project 设置…", "Applying Project setting…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if state.Error != "" {
			message := state.Error
			if state.RestartCancelled {
				message = providerCopy("已取消重启 Harness，Project 设置保持不变。", "Harness restart was cancelled; Project setting was kept")
			}
			ui.Text(c, message).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
	}
	featureToggleRow(c, featureRow{
		key:      "features-remote-workspace",
		title:    providerCopy("远程工作区（SSH）", "Remote workspace (SSH)"),
		desc:     providerCopy("通过 SSH 把远程机器挂为工作区：在「添加工作区」选择远程目录，本地镜像与远端双向同步。", "Mount a remote machine as a workspace over SSH: pick a remote directory in Add workspace; the local mirror and the remote sync both ways."),
		hint:     restartHint,
		value:    a.RemoteWorkspaceEnabled,
		disabled: a.OnSetRemoteWorkspace == nil,
		changed:  a.OnSetRemoteWorkspace,
		last:     true,
	}, fg, muted)
}
