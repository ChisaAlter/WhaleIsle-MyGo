package frame

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// ConversationSessionHeader: titleCluster (crumbs + actions), utilities, corner.
// Occupants share the original 560 DIP title-row compact threshold.
func conversationHeaderTitleRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors) {
	compact := m.Sizes.Center-48 <= 560
	gap := float32(8)
	utilitiesMargin := float32(12)
	if compact {
		gap = 4
		utilitiesMargin = 8
	}
	rightInset := float32(28)
	if m.OnOpenSettings != nil {
		rightInset = 12
	}
	muted := chrome.Color(colors.Muted)
	ui.Row(c).Key("conversation-title-band").Label("Conversation title band").FillWidth().MinHeight(40).Padding(8, rightInset, 0, 20).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Row(c).Key("conversation-title-cluster").Grow(1).MinWidth(0).AlignItems(ui.Center).Gap(gap).Children(func() {
			ui.Box(c).Key("conversation-title-crumbs").Label("Conversation session title").Grow(1).Shrink(1).MaxWidth(280).MinWidth(0).Clip().Children(func() {
				if snap.SelectedID != "" {
					sessionAncestryHeader(c, m, snap)
				} else {
					ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).TextColor(chrome.Color(colors.FG))
				}
			})
			ui.Row(c).Key("conversation-header-actions").Shrink(0).AlignItems(ui.Center).Gap(gap).Children(func() {
				preset := strings.TrimSpace(snap.SelectedAgentPreset)
				if preset == "" && m.AgentPresetPick != nil {
					preset = strings.TrimSpace(*m.AgentPresetPick)
				}
				if preset != "" {
					ui.Row(c).Key("conversation-active-preset").Label("Conversation agent preset").Tooltip(preset).Height(22).MaxWidth(180).MinWidth(14).Shrink(0).Gap(4).AlignItems(ui.Center).Padding(0, 2, 0, 0).Children(func() {
						ui.Icon(c, conversationHeaderIcons["AgentPresetOutline"]).Width(14).Height(14).Shrink(0).TextColor(muted).Opacity(0.7)
						if !compact {
							ui.Text(c, preset).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted).MinWidth(0).MaxWidth(160).NoWrap().Ellipsis("…")
						}
					})
				}
				conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, muted, m.OnOpenLauncher, 0)
			})
		})
		ui.Row(c).Key("conversation-header-utilities").Shrink(0).AlignItems(ui.Center).Gap(gap).Padding(0, 0, 0, utilitiesMargin).Children(func() {
			if m.Store != nil {
				jobs := m.Store.NativeJobs()
				state := jobs.Snapshot()
				button := conversationHeaderButton(c, "conversation-jobs", archiveText("后台任务", "Jobs"), "ClockOutline", compact, muted, func() { jobs.Open(state.Generation) }, 0)
				button.Disabled(!state.Ready)
			}
			openAgents := m.OnOpenAgents
			if openAgents == nil {
				openAgents = m.OnOpenAssistant
			}
			conversationHeaderButton(c, "conversation-agents", i18n.T(i18n.KeyFrameCaptionAgents), "AgentPresetOutline", compact, muted, openAgents, 0)
			if !m.TitlebarPreferences.HideSurfacesToggle {
				conversationHeaderButton(c, "conversation-docs", i18n.T(i18n.KeyFrameCaptionProjectDocs), "PanelLeftOutline", true, muted, m.OnToggleTools, 180)
			}
			if !m.TitlebarPreferences.HideTerminalToggle {
				conversationHeaderButton(c, "conversation-host-terminal", termui.HostTitle(), "CodeOutline", compact, muted, m.OnOpenHostTerminal, 0)
				conversationHeaderButton(c, "conversation-terminal", i18n.T(i18n.KeyTerminalTitle), "PanelLeftOutline", true, muted, m.OnToggleTerminal, 90)
			}
		})
		ui.Row(c).Key("conversation-header-corner").Shrink(0).AlignItems(ui.Center).Children(func() {
			conversationHeaderButton(c, "conversation-settings", i18n.T(i18n.KeySettingsOpen), "SettingsOutline", true, muted, m.OnOpenSettings, 0)
		})
	})
}

func conversationHeaderButton(c *ui.Context, key, label, icon string, compact bool, ink ui.Color, action func(), rotation float32) ui.Element {
	if action == nil {
		return ui.Element{}
	}
	button := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Height(28).MinWidth(28).Shrink(0).Padding(0, 6).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(ink)
	button.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
			ui.Icon(c, conversationHeaderIcons[icon]).Width(16).Height(16).TextColor(ink).Rotate(rotation)
			if !compact {
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(20)).NoWrap().TextColor(ink)
			}
		})
	})
	if button.Clicked() {
		action()
	}
	return button
}
