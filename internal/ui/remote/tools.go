package remote

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

func viewMachineTools(c *ui.Context, m Model) {
	s, e := m.Machines.Snapshot(), m.MachineEditor
	v := s.Tools
	if !v.Loaded && !s.Busy && m.HarnessReady && m.RemoteWorkspaceEnabled {
		m.Machines.LoadTools()
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, label("SSH 设置与端口转发", "SSH settings and port forwards")).Bold().Grow(1)
		if ui.Button(c, label("刷新 SSH 设置", "Refresh SSH settings")).Disabled(s.Busy || !m.HarnessReady).Clicked() {
			m.Machines.LoadTools()
		}
	})
	if v.Error != "" {
		ui.Text(c, v.Error)
	}
	if v.Notice != "" {
		ui.Text(c, v.Notice)
	}
	if s.Status.Workspace != "" {
		ui.Text(c, label("远程工作区：", "Remote workspace: ")+s.Status.Workspace)
	}
	if s.Status.LocalMirror != "" {
		ui.Text(c, label("本地镜像：", "Local mirror: ")+s.Status.LocalMirror)
	}
	if s.CurrentID != "" {
		key := label("主机密钥尚未登记", "Host key is not recorded")
		if s.Status.HostKeyKnown {
			key = label("主机密钥已登记", "Host key is recorded")
		}
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, key).Grow(1)
			if ui.Button(c, label("忘记主机密钥", "Forget host key")).Disabled(s.Busy).Clicked() {
				e.PendingForget = s.CurrentID
			}
		})
		if e.PendingForget == s.CurrentID {
			ui.Text(c, label("忘记后需要重新核对服务器指纹。确认清除此机器的记录？", "Verify the server fingerprint again after forgetting its key. Clear this machine's key record?"))
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, label("确认忘记", "Confirm forget")).Disabled(s.Busy).Clicked() {
					m.Machines.ForgetHostKey()
					e.PendingForget = ""
				}
				if ui.Button(c, label("取消", "Cancel")).Clicked() {
					e.PendingForget = ""
				}
			})
		}
	}
	ui.Text(c, label("从 SSH 配置导入", "Import from SSH config")).Bold()
	if e.ForwardError != "" {
		ui.Text(c, e.ForwardError)
	}
	if v.Loaded && len(v.SSHEntries) == 0 {
		ui.Text(c, label("没有可导入的主机配置。", "No importable SSH host entries."))
	}
	for _, entry := range v.SSHEntries {
		entry := entry
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, entry.Host+" · "+entry.User+"@"+entry.HostName).Grow(1)
			if ui.Button(c, label("导入表单", "Import to form")).Disabled(s.Busy).Clicked() {
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
	ui.Text(c, label("端口转发", "Port forwards")).Bold()
	if s.CurrentID == "" {
		ui.Text(c, label("先选择远程机器。", "Select a remote machine first."))
	} else {
		count := 0
		for _, f := range v.Forwards {
			if f.MachineID != "" && f.MachineID != s.CurrentID {
				continue
			}
			count++
			f := f
			ui.Row(c).Gap(8).Children(func() {
				state := label("已停止", "Stopped")
				action, actionLabel := "start", label("启动", "Start")
				if f.Active {
					state, action, actionLabel = label("运行中", "Running"), "stop", label("停止", "Stop")
				}
				ui.Text(c, fmt.Sprintf("%s :%d → %s:%d · %s", f.Direction, f.ListenPort, f.TargetHost, f.TargetPort, state)).Grow(1)
				if ui.Button(c, actionLabel).Disabled(s.Busy).Clicked() {
					m.Machines.ForwardAction(f.ID, action)
				}
				if ui.Button(c, label("移除转发", "Remove forward")).Disabled(s.Busy).Clicked() {
					m.Machines.ForwardAction(f.ID, "remove")
				}
			})
		}
		if count == 0 {
			ui.Text(c, label("此机器暂无转发。", "No forwards for this machine."))
		}
		ui.Row(c).Gap(6).Children(func() {
			for _, choice := range []struct{ id, title string }{{"local", label("本地转发", "Local")}, {"reverse", label("反向转发", "Reverse")}} {
				title := choice.title
				current := e.Forward.Direction
				if current == "" {
					current = "local"
				}
				if current == choice.id {
					title = "✓ " + title
				}
				if ui.Button(c, title).Disabled(s.Busy).Clicked() {
					e.Forward.Direction = choice.id
				}
			}
		})
		ui.Text(c, label("反向转发会将本地目标端口提供给远端访问。", "Reverse forwarding makes the local target port accessible from the remote machine."))
		for _, f := range []struct {
			title string
			value *string
		}{
			{label("监听端口", "Listen port"), &e.Forward.ListenPort},
			{label("目标主机（默认 127.0.0.1）", "Target host (default 127.0.0.1)"), &e.Forward.TargetHost},
			{label("目标端口（默认同监听）", "Target port (default same as listen)"), &e.Forward.TargetPort},
		} {
			ui.Row(c).Gap(8).Children(func() { ui.Text(c, f.title).Width(230); ui.TextInput(c, f.value).FillWidth().Disabled(s.Busy) })
		}
		ui.Checkbox(c, &e.Forward.AutoStart, label("连接后自动启动本地转发", "Start local forward automatically on connection")).Disabled(s.Busy || e.Forward.Direction == "reverse")
		if e.ForwardError != "" {
			ui.Text(c, e.ForwardError)
		}
		if ui.Button(c, label("添加转发定义", "Add forward definition")).Disabled(s.Busy).Clicked() {
			d := e.Forward
			if d.Direction == "reverse" {
				d.AutoStart = false
			}
			if err := m.Machines.DefineForward(d); err != nil {
				e.ForwardError = err.Error()
			} else {
				e.ForwardError = ""
			}
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, label("远程审计日志", "Remote audit log")).Bold().Grow(1)
		title := label("显示最近 100 行", "Show last 100 lines")
		if e.ShowAudit {
			title = label("隐藏日志", "Hide log")
		}
		if ui.Button(c, title).Disabled(s.Busy).Clicked() {
			e.ShowAudit = !e.ShowAudit
			if e.ShowAudit {
				m.Machines.LoadAudit()
			}
		}
	})
	if e.ShowAudit && v.AuditLoaded {
		if !v.AuditEnabled {
			ui.Text(c, label("远程审计日志未启用。", "Remote audit logging is disabled."))
		} else if len(v.AuditLines) == 0 {
			ui.Text(c, label("暂无审计记录。", "No audit entries."))
		} else {
			ui.Scroll(c).Height(180).Children(func() { ui.Text(c, strings.Join(v.AuditLines, "\n")).FontSize(fonttheme.InterfaceSize(11)) })
		}
	}
}
