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
	RoomCreate                RoomCreateModel
	BotVisibility             BotVisibilityModel
	Contacts                  ContactsModel
	BotCreate                 BotCreateModel
	BotDelete                 BotDeleteModel
	BotDuplicate              BotDuplicateModel
	RoomEdit                  RoomEditModel
	RoomRuntime               RoomRuntimeModel
	RoutineCreate             RoutineCreateModel
	Routines                  RoutineModel
	WhaleSettings             WhaleSettingsModel
	WhaleTasks                WhaleTasksModel
	Proactive                 ProactiveModel
	Tasks                     TaskModel
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
	BotNotificationStatus     string
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
		ui.Row(c).Padding(12, 16, 12, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAssistantTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(fg).Grow(1)
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyAssistantBack)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(.5).Background(border)

		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(12, 16, 12, 16).Gap(12).Children(func() {
				if m.Proactive.Open {
					proactiveBlock(c, m.Proactive, fg, muted, border, accent)
					return
				}
				ui.Text(c, i18n.T(i18n.KeyAssistantStubNote)).FontSize(theme.InterfaceSize(12)).TextColor(muted)

				modeBlock(c, m, fg, muted, border, accent)
				inboxLabel := whaleTaskCopy("任务收件箱", "Task inbox")
				if unread := proactiveUnread(m.Proactive.State.Notices); unread > 0 {
					inboxLabel += " (" + strconv.Itoa(unread) + ")"
				}
				if m.Proactive.OnToggle != nil && ui.Button(c, inboxLabel).Clicked() {
					m.Proactive.OnToggle()
				}
				whaleTasksBlock(c, m, fg, muted, border, accent)
				whaleSettingsBlock(c, m, fg, muted, field, border, accent)
				dshbotRosterBlock(c, m, fg, muted, field, border, accent)
				if m.Mode == assiststore.ModeDshbot && m.BotNotificationStatus != "" {
					ui.Text(c, m.BotNotificationStatus).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				botVisibilityBlock(c, m, fg, muted, field, border, accent)
				roomCreateBlock(c, m, fg, muted, field, border, accent)
				botCreateBlock(c, m, fg, muted, field, border, accent)
				botDeleteBlock(c, m, fg, muted, field, border, accent)
				botDuplicateBlock(c, m, fg, muted, field, border, accent)
				roomEditBlock(c, m, fg, muted, field, border, accent)
				roomRuntimeBlock(c, m, fg, muted, field, border, accent)
				taskControlBlock(c, m, fg, muted, field, border, accent)
				routineControlBlock(c, m, fg, muted, field, border, accent)
				routineCreateBlock(c, m, fg, muted, field, border, accent)
				composeBlock(c, m, fg, muted, field, border, accent)
				pluginRPCBlock(c, m, fg, muted, border, accent)
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantFullHistoryStreamingRepliesApprovals)).FontSize(theme.InterfaceSize(11)).TextColor(muted)

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

				section(c, i18n.T(i18n.KeyAssistantWhaleTitle), fg, border, m.WhaleEnabled, m.OnToggleWhale,
					m.HarnessReady && m.WhaleEnabled, m.OnOpenWhaleWeb,
					i18n.T(i18n.KeyAssistantOpenWhaleWeb), i18n.T(i18n.KeyAssistantToggleOn), i18n.T(i18n.KeyAssistantToggleOff))

				section(c, i18n.T(i18n.KeyAssistantDshbotTitle), fg, border, m.DshbotEnabled, m.OnToggleDshbot,
					m.HarnessReady && m.DshbotEnabled, m.OnOpenDshbotWeb,
					i18n.T(i18n.KeyAssistantOpenDshbotWeb), i18n.T(i18n.KeyAssistantToggleOn), i18n.T(i18n.KeyAssistantToggleOff))

				ui.Text(c, i18n.T(i18n.KeyAssistantRestartNote)).FontSize(theme.InterfaceSize(11)).TextColor(accent)
			})
		})
	})
}

type kv struct{ label, value string }

func modeBlock(c *ui.Context, m Model, fg, muted, border, accent ui.Color) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantModeTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyAssistantModeHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			whaleLabel := i18n.T(i18n.KeyAssistantModeWhale)
			dshLabel := i18n.T(i18n.KeyAssistantModeDshbot)
			if m.Mode == assiststore.ModeWhale {
				whaleLabel = "• " + whaleLabel
			}
			if m.Mode == assiststore.ModeDshbot {
				dshLabel = "• " + dshLabel
			}
			if m.OnSetMode != nil && ui.Button(c, whaleLabel).Height(28).Padding(0, 12).Radius(999).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(m.Busy || m.PluginRPCBusy).Clicked() {
				m.OnSetMode(assiststore.ModeWhale)
			}
			if m.OnSetMode != nil && ui.Button(c, dshLabel).Height(28).Padding(0, 12).Radius(999).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(m.Busy || m.PluginRPCBusy).Clicked() {
				m.OnSetMode(assiststore.ModeDshbot)
			}
			if m.HarnessReady && m.OnOpenModeWeb != nil && ui.Button(c, i18n.T(i18n.KeyAssistantOpenModeWeb)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
				m.OnOpenModeWeb()
			}
			mode := assiststore.NormalizeMode(m.Mode)
			canOpenSess := m.HarnessReady && !m.Busy && !m.PluginRPCBusy && m.OnOpenSession != nil && ((mode == assiststore.ModeWhale && m.WhaleEnabled) || (mode == assiststore.ModeDshbot && m.DshbotEnabled && m.DshbotSelectedID != ""))
			if canOpenSess && ui.Button(c, i18n.T(i18n.KeyAssistantOpenSession)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
				m.OnOpenSession(m.SessionID)
			}
		})
	})
}

func statusBlock(c *ui.Context, field, border, fg, muted ui.Color, rows []kv) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantStatusTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		card := ui.Column(c).Border(.5, border).Radius(12).Padding(12, 16, 12, 16).Gap(6)
		chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
		card.Children(func() {
			for _, row := range rows {
				ui.Row(c).Gap(12).Children(func() {
					ui.Text(c, row.label).FontSize(theme.InterfaceSize(12)).TextColor(muted).Width(220)
					ui.Text(c, row.value).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1)
				})
			}
		})
	})
}

func dshbotRosterBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot {
		return
	}
	if m.Contacts.OnRefresh != nil {
		contactsView(c, m, fg, muted, field, border, accent)
		return
	}
	if m.OnDshbotRefreshRoster == nil && len(m.DshbotRoster) == 0 {
		return
	}
	busy := m.PluginRPCBusy || m.Busy
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if m.OnDshbotRefreshRoster != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotRosterRefresh)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotRefreshRoster()
			}
		})
		if fetched := strings.TrimSpace(m.DshbotRosterFetched); fetched != "" {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterUpdated)+": "+fetched).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if len(m.DshbotRoster) == 0 {
			ui.Text(c, i18n.T(i18n.KeyAssistantDshbotRosterEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			return
		}
		limit := len(m.DshbotRoster)
		card := ui.Column(c).Border(.5, border).Radius(12).Padding(12, 16, 12, 16).Gap(8)
		chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
		card.Children(func() {
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
						ui.Text(c, kindLabel).FontSize(theme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, border).TextColor(muted).Width(88)
						ui.Text(c, clip(label, 120)).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).Grow(1)
						if m.OnDshbotSelect != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantSelect)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled || id == m.DshbotSelectedID).Clicked() {
							m.OnDshbotSelect(id)
						}
						if canOpen && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotRosterOpen)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled).Clicked() {
							m.OnDshbotOpenRosterSession(id)
						}
					})
					if d := strings.TrimSpace(row.Detail); d != "" {
						ui.Text(c, clip(d, 140)).FontSize(theme.InterfaceSize(12)).TextColor(accent)
					}
				})
			}
		})
	})
}

func composeBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if m.Draft == nil {
		return
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantComposeTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantOpenTheCanonicalChatToSendFollowR)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.TextArea(c, m.Draft).Lines(2, 4).Radius(12).Border(.5, border).Background(field).Padding(10, 12).FontSize(theme.InterfaceSize(13)).Placeholder(i18n.T(i18n.KeyAssistantComposePlaceholder))
		available := m.HarnessReady && m.WhaleEnabled
		if assiststore.NormalizeMode(m.Mode) == assiststore.ModeDshbot {
			available = m.HarnessReady && m.DshbotEnabled && m.DshbotSelectedID != ""
			ui.Text(c, i18n.T(i18n.KeyW4AssistantChooseABotOrGroupAboveFirstMessagesG)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		empty := strings.TrimSpace(*m.Draft) == "" || m.OnNativePrompt == nil || m.Busy || m.PluginRPCBusy || !available
		if ui.Button(c, i18n.T(i18n.KeyAssistantSend)).Height(28).Padding(0, 12).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(empty).Clicked() {
			text := strings.TrimSpace(*m.Draft)
			if text != "" && m.OnNativePrompt != nil {
				m.OnNativePrompt(text)
				// The submitted prompt is now owned by the canonical session. Clear
				// the local draft so a second click cannot resend it accidentally.
				*m.Draft = ""
			}
		}
		if m.LastNativeAction != "" {
			ui.Text(c, m.LastNativeAction).FontSize(theme.InterfaceSize(11)).TextColor(accent)
		}
		if m.LastErr != "" {
			ui.Text(c, m.LastErr).FontSize(theme.InterfaceSize(11)).TextColor(accent)
		}
	})
}

func pluginRPCBlock(c *ui.Context, m Model, fg, muted, border, accent ui.Color) {
	if m.OnWhaleEnsure == nil && m.OnDshbotDescribe == nil {
		return
	}
	busy := m.PluginRPCBusy || m.Busy
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAssistantPluginRPCTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyAssistantPluginRPCHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if m.OnWhaleEnsure != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhaleEnsure)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhaleEnsure()
			}
			if m.OnWhaleCatalog != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhaleCatalog)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhaleCatalog()
			}
			if m.OnWhalePetChat != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetChat)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || m.Mode != assiststore.ModeWhale || !m.WhaleEnabled || m.Draft == nil || strings.TrimSpace(*m.Draft) == "").Clicked() {
				m.OnWhalePetChat(strings.TrimSpace(*m.Draft))
			}
			if m.OnWhalePetState != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetState)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhalePetState()
			}
			if m.OnWhalePetLook != nil && ui.Button(c, i18n.T(i18n.KeyAssistantWhalePetLook)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.WhaleEnabled).Clicked() {
				m.OnWhalePetLook()
			}
			if m.OnDshbotDescribe != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotDescribe)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotDescribe()
			}
			if m.OnDshbotCatalog != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotCatalog)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotCatalog()
			}
			if m.OnDshbotActivity != nil && ui.Button(c, i18n.T(i18n.KeyAssistantDshbotActivity)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(busy || !m.DshbotEnabled).Clicked() {
				m.OnDshbotActivity()
			}
		})
		if m.PluginRPCDetail != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.PluginRPCDetail).FontSize(theme.InterfaceSize(11)).TextColor(accent).Grow(1)
				if ui.Button(c, i18n.T(i18n.KeyChatCopy)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(11)).Clicked() {
					mygo.Clipboard.WriteText(m.PluginRPCDetail)
				}
			})
		}
	})
}

func section(
	c *ui.Context,
	title string,
	fg, border ui.Color,
	enabled bool,
	onToggle func(bool),
	canOpen bool,
	onOpen func(),
	openLabel, onLabel, offLabel string,
) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Row(c).Gap(8).Children(func() {
			if onToggle != nil {
				label := onLabel
				next := true
				if enabled {
					label = offLabel
					next = false
				}
				if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
					onToggle(next)
				}
			}
			if canOpen && onOpen != nil && ui.Button(c, openLabel).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
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
