package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func whaleSectionCopy(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
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
		if !s.Loaded && !s.Busy && s.Error == "" && s.Notice == "" && w.OnRefresh != nil {
			w.OnRefresh()
		}
		if !ps.Loaded && !ps.Busy && ps.Error == "" && ps.Notice == "" && p.OnRefresh != nil {
			p.OnRefresh()
		}
	}
	anchor := ui.Column(c.Key("pet-whale-feature-" + s.HostKey)).FillWidth().MinWidth(0).Gap(4)
	dialog := ui.Local(anchor, "dialog", func() string { return "" })
	blocked := !m.HarnessReady || !m.WhaleEnabled || m.Busy || m.PluginRPCBusy || s.Busy || s.HostKey == ""
	disabled := blocked || !s.Loaded
	prefDisabled := blocked || ps.Busy || !ps.Loaded || ps.Host == ""
	anchor.Children(func() {
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantReadAssistantSettings), "Refresh assistant settings")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(blocked || w.OnRefresh == nil).Clicked() {
				w.OnRefresh()
				if p.OnRefresh != nil {
					p.OnRefresh()
				}
			}
		})
		if s.Busy || ps.Busy {
			ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantWaitingForTheHost), "Waiting for Host…")).FillWidth().TextColor(muted)
		}
		for _, message := range []string{s.Error, ps.Error, ps.ModelsError, s.Notice, ps.Notice} {
			if message != "" {
				ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		}
		if !s.Loaded {
			ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantReadTheAssistantSettingsForTheCurrent), "Refresh to read this Host's assistant settings.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		whaleSectionRow(c, "pet-whale-identity", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantNameAndTitle), "Name and title"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantTheNameAndTitleAreEachLimitedTo64Ch), "Name and title are limited to 64 characters each."), fg, muted, border, func() {
			name, title := s.Name, s.UserTitle
			nameInput := ui.TextInput(c.Key("whale-name"), &name).FillWidth().Height(32).Radius(8).Border(.5, border).Label("Whale name").Disabled(disabled).OnChange(func() {
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
			titleInput := ui.TextInput(c.Key("whale-title"), &title).FillWidth().Height(32).Radius(8).Border(.5, border).Label("Whale user title").Disabled(disabled).OnChange(func() {
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
			ui.Select(c, &selected, labels).FillWidth().Height(32).Radius(8).Border(.5, border).Label("Whale model").Disabled(prefDisabled || ps.ModelsError != "").OnChange(func() {
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
			effort := route.ReasoningEffort
			ui.Select(c, &effort, efforts).FillWidth().Height(32).Radius(8).Border(.5, border).Label("Whale reasoning effort").Disabled(prefDisabled).OnChange(func() {
				route.ReasoningEffort = effort
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
			if ui.Switch(c, &im).Label("Whale default IM").Disabled(prefDisabled).Changed() && p.OnEdit != nil {
				p.OnEdit(ps.Route, im)
				if p.OnSave != nil {
					p.OnSave("im")
				}
			}
		})
		for _, item := range []struct{ id, title, hint string }{{"persona", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantExtraPersona), "Extra persona"), s.PersonaText}, {"memory", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantLongTermMemory), "Long-term memory"), whaleSectionCopy(i18n.T(i18n.KeyW4AssistantInjectedWithHerResidentSession), "Injected into her resident conversation.")}, {"skills", whaleSectionCopy(i18n.T(i18n.KeyW4AssistantSkills), "Skills"), fmt.Sprintf(whaleSectionCopy(i18n.T(i18n.KeyW4AssistantDSkillsOnTheCurrentHost), "%d skills on this Host"), len(ps.Server.Skills))}} {
			whaleSectionRow(c, "pet-whale-"+item.id, item.title, item.hint, fg, muted, border, func() {
				if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantManage), "Manage")).Label("manage-whale-"+item.id).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(disabled).Clicked() {
					*dialog = item.id
				}
			})
		}
	})
	if *dialog == "" {
		return
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .24))
		vw, vh := c.Size()
		panel.Width(min(float32(560), max(float32(0), vw-48))).Height(min(float32(600), max(float32(0), vh-48))).Radius(28).Clip().Background(chrome.Color(colors.BG)).Children(func() {
			ui.Scroll(c).Fill().MinWidth(0).MinHeight(0).Padding(20).Gap(10).Children(func() {
				ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantAssistantSettings), "Assistant settings")).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(fg)
				if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClose), "Close")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
					*dialog = ""
				}
				if *dialog == "skills" {
					if len(ps.Server.Skills) == 0 {
						ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantNoSkillsYet), "No skills yet.")).FillWidth().TextColor(muted)
					}
					for _, skill := range ps.Server.Skills {
						enabled := skill.Enabled
						whaleSectionRow(c, "whale-skill-"+skill.Name, skill.Name, "", fg, muted, border, func() {
							if ui.Switch(c, &enabled).Label(skill.Name).Disabled(prefDisabled || p.OnToggleSkill == nil).Changed() {
								p.OnToggleSkill(skill.Name, enabled)
							}
						})
					}
					return
				}
				field := "persona"
				value := s.PersonaText
				limit := 4000
				if *dialog == "memory" {
					field = "memory"
					value = s.Memory
					limit = 40000
				}
				ui.TextArea(c, &value).FillWidth().Lines(6, 14).Radius(12).Border(.5, border).Label("Whale " + field).Disabled(disabled).OnChange(func() {
					if w.OnEdit != nil {
						w.OnEdit(field, value)
					}
				})
				ui.Text(c, fmt.Sprintf("%d / %d", pluginrpc.WhaleTextLength(value), limit)).TextColor(muted)
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantSave), "Save")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(disabled || w.OnSave == nil || pluginrpc.WhaleTextLength(value) > limit).Clicked() {
						w.OnSave(field)
					}
					if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantRestoreDraft), "Restore draft")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(disabled || w.OnRestore == nil).Clicked() {
						w.OnRestore(field)
					}
					if field == "memory" && w.OnReviewClear != nil && ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClearMemory), "Clear memory…")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(disabled).Clicked() {
						w.OnReviewClear()
					}
				})
				if field == "memory" && s.ConfirmClear {
					ui.Text(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantClearHerLongTermMemoryThisCannotBeUn), "Clear her long-term memory? This cannot be undone.")).FillWidth().TextColor(fg)
					ui.Row(c).Wrap().Gap(8).Children(func() {
						if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantCancelClear), "Cancel clear")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(blocked || w.OnCancelClear == nil).Clicked() {
							w.OnCancelClear()
						}
						if ui.Button(c, whaleSectionCopy(i18n.T(i18n.KeyW4AssistantConfirmClearingLongTermMemory), "Confirm clear memory")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(disabled || w.OnConfirmClear == nil).Clicked() {
							w.OnConfirmClear()
						}
					})
				}
			})
		})
		chrome.DialogDecor(c, backdrop, panel, colors, 28)
	})
	if !open {
		*dialog = ""
	}
}
