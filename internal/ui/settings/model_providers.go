package settings

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

// configurationProviders edits the custom (pi-ai) provider profiles. Inside a
// provider's editor card (configurationProvider is set) it is only the body of
// the reference's "自定义设置" fold: no heading, no nested card, the display
// name (declared routes only), the API address with its hint, the protocol
// (declared routes only) and the model catalog.
func configurationProviders(c *ui.Context, m Model, s store.ViewState, ns store.Namespace, fg, muted ui.Color) {
	providers := configurationObject(s.Drafts["providers"])
	fields := store.ConfigurationFields(ns.Schema, "providers", "route")
	models := store.ConfigurationFields(ns.Schema, "providers", "route", "models", "0")
	if m.Models != nil && m.Models.configurationCreateOnly {
		configurationCreateProvider(c, m, s, providers, fields, models, fg, muted)
		return
	}
	editor := m.Models != nil && m.Models.configurationProvider != ""
	if !editor {
		ui.Text(c, providerCopy("供应商与模型配置", "Provider and model configuration")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
		ui.Text(c, providerCopy("编辑后点击保存。凭据仍由凭据管理入口管理。", "Click Save after editing. Credentials remain managed through the credential controls.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	keys := make([]string, 0, len(providers))
	for key := range providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, route := range keys {
		if m.Models != nil && len(m.Models.configurationPath) > 0 && (len(m.Models.configurationPath) != 2 || m.Models.configurationPath[0] != "providers" || m.Models.configurationPath[1] != route) {
			continue
		}
		p, _ := providers[route].(map[string]any)
		if p == nil {
			continue
		}
		route := route
		profile := ui.Column(c.Key("provider-profile/" + route)).FillWidth().MinWidth(0).Gap(12)
		if !editor {
			profile = profile.Gap(8).Padding(14, 16).Radius(12).Border(0.5, c.Theme().Border.Alpha(0.45)).Background(c.Theme().Surface)
		}
		profile.Children(func() {
			entry, hasEntry := configurationProviderMetadata(m, ns.NS, []string{"providers", route})
			if !editor {
				ui.Text(c, route).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
				if hasEntry {
					providerEndpointNotice(c, configurationText(p["baseURL"]), entry, muted)
				}
			}
			save := func() { configurationWrite(m, "providers", providers) }
			for _, key := range []string{"displayName", "baseURL"} {
				if key == "displayName" && !configurationDeclaredProvider(m, ns.NS, route) {
					continue
				}
				if !configurationHas(fields, key) {
					continue
				}
				key := key
				value, hint := p[key], ""
				if m.Models != nil && m.Models.configurationBinding != nil {
					var inherited bool
					value, inherited = m.Models.configurationBinding.ProviderInput([]string{"providers", route, key}, p[key])
					if inherited {
						hint = configurationText(p[key])
					}
				}
				configurationInput(c, m, s, route+"/"+key, key, value, strings.HasPrefix(key, "default"), func(v any) { configurationSet(p, key, v); save() }, fg, muted, hint)
				if editor && key == "baseURL" && hasEntry {
					providerEndpointNotice(c, configurationText(p["baseURL"]), entry, muted)
				}
			}
			for _, field := range fields {
				if field.Key == "api" && configurationDeclaredProvider(m, ns.NS, route) {
					current := configurationText(p["api"])
					if current == "" && len(field.EnumOptions) > 0 {
						current = field.EnumOptions[0]
					}
					labels := make([]string, len(field.EnumOptions))
					selected := ""
					for i, id := range field.EnumOptions {
						labels[i] = configurationProtocolLabel(id)
						if id == current {
							selected = labels[i]
						}
					}
					ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
						modelFieldLabel(c, providerCopy("API 协议", "API protocol"))
						if ui.Select(c, &selected, labels).Label(providerCopy("API 协议", "API protocol")).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(fonttheme.InterfaceSize(14)).Disabled(!s.Writable || s.Saving).Changed() {
							for i, label := range labels {
								if label == selected {
									p["api"] = field.EnumOptions[i]
									save()
								}
							}
						}
					})
				}
			}
			if !editor {
				for _, key := range []string{"defaultContextWindow", "defaultMaxTokens"} {
					if configurationHas(fields, key) {
						key := key
						configurationInput(c, m, s, route+"/"+key, key, p[key], true, func(v any) { configurationSet(p, key, v); save() }, fg, muted)
					}
				}
				if configurationHas(fields, "defaultInput") {
					configurationModalities(c, s, p, "defaultInput", save, fg)
				}
				configurationProviderOptions(c, m, s, ns, route, p, save, fg, muted)
			}
			configurationModels(c, m, s, route, configurationRows(p["models"]), models, func(rows []any) { p["models"] = rows; save() }, func() { delete(p, "models"); save() }, fg, muted)
		})
	}
	if m.Models == nil || m.Models.configurationProvider == "" {
		configurationCreateProvider(c, m, s, providers, fields, models, fg, muted)
	}
}
