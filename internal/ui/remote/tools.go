package remote

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	remotestore "whaleisle/internal/store/remote"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// toolsCard is one bordered section card of the workspace page, matching the
// installed plugin's dshr-card blocks.
func toolsCard(c *ui.Context, key string, border ui.Color, body func()) {
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(10).Padding(12, 14).Radius(10).Border(.5, border).Children(body)
}

func cardTitle(c *ui.Context, title string, right func()) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).Grow(1).MinWidth(0)
		if right != nil {
			right()
		}
	})
}

func ensureToolsLoaded(c *ui.Context, m Model) {
	s := m.Machines.Snapshot()
	if !s.Tools.Loaded && !s.Busy && s.Tools.Error == "" && m.HarnessReady && m.RemoteWorkspaceEnabled {
		m.Machines.LoadTools()
	}
}

// viewActiveCard is the installed settings.active card: a status dot, the
// current machine's identity and connection facts, then 取消当前/忘记指纹.
func viewActiveCard(c *ui.Context, m Model, border ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	s := m.Machines.Snapshot()
	muted := c.Theme().TextMuted
	var current *struct{ name, host string }
	for _, machine := range s.Machines {
		if machine.ID == s.CurrentID {
			name := machine.Name
			if name == "" {
				name = machine.Host
			}
			current = &struct{ name, host string }{name, machine.Host}
			break
		}
	}
	toolsCard(c, "remote-active-card", border, func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			dot := machineOffline
			if s.Status.Connected {
				dot = machineOnline
			}
			ui.Box(c).Size(8, 8).Shrink(0).Radius(4).Background(dot)
			ui.Text(c, label("当前机器", "Active machine")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600)
			caption := label("未选择机器", "No machine selected")
			if current != nil {
				caption = current.name
			}
			ui.Text(c, caption).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
		})
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			state := label("未连接", "Disconnected")
			if s.Status.Connected {
				state = label("已连接", "Connected")
			}
			ui.Text(c, state).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			if s.Status.Host != "" {
				address := s.Status.Username + "@" + s.Status.Host
				if s.Status.Port > 0 {
					address += fmt.Sprintf(":%d", s.Status.Port)
				}
				ui.Text(c, address).Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
			if s.Status.Connected {
				key := label("主机指纹未记录", "Host key not recorded")
				if s.Status.HostKeyKnown {
					key = label("主机指纹已信任", "Host key trusted")
				}
				ui.Text(c, key).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
		})
		if s.Status.Workspace != "" {
			ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
					ui.Text(c, label("远程目录", "Remote path")+":").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					ui.Text(c, s.Status.Workspace).Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
				})
				if s.Status.LocalMirror != "" {
					ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
						ui.Text(c, label("本地镜像", "Local mirror")+":").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
						ui.Text(c, s.Status.LocalMirror).Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
					})
				}
			})
		}
		if s.CurrentID != "" {
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if widgets.Button(c, colors, label("取消当前", "Clear active"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					m.Machines.Select("")
				}
				if widgets.Button(c, colors, label("忘记指纹", "Forget key"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					m.Machines.ForgetHostKey()
				}
			})
		}
	})
}

// viewSSHConfigCard lists importable ~/.ssh/config aliases; hidden when empty,
// as the installed card only mounts with entries.
func viewSSHConfigCard(c *ui.Context, m Model, e *MachineEditor, border ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	s := m.Machines.Snapshot()
	if !s.Tools.Loaded || len(s.Tools.SSHEntries) == 0 {
		return
	}
	muted := c.Theme().TextMuted
	toolsCard(c, "remote-sshconfig-card", border, func() {
		cardTitle(c, label("从 ~/.ssh/config 导入", "Import from ~/.ssh/config"), nil)
		ui.Text(c, label(fmt.Sprintf("识别到 %d 个主机别名，点击导入填入表单。", len(s.Tools.SSHEntries)),
			fmt.Sprintf("%d host aliases found; click import to fill the form.", len(s.Tools.SSHEntries)))).
			FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		for _, entry := range s.Tools.SSHEntries {
			entry := entry
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				caption := entry.Host
				if entry.User != "" {
					host := entry.HostName
					if host == "" {
						host = entry.Host
					}
					caption += fmt.Sprintf(" (%s@%s", entry.User, host)
					if entry.Port > 0 && entry.Port != 22 {
						caption += fmt.Sprintf(":%d", entry.Port)
					}
					caption += ")"
				}
				ui.Text(c, caption).Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
				if widgets.Button(c, colors, label("导入", "Import"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					d, err := entry.Draft()
					if err != nil {
						e.ForwardError = err.Error()
					} else {
						e.Open, e.Draft, e.Error = true, d, ""
						e.ForwardError = ""
					}
				}
			})
		}
	})
}

// viewForwardsCard is the installed forwards card: a needsMachine gate, one row
// per rule with a state tag and start/stop/remove, then the inline add row.
func viewForwardsCard(c *ui.Context, m Model, e *MachineEditor, border ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	s := m.Machines.Snapshot()
	muted := c.Theme().TextMuted
	toolsCard(c, "remote-forwards-card", border, func() {
		cardTitle(c, label("端口转发", "Port forwards"), nil)
		if e.ForwardError != "" {
			ui.Text(c, e.ForwardError).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Danger)
		}
		if s.CurrentID == "" {
			ui.Text(c, label("先设为当前机器再管理转发", "Set a machine active to manage forwards")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		count := 0
		for _, f := range s.Tools.Forwards {
			if f.MachineID != "" && f.MachineID != s.CurrentID {
				continue
			}
			count++
			f := f
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				dir := "L"
				if f.Direction == "reverse" {
					dir = "R"
				}
				ui.Text(c, fmt.Sprintf("%s :%d → %s:%d", dir, f.ListenPort, f.TargetHost, f.TargetPort)).Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0)
				state := label("已停止", "Stopped")
				action, actionLabel := "start", label("启动", "Start")
				if f.Active {
					state, action, actionLabel = label("运行中", "Running"), "stop", label("停止", "Stop")
				}
				ui.Text(c, state).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if widgets.Button(c, colors, actionLabel, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					m.Machines.ForwardAction(f.ID, action)
				}
				if widgets.Button(c, colors, label("移除", "Remove"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					m.Machines.ForwardAction(f.ID, "remove")
				}
			})
		}
		if count == 0 {
			ui.Text(c, label("当前机器没有转发规则", "No forwards on the active machine")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
			labels := []string{label("本地 → 远程", "Local → remote"), label("远程 → 本地", "Remote → local")}
			active := 0
			if e.Forward.Direction == "reverse" {
				active = 1
			}
			if picked := remoteGatewaySegmented(c, "forward-direction", labels, active, s.Busy); picked >= 0 {
				e.Forward.Direction = []string{"local", "reverse"}[picked]
			}
			ui.TextInput(c, &e.Forward.ListenPort).Width(90).Height(32).Radius(8).Border(.5, border).Padding(0, 10).FontSize(fonttheme.InterfaceSize(13)).Placeholder(label("监听端口", "Listen port")).Disabled(s.Busy)
			ui.TextInput(c, &e.Forward.TargetHost).Width(130).Height(32).Radius(8).Border(.5, border).Padding(0, 10).FontSize(fonttheme.InterfaceSize(13)).Placeholder(label("目标主机", "Target host")).Disabled(s.Busy)
			ui.TextInput(c, &e.Forward.TargetPort).Width(90).Height(32).Radius(8).Border(.5, border).Padding(0, 10).FontSize(fonttheme.InterfaceSize(13)).Placeholder(label("目标端口", "Target port")).Disabled(s.Busy)
			ui.Checkbox(c, &e.Forward.AutoStart, label("自动启动", "Auto start")).Disabled(s.Busy || e.Forward.Direction == "reverse")
			if widgets.Button(c, colors, label("添加转发", "Add forward"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy || strings.TrimSpace(e.Forward.ListenPort) == "").Clicked() {
				d := e.Forward
				if d.Direction == "reverse" {
					d.AutoStart = false
				}
				if err := m.Machines.DefineForward(d); err != nil {
					e.ForwardError = err.Error()
				} else {
					e.ForwardError = ""
					e.Forward = remotestore.ForwardDraft{}
				}
			}
		})
	})
}

// viewAuditCard is the installed audit card: a title, a show/hide toggle and
// the log tail (or its empty/disabled states).
func viewAuditCard(c *ui.Context, m Model, e *MachineEditor, border ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	s := m.Machines.Snapshot()
	muted := c.Theme().TextMuted
	toolsCard(c, "remote-audit-card", border, func() {
		cardTitle(c, label("操作审计", "Audit log"), func() {
			caption := label("查看审计日志", "Show audit log")
			if e.ShowAudit {
				caption = label("收起审计日志", "Hide audit log")
			}
			if widgets.Button(c, colors, caption, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				e.ShowAudit = !e.ShowAudit
				if e.ShowAudit {
					m.Machines.LoadAudit()
				}
			}
		})
		if e.ShowAudit && s.Tools.AuditLoaded {
			switch {
			case !s.Tools.AuditEnabled:
				ui.Text(c, label("审计日志未开启", "Audit log is disabled")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			case len(s.Tools.AuditLines) == 0:
				ui.Text(c, label("暂无审计记录", "No audit entries yet")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			default:
				ui.Scroll(c).Height(180).FillWidth().Children(func() {
					ui.Text(c, strings.Join(s.Tools.AuditLines, "\n")).Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				})
			}
		}
	})
}
