package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

// configurationModelsCustomized reports whether the profile carries its own model
// catalog (or overrides) rather than the adapter's: the stored layer names one,
// or the draft's catalog differs from the layer beneath the user's.
func configurationModelsCustomized(m Model, s store.ViewState) bool {
	if len(s.Namespaces) == 0 {
		return false
	}
	ns := s.Namespaces[0]
	if ns.NS == "llm-deepseek-account" {
		// The account's catalog is edited as per-model overrides.
		return len(configurationObject(s.Drafts["modelOverrides"])) > 0
	}
	if s.Resets["models"] {
		return false
	}
	var layer any
	_ = json.Unmarshal(ns.User, &layer)
	path := []string{}
	if m.Models != nil {
		path = append(path, m.Models.configurationPath...)
	}
	if profile, ok := providerPathValue(layer, path); ok {
		p := configurationObjectValue(profile)
		_, models := p["models"]
		_, overrides := p["modelOverrides"]
		if models || overrides {
			return true
		}
	}
	if len(path) == 0 {
		return s.Drafts["models"] != store.ValueDraft(ns.Base, "models", store.FieldJSON)
	}
	return s.Dirty
}

// configurationModels is the reference's model catalog section: a hairline, 12
// DIP of air, the heading (title, which catalog is in use, restore and fetch),
// one bordered card per model (id and name inputs, the options chevron, the
// trash) and the outlined "add model" pill. reset is nil where the catalog has
// no layer to fall back to (a provider that is still being created).
func configurationModels(c *ui.Context, m Model, s store.ViewState, scope string, rows []any, fields []store.Field, change func([]any), reset func(), fg, muted ui.Color) {
	colors := modelColors(c)
	disabled := !s.Writable || s.Saving
	provider := scope
	if m.Models != nil && m.Models.configurationProvider != "" {
		provider = m.Models.configurationProvider
	}
	inherited, _, _ := m.Store.ConfigurationModels(provider)
	custom := reset != nil && configurationModelsCustomized(m, s)
	hasName := configurationHas(fields, "name")
	ui.Column(c.Key("model-catalog/" + scope)).FillWidth().MinWidth(0).Gap(0).Children(func() {
		ui.Box(c).FillWidth().Height(.667).Background(fonttheme.Color(colors.Border))
		ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(12, 0, 0, 0).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
					modelFieldLabel(c, providerCopy("模型目录", "Models"))
					if reset != nil {
						meta := providerCopy("正在使用适配器默认模型", "Using the adapter defaults")
						if custom {
							meta = providerCopy("已自定义模型目录", "Customized model catalog")
						}
						modelHint(c, meta, "")
					}
				})
				if custom && modelLinkPill(c, providerCopy("恢复默认模型", "Restore defaults"), disabled).Clicked() {
					reset()
				}
				modelListDiscovery(c, m, s, scope, rows, change, fg, muted)
			})
			if message := configurationModelError(rows); message != "" {
				modelHint(c, message, "error")
			}
			if len(rows) == 0 {
				ui.Text(c, providerCopy("模型选择器中将不显示任何模型；目录外 ID 仍可直接发送。", "No models will be shown in the selector. Unlisted IDs can still be sent directly.")).
					FillWidth().Padding(12).Radius(12).Border(.667, fonttheme.Color(colors.BorderL3)).
					FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(fonttheme.Color(colors.Muted)).TextAlign(ui.Center)
			}
			for i, raw := range rows {
				row := configurationObjectValue(raw)
				if row == nil {
					continue
				}
				index := i
				item := ui.Column(c.Key(fmt.Sprintf("%s/model/%d", scope, index))).FillWidth().MinWidth(0).Gap(0).Padding(8).Radius(12).Border(.667, fonttheme.Color(colors.Border))
				expanded := ui.Local(item, "expanded", func() bool { return false })
				item.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
						for _, field := range []string{"id", "name"} {
							if field == "name" && !hasName {
								continue
							}
							field := field
							value := configurationText(row[field])
							grow := float32(1)
							if field == "id" && hasName {
								grow = 1.4
							}
							label := providerCopy("模型 ID", "Model ID")
							placeholder := label
							if field == "name" {
								label = providerCopy("显示名称", "Display name")
								placeholder = label
								if idText := strings.TrimSpace(configurationText(row["id"])); idText != "" {
									placeholder = idText
								}
							}
							input := ui.TextInput(c.Key(fmt.Sprintf("%s/model/%d/%s", scope, index, field)), &value).Label(label).Placeholder(placeholder).Grow(grow).MinWidth(0).Height(36).Radius(8).
								Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(fonttheme.InterfaceSize(14)).Disabled(disabled)
							if input.Changed() {
								if field == "name" && strings.TrimSpace(value) == "" {
									configurationSet(row, field, nil)
								} else {
									configurationSet(row, field, value)
								}
								change(rows)
							}
						}
						chevron, label := bodyIcon("right"), providerCopy("模型选项", "Model options")
						if *expanded {
							chevron = bodyIcon("down")
						}
						if modelIconButton(c, fmt.Sprintf("%s/model/%d/options", scope, index), label, "", chevron, false, false) {
							*expanded = !*expanded
						}
						if modelIconButton(c, fmt.Sprintf("%s/model/%d/remove", scope, index), providerCopy("删除模型", "Remove model"), "", modelTrashIcon, true, disabled) {
							change(append(append([]any{}, rows[:index]...), rows[index+1:]...))
						}
					})
					if !*expanded {
						return
					}
					ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(12, 0, 0, 0).Children(func() {
						configurationModelOptions(c, m, s, scope, index, row, rows, fields, inherited, change, fg, muted)
					})
				})
			}
			if modelPlusPill(c, providerCopy("添加模型", "Add model"), disabled).Clicked() {
				change(append(append([]any{}, rows...), map[string]any{"id": ""}))
			}
		})
	})
}

var reasoningEffortLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

func defaultReasoningEffortValue(level string) any {
	if level == "off" {
		return nil
	}
	return level
}

// configurationModelOptions is the expanded body of one model card: the two
// capacities side by side, the input types and the reasoning levels.
func configurationModelOptions(c *ui.Context, m Model, s store.ViewState, scope string, index int, row map[string]any, rows []any, fields []store.Field, inherited []any, change func([]any), fg, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		for _, key := range []string{"contextWindow", "maxTokens"} {
			if !configurationHas(fields, key) {
				continue
			}
			key := key
			ui.Column(c.Key(fmt.Sprintf("%s/model/%d/cap/%s", scope, index, key))).Grow(1).MinWidth(0).Gap(4).Children(func() {
				configurationInput(c, m, s, fmt.Sprintf("%s/model/%d/%s", scope, index, key), key, row[key], true, func(v any) { configurationSet(row, key, v); change(rows) }, fg, muted, providerCopy("使用提供商默认值", "Use the provider default"))
			})
		}
	})
	for _, inputField := range []string{"input", "inputModalities"} {
		if !configurationHas(fields, inputField) {
			continue
		}
		fallback := []any{"text"}
		if profile := configurationObjectValue(configurationObject(s.Drafts["providers"])[scope]); profile != nil {
			if values := configurationRows(profile["defaultInput"]); len(values) > 0 {
				fallback = values
			}
		}
		for _, v := range inherited {
			model := configurationObjectValue(v)
			if model["id"] == row["id"] {
				values := configurationRows(model[inputField])
				if len(values) == 0 {
					values = configurationRows(model["inputModalities"])
				}
				if len(values) > 0 {
					fallback = values
				}
			}
		}
		configurationModalities(c, s, row, inputField, func() { change(rows) }, fg, fallback)
	}
	if !configurationHas(fields, "reasoningEfforts") {
		return
	}
	configurationModelReasoningEfforts(c, m, s, scope, index, row, rows, change)
}

func configurationModelReasoningEfforts(c *ui.Context, m Model, s store.ViewState, scope string, index int, row map[string]any, rows []any, change func([]any)) {
	disabled := !s.Writable || s.Saving
	efforts, has := row["reasoningEfforts"].(map[string]any)
	section := ui.Column(c.Key(fmt.Sprintf("%s/model/%d/reasoning", scope, index))).FillWidth().MinWidth(0).Gap(8)
	order := ui.Local(section, "order", func() []string { return nil })
	if !has {
		*order = nil
	} else {
		nextOrder := make([]string, 0, len(efforts))
		seen := map[string]bool{}
		for _, level := range *order {
			if _, exists := efforts[level]; exists && !seen[level] {
				nextOrder = append(nextOrder, level)
				seen[level] = true
			}
		}
		for _, level := range reasoningEffortLevels {
			if _, exists := efforts[level]; exists && !seen[level] {
				nextOrder = append(nextOrder, level)
				seen[level] = true
			}
		}
		for level := range efforts {
			if !seen[level] {
				nextOrder = append(nextOrder, level)
				seen[level] = true
			}
		}
		*order = nextOrder
	}
	section.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Row(c).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, providerCopy("推理等级", "Reasoning levels")).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).FontWeight(500).TextColor(fonttheme.Color(modelColors(c).Secondary))
				modelIconButton(
					c,
					fmt.Sprintf("%s/model/%d/reasoning/help", scope, index),
					providerCopy("推理等级说明", "Reasoning levels help"),
					providerCopy(
						"API 参数值会写入请求的推理强度字段（如 OpenAI 的 reasoning_effort），默认与档位名相同；如有不同，以模型提供商的 API 文档为准。Off 表示关闭思考，不需要填写参数值；配置中已写入的值会保留并发送。",
						"The API value is sent in the request's reasoning-effort field (such as OpenAI's reasoning_effort) and defaults to the level name; if your provider uses different spellings, follow its API docs. Off disables thinking and needs no value; a value already present in the config is kept and sent.",
					),
					bodyIcon("help"),
					false,
					false,
				)
			})
			enabled := has
			switcher := settingsPreferenceSwitch(c.Key(fmt.Sprintf("%s/model/%d/reasoning/switch", scope, index)), &enabled, settingsSwitchLabelled).
				Label(providerCopy("推理等级", "Reasoning levels")).Disabled(disabled)
			if switcher.Changed() {
				if enabled {
					defaults := []string{"off", "low", "medium", "high"}
					next := map[string]any{}
					for _, level := range defaults {
						next[level] = defaultReasoningEffortValue(level)
					}
					row["reasoningEfforts"] = next
					*order = append([]string{}, defaults...)
				} else {
					delete(row, "reasoningEfforts")
					*order = nil
				}
				change(rows)
			}
		})
		if !has {
			return
		}
		entries := append([]string{}, (*order)...)
		used := map[string]bool{}
		for _, level := range entries {
			used[level] = true
		}
		commitEntries := func(nextLevels []string, nextMap map[string]any) {
			out := map[string]any{}
			cleanOrder := make([]string, 0, len(nextLevels))
			for _, level := range nextLevels {
				cleanOrder = append(cleanOrder, level)
				out[level] = nextMap[level]
			}
			*order = cleanOrder
			row["reasoningEfforts"] = out
			change(rows)
		}
		if len(entries) > 0 {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					modelFieldLabel(c, providerCopy("档位", "Level"))
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					modelFieldLabel(c, providerCopy("API 参数值", "API value"))
				})
				ui.Box(c).Size(28, 1).Shrink(0)
			})
		}
		for rowIdx, level := range entries {
			rowIdx, level := rowIdx, level
			wire := efforts[level]
			ui.Row(c.Key(fmt.Sprintf("%s/model/%d/reasoning/row/%d", scope, index, rowIdx))).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				var choices []string
				for _, candidate := range reasoningEffortLevels {
					if candidate == level || !used[candidate] {
						choices = append(choices, candidate)
					}
				}
				selectedLevel := level
				if ui.Select(c.Key(fmt.Sprintf("%s/model/%d/reasoning/level/%d", scope, index, rowIdx)), &selectedLevel, choices).
					Label(providerCopy("档位", "Level")).Grow(1).MinWidth(0).Height(36).Radius(8).
					Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(fonttheme.InterfaceSize(14)).Disabled(disabled).Changed() && selectedLevel != level {
					nextLevels := append([]string{}, entries...)
					nextLevels[rowIdx] = selectedLevel
					nextMap := map[string]any{}
					for k, v := range efforts {
						if k != level {
							nextMap[k] = v
						}
					}
					if level != "off" && selectedLevel != "off" && configurationText(wire) != level {
						nextMap[selectedLevel] = wire
					} else {
						nextMap[selectedLevel] = defaultReasoningEffortValue(selectedLevel)
					}
					if m.DraftPtr != nil && selectedLevel != "off" {
						*m.DraftPtr(fmt.Sprintf("configuration/%s/model/%d/reasoning/%s", scope, index, selectedLevel), configurationText(nextMap[selectedLevel])) = configurationText(nextMap[selectedLevel])
					}
					commitEntries(nextLevels, nextMap)
				}
				placeholder := level
				if level == "off" {
					placeholder = providerCopy("关闭思考", "Disable thinking")
				}
				wireText := configurationText(wire)
				var valuePtr *string
				if m.DraftPtr != nil {
					valuePtr = m.DraftPtr(fmt.Sprintf("configuration/%s/model/%d/reasoning/%s", scope, index, level), wireText)
				} else {
					valuePtr = &wireText
				}
				if ui.TextInput(c.Key(fmt.Sprintf("%s/model/%d/reasoning/wire/%s", scope, index, level)), valuePtr).
					Label(fmt.Sprintf("%s (%s)", providerCopy("API 参数值", "API value"), level)).Placeholder(placeholder).
					Grow(1).MinWidth(0).Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).
					FontSize(fonttheme.InterfaceSize(14)).Disabled(disabled || level == "off").Changed() {
					efforts[level] = *valuePtr
					change(rows)
				}
				if modelIconButton(
					c,
					fmt.Sprintf("%s/model/%d/reasoning/remove/%s", scope, index, level),
					providerCopy("删除档位", "Remove level"),
					"",
					modelTrashIcon,
					true,
					disabled,
				) {
					nextLevels := append(append([]string{}, entries[:rowIdx]...), entries[rowIdx+1:]...)
					nextMap := map[string]any{}
					for k, v := range efforts {
						if k != level {
							nextMap[k] = v
						}
					}
					commitEntries(nextLevels, nextMap)
				}
			})
		}
		var nextUnused string
		highestUsedIndex := -1
		for i, candidate := range reasoningEffortLevels {
			if used[candidate] {
				highestUsedIndex = i
			}
		}
		for i := highestUsedIndex + 1; i < len(reasoningEffortLevels); i++ {
			if !used[reasoningEffortLevels[i]] {
				nextUnused = reasoningEffortLevels[i]
				break
			}
		}
		if nextUnused == "" {
			for _, candidate := range reasoningEffortLevels {
				if !used[candidate] {
					nextUnused = candidate
					break
				}
			}
		}
		if modelPlusPill(c, providerCopy("添加档位", "Add level"), disabled || nextUnused == "").Clicked() && nextUnused != "" {
			nextLevels := append(append([]string{}, entries...), nextUnused)
			nextMap := map[string]any{}
			for k, v := range efforts {
				nextMap[k] = v
			}
			nextMap[nextUnused] = defaultReasoningEffortValue(nextUnused)
			if m.DraftPtr != nil && nextUnused != "off" {
				*m.DraftPtr(fmt.Sprintf("configuration/%s/model/%d/reasoning/%s", scope, index, nextUnused), configurationText(nextMap[nextUnused])) = configurationText(nextMap[nextUnused])
			}
			commitEntries(nextLevels, nextMap)
		}
	})
}
