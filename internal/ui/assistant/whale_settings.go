package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type WhaleSettingsModel struct {
	Preferences                                  WhalePreferencesModel
	State                                        assiststore.WhaleSettingsSnapshot
	OnRefresh                                    func()
	OnEdit                                       func(field, value string)
	OnRestore                                    func(field string)
	OnSave                                       func(action string)
	OnReviewClear, OnCancelClear, OnConfirmClear func()
}

func whaleSettingsBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeWhale || m.WhaleSettings.OnRefresh == nil {
		return
	}
	w := m.WhaleSettings
	s := w.State
	if s.HostKey != "" && !s.Loaded && !s.Busy && s.Error == nil && s.Notice == "" && m.HarnessReady && m.WhaleEnabled {
		w.OnRefresh()
	}
	blocked := s.Busy || m.PluginRPCBusy || m.Busy || !m.HarnessReady || !m.WhaleEnabled || s.HostKey == ""
	ui.Column(c.Key("whale-settings")).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWhaleSettings)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if aButton(c, i18n.T(i18n.KeyW4AssistantReadRefreshHostContentRefresh), blocked).Clicked() {
				w.OnRefresh()
			}
		})
		ui.Text(c, i18n.T(i18n.KeyW4AssistantReadingAndSavingAffectOnlyTheCurrentH)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if s.Error != nil {
			ui.Text(c, s.Error.SentenceText()).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		widgets.AckLine(c, "whale_settings:46", s.Notice)
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeEditingOrRetrying2)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		whalePreferencesBlock(c, w.Preferences, blocked, fg, muted, border, accent)
		if s.Server.HomeDir == "" {
			return
		}
		label := "会话："
		if !i18n.Speaks("zh") {
			label = "Session: "
		}
		ui.Text(c, label+s.Server.SessionID).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		disabled := blocked || !s.Loaded
		memCard := ui.Column(c.Key("whale-memory")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
		chrome.GlassSurface(memCard, field, chrome.MaterialPanel, 12)
		memCard.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantLongTermMemory)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantMemoryMdIsInjectedIntoWhaleSSessi)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			memory := s.Memory
			aTextarea(c.Key("whale-memory-editor"), &memory, coss.TextareaOptions{Label: "Whale long-term memory", MaxLines: 8, Disabled: blocked}).MaxHeight(200).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("memory", memory)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 40000 UTF-16 · %s", pluginrpc.WhaleTextLength(memory), whaleDraftLabel(memory != s.Server.Memory))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && aPrimary(c, i18n.T(i18n.KeyMiscAssistantSaveMemory), disabled || pluginrpc.WhaleTextLength(s.Memory) > 40000).Clicked() {
					w.OnSave("memory")
				}
				if w.OnRestore != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantRestoreMemoryDraft), disabled || s.Memory == s.Server.Memory).Clicked() {
					w.OnRestore("memory")
				}
				if w.OnReviewClear != nil && aDangerOutline(c, i18n.T(i18n.KeyMiscAssistantClearMemory), disabled).Clicked() {
					w.OnReviewClear()
				}
			})
			if s.ConfirmClear {
				ui.Column(c.Key("whale-memory-clear-review")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantClearLongTermMemoryThisCannotBeUn)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantCancelKeepsThisDraft)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Row(c).Gap(8).Children(func() {
						if w.OnCancelClear != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantCancelClear), blocked).Clicked() {
							w.OnCancelClear()
						}
						if w.OnConfirmClear != nil && aDanger(c, i18n.T(i18n.KeyMiscAssistantConfirmClearMemory), disabled).Clicked() {
							w.OnConfirmClear()
						}
					})
				})
			}
		})
		personaCard := ui.Column(c.Key("whale-persona")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
		chrome.GlassSurface(personaCard, field, chrome.MaterialPanel, 12)
		personaCard.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantExtraPersona)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantAppendedToHerPersonaLeaveBlankToS)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			persona := s.PersonaText
			aTextarea(c.Key("whale-persona-editor"), &persona, coss.TextareaOptions{Label: "Whale extra persona", MaxLines: 8, Disabled: blocked}).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("persona", persona)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 4000 UTF-16 · %s", pluginrpc.WhaleTextLength(persona), whaleDraftLabel(persona != s.Server.PersonaText))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && aPrimary(c, i18n.T(i18n.KeyMiscAssistantSavePersona), disabled || pluginrpc.WhaleTextLength(s.PersonaText) > 4000).Clicked() {
					w.OnSave("persona")
				}
				if w.OnRestore != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantRestorePersonaDraft), disabled || s.PersonaText == s.Server.PersonaText).Clicked() {
					w.OnRestore("persona")
				}
			})
		})
		idCard := ui.Column(c.Key("whale-identity")).Border(.5, border).Radius(12).Padding(12, 16).Gap(7)
		chrome.GlassSurface(idCard, field, chrome.MaterialPanel, 12)
		idCard.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNameAndTitle)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
			name, title := s.Name, s.UserTitle
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantName)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			nameInput := aInput(c.Key("whale-name"), &name, coss.InputOptions{Label: "Whale name", Disabled: disabled}).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("name", name)
				}
			})
			whaleIdentityCommit(nameInput, s.HostKey, !disabled && name != s.Server.Name && pluginrpc.WhaleTextLength(name) <= 64, func() {
				if w.OnEdit != nil {
					w.OnEdit("name", name)
				}
				if w.OnSave != nil {
					w.OnSave("name")
				}
			})
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHowSheCallsYou)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			titleInput := aInput(c.Key("whale-user-title"), &title, coss.InputOptions{Label: "Whale user title", Disabled: disabled}).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("title", title)
				}
			})
			whaleIdentityCommit(titleInput, s.HostKey, !disabled && title != s.Server.UserTitle && pluginrpc.WhaleTextLength(title) <= 64, func() {
				if w.OnEdit != nil {
					w.OnEdit("title", title)
				}
				if w.OnSave != nil {
					w.OnSave("title")
				}
			})
			ui.Text(c, i18n.T(i18n.KeyW4AssistantBlurTheFieldOrPressEnterToSaveItThe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnRestore != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantRestoreNameDraft), disabled).Clicked() {
					w.OnRestore("identity")
				}
			})
		})
	})
}

// Native inputs report submission and focus per frame. Capture the Host with
// the focus so replacing the editor never commits another Host's old draft.
func whaleIdentityCommit(input ui.Element, host string, dirty bool, save func()) {
	state := ui.Local(input, "commit-focus", func() struct {
		Host    string
		Focused bool
	} {
		return struct {
			Host    string
			Focused bool
		}{}
	})
	focused := input.Focused()
	if state.Host == host && dirty && !input.Composing() && (input.Submitted() || state.Focused && !focused) {
		save()
	}
	state.Host, state.Focused = host, focused
}
func whaleDraftLabel(dirty bool) string {
	if dirty {
		return i18n.T(i18n.KeyMiscAssistantUnsavedDraft)
	}
	return i18n.T(i18n.KeyMiscAssistantMatchesHost)
}
