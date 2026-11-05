package settings

import (
	"context"
	"encoding/json"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

type servedModelRoute struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func servedSubagentModels(c *ui.Context, s *store.Store, b *store.NamespaceBinding, state store.ViewState, fg, muted ui.Color) {
	routes, loading, message := s.ModelConfigurationCatalog()
	if message != "" {
		ui.Text(c, message).FillWidth().TextColor(c.Theme().Danger)
	}
	if ui.Button(c, providerCopy("刷新模型目录", "Refresh model catalog")).Disabled(loading || state.Saving).Clicked() {
		s.RefreshModelConfigurationCatalog(context.Background())
	}
	var selected []servedModelRoute
	_ = json.Unmarshal([]byte(state.Drafts["allowedModels"]), &selected)
	var accepted struct {
		Routes []servedModelRoute `json:"allowedModels"`
	}
	_ = json.Unmarshal(state.Namespaces[0].Value, &accepted)
	known := map[string]bool{}
	for _, route := range routes {
		known[route.Provider+"\x00"+route.Model] = true
	}
	for _, route := range append(accepted.Routes, selected...) {
		id := route.Provider + "\x00" + route.Model
		if !known[id] {
			routes = append(routes, store.VisionRoute{Provider: route.Provider, Model: route.Model, Label: route.Provider + " · " + route.Model + providerCopy("（当前不可用）", " (unavailable)")})
			known[id] = true
		}
	}
	provider := ""
	for _, route := range routes {
		if route.Provider != provider {
			provider = route.Provider
			ui.Text(c, provider).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
		}
		checked := false
		for _, item := range selected {
			if item.Provider == route.Provider && item.Model == route.Model {
				checked = true
			}
		}
		if ui.Checkbox(c, &checked, route.Label).Disabled(!state.ClientReady || !state.Writable || state.Loading || state.Saving).Changed() {
			next := []servedModelRoute{}
			for _, item := range selected {
				if item.Provider != route.Provider || item.Model != route.Model {
					next = append(next, item)
				}
			}
			if checked {
				next = append(next, servedModelRoute{route.Provider, route.Model})
			}
			data, _ := json.Marshal(next)
			b.SetDraft("allowedModels", string(data))
		}
	}
	if len(routes) == 0 && !loading {
		ui.Text(c, providerCopy("没有可用的模型。", "No models available.")).TextColor(muted)
	}
}
