package remote

import (
	"fmt"
	"strconv"
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
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

// Status colours of a machine row, shared with the dot painted in the card.
var (
	machineOnline  = ui.RGB(61, 153, 112)
	machineIdle    = ui.RGB(224, 162, 60)
	machineOffline = ui.RGBA(120, 120, 120, .55)
)

func viewMachines(c *ui.Context, m Model) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
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
	muted, border := c.Theme().TextMuted, c.Theme().Border
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label("远程机器", "Remote machines")).Bold().Grow(1).MinWidth(0)
			if len(s.Machines) > 0 {
				ui.Text(c, fmt.Sprintf("%d", len(s.Machines))).FontSize(fonttheme.InterfaceSize(11)).FontFeatures("tnum").TextColor(muted)
			}
			if widgets.Button(c, colors, label("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(s.Busy || !m.HarnessReady).Clicked() {
				m.Machines.Refresh()
			}
			if widgets.Button(c, colors, label("新增", "Add"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(s.Busy || !m.HarnessReady).Clicked() {
				*e = MachineEditor{Open: true, Draft: remotestore.NewDraft()}
			}
		})
		if s.Notice != "" {
			ui.Text(c, s.Notice).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Danger)
		}
		if s.Busy && len(s.Machines) == 0 {
			ui.Text(c, label("正在读取机器列表…", "Loading machines…")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if s.Loaded && len(s.Machines) == 0 && s.Error == "" && !s.Busy {
			emptyMachineCard(c, muted, border)
		}
		for _, machine := range s.Machines {
			machineCard(c, m, s, e, machine, muted, border)
			if e.PendingRemove == machine.ID {
				removeMachineConfirm(c, m, s, e, machine, muted)
			}
		}
		if e.Open {
			machineEditor(c, m, s, e, muted)
		}
		viewMachineTools(c, m)
	})
}

// emptyMachineCard is the empty state: it says what a machine is and offers the
// one action that changes it.
func emptyMachineCard(c *ui.Context, muted, border ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(20, 16).Radius(10).Border(.5, border).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label("还没有远程机器", "No remote machines yet")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600)
		ui.Text(c, label("新增一台后可以连接、浏览它的目录并把镜像作为工作区。", "Add one to connect, browse its directories and use a mirror as your workspace.")).
			FillWidth().TextAlign(ui.Center).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	})
}

// machineCard is one machine row: a status dot, the identity, the connection
// facts and an overflow menu holding every action.
func machineCard(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, machine remotestore.Machine, muted, border ui.Color) {
	current := machine.ID == s.CurrentID
	status := machineOffline
	switch {
	case current && s.Connected:
		status = machineOnline
	case current:
		status = machineIdle
	}
	card := ui.Row(c.Key("remote-machine-"+machine.ID)).FillWidth().MinWidth(0).Gap(12).Padding(12, 14).Radius(10).Border(.5, border).AlignItems(ui.Center)
	if card.Hovered() {
		card.Background(c.Theme().SurfaceHover)
	}
	title := machine.Name
	if title == "" {
		title = machine.Host
	}
	card.ContextMenu(func(menu *ui.Menu) { machineMenu(c, menu, m, s, e, machine) })
	card.Children(func() {
		ui.Box(c).Size(8, 8).Shrink(0).Radius(4).Background(status)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).MinWidth(0).SingleLine().Tooltip(title).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600)
				if current {
					ui.Text(c, label("当前", "Current")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				if current && s.Connected {
					ui.Text(c, label("已连接", "Connected")).FontSize(fonttheme.InterfaceSize(11)).TextColor(machineOnline)
				}
			})
			ui.Text(c, machineAddress(machine)).FillWidth().MinWidth(0).SingleLine().Tooltip(machineAddress(machine)).
				Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if meta := machineMeta(machine); meta != "" {
				ui.Text(c, meta).FillWidth().MinWidth(0).SingleLine().Tooltip(meta).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
		moreLabel := label("更多操作", "More actions")
		more := ui.Button(c, "").Label(moreLabel).Size(24, 24).Shrink(0).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(moreLabel)
		more.Children(func() { ui.Text(c, "⋯").FontSize(fonttheme.InterfaceSize(14)).TextColor(muted) })
		if more.Hovered() {
			more.Background(c.Theme().SurfaceHover)
		}
		more.Menu(func(menu *ui.Menu) { machineMenu(c, menu, m, s, e, machine) })
	})
}

// machineAddress is the user@host:port line of a card.
func machineAddress(machine remotestore.Machine) string {
	user := machine.Username
	if user == "" {
		user = "-"
	}
	return user + "@" + machine.Host + ":" + strconv.Itoa(machine.Port)
}

// machineMeta is the optional third line: latency, last connection and workspace.
func machineMeta(machine remotestore.Machine) string {
	parts := []string{}
	if machine.LatencyMS > 0 {
		parts = append(parts, fmt.Sprintf("%.0f ms", machine.LatencyMS))
	}
	if machine.LastConnectedAt != "" {
		stamp := strings.ReplaceAll(machine.LastConnectedAt, "T", " ")
		if len(stamp) > 16 {
			stamp = stamp[:16]
		}
		parts = append(parts, label("最近连接 ", "Last connected ")+stamp)
	}
	if machine.Workspace != "" {
		parts = append(parts, machine.Workspace)
	}
	if machine.PasswordSet {
		parts = append(parts, label("已保存密码", "Password saved"))
	}
	return strings.Join(parts, " · ")
}

// machineMenu holds every machine action, so the row keeps one control.
func machineMenu(c *ui.Context, menu *ui.Menu, m Model, s remotestore.Snapshot, e *MachineEditor, machine remotestore.Machine) {
	if menu.Item(label("选择此机器", "Select this machine")).Disabled(s.Busy).Chosen() {
		m.Machines.Select(machine.ID)
	}
	if menu.Item(label("测试连接", "Test connection")).Disabled(s.Busy).Chosen() {
		m.Machines.TestMachine(machine.ID)
	}
	if machine.ID == s.CurrentID {
		caption := label("连接 / 重连", "Connect / reconnect")
		if s.Connected {
			caption = label("重新连接", "Reconnect")
		}
		if menu.Item(caption).Disabled(s.Busy).Chosen() {
			m.Machines.Connect()
		}
	}
	menu.Separator()
	if menu.Item(label("编辑…", "Edit…")).Disabled(s.Busy).Chosen() {
		*e = MachineEditor{Open: true, Draft: machine.EditDraft()}
	}
	if menu.Item(label("移除…", "Remove…")).Disabled(s.Busy).Chosen() {
		e.PendingRemove = machine.ID
		e.RemovalGeneration = s.Generation
	}
}

// removeMachineConfirm asks before dropping a machine and its saved credentials.
func removeMachineConfirm(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, machine remotestore.Machine, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := machine.Name
	if title == "" {
		title = machine.Host
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Padding(0, 14).Children(func() {
		ui.Text(c, label("移除机器「"+title+"」？本机保存的凭据也会移除；不会删除远程文件。", "Remove machine \""+title+"\"? Locally saved credentials are removed too; remote files are kept.")).
			Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy || e.Removing).Clicked() {
			e.PendingRemove = ""
		}
		if widgets.Button(c, colors, label("确认移除", "Confirm removal"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
			e.Removing = m.Machines.RemoveAtGeneration(machine.ID, e.RemovalGeneration)
		}
	})
}

// machineEditor is the grouped form: address, credentials, labels. Field errors
// are computed locally so a wrong port or a duplicate name never needs a round
// trip to the host.
func machineEditor(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	draft := &e.Draft
	problem := machineDraftProblem(*draft, s.Machines)
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Padding(14).Radius(10).Border(.5, c.Theme().Border).Children(func() {
		ui.Text(c, label("机器信息", "Machine")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(500)
		ui.Text(c, label("密码留空会保留已有凭据；私钥只填写文件路径。", "A blank password keeps saved credentials; private keys are referenced by file path.")).
			FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		editorGroup(c, label("标签", "Label"), func() {
			editorField(c, label("名称", "Name"), &draft.Name, s.Busy, label("例如 构建机", "e.g. build box"))
			editorField(c, label("工作区路径", "Workspace path"), &draft.Workspace, s.Busy, "/srv/app")
		})
		editorGroup(c, label("地址", "Address"), func() {
			editorField(c, label("主机", "Host"), &draft.Host, s.Busy, "10.0.0.12")
			editorField(c, label("端口", "Port"), &draft.Port, s.Busy, "22")
		})
		editorGroup(c, label("认证", "Credentials"), func() {
			editorField(c, label("用户", "Username"), &draft.Username, s.Busy, "deploy")
			editorField(c, label("私钥文件路径", "Private key path"), &draft.PrivateKeyPath, s.Busy, "~/.ssh/id_ed25519")
		})
		viewConnectionOptions(c, e, s.Busy)
		if problem != "" {
			ui.Text(c, problem).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				e.Open = false
				e.Draft = remotestore.Draft{}
				e.Error = ""
			}
			ui.Box(c).Grow(1)
			if widgets.Button(c, colors, label("测试表单", "Test draft"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(s.Busy || problem != "").Clicked() {
				if err := m.Machines.TestDraft(e.Draft); err != nil {
					e.Error = err.Error()
				} else {
					e.Error = ""
				}
			}
			if ui.PrimaryButton(c, label("保存", "Save")).Disabled(s.Busy || problem != "").Clicked() {
				if err := m.Machines.Save(e.Draft); err != nil {
					e.Error = err.Error()
				} else {
					e.Error = ""
					e.Saving = true
				}
			}
		})
	})
}

// editorGroup is one labelled block of the machine form.
func editorGroup(c *ui.Context, title string, body func()) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
		ui.Text(c, title).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().TextMuted)
		body()
	})
}

// editorField is one labelled input row of the machine form.
func editorField(c *ui.Context, title string, value *string, disabled bool, placeholder string) {
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).Width(120).Shrink(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		ui.TextInput(c, value).Grow(1).MinWidth(160).Disabled(disabled).Placeholder(placeholder)
	})
}

// machineDraftProblem is the form's local validation: the things the host would
// otherwise reject, said before the save round trip.
func machineDraftProblem(draft remotestore.Draft, machines []remotestore.Machine) string {
	if strings.TrimSpace(draft.Host) == "" {
		return label("请填写主机地址。", "Enter a host address.")
	}
	if port, err := strconv.Atoi(strings.TrimSpace(draft.Port)); err != nil || port < 1 || port > 65535 {
		return label("端口需为 1–65535 之间的整数。", "The port must be an integer between 1 and 65535.")
	}
	name := strings.TrimSpace(draft.Name)
	if name != "" {
		for _, machine := range machines {
			if machine.ID != draft.ID && strings.EqualFold(strings.TrimSpace(machine.Name), name) {
				return label("名称已被另一台机器使用。", "Another machine already uses this name.")
			}
		}
	}
	return ""
}
