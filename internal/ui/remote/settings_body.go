package remote

import (
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func settingsView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	content := func() {
		ui.Column(c.Key("remote-settings-body")).FillWidth().MinWidth(0).Gap(16).Children(func() {
			if m.SettingsPage == "workspace" {
				ui.Text(c, label("远程工作区", "Remote workspace")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				ui.Text(c, label("管理远程机器、SSH 配置和端口转发。", "Manage remote machines, SSH settings and port forwards.")).FontSize(theme.InterfaceSize(14)).TextColor(muted)
				if !m.HarnessReady {
					ui.Text(c, label("连接 Host 后即可管理远程机器。", "Connect to a Host to manage remote machines.")).TextColor(muted)
					return
				}
				if !m.RemoteWorkspaceEnabled {
					ui.Text(c, label("远程工作区尚未启用。", "Remote workspace is disabled.")).TextColor(muted)
					if m.OnToggleWorkspace != nil && widgets.Button(c, colors, label("启用远程工作区", "Enable remote workspace"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
						m.OnToggleWorkspace(true)
					}
					return
				}
				viewMachines(c, m)
				return
			}
			ui.Text(c, label("远程", "Remote")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
			ui.Text(c, label("管理远程连接网关与消息渠道。", "Manage your remote gateway and messaging channels.")).FontSize(theme.InterfaceSize(14)).TextColor(muted)
			tabs := ui.Row(c.Key("remote-tabs")).FillWidth().Gap(8).Role(ui.RoleTabList)
			active := m.ActiveTab
			if active == nil {
				active = ui.Local(tabs, "active", func() string { return "gateway" })
			}
			if *active != "channels" || m.ChannelsContent == nil {
				*active = "gateway"
			}
			tabs.Children(func() {
				entries := []struct{ id, title string }{{"gateway", label("网关", "Gateway")}}
				if m.ChannelsContent != nil {
					entries = append(entries, struct{ id, title string }{"channels", label("消息渠道", "Channels")})
				}
				buttons := make([]ui.Element, len(entries))
				choose := func(index int) { *active = entries[index].id; buttons[index].Focus() }
				for index, tab := range entries {
					b := ui.Button(c.Key(tab.id), tab.title).Height(34).BorderWidth(0).Role(ui.RoleTab).Selected(*active == tab.id)
					buttons[index] = b
					b.OnShortcut(0, ui.KeyRight, func() { choose((index + 1) % len(entries)) })
					b.OnShortcut(0, ui.KeyLeft, func() { choose((index + len(entries) - 1) % len(entries)) })
					b.OnShortcut(0, ui.KeyHome, func() { choose(0) })
					b.OnShortcut(0, ui.KeyEnd, func() { choose(len(entries) - 1) })
					if *active == tab.id {
						b = b.Background(chrome.Color(colors.Field)).TextColor(fg)
					} else {
						b = b.Background(ui.Transparent).TextColor(muted)
					}
					if b.Clicked() {
						*active = tab.id
					}
				}
			})
			ui.Box(c).FillWidth().Height(1).Background(chrome.Color(colors.Border))
			if *active == "channels" {
				m.ChannelsContent(c)
				return
			}
			gatewaySettingsBody(c, m, fg, muted)
		})
	}
	if m.Embedded {
		content()
		return
	}
	ui.Scroll(c).Grow(1).MinHeight(0).Padding(20).Children(content)
}

func gatewaySettingsBody(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if m.OnSaveGateway == nil || m.GatewayEditor == nil {
		ui.Text(c, label("当前桌面连接未提供网关设置。", "Gateway settings are unavailable on this desktop connection.")).TextColor(muted)
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() {
		ui.Text(c, label("连接状态", "Connection status")).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
		status := m.PhoneRuntime.Status
		if !m.RemoteEnabled {
			status = label("已停用", "Disabled")
		}
		if status == "" {
			status = label("未启动", "Stopped")
		}
		ui.Text(c, status).FontSize(theme.InterfaceSize(13)).TextColor(muted)
		if m.PhoneRuntime.Message != "" && !strings.Contains(m.PhoneActionMsg, m.PhoneRuntime.Message) {
			ui.Text(c, m.PhoneRuntime.Message).TextColor(fg).Selectable()
		}
		if m.PhoneActionMsg != "" {
			ui.Text(c, m.PhoneActionMsg).TextColor(fg)
		}
		gatewaySettingsFields(c, m, fg, muted)
		e := m.GatewayEditor
		if m.GatewayProtocol == "chisacode-v2" {
			legacy := ui.Column(c.Key("legacy-remote-settings")).FillWidth().Gap(8)
			open := ui.Local(legacy, "open", func() bool { return false })
			legacy.Children(func() {
				if widgets.Button(c, colors, label("旧版配置", "Legacy configuration"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					*open = !*open
				}
				if *open {
					ui.Text(c, label("旧版中继令牌不参与当前设备配对。", "Legacy relay tokens are not used for current device pairing.")).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					if !m.LegacyRelayTokenSet {
						ui.Text(c, label("没有保存的旧版令牌。", "No legacy token is stored.")).TextColor(muted)
					}
					if m.LegacyRelayTokenSet && m.OnSetRelayToken != nil && widgets.Button(c, colors, label("清除旧版令牌", "Clear legacy token"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Clicked() {
						if err := m.OnSetRelayToken(""); err != nil {
							e.Error = err.Error()
						} else {
							e.Error = ""
						}
					}
				}
			})
		} else if m.OnSetRelayToken != nil {
			ui.Text(c, label("中继访问令牌", "Relay access token")).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
			gatewayReadRow(c, label("令牌状态", "Token status"), credLabel(m.HasRemoteRelayToken), fg, muted)
			ui.TextInput(c, &e.RelayToken).Password().FillWidth().Placeholder(label("输入新的中继令牌", "Enter a new relay token"))
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if widgets.Button(c, colors, label("保存令牌", "Save token"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(strings.TrimSpace(e.RelayToken) == "").Clicked() {
					if err := m.OnSetRelayToken(strings.TrimSpace(e.RelayToken)); err != nil {
						e.Error = err.Error()
					} else {
						e.RelayToken = ""
						e.Error = ""
					}
				}
				if widgets.Button(c, colors, label("清除令牌", "Clear token"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Disabled(!m.HasRemoteRelayToken).Clicked() {
					if err := m.OnSetRelayToken(""); err != nil {
						e.Error = err.Error()
					} else {
						e.RelayToken = ""
						e.Error = ""
					}
				}
			})
		}
		if m.OnRotateToken != nil {
			ui.Text(c, label("轮换连接令牌", "Rotate connection token")).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
			ui.Text(c, label("重新生成配对连接；已配对设备保持绑定。", "Renew the pairing offer; paired devices remain bound.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if widgets.Button(c, colors, label("轮换令牌", "Rotate token"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(m.PhoneRuntime.DeviceAction.Busy || m.PhoneRuntime.Status == "starting").Clicked() {
				if err := m.OnRotateToken(); err != nil {
					e.Error = err.Error()
				} else {
					e.Error = ""
				}
			}
		}
		if e.Error != "" {
			ui.Text(c, e.Error).TextColor(fg)
		}
	})
}
func gatewayReadRow(c *ui.Context, title, value string, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(6, 0).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(14)).TextColor(fg)
		ui.Text(c, value).FontSize(theme.InterfaceSize(13)).TextColor(muted)
	})
}

func gatewaySettingsFields(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	e := m.GatewayEditor
	current := config.RemoteGatewayPreferences{Mode: m.RemoteMode, Port: strconv.Itoa(m.RemotePort), BindAddress: m.RemoteBindAddress, RelayEndpoint: m.RemoteRelayEndpoint, RelayTLS: m.RemoteRelayUseTLS}
	if !e.Open {
		e.Expected, e.Draft = current, current
	}
	busy := m.PhoneRuntime.Status == "starting" || m.PhoneRuntime.DeviceAction.Busy
	// Original settings commits one row at a time. An unrelated text draft
	// must never hitchhike on a mode/bind change or a different row's Save.
	save := func(next config.RemoteGatewayPreferences) {
		if err := m.OnSaveGateway(current, next); err != nil {
			e.Error = err.Error()
			return
		}
		e.Error = ""
		if next.RelayEndpoint != current.RelayEndpoint {
			e.Draft.RelayEndpoint = next.RelayEndpoint
		}
		if next.Port != current.Port {
			e.Draft.Port = next.Port
		}
		e.Open = e.Draft.RelayEndpoint != next.RelayEndpoint || e.Draft.Port != next.Port
	}
	gatewaySettingRow(c, label("中继地址", "Relay endpoint"), label("使用中继 host[:port]；留空使用默认地址。", "Relay host[:port]; leave blank to use the default."), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.RelayEndpoint).Label("Gateway relay endpoint").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if widgets.Button(c, colors, label("保存", "Save"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
				next := current
				next.RelayEndpoint = e.Draft.RelayEndpoint
				save(next)
			}
			if widgets.Button(c, colors, label("使用默认地址", "Use default relay"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
				next := current
				next.RelayEndpoint = ""
				save(next)
			}
		})
	})
	gatewaySettingRow(c, label("连接模式", "Connection mode"), label("选择局域网或中继连接。", "Choose a LAN or relay connection."), fg, muted, func() {
		mode := current.Mode
		if ui.Select(c, &mode, []string{"lan", "relay"}).Label("Gateway connection mode").FillWidth().Disabled(busy).Changed() {
			next := current
			next.Mode = mode
			save(next)
		}
	})
	gatewaySettingRow(c, label("端口", "Port"), label("网关监听端口，范围 1024–65535。", "Gateway listen port, from 1024 to 65535."), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.Port).Label("Gateway port").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
		if widgets.Button(c, colors, label("保存", "Save"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
			next := current
			next.Port = e.Draft.Port
			save(next)
		}
	})
	gatewaySettingRow(c, label("监听范围", "Bind scope"), label("127.0.0.1、0.0.0.0 或本机 IPv4 地址。", "127.0.0.1, 0.0.0.0, or a local IPv4 address."), fg, muted, func() {
		options := []string{"0.0.0.0", "127.0.0.1"}
		for _, address := range append(append([]string(nil), m.PhoneRuntime.BindAddresses...), current.BindAddress) {
			present := false
			for _, option := range options {
				if option == address {
					present = true
					break
				}
			}
			if !present && address != "" {
				options = append(options, address)
			}
		}
		bind := current.BindAddress
		if bind == "" {
			bind = "0.0.0.0"
		}
		selected := gatewayBindLabel(bind, m.PhoneRuntime.BindAddresses)
		labels := make([]string, len(options))
		values := make(map[string]string, len(options))
		for i, address := range options {
			labels[i] = gatewayBindLabel(address, m.PhoneRuntime.BindAddresses)
			values[labels[i]] = address
		}
		if ui.Select(c, &selected, labels).Label("Gateway bind address").FillWidth().Disabled(busy).Changed() {
			if address, ok := values[selected]; ok && address != bind {
				next := current
				next.BindAddress = address
				save(next)
			}
		}
	})
	gatewayReadRow(c, label("局域网传输", "LAN transport"), label("局域网连接使用 TLS 加密。", "LAN connections use TLS encryption."), fg, muted)
}
func gatewaySettingRow(c *ui.Context, title, hint string, fg, muted ui.Color, control func()) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(24).Wrap().Padding(8, 0).Children(func() {
		ui.Column(c).Grow(1).MinWidth(200).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(14)).TextColor(fg)
			ui.Text(c, hint).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Column(c).Grow(1).MinWidth(180).Gap(8).Children(control)
	})
}

// A detected empty list differs from an unknown scan. Saved per-NIC choices
// remain selectable, with the same stale-address feedback as the original UI.
func gatewayBindLabel(address string, detected []string) string {
	switch address {
	case "0.0.0.0":
		return label("整个局域网", "Whole LAN")
	case "127.0.0.1":
		return label("仅本机", "This machine only")
	}
	if detected != nil {
		for _, current := range detected {
			if current == address {
				return address
			}
		}
		return address + " · " + label("已失效", "unavailable")
	}
	return address
}
