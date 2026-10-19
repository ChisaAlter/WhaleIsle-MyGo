package remote

import (
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
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
					if m.OnToggleWorkspace != nil && ui.Button(c, label("启用远程工作区", "Enable remote workspace")).Clicked() {
						m.OnToggleWorkspace(true)
					}
					return
				}
				viewMachines(c, m)
				return
			}
			ui.Text(c, label("远程", "Remote")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
			ui.Text(c, label("管理远程连接网关与消息渠道。", "Manage your remote gateway and messaging channels.")).FontSize(theme.InterfaceSize(14)).TextColor(muted)
			tabs := ui.Row(c.Key("remote-tabs")).FillWidth().Gap(8)
			active := m.ActiveTab
			if active == nil {
				active = ui.Local(tabs, "active", func() string { return "gateway" })
			}
			if *active != "channels" || m.ChannelsContent == nil {
				*active = "gateway"
			}
			tabs.Children(func() {
				for _, tab := range []struct{ id, title string }{{"gateway", label("网关", "Gateway")}, {"channels", label("消息渠道", "Channels")}} {
					if tab.id == "channels" && m.ChannelsContent == nil {
						continue
					}
					b := ui.Button(c, tab.title).Height(34).BorderWidth(0)
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
				if ui.Button(c, label("旧版配置", "Legacy configuration")).Clicked() {
					*open = !*open
				}
				if *open {
					ui.Text(c, label("旧版中继令牌不参与当前设备配对。", "Legacy relay tokens are not used for current device pairing.")).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					if !m.LegacyRelayTokenSet {
						ui.Text(c, label("没有保存的旧版令牌。", "No legacy token is stored.")).TextColor(muted)
					}
					if m.LegacyRelayTokenSet && m.OnSetRelayToken != nil && ui.Button(c, label("清除旧版令牌", "Clear legacy token")).Clicked() {
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
				if ui.Button(c, label("保存令牌", "Save token")).Disabled(strings.TrimSpace(e.RelayToken) == "").Clicked() {
					if err := m.OnSetRelayToken(strings.TrimSpace(e.RelayToken)); err != nil {
						e.Error = err.Error()
					} else {
						e.RelayToken = ""
						e.Error = ""
					}
				}
				if ui.Button(c, label("清除令牌", "Clear token")).Disabled(!m.HasRemoteRelayToken).Clicked() {
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
			if ui.Button(c, label("轮换令牌", "Rotate token")).Disabled(m.PhoneRuntime.DeviceAction.Busy || m.PhoneRuntime.Status == "starting").Clicked() {
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
	e := m.GatewayEditor
	if !e.Open {
		e.Expected = config.RemoteGatewayPreferences{Mode: m.RemoteMode, Port: strconv.Itoa(m.RemotePort), BindAddress: m.RemoteBindAddress, RelayEndpoint: m.RemoteRelayEndpoint, RelayTLS: m.RemoteRelayUseTLS}
		e.Draft = e.Expected
	}
	busy := m.PhoneRuntime.Status == "starting" || m.PhoneRuntime.DeviceAction.Busy
	gatewaySettingRow(c, label("中继地址", "Relay endpoint"), label("使用中继 host[:port]；留空使用默认地址。", "Relay host[:port]; leave blank to use the default."), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.RelayEndpoint).Label("Gateway relay endpoint").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
	})
	gatewaySettingRow(c, label("连接模式", "Connection mode"), label("选择局域网或中继连接。", "Choose a LAN or relay connection."), fg, muted, func() {
		if ui.Select(c, &e.Draft.Mode, []string{"lan", "relay"}).Label("Gateway connection mode").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
	})
	gatewaySettingRow(c, label("端口", "Port"), label("网关监听端口，范围 1024–65535。", "Gateway listen port, from 1024 to 65535."), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.Port).Label("Gateway port").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
	})
	gatewaySettingRow(c, label("监听范围", "Bind scope"), label("127.0.0.1、0.0.0.0 或本机 IPv4 地址。", "127.0.0.1, 0.0.0.0, or a local IPv4 address."), fg, muted, func() {
		if ui.TextInput(c, &e.Draft.BindAddress).Label("Gateway bind address").FillWidth().Disabled(busy).Changed() {
			e.Open = true
		}
	})
	gatewayReadRow(c, label("局域网传输", "LAN transport"), label("局域网连接使用 TLS 加密。", "LAN connections use TLS encryption."), fg, muted)
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c, label("保存并重新连接", "Save and reconnect")).Disabled(!e.Open || busy).Clicked() {
			if err := m.OnSaveGateway(e.Expected, e.Draft); err != nil {
				e.Error = err.Error()
			} else {
				e.Open = false
				e.Error = ""
			}
		}
		if e.Open && ui.Button(c, label("取消", "Cancel")).Disabled(busy).Clicked() {
			e.Draft = e.Expected
			e.Open = false
			e.Error = ""
		}
	})
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
