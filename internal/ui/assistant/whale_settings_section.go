package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func whaleSectionCopy(zh, en string) string {
	if i18n.Speaks("zh") {
		return zh
	}
	return en
}
func whaleSectionRow(c *ui.Context, key, title, hint string, fg, muted, border ui.Color, controls func()) {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0)
	width := row.Bounds().W
	row.Draw(func(_ *ui.Painter, b ui.Rect) {
		if b.W != width {
			c.Invalidate()
		}
	})
	row.Children(func() {
		body := func() {
			copy := ui.Column(c).MinWidth(0).Gap(4)
			if width >= 420 {
				copy.Grow(1)
			} else {
				copy.FillWidth()
			}
			copy.Children(func() {
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				if hint != "" {
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
			})
			control := ui.Column(c).MinWidth(0).Gap(6)
			if width >= 420 {
				control.Width(180).Shrink(0)
			} else {
				control.FillWidth()
			}
			control.Children(controls)
		}
		if width >= 420 {
			ui.Row(c).FillWidth().Gap(24).Padding(14, 0).AlignItems(ui.Center).Children(body)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(14, 0).Children(body)
		}
		ui.Box(c).FillWidth().Height(.5).Background(border)
	})
}

type whaleSectionDialog struct {
	field, pending string
	completed      uint64
}

// WhaleSettingsSection renders the original settings.pet.item contribution.
// It reuses the application-owned Host editor and does not create sessions or
// invoke a model merely by mounting the section. Only settings/catalog reads
// are made automatically, as in the original contribution.
func WhaleSettingsSection(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	w := m.WhaleSettings
	s := w.State
	p := w.Preferences
	ps := p.State
	if m.HarnessReady && m.WhaleEnabled && s.HostKey != "" {
		if !s.Loaded && !s.Busy && s.Error == nil && s.Notice == "" && w.OnRefresh != nil {
			w.OnRefresh()
		}
		if !ps.Loaded && !ps.Busy && ps.Error == nil && ps.Notice == "" && p.OnRefresh != nil {
			p.OnRefresh()
		}
	}
	anchor := ui.Column(c.Key("pet-whale-feature-" + s.HostKey)).FillWidth().MinWidth(0).Gap(4)
	dialog := ui.Local(anchor, "dialog", func() whaleSectionDialog { return whaleSectionDialog{} })
	if dialog.pending != "" && s.CompletedSerial > dialog.completed && s.CompletedAction == dialog.pending {
		dialog.field, dialog.pending = "", ""
	}
	blocked := !m.HarnessReady || !m.WhaleEnabled || m.Busy || m.PluginRPCBusy || s.Busy || s.HostKey == ""
	disabled := blocked || !s.Loaded
	prefDisabled := blocked || ps.Busy || !ps.Loaded || ps.Host == ""
	anchor.Children(func() {
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			label := whaleSectionCopy(i18n.T(i18n.KeyW4AssistantReadAssistantSettings), "Refresh assistant settings")
			if s.Busy || ps.Busy {
				label = whaleSectionCopy(i18n.T(i18n.KeyW4AssistantWaitingForTheHost), "Waiting for Host…")
			}
			if aButton(c, label, blocked || ps.Busy || w.OnRefresh == nil).Clicked() {
				w.OnRefresh()
				if p.OnRefresh != nil {
					p.OnRefresh()
				}
			}
		})
		for _, message := range []string{s.Error.SentenceText(), ps.Error.SentenceText(), ps.ModelsError.SentenceText(), s.Notice, ps.Notice} {
			if message != "" {
				ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		}
		if !s.Loaded {
			ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantReadTheAssistantSettingsForTheCurrent), "Refresh to read this Host's assistant settings.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		whaleSectionRow(c, "pet-whale-identity", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantNameAndTitle), "Name and title"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantTheNameAndTitleAreEachLimitedTo64Ch), "Name and title are limited to 64 characters each."), fg, muted, border, func() {
			name, title := s.Name, s.UserTitle
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
			titleInput := aInput(c.Key("whale-title"), &title, coss.InputOptions{Label: "Whale user title", Disabled: disabled}).OnChange(func() {
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
		})
		whaleSectionRow(c, "pet-whale-model", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantModelAndReasoningEffort), "Model and reasoning effort"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantTheDefaultModelForHerOwnSessionLeave), "The default model for her conversation; leave blank to follow the session default."), fg, muted, border, func() {
			route := ps.Route
			labels := []string{whaleSectionCopy(i18n.T(i18n.KeyW4AssistantFollowSessionDefault), "Follow session default")}
			choices := map[string]pluginrpc.WhaleModelRoute{labels[0]: {ReasoningEffort: route.ReasoningEffort}}
			selected := labels[0]
			found := route.Provider == "" && route.Model == ""
			efforts := []string{"", "low", "medium", "high"}
			for _, group := range ps.Catalog.Groups {
				for _, model := range group.Models {
					label := fmt.Sprintf("%d · %s / %s", len(labels), group.Id, model.Id)
					labels = append(labels, label)
					choices[label] = pluginrpc.WhaleModelRoute{Provider: group.Id, Model: model.Id, ReasoningEffort: route.ReasoningEffort}
					if route.Provider == group.Id && route.Model == model.Id {
						selected = label
						found = true
						if model.Reasoning != nil {
							for _, effort := range model.Reasoning.Efforts {
								present := false
								for _, option := range efforts {
									if option == effort.Id {
										present = true
										break
									}
								}
								if !present {
									efforts = append(efforts, effort.Id)
								}
							}
						}
					}
				}
			}
			if !found {
				selected = route.Provider + " / " + route.Model
				labels = append(labels, selected)
			}
			aSelect(c, &selected, labels, "Whale model", prefDisabled || ps.ModelsError != nil).FillWidth().OnChange(func() {
				if v, ok := choices[selected]; ok && p.OnEdit != nil {
					p.OnEdit(v, ps.IMDefault)
					if p.OnSave != nil {
						p.OnSave("model")
					}
				}
			})
			present := false
			for _, effort := range efforts {
				if effort == route.ReasoningEffort {
					present = true
				}
			}
			if !present {
				efforts = append(efforts, route.ReasoningEffort)
			}
			defaultEffort := i18n.T(i18n.KeyMiscAssistantDefaultEffort)
			effortLabels := []string{defaultEffort}
			effortValues := map[string]string{defaultEffort: ""}
			effort := defaultEffort
			for _, value := range efforts {
				if value != "" {
					effortLabels = append(effortLabels, value)
					effortValues[value] = value
					if value == route.ReasoningEffort {
						effort = value
					}
				}
			}
			aSelect(c, &effort, effortLabels, "Whale reasoning effort", prefDisabled).FillWidth().OnChange(func() {
				route.ReasoningEffort = effortValues[effort]
				if p.OnEdit != nil {
					p.OnEdit(route, ps.IMDefault)
					if p.OnSave != nil {
						p.OnSave("model")
					}
				}
			})
		})
		whaleSectionRow(c, "pet-whale-im", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantImDefaultChat), "Default IM conversation"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantImBotsWithoutAnotherPresetJoinHerResi), "IM bots without another preset join her resident conversation."), fg, muted, border, func() {
			im := ps.IMDefault
			if coss.Switch(c, &im, "", coss.CheckOptions{Label: "Whale default IM", Disabled: prefDisabled}).Changed() && p.OnEdit != nil {
				p.OnEdit(ps.Route, im)
				if p.OnSave != nil {
					p.OnSave("im")
				}
			}
		})
		for _, item := range []struct{ id, title, hint string }{{"persona", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantExtraPersona), "Extra persona"), s.PersonaText}, {"memory", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantLongTermMemory), "Long-term memory"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantInjectedWithHerResidentSession), "Injected into her resident conversation.")}, {"skills", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantSkills), "Skills"), fmt.Sprintf(whaleSectionCopy(i18n.T(i18n.KeyW4AssistantDSkillsOnTheCurrentHost), "%d skills on this Host"), len(ps.Server.Skills))}} {
			whaleSectionRow(c, "pet-whale-"+item.id, item.title, item.hint, fg, muted, border, func() {
				if aButton(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantManage), "Manage"), disabled).Label("manage-whale-" + item.id).Clicked() {
					if item.id == "memory" && w.OnCancelClear != nil {
						w.OnCancelClear()
					}
					if item.id == "persona" && w.OnRestore != nil {
						w.OnRestore("persona")
					}
					dialog.field, dialog.pending = item.id, ""
				}
			})
		}
	})
	if dialog.field == "" {
		return
	}
	open := true
	dismissBusy := s.Busy || ps.Busy
	dismiss := func() {
		if dismissBusy {
			return
		}
		if dialog.field == "memory" && s.ConfirmClear {
			if w.OnCancelClear != nil {
				w.OnCancelClear()
			}
			return
		}
		dialog.field, dialog.pending = "", ""
	}
	ui.DialogBaseWithOptions(c, &open, ui.DialogOptions{DismissDisabled: dismissBusy, OnDismiss: dismiss}, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .24))
		vw, vh := c.Size()
		chrome.FrostSurface(panel.Width(min(float32(560), max(float32(0), vw-48))).Height(min(float32(600), max(float32(0), vh-48))).Clip(), colors, m.SystemDark, chrome.Color(colors.BG), theme.ContextRadii(c).Panel).Children(func() {
			ui.Scroll(c).Fill().MinWidth(0).MinHeight(0).Padding(20).Gap(10).Children(func() {
				title := whaleSectionCopy(i18n.T(i18n.KeyW4AssistantExtraPersona), "Extra persona")
				if dialog.field == "memory" {
					title = whaleSectionCopy(i18n.T(i18n.KeyW4AssistantLongTermMemory), "Long-term memory")
					if s.ConfirmClear {
						title = whaleSectionCopy("清空长期记忆", "Clear long-term memory")
					}
				} else if dialog.field == "skills" {
					title = whaleSectionCopy(i18n.T(i18n.KeyW4AssistantSkills), "Skills")
				}
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(fg)
				if aButton(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClose), "Close"), dismissBusy).Clicked() {
					dismiss()
				}
				for _, message := range []string{s.Error.SentenceText(), ps.Error.SentenceText(), ps.ModelsError.SentenceText(), s.Notice, ps.Notice} {
					if message != "" {
						ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
				}
				if aButton(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantReadAssistantSettings), "Refresh assistant settings"), blocked || ps.Busy || w.OnRefresh == nil).Clicked() {
					w.OnRefresh()
					if p.OnRefresh != nil {
						p.OnRefresh()
					}
				}
				if dialog.field == "skills" {
					if len(ps.Server.Skills) == 0 {
						ui.Text(c, whaleSectionCopy("还没有技能。请在下方助理目录的 skills/<名称>/SKILL.md 中放入技能，然后刷新。", "No skills yet. Add skills/<name>/SKILL.md under the assistant directory below, then refresh.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						homeDir := ps.Server.HomeDir
						if homeDir == "" {
							homeDir = whaleSectionCopy("目录尚未读取，请刷新助理设置。", "Refresh assistant settings to read the directory.")
						}
						ui.Text(c, homeDir).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg).Selectable()
					}
					for _, skill := range ps.Server.Skills {
						enabled := skill.Enabled
						whaleSectionRow(c, "whale-skill-"+skill.Name, skill.Name, "", fg, muted, border, func() {
							if coss.Switch(c, &enabled, "", coss.CheckOptions{Label: skill.Name, Disabled: prefDisabled || p.OnToggleSkill == nil}).Changed() {
								p.OnToggleSkill(skill.Name, enabled)
							}
						})
					}
					return
				}
				field := "persona"
				value := s.PersonaText
				limit := 4000
				if dialog.field == "memory" {
					field = "memory"
					value = s.Memory
					limit = 40000
				}
				if !(field == "memory" && s.ConfirmClear) {
					draftLen := pluginrpc.WhaleTextLength(value)
					aTextarea(c, &value, coss.TextareaOptions{Label: "Whale " + field, MaxLines: 14, Disabled: disabled}).OnChange(func() {
						if w.OnEdit != nil {
							w.OnEdit(field, value)
						}
					})
					ui.Text(c, fmt.Sprintf("%d / %d", draftLen, limit)).TextColor(muted)
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						saveLabel := whaleSectionCopy(i18n.T(i18n.KeyW4AssistantSave), "Save")
						if s.Busy {
							saveLabel = whaleSectionCopy(i18n.T(i18n.KeyW4AssistantWaitingForTheHost), "Waiting for Host…")
						}
						if aPrimary(c, saveLabel, disabled || w.OnSave == nil || draftLen > limit).Clicked() {
							dialog.pending, dialog.completed = field, s.CompletedSerial
							w.OnSave(field)
						}
						if aButton(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantRestoreDraft), "Restore draft"), disabled || w.OnRestore == nil).Clicked() {
							w.OnRestore(field)
						}
						if field == "memory" && w.OnReviewClear != nil && aDangerOutline(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClearMemory), "Clear memory…"), disabled).Clicked() {
							w.OnReviewClear()
						}
					})
				}
				if field == "memory" && s.ConfirmClear {
					ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClearHerLongTermMemoryThisCannotBeUn), "Clear her long-term memory? This cannot be undone.")).FillWidth().TextColor(fg)
					ui.Row(c).Wrap().Gap(8).Children(func() {
						if aButton(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantCancelClear), "Cancel clear"), dismissBusy || w.OnCancelClear == nil).Clicked() {
							w.OnCancelClear()
						}
						if aDanger(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantConfirmClearingLongTermMemory), "Confirm clear memory"), disabled || w.OnConfirmClear == nil).Clicked() {
							dialog.pending, dialog.completed = "clear", s.CompletedSerial
							w.OnConfirmClear()
						}
					})
				}
			})
		})
		chrome.DialogDecor(c, backdrop, panel, colors, 28)
	})
	if !open {
		dismiss()
	}
}
