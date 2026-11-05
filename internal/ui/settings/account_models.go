package settings

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
)

func accountModelOverrides(legacy []any, overrides map[string]any, defaults []any) map[string]any {
	next := map[string]any{}
	if legacy != nil {
		ids := map[string]bool{}
		for _, raw := range legacy {
			row, _ := raw.(map[string]any)
			id := configurationText(row["id"])
			ids[id] = true
			fields := map[string]any{}
			for k, v := range row {
				if k != "id" {
					fields[k] = v
				}
			}
			next[id] = fields
		}
		for _, raw := range defaults {
			row, _ := raw.(map[string]any)
			id := configurationText(row["id"])
			if !ids[id] {
				next[id] = map[string]any{"disabled": true}
			}
		}
	}
	for id, raw := range overrides {
		fields, _ := next[id].(map[string]any)
		if fields == nil {
			fields = map[string]any{}
		}
		for k, v := range configurationObjectValue(raw) {
			if list, ok := v.([]any); !ok || len(list) > 0 {
				fields[k] = v
			}
		}
		next[id] = fields
	}
	return next
}
func configurationObjectValue(v any) map[string]any { x, _ := v.(map[string]any); return x }
func accountModelRows(catalog []any, overrides map[string]any) []any {
	rows := []any{}
	seen := map[string]bool{}
	appendRow := func(id string, raw map[string]any) {
		fields := configurationObjectValue(overrides[id])
		seen[id] = true
		if fields["disabled"] == true {
			return
		}
		row := map[string]any{"id": id}
		for k, v := range raw {
			row[k] = v
		}
		for k, v := range fields {
			if k != "disabled" && k != "id" {
				row[k] = v
			}
		}
		rows = append(rows, row)
	}
	for _, raw := range catalog {
		row, _ := raw.(map[string]any)
		appendRow(configurationText(row["id"]), row)
	}
	ids := make([]string, 0, len(overrides))
	for id := range overrides {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !seen[id] {
			appendRow(id, nil)
		}
	}
	return rows
}
func editedAccountOverrides(before, after []any, original map[string]any) map[string]any {
	next := map[string]any{}
	raw, _ := json.Marshal(original)
	_ = json.Unmarshal(raw, &next)
	if next == nil {
		next = map[string]any{}
	}
	old := map[string]map[string]any{}
	kept := map[string]bool{}
	for _, r := range before {
		v := configurationObjectValue(r)
		old[configurationText(v["id"])] = v
	}
	for _, r := range after {
		v := configurationObjectValue(r)
		id := configurationText(v["id"])
		kept[id] = true
		fields := configurationObjectValue(next[id])
		if fields == nil {
			fields = map[string]any{}
		}
		if fields["disabled"] == true {
			fields["disabled"] = false
		}
		prev := old[id]
		for k, val := range v {
			if k != "id" && !reflect.DeepEqual(prev[k], val) {
				fields[k] = val
			}
		}
		for k := range prev {
			if k != "id" {
				if _, ok := v[k]; !ok {
					delete(fields, k)
				}
			}
		}
		if len(fields) > 0 {
			next[id] = fields
		} else {
			delete(next, id)
		}
	}
	for id := range old {
		if !kept[id] {
			if id == "" {
				delete(next, id)
				continue
			}
			fields := configurationObjectValue(next[id])
			if fields == nil {
				fields = map[string]any{}
			}
			fields["disabled"] = true
			next[id] = fields
		}
	}
	return next
}
func configurationAccountModels(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, fg, muted ui.Color) {
	m.Store.EnsureModelConfigurationCatalog(context.Background())
	catalog, loading, message := m.Store.ConfigurationModels("deepseek-account")
	settingsNotice(c, "account_models:151", "", message != "", func() string { return message })
	if modelLinkPill(c, providerCopy("刷新账户模型目录", "Refresh account model catalog"), loading || s.Saving).AlignSelf(ui.Start).Clicked() {
		m.Store.RefreshModelConfigurationCatalog(context.Background())
	}
	var legacy, defaults []any
	_ = json.Unmarshal([]byte(s.Drafts["models"]), &legacy)
	var base map[string]json.RawMessage
	_ = json.Unmarshal(ns.Base, &base)
	defaults = store.AccountModelDefaults(ns.Schema)
	if len(defaults) == 0 {
		_ = json.Unmarshal(base["models"], &defaults)
	}
	if len(catalog) == 0 {
		catalog = defaults
	} else {
		for _, raw := range catalog {
			row := configurationObjectValue(raw)
			for _, fallback := range defaults {
				item := configurationObjectValue(fallback)
				if item["id"] == row["id"] {
					for key, value := range item {
						if _, exists := row[key]; !exists {
							row[key] = value
						}
					}
				}
			}
		}
	}
	overrides := accountModelOverrides(legacy, configurationObject(s.Drafts["modelOverrides"]), defaults)
	rows := accountModelRows(catalog, overrides)
	if pending := s.Drafts["@account-model-rows"]; pending != "" {
		_ = json.Unmarshal([]byte(pending), &rows)
	}
	var before []any
	raw, _ := json.Marshal(rows)
	_ = json.Unmarshal(raw, &before)
	configurationModels(c, m, s, ns.NS, rows, store.ConfigurationFields(ns.Schema, "models", "0"), func(after []any) {
		raw, _ := json.Marshal(after)
		configurationSetForm(m, "@account-model-rows", string(raw))
		configurationWrite(m, "modelOverrides", editedAccountOverrides(before, after, overrides))
		configurationReset(m, "models")
	}, func() {
		configurationSetForm(m, "@account-model-rows", "")
		configurationReset(m, "models")
		configurationReset(m, "modelOverrides")
	}, fg, muted)
}
