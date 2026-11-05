package remote

import (
	"fmt"
	"strconv"
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/chrome"
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
	toolsCard(c, "remote-machines-card", border, func() {
		cardTitle(c, label("SSH 机器", "SSH machines"), func() {
			if len(s.Machines) > 0 {
				ui.Text(c, fmt.Sprintf("%d", len(s.Machines))).FontSize(fonttheme.InterfaceSize(11)).FontFeatures("tnum").TextColor(muted)
			}
			if widgets.Button(c, colors, label("添加机器", "Add machine"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy || !m.HarnessReady).Clicked() {
				if e.Open && e.Draft.ID == "" {
					e.Open = false
					return
				}
				*e = MachineEditor{Open: true, Draft: remotestore.NewDraft()}
			}
		})
		widgets.Notice(c, "machines:77", "", s.Notice != "", func() string { return s.Notice })
		if s.Error != "" {
			ui.Text(c, s.Error).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Danger)
		}
		if e.Open {
			machineEditor(c, m, s, e, muted, border)
		}
		if s.Loaded && len(s.Machines) == 0 && s.Error == "" && !s.Busy && !e.Open {
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(12, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label("还没有保存的 SSH 机器", "No saved SSH machines yet")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600)
				ui.Text(c, label("添加一台机器后，新建工作区时可以选择「远程」标签页。", "Once a machine is added, the \"Remote\" tab appears when creating a workspace.")).
					FillWidth().TextAlign(ui.Center).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			})
		}
		for _, machine := range s.Machines {
			machineRow(c, m, s, e, machine, muted)
			if e.PendingRemove == machine.ID {
				removeMachineConfirm(c, m, s, e, machine, muted)
			}
		}
	})
}

// machineRow is one flat list row of the installed machine card: the identity
// line, the mono facts and inline ghost actions.
func machineRow(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, machine remotestore.Machine, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	current := machine.ID == s.CurrentID
	row := ui.Row(c.Key("remote-machine-"+machine.ID)).FillWidth().MinWidth(0).Gap(8).Padding(8, 0).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(c.Theme().SurfaceHover)
	}
	title := machine.Name
	if title == "" {
		title = machine.Host
	}
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).MinWidth(0).SingleLine().Tooltip(title).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600)
				if current {
					ui.Text(c, label("当前", "Active")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				if machine.PasswordSet {
					ui.Text(c, label("已存密码", "Password saved")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			})
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, machineAddress(machine)).MinWidth(0).SingleLine().Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if meta := machineMeta(machine); meta != "" {
					ui.Text(c, meta).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			})
		})
		ui.Row(c).Shrink(0).Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
			if !current && widgets.Button(c, colors, label("设为当前", "Set active"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				m.Machines.Select(machine.ID)
			}
			if widgets.Button(c, colors, label("测试连接", "Test connection"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				m.Machines.TestMachine(machine.ID)
			}
			if widgets.Button(c, colors, label("编辑", "Edit"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				*e = MachineEditor{Open: true, Draft: machine.EditDraft()}
			}
			if widgets.Button(c, colors, label("删除", "Delete"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				e.PendingRemove = machine.ID
				e.RemovalGeneration = s.Generation
			}
		})
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
	return strings.Join(parts, " · ")
}

// removeMachineConfirm asks before dropping a machine and its saved credentials.
func removeMachineConfirm(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, machine remotestore.Machine, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := machine.Name
	if title == "" {
		title = machine.Host
	}
	open := true
	ui.DialogBaseWithOptions(c.Key("remove-remote-machine"), &open, ui.DialogOptions{DismissDisabled: s.Busy || e.Removing, OnDismiss: func() { e.PendingRemove = "" }}, func(backdrop, panel ui.Element) {
		_, height := c.Size()
		chrome.DialogSurface(c, backdrop, panel, colors, 440, 20).MaxHeight(height - 24).Gap(14).Children(func() {
			ui.Text(c, label("移除远程机器？", "Remove remote machine?")).Bold()
			ui.Text(c, label("删除机器「"+title+"」？本机保存的凭据会一并移除。", "Delete machine \""+title+"\"? Locally stored credentials are removed too.")).
				FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			if s.Error != "" {
				ui.Text(c, s.Error).FillWidth().TextColor(c.Theme().Danger).Selectable()
			}
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy || e.Removing).Clicked() {
					e.PendingRemove = ""
				}
				if widgets.Button(c, colors, label("确认移除", "Confirm removal"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
					e.Removing = m.Machines.RemoveAtGeneration(machine.ID, e.RemovalGeneration)
				}
			})
		})
	})
}

// machineEditor is the installed flat form: labelled fields in the plugin's
// order — identity, 认证方式 (secrets, fingerprint policy, toggles), 跳板机 —
// then the test/cancel/save row. Local validation keeps bad drafts off the wire.
func machineEditor(c *ui.Context, m Model, s remotestore.Snapshot, e *MachineEditor, muted, border ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	draft := &e.Draft
	if draft.Options == nil {
		draft.Options = &remotestore.ConnectionDraft{ProxyPort: "22"}
	}
	o := draft.Options
	problem := machineDraftProblem(*draft, s.Machines)
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(10, 0).Children(func() {
		formTitle := label("编辑 SSH 机器", "Edit SSH machine")
		if strings.TrimSpace(draft.ID) == "" {
			formTitle = label("添加 SSH 机器", "Add SSH machine")
		}
		ui.Text(c, formTitle).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600)
		editorField(c, label("名称", "Name"), &draft.Name, s.Busy, label("可选，默认同主机", "Optional; defaults to host"), false, border)
		editorField(c, label("主机", "Host"), &draft.Host, s.Busy, label("example.com 或 192.168.1.10", "example.com or 192.168.1.10"), false, border)
		editorField(c, label("端口", "Port"), &draft.Port, s.Busy, "22", false, border)
		editorField(c, label("用户名", "Username"), &draft.Username, s.Busy, "", false, border)
		ui.Text(c, label("认证方式", "Authentication")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600)
		passwordPlaceholder := label("SSH 密码", "SSH password")
		if draft.ID != "" {
			passwordPlaceholder = label("留空保持不变", "Leave blank to keep current")
		}
		editorField(c, label("密码", "Password"), &o.Password, s.Busy, passwordPlaceholder, true, border)
		editorField(c, label("私钥路径", "Private key path"), &draft.PrivateKeyPath, s.Busy, "~/.ssh/id_ed25519", false, border)
		editorField(c, label("私钥口令", "Key passphrase"), &o.Passphrase, s.Busy, label("私钥加密时填写", "If the key is encrypted"), true, border)
		ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, label("主机指纹校验", "Host key verification")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			labels := []string{label("首次信任（TOFU）", "Trust on first use (TOFU)"), label("严格校验", "Strict verify"), label("不校验", "Disabled")}
			active := 0
			switch o.HostKeyMode {
			case "verify":
				active = 1
			case "off":
				active = 2
			}
			if picked := segmentedRow(c, "machine-hostkey", labels, active, s.Busy); picked >= 0 {
				o.HostKeyMode = []string{"", "verify", "off"}[picked]
			}
		})
		editorField(c, label("默认远程目录", "Default remote path"), &draft.Workspace, s.Busy, label("可选，如 /srv/app", "Optional, e.g. /srv/app"), false, border)
		ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
			ui.Checkbox(c, &o.UseAgent, label("使用 SSH Agent", "Use SSH agent")).Disabled(s.Busy)
			ui.Checkbox(c, &o.KeyboardInteractive, label("键盘交互（OTP/二次验证）", "Keyboard-interactive (OTP/2FA)")).Disabled(s.Busy)
			ui.Switch(c, &o.EncryptPassword).Label(label("存入系统钥匙串", "Store in system keychain")).Disabled(s.Busy)
		})
		ui.Text(c, label("跳板机", "Jump host")).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600)
		editorField(c, label("跳板主机", "Jump host"), &o.ProxyHost, s.Busy, "", false, border)
		editorField(c, label("跳板端口", "Jump port"), &o.ProxyPort, s.Busy, "22", false, border)
		editorField(c, label("跳板用户名", "Jump username"), &o.ProxyUsername, s.Busy, "", false, border)
		editorField(c, label("跳板密码", "Jump password"), &o.ProxyPassword, s.Busy, "", true, border)
		editorField(c, label("跳板私钥路径", "Jump key path"), &o.ProxyKeyPath, s.Busy, "", false, border)
		if problem != "" {
			ui.Text(c, problem).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(c.Theme().Danger)
		}
		ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
			if widgets.Button(c, colors, label("测试连接", "Test connection"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy || problem != "").Clicked() {
				if err := m.Machines.TestDraft(e.Draft); err != nil {
					e.Error = err.Error()
				} else {
					e.Error = ""
				}
			}
			if widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(s.Busy).Clicked() {
				e.Open = false
				e.Draft = remotestore.Draft{}
				e.Error = ""
			}
			if widgets.Button(c, colors, label("保存", "Save"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(s.Busy || problem != "").Clicked() {
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

// segmentedRow is the installed SettingsSelect block: equal-width segments,
// the chosen one inverted.
func segmentedRow(c *ui.Context, key string, labels []string, active int, disabled bool) int {
	const height float32 = 28
	root := ui.Row(c.Key(key)).FillWidth().MinWidth(0).MaxWidth(340).Height(height).Shrink(0).Gap(8).AlignItems(ui.Stretch)
	picked := -1
	root.Children(func() {
		width := (root.Bounds().W - float32(len(labels)-1)*8) / float32(len(labels))
		if width < 60 {
			width = 60
		}
		for index, caption := range labels {
			i, text := index, caption
			segment := ui.Button(c.Key(key+"-"+strconv.Itoa(i)), "").Label(text).
				Width(width).Height(height).Padding(0).Radius(8).
				BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center)
			if disabled {
				segment = segment.Disabled(true).Opacity(0.5)
			}
			labelColor := c.Theme().Text
			if i == active {
				segment.Background(labelColor)
				labelColor = c.Theme().Background
			}
			segment.Children(func() {
				ui.Text(c, text).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(labelColor)
			})
			if segment.Clicked() && !disabled {
				picked = i
			}
		}
	})
	return picked
}

// editorField is one label-above input of the installed machine form.
func editorField(c *ui.Context, title string, value *string, disabled bool, placeholder string, password bool, border ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Text(c, title).FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		input := ui.TextInput(c, value).FillWidth().MinWidth(0).Height(32).Radius(8).Border(.5, border).Padding(0, 10).FontSize(fonttheme.InterfaceSize(13)).Disabled(disabled).Placeholder(placeholder)
		if password {
			input.Password()
		}
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
