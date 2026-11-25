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
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the IM panel to store and harness status.
type Model struct {
	Embedded        bool
	Store           *imstore.Store
	ThemeID         string
	SystemDark      bool
	HarnessReady    bool
	MuxConnected    bool
	PluginDisabled  bool
	Draft           *string
	OnClose         func()
	OnRefresh       func()
	OnOpenPluginWeb func()
	OnEnablePlugin  func()
	OnSend          func(channelID, text string)
}

// View paints the IM panel with channel list, local history, and composer.
func View(c *ui.Context, m Model) {
	if m.Embedded {
		SettingsView(c, m)
		return
	}
	if m.Store != nil && m.Store.Onboarding.Snapshot().Value.Open {
		onboardingView(c, m)
		return
	}
	if m.Store != nil && m.Store.Office.Snapshot().Value.Open {
		officeSettingsView(c, m)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	var snap imstore.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot().Value
	}

	ui.Column(c).Grow(1).MinHeight(0).Background(ui.Transparent).Children(func() {
		if !m.Embedded {
			leftPad, rightPad := chrome.TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
			ui.Row(c.Key("im-header")).FillWidth().MinWidth(0).Shrink(0).Padding(design.Sp(2), rightPad, design.Sp(2), leftPad).Gap(design.Sp(2)).AlignItems(ui.Center).DragWindow().Children(func() {
				ui.Text(c, i18n.T(i18n.KeyIMTitle)).FontSize(theme.InterfaceSize(15)).Bold().TextColor(fg).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
				if m.Store != nil && !m.PluginDisabled && coss.Button(c, channelText("平台接入 / 修复", "Channel setup / repair"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "im-onboarding"}).Shrink(0).Clicked() {
					m.Store.Onboarding.Show("feishu")
				}
				if m.Store != nil && !m.PluginDisabled && coss.Button(c, "AI Office", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "im-office"}).Shrink(0).Clicked() {
					m.Store.Office.Show(context.Background())
				}
				if m.PluginDisabled && m.OnEnablePlugin != nil {
					if coss.Button(c, i18n.T(keyIMEnablePlugin), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Key: "im-enable-plugin"}).Shrink(0).Clicked() {
						m.OnEnablePlugin()
					}
				}
				if m.OnOpenPluginWeb != nil && coss.Button(c, i18n.T(i18n.KeyIMOpenWeb), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "im-open-web"}).Shrink(0).Clicked() {
					m.OnOpenPluginWeb()
				}
				if m.Store != nil && coss.Button(c, i18n.T(i18n.KeyIMRefresh), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: snap.ConnectionBusy, Key: "im-refresh"}).Shrink(0).Clicked() {
					if m.OnRefresh != nil {
						m.OnRefresh()
					} else {
						m.Store.Refresh(context.Background())
					}
				}
				if m.OnClose != nil && coss.Button(c, i18n.T(i18n.KeyIMBack), coss.ButtonOptions{Variant: coss.ButtonSecondary, Size: coss.ButtonSM, Key: "im-back"}).Shrink(0).Clicked() {
					m.OnClose()
				}
			})
			ui.Box(c).FillWidth().Height(1).Shrink(0).Background(border)

		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Gap(10).Children(func() {
			if !snap.HasIMRPC {
				note := i18n.T(i18n.KeyIMStubNote)
				if m.HarnessReady {
					// The Host answered, so its plugin is the missing piece;
					// do not tell a connected user to connect again.
					note = i18n.T(keyIMPluginNotReady)
				}
				ui.Text(c, note).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}

			if !m.Embedded {
				rows := []struct{ label, value string }{
					{i18n.T(i18n.KeyIMPluginDisabled), onOff(m.PluginDisabled)},
					{i18n.T(i18n.KeyIMMux), muxLabel(m.HarnessReady, m.MuxConnected)},
					{i18n.T(i18n.KeyIMGeneratedRPC), rpcLabel(snap.HasIMRPC, len(snap.Methods))},
					{i18n.T(i18n.KeyIMBridge), bridgeLabel(snap.BridgeMode)},
				}
				chrome.FrostSurface(ui.Column(c).Border(1, border).Padding(10, 12, 10, 12).Gap(8), colors, m.SystemDark, field, 8).Children(func() {
					for _, row := range rows {
						ui.Row(c).Gap(12).Children(func() {
							ui.Text(c, row.label).FontSize(theme.InterfaceSize(12)).TextColor(muted).Width(160)
							ui.Text(c, row.value).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1)
						})
					}
				})

			}
			if m.PluginDisabled {
				ui.Text(c, i18n.T(i18n.KeyIMPluginOff)).FontSize(theme.InterfaceSize(12)).TextColor(accent)
			}
			if snap.Err != nil {
				ui.Text(c, snap.Err.SentenceText()).FontSize(theme.InterfaceSize(12)).TextColor(accent)
			}
			if snap.PluginDiag != nil {
				ui.Text(c, i18n.T(i18n.KeyIMPluginDiag)+": "+snap.PluginDiag.SentenceText()).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}

			if snap.LocalMode {
				ui.Text(c, i18n.T(i18n.KeyIMLocalMode)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			} else {
				ui.Text(c, i18n.T(i18n.KeyIMListVia)+": "+snap.ListMethod).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
				if snap.HistoryMethod != "" {
					ui.Text(c, i18n.T(i18n.KeyIMHistoryVia)+": "+snap.HistoryMethod).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if snap.SyncedRemote {
					ui.Text(c, i18n.T(i18n.KeyIMHistorySynced)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			}

			ui.Row(c).Fill().MinHeight(0).Gap(12).Children(func() {
				// Channels column
				ui.Column(c).Width(220).Shrink(0).MinHeight(0).Gap(6).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyIMChannels)).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg)
					if len(snap.Rows) == 0 {
						if snap.RawListPreview != "" {
							ui.Text(c, snap.RawListPreview).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						} else {
							ui.Text(c, i18n.T(i18n.KeyIMListEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
					} else {
						chrome.FrostSurface(ui.Column(c).Fill().MinHeight(0).Border(1, border).
							Padding(8, 10, 8, 10).Gap(4), colors, m.SystemDark, field, 8).Children(func() {
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
								if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Pressed: selected}).FillWidth().Justify(ui.Start).Clicked() && m.Store != nil {
									if m.Draft != nil {
										m.Store.SetDraft(snap.SelectedID, *m.Draft)
									}
									m.Store.Select(id)
									if m.Draft != nil {
										*m.Draft = m.Store.Snapshot().Value.Draft
									}
								}
								if entry.Detail != "" {
									ui.Text(c, entry.Detail).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
						ui.Text(c, "机器人连接与会话配置").FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg)
						if selected.BotID == "" {
							ui.Text(c, "请选择机器人；添加账号请打开插件页面。").FontSize(theme.InterfaceSize(12)).TextColor(muted)
							return
						}
						ui.Text(c, selected.Label+" · "+selected.Channel+" · "+selected.State).FontSize(theme.InterfaceSize(12)).TextColor(fg)
						ui.Text(c, selected.Detail).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						disabled := snap.ConnectionBusy || snap.Loading || m.PluginDisabled || m.Store == nil
						if coss.Button(c, "重新连接", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
							_ = m.Store.ReconnectBot(context.Background(), selected.ID)
						}
						ui.Text(c, "重连不会发送测试消息。账号凭证请在插件页面管理。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
						ui.Text(c, "工作区绝对路径").FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
						workspace := snap.WorkspaceDraft
						if coss.Input(c, &workspace, coss.InputOptions{Placeholder: "/path/to/workspace 或 C:\\workspace"}).Changed() && m.Store != nil {
							m.Store.SetWorkspaceDraft(selected.ID, workspace)
						}
						if coss.Button(c, "保存工作区", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled || strings.TrimSpace(workspace) == selected.Workspace}).Clicked() {
							_ = m.Store.SaveWorkspace(context.Background(), selected.ID, workspace)
						}
						currentPreset := selected.AgentPreset
						if currentPreset == "" {
							currentPreset = "跟随 Host 默认"
						}
						ui.Text(c, "Agent Preset："+currentPreset).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
						ui.Row(c).Gap(6).Children(func() {
							if coss.Button(c, "跟随 Host 默认", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled || selected.AgentPreset == ""}).Clicked() {
								_ = m.Store.SaveAgentPreset(context.Background(), selected.ID, "")
							}
							for _, preset := range selected.Presets {
								if coss.Button(c, preset.Label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled || selected.AgentPreset == preset.ID}).Clicked() {
									_ = m.Store.SaveAgentPreset(context.Background(), selected.ID, preset.ID)
								}
							}
						})
						ui.Text(c, "配置影响新会话；已有聊天请在对应 IM 中发送 /new 后再发送普通消息。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if snap.ConnectionNote != "" {
							ui.Text(c, snap.ConnectionNote).FontSize(theme.InterfaceSize(12)).TextColor(accent)
						}
						return
					}
					ui.Text(c, i18n.T(i18n.KeyIMHistory)).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg)
					if snap.SelectedID == "" {
						ui.Text(c, i18n.T(i18n.KeyIMNoChannel)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					} else if len(snap.Messages) == 0 {
						ui.Text(c, i18n.T(i18n.KeyIMHistoryEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					} else {
						chrome.FrostSurface(ui.Column(c).Fill().MinHeight(0).Border(1, border).
							Padding(8, 10, 8, 10).Gap(6), colors, m.SystemDark, field, 8).Children(func() {
							for _, msg := range snap.Messages {
								entry := msg
								prefix := i18n.T(i18n.KeyIMYou)
								if !entry.Outbound {
									prefix = i18n.T(i18n.KeyIMPeer)
								}
								when := formatIMTime(entry.At)
								line := prefix + " · " + when
								ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
									ui.Text(c, line).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
									if state := imMessageStateLabel(entry.State); state != "" {
										color := muted
										if entry.State == imstore.MessageStateFailed || entry.State == imstore.MessageStateUnknown {
											color = accent
										}
										ui.Text(c, state).FontSize(theme.InterfaceSize(11)).TextColor(color).Shrink(0)
									}
									if strings.TrimSpace(entry.Text) != "" {
										if coss.Button(c, i18n.T(i18n.KeyChatCopy), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
											mygo.Clipboard.WriteText(entry.Text)
										}
									}
								})
								ui.Text(c, entry.Text).FontSize(theme.InterfaceSize(12)).TextColor(fg)
							}
						})
					}

					if m.Draft != nil {
						ui.Text(c, i18n.T(i18n.KeyIMCompose)).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
						if coss.Textarea(c, m.Draft, coss.TextareaOptions{MaxLines: 4}).Changed() && m.Store != nil && snap.SelectedID != "" {
							m.Store.SetDraft(snap.SelectedID, *m.Draft)
						}
						send := coss.Button(c, i18n.T(i18n.KeyIMSend), coss.ButtonOptions{
							Variant:  coss.ButtonDefault,
							Size:     coss.ButtonSM,
							Disabled: snap.SelectedID == "" || strings.TrimSpace(*m.Draft) == "",
						})
						if send.Clicked() && m.OnSend != nil {
							m.OnSend(snap.SelectedID, strings.TrimSpace(*m.Draft))
							if m.Store != nil {
								*m.Draft = m.Store.Snapshot().Value.Draft
							} else {
								*m.Draft = ""
							}
						}
						if snap.LastSend != nil {
							ui.Text(c, snap.LastSend.SentenceText()).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						}
					}
				})
			})

			if len(snap.Methods) > 0 {
				ui.Text(c, i18n.T(i18n.KeyIMMethods)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, strings.Join(snap.Methods, ", ")).FontSize(theme.InterfaceSize(11)).TextColor(fg)
			}
		})
	})
}

// imMessageStateLabel names the real delivery state of an outbound message.
// Legacy entries carry no state and stay unlabelled.
func imMessageStateLabel(state string) string {
	switch state {
	case imstore.MessageStatePending:
		return i18n.T(keyIMSendPending)
	case imstore.MessageStateFailed:
		return i18n.T(keyIMSendFailed)
	case imstore.MessageStateLocal:
		return i18n.T(keyIMSendLocal)
	case imstore.MessageStateUnknown:
		return i18n.T(keyIMSendUnknown)
	default:
		return ""
	}
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
