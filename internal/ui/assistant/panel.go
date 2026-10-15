// Package assistant is the M10 native panel for dsh-whale and dshbot.
package assistant

import (
	"strconv"
	"strings"

	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Model wires whale assistant / dshbot status and actions.
type Model struct {
	ThemeID                   string
	SystemDark                bool
	Mode                      assiststore.Mode
	WhaleEnabled              bool
	DshbotEnabled             bool
	WhaleVendorPresent        bool
	DshbotVendorPresent       bool
	HarnessReady              bool
	Busy                      bool
	LastWebViewURL            string
	LastNativeAction          string
	LastErr                   string
	SessionID                 string
	History                   []assiststore.Turn
	Draft                     *string
	OnClose                   func()
	OnSetMode                 func(mode assiststore.Mode)
	OnToggleWhale             func(enabled bool)
	OnToggleDshbot            func(enabled bool)
	OnOpenModeWeb             func() // opens WebView for current Mode
	OnOpenWhaleWeb            func()
	OnOpenDshbotWeb           func()
	OnNativePrompt            func(text string)
	OnOpenSession             func(sessionID string)
	OnWhaleEnsure             func()
	OnWhaleCatalog            func()
	OnWhalePetChat            func(text string)
	OnWhalePetState           func()
	OnWhalePetLook            func()
	OnDshbotDescribe          func()
	OnDshbotCatalog           func()
	OnDshbotActivity          func()
	OnDshbotRefreshRoster     func()
	OnDshbotOpenRosterSession func(botID string)
	OnDshbotSelect            func(botID string)
	DshbotSelectedID          string
	DshbotRoster              []dshbotstore.Row
	DshbotRosterFetched       string
	PluginRPCBusy             bool
	PluginRPCDetail           string
}

// View paints the M10 assistant / dshbot panel.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAssistantTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyAssistantBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(12, 16, 12, 16).Gap(12).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyAssistantStubNote)).FontSize(12).TextColor(muted)

				modeBlock(c, m, fg, muted, accent)
				dshbotRosterBlock(c, m, fg, muted, field, border, accent)
				composeBlock(c, m, fg, muted, accent)
				pluginRPCBlock(c, m, fg, muted, accent)
				historyBlock(c, m, fg, muted, field, border, accent)

				statusBlock(c, field, border, fg, muted, []kv{
					{i18n.T(i18n.KeyAssistantModeLabel), modeLabel(m.Mode)},
					{i18n.T(i18n.KeyAssistantWhaleTitle) + " · " + i18n.T(i18n.KeyAssistantConfigEnabled), onOff(m.WhaleEnabled)},
					{i18n.T(i18n.KeyAssistantWhaleTitle) + " · " + i18n.T(i18n.KeyAssistantVendorPresent), onOff(m.WhaleVendorPresent)},
					{i18n.T(i18n.KeyAssistantDshbotTitle) + " · " + i18n.T(i18n.KeyAssistantConfigEnabled), onOff(m.DshbotEnabled)},
					{i18n.T(i18n.KeyAssistantDshbotTitle) + " · " + i18n.T(i18n.KeyAssistantVendorPresent), onOff(m.DshbotVendorPresent)},
					{i18n.T(i18n.KeyAssistantLastWebURL), dash(m.LastWebViewURL)},
					{i18n.T(i18n.KeyAssistantLastAction), dash(m.LastNativeAction)},
					{i18n.T(i18n.KeyAssistantLastErr), dash(m.LastErr)},
					{i18n.T(i18n.KeyAssistantSessionID), dash(m.SessionID)},
				})

				section(c, i18n.T(i18n.KeyAssistantWhaleTitle), fg, m.WhaleEnabled, m.OnToggleWhale,
					m.HarnessReady && m.WhaleEnabled, m.OnOpenWhaleWeb,
					i18n.T(i18n.KeyAssistantOpenWhaleWeb), i18n.T(i18n.KeyAssistantToggleOn), i18n.T(i18n.KeyAssistantToggleOff))

				section(c, i18n.T(i18n.KeyAssistantDshbotTitle), fg, m.DshbotEnabled, m.OnToggleDshbot,
					m.HarnessReady && m.DshbotEnabled, m.OnOpenDshbotWeb,
					i18n.T(i18n.KeyAssistantOpenDshbotWeb), i18n.T(i18n.KeyAssistantToggleOn), i18n.T(i18n.KeyAssistantToggleOff))

				ui.Text(c, i18n.T(i18n.KeyAssistantRestartNote)).FontSize(11).TextColor(accent)
			})
		})
	})
}

type kv struct{ label, value string }

func modeBlock(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantModeTitle)).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyAssistantModeHint)).FontSize(11).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			whaleLabel := i18n.T(i18n.KeyAssistantModeWhale)
			dshLabel := i18n.T(i18n.KeyAssistantModeDshbot)
			if m.Mode == assiststore.ModeWhale {
				whaleLabel = "• " + whaleLabel
			}
			if m.Mode == assiststore.ModeDshbot {
				dshLabel = "• " + dshLabel
			}
			if m.OnSetMode != nil && ui.Button(c, whaleLabel).Disabled(m.Busy || m.PluginRPCBusy).Clicked() {
				m.OnSetMode(assiststore.ModeWhale)
			}
			if m.OnSetMode != nil && ui.Button(c, dshLabel).Disabled(m.Busy || m.PluginRPCBusy).Clicked() {
				m.OnSetMode(assiststore.ModeDshbot)
			}
			if m.HarnessReady && m.OnOpenModeWeb != nil && ui.Button(c, i18n.T(i18n.KeyAssistantOpenModeWeb)).Clicked() {
				m.OnOpenModeWeb()
			}
			canOpenSess := strings.TrimSpace(m.SessionID) != "" && m.OnOpenSession != nil
			if canOpenSess && ui.Button(c, i18n.T(i18n.KeyAssistantOpenSession)).Clicked() {
				m.OnOpenSession(m.SessionID)
			}
		})
		if m.Busy {
			ui.Text(c, i18n.T(i18n.KeyAssistantBusy)).FontSize(11).TextColor(accent)
		}
	})
}

func historyBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantHistoryTitle)).FontSize(13).Bold().TextColor(fg)
		if len(m.History) == 0 {
			ui.Text(c, i18n.T(i18n.KeyAssistantHistoryEmpty)).FontSize(12).TextColor(muted)
			return
		}
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
			limit := 8
			if len(m.History) < limit {
				limit = len(m.History)
			}
			for i := 0; i < limit; i++ {
				t := m.History[i]
				ui.Column(c).Gap(2).Children(func() {
					head := modeLabel(t.Mode)
					if t.Source != "" {
						head += " · " + t.Source
					}
					if t.At != "" {
						head += " · " + t.At
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, head).FontSize(11).TextColor(muted).Grow(1)
						copyBody := t.Response
						if copyBody == "" {
							copyBody = t.Prompt
						}
						if strings.TrimSpace(copyBody) != "" {
							if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
								mygo.Clipboard.WriteText(copyBody)
							}
						}
					})
					ui.Text(c, clip(t.Prompt, 160)).FontSize(12).TextColor(fg)
					if t.Response != "" {
						ui.Text(c, clip(t.Response, 240)).FontSize(12).TextColor(accent)
					}
					if t.Err != "" {
						ui.Text(c, t.Err).FontSize(11).TextColor(accent)
					}
				})
			}
		})
	})
}

func statusBlock(c *ui.Context, field, border, fg, muted ui.Color, rows []kv) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantStatusTitle)).FontSize(13).Bold().TextColor(fg)
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			for _, row := range rows {
				ui.Row(c).Gap(12).Children(func() {
					ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(220)
					ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
				})
			}
		})
	})
}

func dshbotRosterBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot {
		return
	}
	if m.OnDshbotRefreshRoster == nil && len(m.DshbotRoster) == 0 {
		return
	}
	busy := m.PluginRPCBusy || m.Busy
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterTitle)).FontSize(13).Bold().TextColor(fg).Grow(1)
			if m.OnDshbotRefreshRoster != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotRosterRefresh)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotRefreshRoster()
			}
		})
		if fetched := strings.TrimSpace(m.DshbotRosterFetched); fetched != "" {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterUpdated)+": "+fetched).FontSize(11).TextColor(muted)
		}
		if len(m.DshbotRoster) == 0 {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterEmpty)).FontSize(12).TextColor(muted)
			return
		}
		limit := len(m.DshbotRoster)
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(6).Children(func() {
			for i := 0; i < limit; i++ {
				row := m.DshbotRoster[i]
				kindLabel := i18n.T(i18n.KeyAssistantDshbotRosterBot)
				if row.Kind == "room" {
					kindLabel = i18n.T(i18n.KeyAssistantDshbotRosterRoom)
					if row.MemberCount > 0 {
						kindLabel += " (" + strconv.Itoa(row.MemberCount) + ")"
					}
				}
				label := row.DisplayLabel
				if label == "" {
					label = row.Name
				}
				if row.Pinned {
					label = "★ " + label
				}
				id := strings.TrimSpace(row.ID)
				canOpen := id != "" && m.OnDshbotOpenRosterSession != nil
				if id == m.DshbotSelectedID {
					label = "• " + label
				}
				ui.Column(c).Gap(2).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, kindLabel).FontSize(11).TextColor(muted).Width(88)
						ui.Text(c, clip(label, 120)).FontSize(12).TextColor(fg).Grow(1)
						if m.OnDshbotSelect != nil && ui.Button(c, "选择 / Select").Disabled(busy || !m.DshbotEnabled || id == m.DshbotSelectedID).Clicked() {
							m.OnDshbotSelect(id)
						}
						if canOpen && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotRosterOpen)).Disabled(busy || !m.DshbotEnabled).Clicked() {
							m.OnDshbotOpenRosterSession(id)
						}
					})
					if d := strings.TrimSpace(row.Detail); d != "" {
						ui.Text(c, clip(d, 140)).FontSize(11).TextColor(accent)
					}
				})
			}
		})
	})
}

func composeBlock(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	if m.Draft == nil {
		return
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantComposeTitle)).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyAssistantComposeHint)).FontSize(11).TextColor(muted)
		ui.TextArea(c, m.Draft).Lines(2, 4).Placeholder(i18n.T(i18n.KeyAssistantComposePlaceholder))
		available := m.HarnessReady && m.WhaleEnabled
		if assiststore.NormalizeMode(m.Mode) == assiststore.ModeDshbot {
			available = m.HarnessReady && m.DshbotEnabled && m.DshbotSelectedID != ""
			ui.Text(c, "先选择上方 Bot / 群聊；消息发送到该条目的专属会话。").FontSize(11).TextColor(muted)
		}
		empty := strings.TrimSpace(*m.Draft) == "" || m.OnNativePrompt == nil || m.Busy || m.PluginRPCBusy || !available
		if ui.Button(c, i18n.T(i18n.KeyAssistantSend)).Disabled(empty).Clicked() {
			text := strings.TrimSpace(*m.Draft)
			if text != "" && m.OnNativePrompt != nil {
				m.OnNativePrompt(text)
			}
		}
		if m.LastNativeAction != "" {
			ui.Text(c, m.LastNativeAction).FontSize(11).TextColor(accent)
		}
		if m.LastErr != "" {
			ui.Text(c, m.LastErr).FontSize(11).TextColor(accent)
		}
	})
}

func pluginRPCBlock(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	if m.OnWhaleEnsure == nil && m.OnDshbotDescribe == nil {
		return
	}
	busy := m.PluginRPCBusy || m.Busy
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantPluginRPCTitle)).FontSize(13).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyAssistantPluginRPCHint)).FontSize(11).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			if m.OnWhaleEnsure != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhaleEnsure)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhaleEnsure()
			}
			if m.OnWhaleCatalog != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhaleCatalog)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhaleCatalog()
			}
			if m.OnWhalePetChat != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetChat)).Disabled(busy || m.Mode != assiststore.ModeWhale || !m.WhaleEnabled || m.Draft == nil || strings.TrimSpace(*m.Draft) == "").Clicked() {
				m.OnWhalePetChat(strings.TrimSpace(*m.Draft))
			}
			if m.OnWhalePetState != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetState)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhalePetState()
			}
			if m.OnWhalePetLook != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetLook)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhalePetLook()
			}
			if m.OnDshbotDescribe != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotDescribe)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotDescribe()
			}
			if m.OnDshbotCatalog != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotCatalog)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotCatalog()
			}
			if m.OnDshbotActivity != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotActivity)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotActivity()
			}
		})
		if m.PluginRPCDetail != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.PluginRPCDetail).FontSize(11).TextColor(accent).Grow(1)
				if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
					mygo.Clipboard.WriteText(m.PluginRPCDetail)
				}
			})
		}
	})
}

func section(
	c *ui.Context,
	title string,
	fg ui.Color,
	enabled bool,
	onToggle func(bool),
	canOpen bool,
	onOpen func(),
	openLabel, onLabel, offLabel string,
) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, title).FontSize(13).Bold().TextColor(fg)
		ui.Row(c).Gap(8).Children(func() {
			if onToggle != nil {
				label := onLabel
				next := true
				if enabled {
					label = offLabel
					next = false
				}
				if ui.Button(c, label).Clicked() {
					onToggle(next)
				}
			}
			if canOpen && onOpen != nil && ui.Button(c, openLabel).Clicked() {
				onOpen()
			}
		})
	})
}

func modeLabel(mode assiststore.Mode) string {
	if assiststore.NormalizeMode(mode) == assiststore.ModeDshbot {
		return i18n.T(i18n.KeyAssistantModeDshbot)
	}
	return i18n.T(i18n.KeyAssistantModeWhale)
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyAssistantOn)
	}
	return i18n.T(i18n.KeyAssistantOff)
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
