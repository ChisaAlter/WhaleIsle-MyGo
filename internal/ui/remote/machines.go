package remote

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/i18n"
)

// MachineEditor keeps transient form values in this window; the host owns saved credentials.
type MachineEditor struct {
	Open              bool
	Draft             remotestore.Draft
	Error             string
	PendingRemove     string
	RemovalGeneration uint64
	Removing          bool
	Saving            bool
	Forward           remotestore.ForwardDraft
	ForwardError      string
	PendingForget     string
	ShowAudit         bool
}

func label(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}
func viewMachines(c *ui.Context, m Model) {
	if m.Machines == nil || m.MachineEditor == nil {
		return
	}
	s, e := m.Machines.Snapshot(), m.MachineEditor
	e.syncRemoval(s)
	if !s.Loaded && !s.Busy && s.Error == "" && m.HarnessReady && m.RemoteWorkspaceEnabled {
		m.Machines.Refresh()
	}
	if e.Saving && !s.Busy {
		e.Saving = false
		if s.MutationApplied {
			e.Open = false
			e.Draft = remotestore.Draft{}
		}
	}
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			ui.Text(c, label("远程机器", "Remote machines")).Bold().Grow(1).MinWidth(0)
			if ui.Button(c, label("刷新", "Refresh")).Disabled(s.Busy || !m.HarnessReady).Clicked() {
				m.Machines.Refresh()
			}
			if ui.Button(c, label("新增", "Add")).Disabled(s.Busy || !m.HarnessReady).Clicked() {
				*e = MachineEditor{Open: true, Draft: remotestore.NewDraft()}
			}
		})
		if s.Busy {
			ui.Text(c, label("正在处理…", "Working…"))
		}
		if s.Error != "" {
			ui.Text(c, s.Error)
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice)
		}
		if s.Loaded && len(s.Machines) == 0 && s.Error == "" {
			ui.Text(c, label("尚无远程机器。新增后可选择并连接。", "No remote machines. Add one, then select and connect."))
		}
		if s.CurrentID != "" {
			status := label("未连接", "Disconnected")
			if s.Connected {
				status = label("已连接", "Connected")
			}
			ui.Row(c).Gap(8).Wrap().Children(func() {
				ui.Text(c, status).Grow(1).MinWidth(0)
				if ui.Button(c, label("连接 / 重连", "Connect / reconnect")).Disabled(s.Busy).Clicked() {
					m.Machines.Connect()
				}
				if ui.Button(c, label("清除当前选择", "Clear current")).Disabled(s.Busy).Clicked() {
					m.Machines.Select("")
				}
			})
		}
		for _, machine := range s.Machines {
			machine := machine
			title := machine.Name
			if title == "" {
				title = machine.Host
			}
			if machine.ID == s.CurrentID {
				title = "✓ " + title
			}
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(8, 0).Children(func() {
				ui.Text(c, title+" · "+machine.Username+"@"+machine.Host+":"+strconv.Itoa(machine.Port)).FillWidth().MinWidth(0)
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if ui.Button(c, label("选择", "Select")).Disabled(s.Busy).Clicked() {
						m.Machines.Select(machine.ID)
					}
					if ui.Button(c, label("测试连接", "Test connection")).Disabled(s.Busy).Clicked() {
						m.Machines.TestMachine(machine.ID)
					}
					if ui.Button(c, label("编辑", "Edit")).Disabled(s.Busy).Clicked() {
						*e = MachineEditor{Open: true, Draft: machine.EditDraft()}
					}
					if ui.Button(c, label("移除", "Remove")).Disabled(s.Busy).Clicked() {
						e.PendingRemove = machine.ID
						e.RemovalGeneration = s.Generation
					}
				})
			})
			if e.PendingRemove == machine.ID {
				ui.Row(c).Gap(8).Wrap().Children(func() {
					ui.Text(c, label("移除机器「"+title+"」？本机保存的凭据也会移除；不会删除远程文件。", "Remove machine \""+title+"\"? Locally saved credentials will also be removed; remote files are retained.")).Grow(1).MinWidth(0)
					if ui.Button(c, label("确认移除", "Confirm removal")).Disabled(s.Busy).Clicked() {
						e.Removing = m.Machines.RemoveAtGeneration(machine.ID, e.RemovalGeneration)
					}
					if ui.Button(c, label("取消", "Cancel")).Disabled(s.Busy || e.Removing).Clicked() {
						e.PendingRemove = ""
					}
				})
			}
		}
		if e.Open {
			ui.Text(c, label("密码留空会保留已有凭据；私钥只填写文件路径。", "Blank passwords keep saved credentials; private keys are referenced by file path."))
			for _, f := range []struct {
				title string
				value *string
			}{
				{label("名称", "Name"), &e.Draft.Name}, {label("主机", "Host"), &e.Draft.Host}, {label("端口", "Port"), &e.Draft.Port},
				{label("用户", "Username"), &e.Draft.Username}, {label("私钥文件路径", "Private key path"), &e.Draft.PrivateKeyPath}, {label("工作区路径", "Workspace path"), &e.Draft.Workspace},
			} {
				ui.Row(c).Gap(8).Wrap().Children(func() { ui.Text(c, f.title).Width(130); ui.TextInput(c, f.value).Grow(1).MinWidth(0).Disabled(s.Busy) })
			}
			viewConnectionOptions(c, e, s.Busy)
			if e.Error != "" {
				ui.Text(c, e.Error)
			}
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if ui.Button(c, label("测试表单", "Test draft")).Disabled(s.Busy).Clicked() {
					if err := m.Machines.TestDraft(e.Draft); err != nil {
						e.Error = err.Error()
					} else {
						e.Error = ""
					}
				}
				if ui.Button(c, label("保存", "Save")).Disabled(s.Busy).Clicked() {
					if err := m.Machines.Save(e.Draft); err != nil {
						e.Error = err.Error()
					} else {
						e.Error = ""
						e.Saving = true
					}
				}
				if ui.Button(c, label("取消", "Cancel")).Disabled(s.Busy).Clicked() {
					e.Open = false
					e.Draft = remotestore.Draft{}
					e.Error = ""
				}
			})
		}
		viewMachineTools(c, m)
	})
}
