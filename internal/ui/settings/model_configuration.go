package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	case ns.NS == "llm-pi-ai" && f.Key == "providers":
		configurationProviders(c, m, s, ns, fg, muted)
		return true
	case (ns.NS == "llm-deepseek" || ns.NS == "llm-deepseek-api-key" || ns.NS == "llm-deepseek-account") && f.Key == "models":
		var rows []any
		_ = json.Unmarshal([]byte(s.Drafts[f.Key]), &rows)
		configurationModels(c, m, s, ns.NS, rows, store.ConfigurationFields(ns.Schema, "models", "0"), func(v []any) { configurationWrite(m, f.Key, v) }, fg, muted)
		if ui.Button(c, providerCopy("继承模型目录", "Inherit model catalog")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
			configurationReset(m, f.Key)
		}
		return true
	case ns.NS == "subagent-model-selection-settings" && f.Key == "allowedModels":
		configurationSubagent(c, m, s, fg, muted)
		return true
	}
	return false
}

func configurationInvalid(s store.ViewState) bool {
	if s.SelectedNS == "subagent-model-selection-settings" && s.Drafts["enabled"] == "true" {
		var rows []any
		_ = json.Unmarshal([]byte(s.Drafts["allowedModels"]), &rows)
		return len(rows) == 0
	}
	if s.SelectedNS == "llm-pi-ai" {
		for _, raw := range configurationObject(s.Drafts["providers"]) {
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

func configurationInput(c *ui.Context, m Model, s store.ViewState, key, label string, value any, number bool, change func(any), fg, muted ui.Color) {
	ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	text := configurationText(value)
	if m.DraftPtr != nil {
		p := m.DraftPtr("configuration/"+key, text)
		if ui.TextInput(c, p).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!s.Writable || s.Saving).Changed() {
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

func configurationModalities(c *ui.Context, s store.ViewState, row map[string]any, key string, change func(), fg ui.Color) {
	ui.Text(c, key).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	for _, choice := range []string{"text", "image"} {
		values := configurationRows(row[key])
		selected := false
		for _, v := range values {
			if v == choice {
				selected = true
			}
		}
		if ui.Checkbox(c, &selected, choice).Disabled(!s.Writable || s.Saving).Changed() {
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
			change()
		}
	}
	if ui.Button(c, providerCopy("继承输入类型", "Inherit input types")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
		delete(row, key)
		change()
	}
}

func configurationModels(c *ui.Context, m Model, s store.ViewState, scope string, rows []any, fields []store.Field, change func([]any), fg, muted ui.Color) {
	ui.Text(c, providerCopy("模型目录与元数据", "Model catalog and metadata")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	if message := configurationModelError(rows); message != "" {
		ui.Text(c, message).TextColor(fg)
	}
	for i, raw := range rows {
		row, _ := raw.(map[string]any)
		if row == nil {
			continue
		}
		index := i
		ui.Column(c).Gap(6).Padding(14, 16).Radius(12).Border(0.5, c.Theme().Border.Alpha(0.45)).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, fmt.Sprintf("Model %d", index+1)).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
			for _, key := range []string{"id", "name", "contextWindow", "maxTokens"} {
				if key != "id" && !configurationHas(fields, key) {
					continue
				}
				field := key
				configurationInput(c, m, s, fmt.Sprintf("%s/model/%d/%s", scope, index, field), field, row[field], field == "contextWindow" || field == "maxTokens", func(v any) { configurationSet(row, field, v); change(rows) }, fg, muted)
			}
			if configurationHas(fields, "input") {
				configurationModalities(c, s, row, "input", func() { change(rows) }, fg)
			}
			if configurationHas(fields, "reasoningEfforts") {
				mode := "inherit"
				efforts, has := row["reasoningEfforts"].(map[string]any)
				if has {
					mode = "custom"
				} else if row["reasoningEfforts"] == false {
					mode = "disabled"
				}
				ui.Text(c, providerCopy("推理级别", "Reasoning levels")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				if ui.Select(c, &mode, []string{"inherit", "disabled", "custom"}).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!s.Writable || s.Saving).Changed() {
					switch mode {
					case "inherit":
						delete(row, "reasoningEfforts")
					case "disabled":
						row["reasoningEfforts"] = false
					case "custom":
						row["reasoningEfforts"] = map[string]any{"off": nil, "low": "low", "medium": "medium", "high": "high"}
					}
					change(rows)
				}
				if has {
					for _, level := range []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"} {
						level := level
						_, selected := efforts[level]
						if ui.Checkbox(c, &selected, level).Disabled(!s.Writable || s.Saving).Changed() {
							if selected {
								if level == "off" {
									efforts[level] = nil
								} else {
									efforts[level] = level
									if m.DraftPtr != nil {
										*m.DraftPtr(fmt.Sprintf("configuration/%s/model/%d/reasoning/%s", scope, index, level), level) = level
									}
								}
							} else {
								delete(efforts, level)
							}
							change(rows)
						}
						if selected && level != "off" && m.DraftPtr != nil {
							p := m.DraftPtr(fmt.Sprintf("configuration/%s/model/%d/reasoning/%s", scope, index, level), configurationText(efforts[level]))
							if ui.TextInput(c, p).FillWidth().Height(28).Radius(6).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Disabled(!s.Writable || s.Saving).Changed() {
								efforts[level] = strings.TrimSpace(*p)
								change(rows)
							}
						}
					}
				}
			}
			if ui.Button(c, providerCopy("移除模型", "Remove model")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
				change(append(append([]any{}, rows[:index]...), rows[index+1:]...))
			}
		})
	}
	if ui.Button(c, providerCopy("添加模型", "Add model")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
		change(append(rows, map[string]any{"id": ""}))
	}
}

func configurationProviders(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, fg, muted ui.Color) {
	providers := configurationObject(s.Drafts["providers"])
	fields := store.ConfigurationFields(ns.Schema, "providers", "route")
	models := store.ConfigurationFields(ns.Schema, "providers", "route", "models", "0")
	ui.Text(c, providerCopy("供应商与模型配置", "Provider and model configuration")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("编辑后点击保存。凭据仍由凭据管理入口管理。", "Click Save after editing. Credentials remain managed through the credential controls.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	keys := make([]string, 0, len(providers))
	for key := range providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, route := range keys {
		p, _ := providers[route].(map[string]any)
		if p == nil {
			continue
		}
		route := route
		ui.Column(c).Gap(8).Padding(14, 16).Radius(12).Border(0.5, c.Theme().Border.Alpha(0.45)).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, route).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
			save := func() { configurationWrite(m, "providers", providers) }
			for _, key := range []string{"displayName", "baseURL", "defaultContextWindow", "defaultMaxTokens"} {
				if !configurationHas(fields, key) {
					continue
				}
				key := key
				configurationInput(c, m, s, route+"/"+key, key, p[key], strings.HasPrefix(key, "default"), func(v any) { configurationSet(p, key, v); save() }, fg, muted)
			}
			for _, field := range fields {
				if field.Key == "api" {
					current := configurationText(p["api"])
					options := append([]string{""}, field.EnumOptions...)
					ui.Text(c, providerCopy("协议（空白继承）", "Protocol (blank inherits)")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					if ui.Select(c, &current, options).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!s.Writable || s.Saving).Changed() {
						if current == "" {
							delete(p, "api")
						} else {
							p["api"] = current
						}
						save()
					}
				}
			}
			if configurationHas(fields, "defaultInput") {
				configurationModalities(c, s, p, "defaultInput", save, fg)
			}
			configurationProviderOptions(c, m, s, ns, route, p, save, fg, muted)
			configurationModels(c, m, s, route, configurationRows(p["models"]), models, func(rows []any) { p["models"] = rows; save() }, fg, muted)
			if ui.Button(c, providerCopy("继承供应商模型目录", "Inherit provider model catalog")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving).Clicked() {
				delete(p, "models")
				save()
			}
		})
	}
	configurationCreateProvider(c, m, s, providers, fields, models, fg, muted)
}

func configurationCreateProvider(c *ui.Context, m Model, s store.ViewState, providers map[string]any, fields, models []store.Field, fg, muted ui.Color) {
	ui.Text(c, providerCopy("添加自定义供应商", "Add custom provider")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	const key = "@provider-create"
	draft := configurationObject(s.Drafts[key])
	save := func() { raw, _ := json.Marshal(draft); configurationSetForm(m, key, string(raw)) }
	for _, field := range []string{"route", "displayName", "baseURL"} {
		field := field
		configurationInput(c, m, s, "create/"+field, field, draft[field], false, func(v any) { configurationSet(draft, field, v); save() }, fg, muted)
	}
	protocol := configurationText(draft["api"])
	var protocols []string
	for _, field := range fields {
		if field.Key == "api" {
			protocols = field.EnumOptions
		}
	}
	ui.Text(c, providerCopy("Host 支持的协议", "Host-supported protocol")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if ui.Select(c, &protocol, append([]string{""}, protocols...)).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!s.Writable || s.Saving).Changed() {
		draft["api"] = protocol
		save()
	}
	rows := configurationRows(draft["models"])
	configurationModels(c, m, s, "create", rows, models, func(v []any) { draft["models"] = v; save() }, fg, muted)
	route := strings.TrimSpace(configurationText(draft["route"]))
	endpoint := strings.TrimSpace(configurationText(draft["baseURL"]))
	message := configurationCreateError(route, endpoint, protocol, rows, providers)
	if message != "" {
		ui.Text(c, message).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c, providerCopy("添加到草稿", "Add to draft")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(!s.Writable || s.Saving || message != "").Clicked() {
			profile := map[string]any{"api": protocol, "baseURL": endpoint, "models": rows}
			if name := configurationText(draft["displayName"]); name != "" {
				profile["displayName"] = name
			}
			providers[route] = profile
			configurationWrite(m, "providers", providers)
			configurationSetForm(m, key, "{}")
		}
		if ui.Button(c, providerCopy("清空创建表单", "Clear create form")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Saving).Clicked() {
			configurationSetForm(m, key, "{}")
		}
	})
}

func configurationSubagent(c *ui.Context, m Model, s store.ViewState, fg, muted ui.Color) {
	ui.Text(c, providerCopy("子代理可选模型集合", "Selectable subagent models")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("启用后须至少选择一个模型；此设置影响新会话。", "When enabled, select at least one model. This setting applies to new sessions.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	routes, loading, message := m.Store.ModelConfigurationCatalog()
	if ui.Button(c, providerCopy("刷新模型目录", "Refresh model catalog")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(loading || s.Saving).Clicked() {
		m.Store.RefreshModelConfigurationCatalog(context.Background())
	}
	if loading {
		ui.Text(c, providerCopy("正在读取目录…", "Loading catalog…")).TextColor(muted)
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
