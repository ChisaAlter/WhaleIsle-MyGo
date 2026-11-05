package settings

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
)

func configurationWrite(m Model, key string, value any) {
	raw, _ := json.Marshal(value)
	if m.Models != nil && m.Models.configurationBinding != nil {
		m.Models.configurationBinding.SetDraft(key, string(raw))
	} else {
		m.Store.SetDraft(key, string(raw))
	}
}
func configurationField(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, f store.Field, fg, muted ui.Color) bool {
	if m.Store == nil {
		return false
	}
	if configurationProviderField(c, m, s, ns, f, fg, muted) {
		return true
	}
	switch {
	case ns.NS == "llm-deepseek-account" && f.Key == "modelOverrides":
		return true
	case ns.NS == "llm-deepseek-account" && f.Key == "models":
		configurationAccountModels(c, m, s, ns, fg, muted)
		return true
	case ns.NS == "llm-pi-ai" && f.Key == "providers":
		configurationProviders(c, m, s, ns, fg, muted)
		return true
	case (ns.NS == "llm-deepseek" || ns.NS == "llm-deepseek-api-key" || ns.NS == "llm-deepseek-account") && f.Key == "models":
		var rows []any
		_ = json.Unmarshal([]byte(s.Drafts[f.Key]), &rows)
		configurationModels(c, m, s, ns.NS, rows, store.ConfigurationFields(ns.Schema, "models", "0"), func(v []any) { configurationWrite(m, f.Key, v) }, func() { configurationReset(m, f.Key) }, fg, muted)
		return true
	case ns.NS == "subagent-model-selection-settings" && f.Key == "allowedModels":
		configurationSubagent(c, m, s, fg, muted)
		return true
	}
	return false
}

func configurationInvalid(s store.ViewState, paths ...[]string) bool {
	if s.SelectedNS == "llm-deepseek-account" {
		if pending := s.Drafts["@account-model-rows"]; pending != "" {
			var rows []any
			if json.Unmarshal([]byte(pending), &rows) != nil || configurationModelError(rows) != "" {
				return true
			}
		}
		rows := []any{}
		for id, raw := range configurationObject(s.Drafts["modelOverrides"]) {
			fields := configurationObjectValue(raw)
			if fields["disabled"] == true {
				continue
			}
			row := map[string]any{"id": id}
			for k, v := range fields {
				row[k] = v
			}
			rows = append(rows, row)
		}
		if configurationModelError(rows) != "" {
			return true
		}
	}

	if s.SelectedNS == "subagent-model-selection-settings" && s.Drafts["enabled"] == "true" {
		var rows []any
		_ = json.Unmarshal([]byte(s.Drafts["allowedModels"]), &rows)
		return len(rows) == 0
	}
	if s.SelectedNS == "llm-pi-ai" {
		for route, raw := range configurationObject(s.Drafts["providers"]) {
			if len(paths) > 0 && len(paths[0]) == 2 && paths[0][0] == "providers" && paths[0][1] != route {
				continue
			}
			p, _ := raw.(map[string]any)
			if configurationModelError(configurationRows(p["models"])) != "" {
				return true
			}
		}
	}
	if strings.HasPrefix(s.SelectedNS, "llm-deepseek") {
		var rows []any
		_ = json.Unmarshal([]byte(s.Drafts["models"]), &rows)
		return configurationModelError(rows) != ""
	}
	return false
}

func configurationInput(c *ui.Context, m Model, s store.ViewState, key, label string, value any, number bool, change func(any), fg, muted ui.Color, placeholder ...string) {
	label = configurationLabel(label)
	text := configurationText(value)
	if number {
		text = configurationFormatCapacity(value)
	}
	ui.Column(c.Key("configuration-input/" + key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
		modelFieldLabel(c, label)
		if m.DraftPtr == nil {
			return
		}
		p := m.DraftPtr("configuration/"+key, text)
		input := ui.TextInput(c, p).Label(label).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(fonttheme.InterfaceSize(14)).Disabled(!s.Writable || s.Saving)
		if len(placeholder) > 0 {
			input.Placeholder(placeholder[0])
		}
		if input.Changed() {
			text = *p
			if number {
				v, err := configurationCapacity(text)
				if err == nil {
					change(v)
				} else {
					change(text)
				}
			} else {
				if strings.TrimSpace(text) == "" {
					change(nil)
				} else {
					change(text)
				}
			}
		}
	})
}
func configurationLabel(key string) string {
	switch key {
	case "route":
		return providerCopy("Provider ID", "Provider ID")
	case "displayName":
		return providerCopy("显示名称", "Display name")
	case "baseURL":
		return providerCopy("API 地址", "API address")
	case "id":
		return providerCopy("模型 ID", "Model ID")
	case "name":
		return providerCopy("模型名称", "Model name")
	case "contextWindow":
		return providerCopy("上下文窗口", "Context window")
	case "maxTokens":
		return providerCopy("最大输出 token 数", "Max output tokens")
	case "defaultContextWindow":
		return providerCopy("默认上下文窗口", "Default context window")
	case "defaultMaxTokens":
		return providerCopy("默认最大输出 token 数", "Default max output tokens")
	default:
		return key
	}
}
func configurationSet(row map[string]any, key string, value any) {
	if value == nil {
		delete(row, key)
	} else {
		row[key] = value
	}
}
func configurationHas(fields []store.Field, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func configurationModalityLabel(choice string) string {
	switch choice {
	case "text":
		return providerCopy("文本", "Text")
	case "image":
		return providerCopy("图片", "Image")
	default:
		return choice
	}
}

func configurationModalities(c *ui.Context, s store.ViewState, row map[string]any, key string, change func(), fg ui.Color, fallback ...[]any) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		modelFieldLabel(c, providerCopy("输入类型", "Input types"))
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(16).MinHeight(28).AlignItems(ui.Center).Children(func() {
			for _, choice := range []string{"text", "image"} {
				choice := choice
				values := configurationRows(row[key])
				if len(values) == 0 {
					values = []any{"text"}
					if len(fallback) > 0 && len(fallback[0]) > 0 {
						values = fallback[0]
					}
				}
				selected := false
				for _, v := range values {
					if v == choice {
						selected = true
					}
				}
				label := configurationModalityLabel(choice)
				if ui.Checkbox(c.Key("modality/"+key+"/"+choice), &selected, label).Disabled(!s.Writable || s.Saving || selected && len(values) == 1).Changed() {
					next := []any{}
					for _, v := range values {
						if v != choice {
							next = append(next, v)
						}
					}
					if selected {
						next = append(next, choice)
					}
					row[key] = next
					if key == "inputModalities" {
						hasImage := false
						for _, v := range next {
							hasImage = hasImage || v == "image"
						}
						if !hasImage {
							delete(row, "imagePixelBudget")
							delete(row, "imageMaxBytes")
						}
					}
					change()
				}
			}
		})
	})
}

func configurationSubagent(c *ui.Context, m Model, s store.ViewState, fg, muted ui.Color) {
	ui.Text(c, providerCopy("子代理可选模型集合", "Selectable subagent models")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("启用后须至少选择一个模型；此设置影响新会话。", "When enabled, select at least one model. This setting applies to new sessions.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	routes, loading, message := m.Store.ModelConfigurationCatalog()
	if ui.Button(c, providerCopy("刷新模型目录", "Refresh model catalog")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(loading || s.Saving).Clicked() {
		m.Store.RefreshModelConfigurationCatalog(context.Background())
	}
	if message != "" {
		ui.Text(c, message).TextColor(fg)
	}
	var allowed []any
	_ = json.Unmarshal([]byte(s.Drafts["allowedModels"]), &allowed)
	known := map[string]bool{}
	for _, route := range routes {
		known[route.Provider+"\x00"+route.Model] = true
	}
	for _, raw := range allowed {
		item, _ := raw.(map[string]any)
		p, model := configurationText(item["provider"]), configurationText(item["model"])
		if !known[p+"\x00"+model] {
			routes = append(routes, store.VisionRoute{Provider: p, Model: model, Label: p + " · " + model + providerCopy("（目录当前未列出）", " (not currently listed)")})
			known[p+"\x00"+model] = true
		}
	}
	for _, route := range routes {
		selected := false
		for _, raw := range allowed {
			item, _ := raw.(map[string]any)
			if item["provider"] == route.Provider && item["model"] == route.Model {
				selected = true
			}
		}
		if ui.Checkbox(c, &selected, route.Label).Disabled(!s.Writable || s.Saving).Changed() {
			next := []any{}
			for _, raw := range allowed {
				item, _ := raw.(map[string]any)
				if item["provider"] != route.Provider || item["model"] != route.Model {
					next = append(next, raw)
				}
			}
			if selected {
				next = append(next, map[string]any{"provider": route.Provider, "model": route.Model})
			}
			configurationWrite(m, "allowedModels", next)
		}
	}
	if len(routes) == 0 && !loading {
		ui.Text(c, providerCopy("请刷新目录以选择模型。", "Refresh the catalog to choose models.")).TextColor(muted)
	}
	enabled, _ := strconv.ParseBool(s.Drafts["enabled"])
	if enabled && len(allowed) == 0 {
		ui.Text(c, providerCopy("请选择至少一个模型后保存。", "Select at least one model before saving.")).TextColor(fg)
	}
	if ui.Button(c, providerCopy("继承可选模型集合", "Inherit selectable models")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
		configurationReset(m, "allowedModels")
	}
}

func configurationReset(m Model, key string) {
	if m.Models != nil && m.Models.configurationBinding != nil {
		m.Models.configurationBinding.Reset(key)
	} else if m.Store != nil {
		m.Store.ResetDraft(key)
	}
}
func configurationSetForm(m Model, key, value string) {
	if m.Models != nil && m.Models.configurationBinding != nil {
		m.Models.configurationBinding.SetForm(key, value)
	} else if m.Store != nil {
		m.Store.SetConfigurationForm(key, value)
	}
}
