package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	fonttheme "whaleisle/internal/ui/theme"
)

type WhalePreferencesModel struct {
	State         assiststore.WhalePreferencesSnapshot
	OnRefresh     func()
	OnEdit        func(pluginrpc.WhaleModelRoute, bool)
	OnSave        func(string)
	OnToggleSkill func(string, bool)
}

func whalePreferencesBlock(c *ui.Context, m WhalePreferencesModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy || s.Host == ""
	ui.Column(c.Key("whale-own-preferences")).Gap(8).Children(func() {
		ui.Text(c, "鲸鱼娘 · 模型、IM 与技能 / Whale preferences").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		if ui.Button(c, "读取模型、IM 与技能 / Refresh preferences").Disabled(blocked).Clicked() {
			m.OnRefresh()
		}
		if s.Busy {
			ui.Text(c, "正在等待 Host… / Waiting for Host…").TextColor(muted)
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(muted).Selectable()
		}
		if s.Server.HomeDir == "" {
			return
		}
		disabled := blocked || !s.Loaded
		route := s.Route
		ui.Text(c, "她自身会话的默认模型；保存后 Host 同步已有常驻会话。留空跟随会话默认。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		labels := []string{"跟随会话默认 / Follow session default"}
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
			selected = "当前配置 / Current: " + route.Provider + " / " + route.Model
			labels = append(labels, selected)
			choices[selected] = route
		}
		if s.ModelsError != "" {
			ui.Text(c, "模型目录读取失败 / Model catalog: "+s.ModelsError).TextColor(accent).Selectable()
		}
		ui.Select(c.Key("whale-own-model"), &selected, labels).FillWidth().Disabled(disabled || s.ModelsError != "").OnChange(func() {
			if v, ok := choices[selected]; ok && m.OnEdit != nil {
				m.OnEdit(v, s.IMDefault)
			}
		})
		effortLabels := []string{"默认推理档 / Default effort"}
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
		ui.Select(c.Key("whale-own-effort"), &effortSelected, effortLabels).FillWidth().Disabled(disabled).OnChange(func() {
			route.ReasoningEffort = effortValues[effortSelected]
			if m.OnEdit != nil {
				m.OnEdit(route, s.IMDefault)
			}
		})
		if ui.Button(c, "保存 Whale 模型与推理档 / Save Whale model").Disabled(disabled || route == s.Server.Route()).Clicked() && m.OnSave != nil {
			m.OnSave("model")
		}
		im := s.IMDefault
		if ui.Checkbox(c.Key("whale-im-default"), &im, "IM 接入同一对话 / Share conversation with IM").Disabled(disabled).Changed() && m.OnEdit != nil {
			m.OnEdit(s.Route, im)
		}
		ui.Text(c, "未指定其他预设的 IM 机器人进入她的常驻会话，与桌面共用记录。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "保存 IM 默认 / Save IM default").Disabled(disabled || im == s.Server.IMDefault).Clicked() && m.OnSave != nil {
			m.OnSave("im")
		}
		enabled := 0
		for _, skill := range s.Server.Skills {
			if skill.Enabled {
				enabled++
			}
		}
		ui.Text(c, fmt.Sprintf("她的技能 / Whale skills · %d / %d 已启用", enabled, len(s.Server.Skills))).Bold().TextColor(fg)
		if len(s.Server.Skills) == 0 {
			ui.Text(c, "还没有技能；在此 Host 的 Whale 目录下放入 skills/<名称>/SKILL.md 后刷新。\n"+s.Server.HomeDir).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
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
