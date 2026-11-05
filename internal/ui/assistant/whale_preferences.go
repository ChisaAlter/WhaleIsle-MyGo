package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type WhalePreferencesModel struct {
	State         assiststore.WhalePreferencesSnapshot
	OnRefresh     func()
	OnEdit        func(pluginrpc.WhaleModelRoute, bool)
	OnSave        func(string)
	OnToggleSkill func(string, bool)
}

func whalePreferencesBlock(c *ui.Context, m WhalePreferencesModel, blocked bool, fg, muted, border, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	if !blocked && s.Host != "" && !s.Loaded && !s.Busy && s.Error == "" && s.Notice == "" {
		m.OnRefresh()
	}
	blocked = blocked || s.Busy || s.Host == ""
	ui.Column(c.Key("whale-own-preferences")).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantWhalePreferences)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshPreferences)).AlignSelf(ui.Start).Height(28).Padding(0, 12).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "whale_preferences:39", "", s.Notice != "", func() string { return s.Notice })
		if s.Server.HomeDir == "" {
			return
		}
		disabled := blocked || !s.Loaded
		route := s.Route
		ui.Text(c, i18n.T(i18n.KeyW4AssistantTheDefaultModelForHerOwnSessionAfter)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		labels := []string{i18n.T(i18n.KeyMiscAssistantFollowSessionDefault)}
		choices := map[string]pluginrpc.WhaleModelRoute{labels[0]: {ReasoningEffort: route.ReasoningEffort}}
		selected := labels[0]
		found := route.Provider == "" && route.Model == ""
		efforts := []string{"", "low", "medium", "high"}
		for _, group := range s.Catalog.Groups {
			for _, model := range group.Models {
				label := fmt.Sprintf("%d · %s / %s", len(labels), group.Id, model.Id)
				labels = append(labels, label)
				choices[label] = pluginrpc.WhaleModelRoute{Provider: group.Id, Model: model.Id, ReasoningEffort: route.ReasoningEffort}
				if group.Id == route.Provider && model.Id == route.Model {
					selected = label
					found = true
					if model.Reasoning != nil {
						for _, effort := range model.Reasoning.Efforts {
							present := false
							for _, v := range efforts {
								if v == effort.Id {
									present = true
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
			selected = i18n.T(i18n.KeyMiscAssistantCurrent) + route.Provider + " / " + route.Model
			labels = append(labels, selected)
			choices[selected] = route
		}
		if s.ModelsError != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantModelCatalog)+s.ModelsError).TextColor(accent).Selectable()
		}
		ui.Select(c.Key("whale-own-model"), &selected, labels).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled || s.ModelsError != "").OnChange(func() {
			if v, ok := choices[selected]; ok && m.OnEdit != nil {
				m.OnEdit(v, s.IMDefault)
				if m.OnSave != nil {
					m.OnSave("model")
				}
			}
		})
		effortLabels := []string{i18n.T(i18n.KeyMiscAssistantDefaultEffort)}
		effortValues := map[string]string{effortLabels[0]: ""}
		effortSelected := effortLabels[0]
		effortFound := route.ReasoningEffort == ""
		for _, e := range efforts {
			if e == "" {
				continue
			}
			effortLabels = append(effortLabels, e)
			effortValues[e] = e
			if route.ReasoningEffort == e {
				effortSelected = e
				effortFound = true
			}
		}
		if !effortFound {
			effortSelected = route.ReasoningEffort
			effortLabels = append(effortLabels, effortSelected)
			effortValues[effortSelected] = effortSelected
		}
		ui.Select(c.Key("whale-own-effort"), &effortSelected, effortLabels).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled).OnChange(func() {
			route.ReasoningEffort = effortValues[effortSelected]
			if m.OnEdit != nil {
				m.OnEdit(route, s.IMDefault)
				if m.OnSave != nil {
					m.OnSave("model")
				}
			}
		})
		im := s.IMDefault
		if ui.Checkbox(c.Key("whale-im-default"), &im, i18n.T(i18n.KeyMiscAssistantShareConversationWithIm)).Disabled(disabled).Changed() && m.OnEdit != nil {
			m.OnEdit(s.Route, im)
			if m.OnSave != nil {
				m.OnSave("im")
			}
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantImBotsWithoutAnotherPresetJoinHerResi2)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		enabled := 0
		for _, skill := range s.Server.Skills {
			if skill.Enabled {
				enabled++
			}
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantWhaleSkills), enabled, len(s.Server.Skills))).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		if len(s.Server.Skills) == 0 {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantNoSkillsYetPutSkillsNameSkillMdUnder)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			label := "技能目录："
			if !strings.HasPrefix(i18n.Locale(), "zh") {
				label = "Skills directory: "
			}
			ui.Text(c, label+s.Server.HomeDir).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, skill := range s.Server.Skills {
			skill := skill
			on := skill.Enabled
			if ui.Checkbox(c.Key("whale-skill-"+skill.Name), &on, skill.Name).Disabled(disabled).Changed() && m.OnToggleSkill != nil {
				m.OnToggleSkill(skill.Name, on)
			}
		}
	})
}
