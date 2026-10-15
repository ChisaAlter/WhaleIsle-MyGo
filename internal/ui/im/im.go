// Package im is the M9 native instant-messaging panel (dsh-im).
package im

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the IM panel to store and harness status.
type Model struct {
	Store            *imstore.Store
	ThemeID          string
	SystemDark       bool
	HarnessReady     bool
	MuxConnected     bool
	HarnessIMOverlay bool
	PluginDisabled   bool
	Draft            *string
	OnClose          func()
	OnRefresh        func()
	OnOpenPluginWeb  func()
	OnSend           func(channelID, text string)
}

// View paints the IM panel with channel list, local history, and composer.
func View(c *ui.Context, m Model) {
	if m.Store != nil && m.Store.Onboarding.Snapshot().Open {
		onboardingView(c, m)
		return
	}
	if m.Store != nil && m.Store.Office.Snapshot().Open {
		officeSettingsView(c, m)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	var snap imstore.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot()
	}

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyIMTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if m.Store != nil && ui.Button(c, "平台接入 / 修复").Disabled(m.PluginDisabled).Clicked() {
				m.Store.Onboarding.Show("feishu")
			}
			if m.Store != nil && ui.Button(c, "AI Office").Disabled(m.PluginDisabled).Clicked() {
				m.Store.Office.Show(context.Background())
			}
			if snap.Loading {
				ui.Text(c, i18n.T(i18n.KeyIMLoading)).FontSize(12).TextColor(muted)
			}
			if m.OnOpenPluginWeb != nil && ui.Button(c, i18n.T(i18n.KeyIMOpenWeb)).Clicked() {
				m.OnOpenPluginWeb()
			}
			if m.Store != nil && ui.Button(c, i18n.T(i18n.KeyIMRefresh)).Disabled(snap.ConnectionBusy).Clicked() {
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background())
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyIMBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Column(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Gap(10).Children(func() {
			if !snap.HasIMRPC {
				ui.Text(c, i18n.T(i18n.KeyIMStubNote)).FontSize(12).TextColor(muted)
			}

			rows := []struct{ label, value string }{
				{i18n.T(i18n.KeyIMHarnessOverlay), onOff(m.HarnessIMOverlay)},
				{i18n.T(i18n.KeyIMPluginDisabled), onOff(m.PluginDisabled)},
				{i18n.T(i18n.KeyIMMux), muxLabel(m.HarnessReady, m.MuxConnected)},
				{i18n.T(i18n.KeyIMGeneratedRPC), rpcLabel(snap.HasIMRPC, len(snap.Methods))},
				{i18n.T(i18n.KeyIMBridge), bridgeLabel(snap.BridgeMode)},
			}
			ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
				for _, row := range rows {
					ui.Row(c).Gap(12).Children(func() {
						ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(160)
						ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
					})
				}
			})

			if m.PluginDisabled {
				ui.Text(c, i18n.T(i18n.KeyIMPluginOff)).FontSize(12).TextColor(accent)
			}
			if snap.Err != "" {
				ui.Text(c, snap.Err).FontSize(12).TextColor(accent)
			}
			if snap.PluginDiag != "" {
				ui.Text(c, i18n.T(i18n.KeyIMPluginDiag)+": "+snap.PluginDiag).FontSize(11).TextColor(muted)
			}

			if snap.LocalMode {
				ui.Text(c, i18n.T(i18n.KeyIMLocalMode)).FontSize(12).TextColor(muted)
			} else {
				ui.Text(c, i18n.T(i18n.KeyIMListVia)+": "+snap.ListMethod).FontSize(12).Bold().TextColor(fg)
				if snap.HistoryMethod != "" {
					ui.Text(c, i18n.T(i18n.KeyIMHistoryVia)+": "+snap.HistoryMethod).FontSize(12).TextColor(fg)
				}
				if snap.SyncedRemote {
					ui.Text(c, i18n.T(i18n.KeyIMHistorySynced)).FontSize(11).TextColor(muted)
				}
			}

			ui.Row(c).Fill().MinHeight(0).Gap(12).Children(func() {
				// Channels column
				ui.Column(c).Width(220).Shrink(0).MinHeight(0).Gap(6).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyIMChannels)).FontSize(13).Bold().TextColor(fg)
					if len(snap.Rows) == 0 {
						if snap.RawListPreview != "" {
							ui.Text(c, snap.RawListPreview).FontSize(11).TextColor(muted)
						} else {
							ui.Text(c, i18n.T(i18n.KeyIMListEmpty)).FontSize(12).TextColor(muted)
						}
					} else {
						ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
							Padding(8, 10, 8, 10).Gap(4).Children(func() {
							for _, row := range snap.Rows {
								entry := row
								id := entry.ID
								if id == "" {
									id = entry.Label
								}
								selected := snap.SelectedID != "" && (entry.ID == snap.SelectedID || entry.Label == snap.SelectedID)
								label := entry.Label
								if selected {
									label = "• " + label
								}
								if ui.Button(c, label).Clicked() && m.Store != nil {
									if m.Draft != nil {
										m.Store.SetDraft(snap.SelectedID, *m.Draft)
									}
									m.Store.Select(id)
									if m.Draft != nil {
										*m.Draft = m.Store.Snapshot().Draft
									}
								}
								if entry.Detail != "" {
									ui.Text(c, entry.Detail).FontSize(11).TextColor(muted)
								}
							}
						})
					}
				})

				// History + compose
				ui.Column(c).Grow(1).Fill().MinHeight(0).Gap(8).Children(func() {

					if snap.BridgeMode == "cordis" {
						var selected imstore.Entry
						for _, row := range snap.Rows {
							if row.ID == snap.SelectedID {
								selected = row
								break
							}
						}
						ui.Text(c, "机器人连接与会话配置").FontSize(13).Bold().TextColor(fg)
						if selected.BotID == "" {
							ui.Text(c, "请选择机器人；添加账号请打开插件页面。").FontSize(12).TextColor(muted)
							return
						}
						ui.Text(c, selected.Label+" · "+selected.Channel+" · "+selected.State).FontSize(12).TextColor(fg)
						ui.Text(c, selected.Detail).FontSize(11).TextColor(muted)
						disabled := snap.ConnectionBusy || snap.Loading || m.PluginDisabled || m.Store == nil
						if ui.Button(c, "重新连接").Disabled(disabled).Clicked() {
							_ = m.Store.ReconnectBot(context.Background(), selected.ID)
						}
						ui.Text(c, "重连不会发送测试消息。账号凭证请在插件页面管理。").FontSize(11).TextColor(muted)
						ui.Text(c, "工作区绝对路径").FontSize(12).Bold().TextColor(fg)
						workspace := snap.WorkspaceDraft
						ui.TextInput(c, &workspace).FillWidth().Placeholder("/path/to/workspace 或 C:\\workspace")
						if m.Store != nil && workspace != snap.WorkspaceDraft {
							m.Store.SetWorkspaceDraft(selected.ID, workspace)
						}
						if ui.Button(c, "保存工作区").Disabled(disabled || strings.TrimSpace(workspace) == selected.Workspace).Clicked() {
							_ = m.Store.SaveWorkspace(context.Background(), selected.ID, workspace)
						}
						currentPreset := selected.AgentPreset
						if currentPreset == "" {
							currentPreset = "跟随 Host 默认"
						}
						ui.Text(c, "Agent Preset："+currentPreset).FontSize(12).Bold().TextColor(fg)
						ui.Row(c).Gap(6).Children(func() {
							if ui.Button(c, "跟随 Host 默认").Disabled(disabled || selected.AgentPreset == "").Clicked() {
								_ = m.Store.SaveAgentPreset(context.Background(), selected.ID, "")
							}
							for _, preset := range selected.Presets {
								if ui.Button(c, preset.Label).Disabled(disabled || selected.AgentPreset == preset.ID).Clicked() {
									_ = m.Store.SaveAgentPreset(context.Background(), selected.ID, preset.ID)
								}
							}
						})
						ui.Text(c, "配置影响新会话；已有聊天请在对应 IM 中发送 /new 后再发送普通消息。").FontSize(11).TextColor(muted)
						if snap.ConnectionNote != "" {
							ui.Text(c, snap.ConnectionNote).FontSize(12).TextColor(accent)
						}
						return
					}
					ui.Text(c, i18n.T(i18n.KeyIMHistory)).FontSize(13).Bold().TextColor(fg)
					if snap.SelectedID == "" {
						ui.Text(c, i18n.T(i18n.KeyIMNoChannel)).FontSize(12).TextColor(muted)
					} else if len(snap.Messages) == 0 {
						ui.Text(c, i18n.T(i18n.KeyIMHistoryEmpty)).FontSize(12).TextColor(muted)
					} else {
						ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
							Padding(8, 10, 8, 10).Gap(6).Children(func() {
							for _, msg := range snap.Messages {
								entry := msg
								prefix := i18n.T(i18n.KeyIMYou)
								if !entry.Outbound {
									prefix = i18n.T(i18n.KeyIMPeer)
								}
								when := formatIMTime(entry.At)
								line := prefix + " · " + when
								ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
									ui.Text(c, line).FontSize(11).TextColor(muted).Grow(1)
									if strings.TrimSpace(entry.Text) != "" {
										if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
											mygo.Clipboard.WriteText(entry.Text)
										}
									}
								})
								ui.Text(c, entry.Text).FontSize(12).TextColor(fg)
							}
						})
					}

					if m.Draft != nil {
						ui.Text(c, i18n.T(i18n.KeyIMCompose)).FontSize(12).Bold().TextColor(fg)
						ui.TextArea(c, m.Draft).Fill().Lines(2, 4)
						if m.Store != nil && snap.SelectedID != "" {
							m.Store.SetDraft(snap.SelectedID, *m.Draft)
						}
						send := ui.Button(c, i18n.T(i18n.KeyIMSend))
						if snap.SelectedID == "" || strings.TrimSpace(*m.Draft) == "" {
							send = send.Disabled(true)
						}
						if send.Clicked() && m.OnSend != nil {
							m.OnSend(snap.SelectedID, strings.TrimSpace(*m.Draft))
							*m.Draft = ""
							if m.Store != nil {
								m.Store.SetDraft(snap.SelectedID, "")
							}
						}
						if snap.LastSend != "" {
							ui.Text(c, snap.LastSend).FontSize(11).TextColor(muted)
						}
					}
				})
			})

			if len(snap.Methods) > 0 {
				ui.Text(c, i18n.T(i18n.KeyIMMethods)).FontSize(11).TextColor(muted)
				ui.Text(c, strings.Join(snap.Methods, ", ")).FontSize(11).TextColor(fg)
			}
		})
	})
}

func formatIMTime(at string) string {
	if at == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, at); err == nil {
		return t.Local().Format("15:04:05")
	}
	return at
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyIMOn)
	}
	return i18n.T(i18n.KeyIMOff)
}

func muxLabel(ready, connected bool) string {
	if !ready {
		return i18n.T(i18n.KeyIMMuxWaiting)
	}
	if connected {
		return i18n.T(i18n.KeyIMMuxConnected)
	}
	return i18n.T(i18n.KeyIMMuxDisconnected)
}

func rpcLabel(hasRPC bool, count int) string {
	if hasRPC {
		return i18n.T(i18n.KeyIMRPCYes)
	}
	if count > 0 {
		return i18n.T(i18n.KeyIMRPCPartial)
	}
	return i18n.T(i18n.KeyIMRPCNo)
}

func bridgeLabel(mode string) string {
	switch mode {
	case "rpc":
		return i18n.T(i18n.KeyIMBridgeRPC)
	case "cordis":
		return i18n.T(i18n.KeyIMBridgeCordis)
	case "inventory":
		return i18n.T(i18n.KeyIMBridgeInventory)
	case "local":
		return i18n.T(i18n.KeyIMBridgeLocal)
	default:
		return i18n.T(i18n.KeyIMBridgeIdle)
	}
}
