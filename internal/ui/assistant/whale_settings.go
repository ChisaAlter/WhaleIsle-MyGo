package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
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
	if s.HostKey != "" && !s.Loaded && !s.Busy && s.Error == "" && s.Notice == "" && m.HarnessReady && m.WhaleEnabled {
		w.OnRefresh()
	}
	blocked := s.Busy || m.PluginRPCBusy || m.Busy || !m.HarnessReady || !m.WhaleEnabled || s.HostKey == ""
	ui.Column(c.Key("whale-settings")).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWhaleSettings)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, i18n.T(i18n.KeyW4AssistantReadRefreshHostContentRefresh)).Disabled(blocked).Clicked() {
				w.OnRefresh()
			}
		})
		ui.Text(c, i18n.T(i18n.KeyW4AssistantReadingAndSavingAffectOnlyTheCurrentH)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Busy {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWaitingForHost2)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeEditingOrRetrying2)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		whalePreferencesBlock(c, w.Preferences, blocked, fg, muted, accent)
		if s.Server.HomeDir == "" {
			return
		}
		ui.Text(c, s.Server.HomeDir+" · "+s.Server.SessionID).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		disabled := blocked || !s.Loaded
		ui.Column(c.Key("whale-memory")).Background(field).Border(1, border).Radius(8).Padding(12).Gap(7).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantLongTermMemory)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantMemoryMdIsInjectedIntoWhaleSSessi)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			memory := s.Memory
			ui.TextArea(c.Key("whale-memory-editor"), &memory).Label("Whale long-term memory").FillWidth().Lines(6, 14).Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("memory", memory)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 40000 UTF-16 · %s", pluginrpc.WhaleTextLength(memory), whaleDraftLabel(memory != s.Server.Memory))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantSaveMemory)).Disabled(disabled || pluginrpc.WhaleTextLength(memory) > 40000).Clicked() {
					w.OnSave("memory")
				}
				if w.OnRestore != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRestoreMemoryDraft)).Disabled(disabled || memory == s.Server.Memory).Clicked() {
					w.OnRestore("memory")
				}
				if w.OnReviewClear != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantClearMemory)).Disabled(disabled).Clicked() {
					w.OnReviewClear()
				}
			})
			if s.ConfirmClear {
				ui.Column(c.Key("whale-memory-clear-review")).Border(1, accent).Radius(8).Padding(10).Gap(7).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantClearLongTermMemoryThisCannotBeUn)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantCancelKeepsThisDraft)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Row(c).Gap(8).Children(func() {
						if w.OnCancelClear != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelClear)).Disabled(blocked).Clicked() {
							w.OnCancelClear()
						}
						if w.OnConfirmClear != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmClearMemory)).Disabled(disabled).Clicked() {
							w.OnConfirmClear()
						}
					})
				})
			}
		})
		ui.Column(c.Key("whale-persona")).Background(field).Border(1, border).Radius(8).Padding(12).Gap(7).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantExtraPersona)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantAppendedToHerPersonaLeaveBlankToS)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			persona := s.PersonaText
			ui.TextArea(c.Key("whale-persona-editor"), &persona).Label("Whale extra persona").FillWidth().Lines(3, 8).Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("persona", persona)
				}
			})
			ui.Text(c, fmt.Sprintf("%d / 4000 UTF-16 · %s", pluginrpc.WhaleTextLength(persona), whaleDraftLabel(persona != s.Server.PersonaText))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				if w.OnSave != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantSavePersona)).Disabled(disabled || pluginrpc.WhaleTextLength(persona) > 4000).Clicked() {
					w.OnSave("persona")
				}
				if w.OnRestore != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRestorePersonaDraft)).Disabled(disabled || persona == s.Server.PersonaText).Clicked() {
					w.OnRestore("persona")
				}
			})
		})
		ui.Column(c.Key("whale-identity")).Gap(7).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNameAndTitle)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
			name, title := s.Name, s.UserTitle
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantName)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			nameInput := ui.TextInput(c.Key("whale-name"), &name).Label("Whale name").FillWidth().Disabled(disabled).OnChange(func() {
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
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHowSheCallsYou)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			titleInput := ui.TextInput(c.Key("whale-user-title"), &title).Label("Whale user title").FillWidth().Disabled(disabled).OnChange(func() {
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
				if w.OnRestore != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRestoreNameDraft)).Disabled(disabled).Clicked() {
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
