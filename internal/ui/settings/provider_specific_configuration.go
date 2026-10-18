package settings

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

// Provider controls use the connected Host's schema and the existing draft
// mutation path. Unedited fields and credential references stay in their owner.
func configurationProviderOptions(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, route string, profile map[string]any, change func(), fg, muted ui.Color) {
	path := []string{"providers", route}
	fields := store.ConfigurationFields(ns.Schema, path...)
	ui.Text(c, providerCopy("供应商专用设置", "Provider-specific settings")).Bold().TextColor(fg)
	for _, key := range []string{
		"cacheRetention", "transport", "timeoutMs", "websocketConnectTimeoutMs",
		"streamIdleTimeoutMs", "maxRequestImageBytes", "requestImagePixelBudget", "requestImageMaxBytes",
	} {
		if field, ok := configurationOption(fields, key); ok {
			configurationOptionField(c, m, s, route, profile, field, change, fg, muted)
		}
	}
	for _, key := range []string{"thinkingBudgets", "compat"} {
		if configurationHas(fields, key) {
			configurationOptionObject(c, m, s, ns.Schema, append(append([]string{}, path...), key), route, profile, key, change, fg, muted)
		}
	}
	if configurationHas(fields, "headers") {
		configurationOptionDictionary(c, m, s, ns.Schema, append(append([]string{}, path...), "headers"), route, profile, "headers", false, change, fg, muted)
	}
	if configurationHas(fields, "retryPolicy") {
		configurationRetryPolicy(c, m, s, ns.Schema, append(append([]string{}, path...), "retryPolicy"), route, profile, change, fg, muted)
	}
}

// DeepSeek's scalar options already have native schema rows; its policy object
// is the remaining structured value that otherwise requires raw JSON.
func configurationProviderField(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, field store.Field, fg, muted ui.Color) bool {
	if field.Key != "retryPolicy" || (ns.NS != "llm-deepseek" && ns.NS != "llm-deepseek-api-key" && ns.NS != "llm-deepseek-account") {
		return false
	}
	profile := map[string]any{}
	if value := strings.TrimSpace(s.Drafts[field.Key]); value != "" && value != "null" {
		profile[field.Key] = configurationObject(value)
	}
	configurationRetryPolicy(c, m, s, ns.Schema, []string{field.Key}, ns.NS, profile, func() {
		if policy, exists := profile[field.Key]; exists {
			configurationWrite(m, field.Key, policy)
		} else {
			m.Store.ResetDraft(field.Key)
		}
	}, fg, muted)
	return true
}

func configurationOption(fields []store.Field, key string) (store.Field, bool) {
	for _, field := range fields {
		if field.Key == key {
			return field, true
		}
	}
	return store.Field{}, false
}

func configurationOptionField(c *ui.Context, m Model, s store.ViewState, scope string, row map[string]any, field store.Field, change func(), fg, muted ui.Color) {
	if field.Kind == store.FieldJSON {
		return
	}
	ui.Text(c, field.Label).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if field.Description != "" {
		ui.Text(c, field.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	value := configurationText(row[field.Key])
	disabled := !s.Writable || s.Saving
	set := func(value any) { configurationSet(row, field.Key, value); change() }
	switch field.Kind {
	case store.FieldBool, store.FieldEnum:
		options := append([]string{""}, field.EnumOptions...)
		if field.Kind == store.FieldBool {
			options = []string{"", "true", "false"}
		}
		if ui.Select(c, &value, options).FillWidth().Disabled(disabled).Changed() {
			if value == "" {
				set(nil)
			} else if field.Kind == store.FieldBool {
				set(value == "true")
			} else {
				set(value)
			}
		}
	default:
		if m.DraftPtr != nil {
			text := m.DraftPtr("configuration/"+scope+"/"+field.Key, value)
			input := ui.TextInput(c, text).FillWidth().Disabled(disabled)
			if SecretLikeKey(field.Key) && (m.RevealSecrets == nil || !*m.RevealSecrets) {
				input = input.Password()
			}
			if input.Changed() {
				if strings.TrimSpace(*text) == "" {
					set(nil)
				} else if field.Kind == store.FieldNumber {
					number, err := strconv.ParseFloat(strings.TrimSpace(*text), 64)
					_, jsonErr := json.Marshal(number)
					if err == nil && jsonErr == nil {
						set(number)
					} else {
						set(*text)
					}
				} else {
					set(*text)
				}
			}
		}
	}
	if ui.Button(c, providerCopy("继承默认值", "Inherit default")).FontSize(fonttheme.InterfaceSize(11)).Disabled(disabled).Clicked() {
		set(nil)
	}
}

func configurationOptionObject(c *ui.Context, m Model, s store.ViewState, schema json.RawMessage, path []string, scope string, parent map[string]any, key string, change func(), fg, muted ui.Color) {
	fields := store.ConfigurationFields(schema, path...)
	if len(fields) == 0 {
		return
	}
	row, _ := parent[key].(map[string]any)
	if row == nil {
		row = map[string]any{}
	}
	save := func() { parent[key] = row; change() }
	ui.Column(c).Gap(4).Padding(6, 6, 6, 6).Children(func() {
		ui.Text(c, key).Bold().TextColor(fg)
		for _, field := range fields {
			configurationOptionField(c, m, s, scope+"/"+key, row, field, save, fg, muted)
		}
		for _, child := range []string{"chatTemplateKwargs", "chatTemplateArgs"} {
			if configurationHas(fields, child) {
				configurationOptionDictionary(c, m, s, schema, append(append([]string{}, path...), child), scope+"/"+key, row, child, true, save, fg, muted)
			}
		}
		if ui.Button(c, providerCopy("继承整组设置", "Inherit this group")).Disabled(!s.Writable || s.Saving).Clicked() {
			delete(parent, key)
			change()
		}
	})
}

func configurationRetryPolicy(c *ui.Context, m Model, s store.ViewState, schema json.RawMessage, path []string, scope string, profile map[string]any, change func(), fg, muted ui.Color) {
	fields := store.ConfigurationFields(schema, path...)
	if len(fields) == 0 {
		return
	}
	policy, _ := profile["retryPolicy"].(map[string]any)
	if policy == nil {
		policy = map[string]any{}
	}
	save := func() {
		if configurationText(policy["mode"]) == "" {
			policy["mode"] = "normal"
		}
		profile["retryPolicy"] = policy
		change()
	}
	ui.Column(c).Gap(4).Padding(6, 6, 6, 6).Children(func() {
		ui.Text(c, providerCopy("请求重试策略", "Request retry policy")).Bold().TextColor(fg)
		if field, ok := configurationOption(fields, "mode"); ok {
			configurationOptionField(c, m, s, scope+"/retryPolicy", policy, field, func() {
				if configurationText(policy["mode"]) == "" {
					delete(profile, "retryPolicy")
					change()
				} else {
					save()
				}
			}, fg, muted)
		}
		if policy["mode"] == "always" {
			ui.Text(c, providerCopy("所有模型请求失败都会持续重试，直到成功或取消。", "Every model-request failure retries until success or cancellation.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else {
			if field, ok := configurationOption(fields, "maxRetries"); ok {
				configurationOptionField(c, m, s, scope+"/retryPolicy", policy, field, save, fg, muted)
			}
			if configurationHas(fields, "retryableCodes") {
				configurationOptionList(c, m, s, scope+"/retryPolicy", policy, "retryableCodes", save, fg, muted)
			}
		}
		if configurationHas(fields, "backoff") {
			configurationOptionObject(c, m, s, schema, append(append([]string{}, path...), "backoff"), scope+"/retryPolicy", policy, "backoff", save, fg, muted)
		}
		if ui.Button(c, providerCopy("继承重试策略", "Inherit retry policy")).Disabled(!s.Writable || s.Saving).Clicked() {
			delete(profile, "retryPolicy")
			change()
		}
	})
}

func configurationOptionList(c *ui.Context, m Model, s store.ViewState, scope string, row map[string]any, key string, change func(), fg, muted ui.Color) {
	ui.Text(c, key+providerCopy("（每行一项）", " (one per line)")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	var values []string
	for _, value := range configurationRows(row[key]) {
		values = append(values, configurationText(value))
	}
	if m.DraftPtr != nil {
		text := m.DraftPtr("configuration/"+scope+"/"+key, strings.Join(values, "\n"))
		if ui.TextArea(c, text).FillWidth().Lines(3, 8).Disabled(!s.Writable || s.Saving).Changed() {
			if strings.TrimSpace(*text) == "" {
				delete(row, key)
			} else {
				next := []any{}
				for _, line := range strings.Split(*text, "\n") {
					if value := strings.TrimSpace(line); value != "" {
						next = append(next, value)
					}
				}
				row[key] = next
			}
			change()
		}
	}
	if ui.Button(c, providerCopy("继承列表", "Inherit list")).Disabled(!s.Writable || s.Saving).Clicked() {
		delete(row, key)
		change()
	}
}

func configurationOptionDictionary(c *ui.Context, m Model, s store.ViewState, schema json.RawMessage, path []string, scope string, parent map[string]any, key string, typed bool, change func(), fg, muted ui.Color) {
	row, _ := parent[key].(map[string]any)
	if row == nil {
		row = map[string]any{}
	}
	save := func() { parent[key] = row; change() }
	disabled := !s.Writable || s.Saving
	ui.Text(c, key).Bold().TextColor(fg)
	keys := make([]string, 0, len(row))
	for name := range row {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		name := name
		value := row[name]
		kind := "string"
		switch value.(type) {
		case float64:
			kind = "number"
		case bool:
			kind = "boolean"
		case nil:
			kind = "null"
		case map[string]any:
			kind = "variable"
		}
		ui.Text(c, name).TextColor(fg)
		entryPath := append(append([]string{}, path...), name)
		variableFields := store.ConfigurationFields(schema, entryPath...)
		if typed {
			options := []string{"string", "number", "boolean", "null"}
			if configurationHas(variableFields, "$var") {
				options = append(options, "variable")
			}
			if ui.Select(c, &kind, options).Disabled(disabled).Changed() {
				switch kind {
				case "number":
					row[name] = float64(0)
				case "boolean":
					row[name] = false
				case "null":
					row[name] = nil
				case "variable":
					field, _ := configurationOption(variableFields, "$var")
					if len(field.EnumOptions) > 0 {
						row[name] = map[string]any{"$var": field.EnumOptions[0]}
					}
				default:
					row[name] = ""
				}
				save()
				value = row[name]
			}
		}
		switch kind {
		case "null":
			ui.Text(c, "null").TextColor(muted)
		case "variable":
			variable, _ := value.(map[string]any)
			for _, field := range variableFields {
				configurationOptionField(c, m, s, scope+"/"+key+"/"+name, variable, field, save, fg, muted)
			}
		default:
			fieldKind := store.FieldString
			if kind == "number" {
				fieldKind = store.FieldNumber
			} else if kind == "boolean" {
				fieldKind = store.FieldBool
			}
			configurationOptionField(c, m, s, scope+"/"+key, row, store.Field{Key: name, Label: providerCopy("值", "Value"), Kind: fieldKind}, save, fg, muted)
		}
		if ui.Button(c, providerCopy("移除此项", "Remove entry")).Disabled(disabled).Clicked() {
			delete(row, name)
			save()
		}
	}
	formKey := "@provider-option/" + scope + "/" + key
	name := s.Drafts[formKey]
	if m.DraftPtr != nil {
		draft := m.DraftPtr("configuration/"+formKey, name)
		if ui.TextInput(c, draft).FillWidth().Placeholder(providerCopy("新字段名称", "New field name")).Disabled(disabled).Changed() {
			name = *draft
			m.Store.SetConfigurationForm(formKey, name)
		}
	}
	name = strings.TrimSpace(name)
	_, exists := row[name]
	if ui.Button(c, providerCopy("添加字段", "Add field")).Disabled(disabled || name == "" || exists).Clicked() {
		row[name] = ""
		save()
		m.Store.SetConfigurationForm(formKey, "")
	}
	if ui.Button(c, fmt.Sprintf("%s %s", providerCopy("继承", "Inherit"), key)).Disabled(disabled).Clicked() {
		delete(parent, key)
		change()
	}
}
