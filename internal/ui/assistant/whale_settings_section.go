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
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(fg)
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
// request models merely by mounting the section.
func WhaleSettingsSection(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	w := m.WhaleSettings
	s := w.State
	p := w.Preferences
	ps := p.State
	anchor := ui.Column(c.Key("pet-whale-feature-" + s.HostKey)).FillWidth().MinWidth(0).Gap(4)
	dialog := ui.Local(anchor, "dialog", func() string { return "" })
	blocked := !m.HarnessReady || !m.WhaleEnabled || m.Busy || m.PluginRPCBusy || s.Busy || s.HostKey == ""
	disabled := blocked || !s.Loaded
	prefDisabled := blocked || ps.Busy || !ps.Loaded || ps.Host == ""
	anchor.Children(func() {
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.Button(c, whaleSectionCopy("读取助理设置", "Refresh assistant settings")).Disabled(blocked || w.OnRefresh == nil).Clicked() {
				w.OnRefresh()
				if p.OnRefresh != nil {
					p.OnRefresh()
				}
			}
		})
		if s.Busy || ps.Busy {
			ui.Text(c, whaleSectionCopy("正在等待 Host…", "Waiting for Host…")).FillWidth().TextColor(muted)
		}
		for _, message := range []string{s.Error, ps.Error, ps.ModelsError, s.Notice, ps.Notice} {
			if message != "" {
				ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		}
		if !s.Loaded {
			ui.Text(c, whaleSectionCopy("请先读取当前 Host 的助理设置。", "Refresh to read this Host's assistant settings.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		whaleSectionRow(c, "pet-whale-identity", whaleSectionCopy("名字与称呼", "Name and title"), whaleSectionCopy("名字和称呼各限 64 字符。", "Name and title are limited to 64 characters each."), fg, muted, border, func() {
			name, title := s.Name, s.UserTitle
			ui.TextInput(c, &name).FillWidth().Label("Whale name").Disabled(disabled).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("name", name)
				}
			})
			ui.TextInput(c, &title).FillWidth().Label("Whale user title").Disabled(disabled).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit("title", title)
				}
			})
			if ui.Button(c, whaleSectionCopy("保存名字与称呼", "Save name and title")).Disabled(disabled || w.OnSave == nil || pluginrpc.WhaleTextLength(name) > 64 || pluginrpc.WhaleTextLength(title) > 64).Clicked() {
				w.OnSave("identity")
			}
		})
		whaleSectionRow(c, "pet-whale-model", whaleSectionCopy("模型与推理档", "Model and reasoning effort"), whaleSectionCopy("她自身会话的默认模型；留空跟随会话默认。", "The default model for her conversation; leave blank to follow the session default."), fg, muted, border, func() {
			route := ps.Route
			labels := []string{whaleSectionCopy("跟随会话默认", "Follow session default")}
			choices := map[string]pluginrpc.WhaleModelRoute{labels[0]: {ReasoningEffort: route.ReasoningEffort}}
			selected := labels[0]
			found := route.Provider == "" && route.Model == ""
			efforts := []string{""}
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
								efforts = append(efforts, effort.Id)
							}
						}
					}
				}
			}
			if !found {
				selected = route.Provider + " / " + route.Model
				labels = append(labels, selected)
			}
			ui.Select(c, &selected, labels).FillWidth().Label("Whale model").Disabled(prefDisabled || ps.ModelsError != "").OnChange(func() {
				if v, ok := choices[selected]; ok && p.OnEdit != nil {
					p.OnEdit(v, ps.IMDefault)
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
			ui.Select(c, &effort, efforts).FillWidth().Label("Whale reasoning effort").Disabled(prefDisabled).OnChange(func() {
				route.ReasoningEffort = effort
				if p.OnEdit != nil {
					p.OnEdit(route, ps.IMDefault)
				}
			})
			if ui.Button(c, whaleSectionCopy("保存助理模型", "Save assistant model")).Disabled(prefDisabled || p.OnSave == nil || ps.Route == ps.Server.Route()).Clicked() {
				p.OnSave("model")
			}
		})
		whaleSectionRow(c, "pet-whale-im", whaleSectionCopy("IM 默认对话", "Default IM conversation"), whaleSectionCopy("未指定其他预设的 IM 机器人进入她的常驻会话。", "IM bots without another preset join her resident conversation."), fg, muted, border, func() {
			im := ps.IMDefault
			if ui.Switch(c, &im).Label("Whale default IM").Disabled(prefDisabled).Changed() && p.OnEdit != nil {
				p.OnEdit(ps.Route, im)
			}
			if ui.Button(c, whaleSectionCopy("保存 IM 默认", "Save IM default")).Disabled(prefDisabled || p.OnSave == nil || ps.IMDefault == ps.Server.IMDefault).Clicked() {
				p.OnSave("im")
			}
		})
		for _, item := range []struct{ id, title, hint string }{{"persona", whaleSectionCopy("额外人设", "Extra persona"), s.PersonaText}, {"memory", whaleSectionCopy("长期记忆", "Long-term memory"), whaleSectionCopy("随她的常驻会话注入。", "Injected into her resident conversation.")}, {"skills", whaleSectionCopy("技能", "Skills"), fmt.Sprintf(whaleSectionCopy("当前 Host 的 %d 个技能", "%d skills on this Host"), len(ps.Server.Skills))}} {
			whaleSectionRow(c, "pet-whale-"+item.id, item.title, item.hint, fg, muted, border, func() {
				if ui.Button(c, whaleSectionCopy("管理", "Manage")).Label("manage-whale-" + item.id).Disabled(disabled).Clicked() {
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
		panel.Width(min(float32(560), max(float32(0), vw-48))).Height(min(float32(600), max(float32(0), vh-48))).Radius(16).Clip().Background(chrome.Color(colors.BG)).Children(func() {
			ui.Scroll(c).Fill().MinWidth(0).MinHeight(0).Padding(20).Gap(10).Children(func() {
				ui.Text(c, whaleSectionCopy("助理设置", "Assistant settings")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				if ui.Button(c, whaleSectionCopy("关闭", "Close")).Clicked() {
					*dialog = ""
				}
				if *dialog == "skills" {
					if len(ps.Server.Skills) == 0 {
						ui.Text(c, whaleSectionCopy("还没有技能。", "No skills yet.")).FillWidth().TextColor(muted)
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
				ui.TextArea(c, &value).FillWidth().Lines(6, 14).Label("Whale " + field).Disabled(disabled).OnChange(func() {
					if w.OnEdit != nil {
						w.OnEdit(field, value)
					}
				})
				ui.Text(c, fmt.Sprintf("%d / %d", pluginrpc.WhaleTextLength(value), limit)).TextColor(muted)
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					if ui.Button(c, whaleSectionCopy("保存", "Save")).Disabled(disabled || w.OnSave == nil || pluginrpc.WhaleTextLength(value) > limit).Clicked() {
						w.OnSave(field)
					}
					if ui.Button(c, whaleSectionCopy("还原草稿", "Restore draft")).Disabled(disabled || w.OnRestore == nil).Clicked() {
						w.OnRestore(field)
					}
					if field == "memory" && w.OnReviewClear != nil && ui.Button(c, whaleSectionCopy("清空记忆…", "Clear memory…")).Disabled(disabled).Clicked() {
						w.OnReviewClear()
					}
				})
				if field == "memory" && s.ConfirmClear {
					ui.Text(c, whaleSectionCopy("确定清空她的长期记忆？此操作不可撤销。", "Clear her long-term memory? This cannot be undone.")).FillWidth().TextColor(fg)
					ui.Row(c).Wrap().Gap(8).Children(func() {
						if ui.Button(c, whaleSectionCopy("取消清空", "Cancel clear")).Disabled(blocked || w.OnCancelClear == nil).Clicked() {
							w.OnCancelClear()
						}
						if ui.Button(c, whaleSectionCopy("确认清空长期记忆", "Confirm clear memory")).Disabled(disabled || w.OnConfirmClear == nil).Clicked() {
							w.OnConfirmClear()
						}
					})
				}
			})
		})
	})
	if !open {
		*dialog = ""
	}
}
