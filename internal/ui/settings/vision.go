package settings

import (
	"github.com/egoist/mygo/ui"
	settingsstore "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

type VisionAccess struct {
	Snapshot  settingsstore.VisionView
	OnRefresh func()
	OnSelect  func(provider, model string)
}

func visionPane(c *ui.Context, a *VisionAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	ui.Text(c, providerCopy("识图模型（全局）", "Vision model (global)")).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	if a.OnRefresh != nil && ui.Button(c, providerCopy("刷新识图设置", "Refresh vision settings")).Disabled(s.Loading || s.Saving).Clicked() {
		a.OnRefresh()
	}
	if s.Err != "" {
		ui.Text(c, s.Err).TextColor(fg)
	}
	if s.CatalogErr != "" {
		ui.Text(c, s.CatalogErr).TextColor(fg)
	}
	if !s.Present {
		ui.Text(c, providerCopy("Host 尚未提供识图配置；请刷新。", "The Host has not exposed vision settings. Refresh to retry.")).TextColor(muted)
		return
	}
	routes := append([]settingsstore.VisionRoute{{Label: providerCopy("关闭识图", "Off")}}, s.Routes...)
	selected := routes[0].Label
	if s.Provider != "" && s.Model != "" {
		found := false
		for _, route := range routes[1:] {
			if route.Provider == s.Provider && route.Model == s.Model {
				selected = route.Label
				found = true
				break
			}
		}
		if !found {
			selected = s.Provider + " / " + s.Model + providerCopy("（当前值，目录未列出）", " (current, not in catalog)")
			routes = append(routes, settingsstore.VisionRoute{Provider: s.Provider, Model: s.Model, Label: selected})
		}
	}
	labels := make([]string, len(routes))
	for i, route := range routes {
		labels[i] = route.Label
	}
	ui.Select(c, &selected, labels).FillWidth().Disabled(!s.Writable || s.Saving || s.Loading).OnChange(func() {
		if a.OnSelect == nil {
			return
		}
		for _, route := range routes {
			if route.Label == selected {
				a.OnSelect(route.Provider, route.Model)
				return
			}
		}
	})
	ui.Text(c, providerCopy("为不支持图片的主模型提供识图描述；选择后立即保存。", "Describe images for text-only main models. Changes save immediately.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if s.Saving {
		ui.Text(c, providerCopy("正在保存…", "Saving…")).TextColor(muted)
	} else if s.LastAction != "" {
		ui.Text(c, s.LastAction).TextColor(muted)
	}
}
