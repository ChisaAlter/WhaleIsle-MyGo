package settings

import (
	settingsstore "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

type VisionAccess struct {
	Snapshot  settingsstore.VisionView
	OnRefresh func()
	OnSelect  func(provider, model string)
}

func visionPane(c *ui.Context, a *VisionAccess, fg, muted ui.Color) {
	if a == nil || !a.Snapshot.Present {
		return
	}
	s := a.Snapshot
	failure := s.Err
	if failure == "" {
		failure = s.CatalogErr
	}
	settingsAlert(c, "vision-save-error", failure != "", providerCopy("没能读取视觉模型，请重试。", "Image models could not be read. Try again."), failure)
	// 0.3.13: a bordered 16 DIP card holding the label, the pill select and the
	// explanation. The Host data is refreshed when the page opens, so the old
	// refresh button is gone.
	modelRowCard(c, "vision-card", func() {
		modelFieldLabel(c, providerCopy("识图模型", "Vision model"))
		routes := append([]settingsstore.VisionRoute{{Label: providerCopy("不启用", "Off")}}, s.Routes...)
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
				selected = s.Provider + " · " + s.Model + providerCopy("（当前值，目录未列出）", " (current, not in catalog)")
				routes = append(routes, settingsstore.VisionRoute{Provider: s.Provider, Model: s.Model, Label: selected})
			}
		}
		labels := make([]string, len(routes))
		for i, route := range routes {
			labels[i] = route.Label
		}
		if s.Saving {
			selected = providerCopy("正在保存…", "Saving…")
		}
		kSelect(c, &selected, labels, providerCopy("识图模型", "Vision model"), !s.Writable || s.Saving || s.Loading).FillWidth().OnChange(func() {
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
		ui.Text(c, providerCopy("主模型不支持图片时，先由该模型识别图片内容，再把描述交给主模型处理。请选择一个支持图片输入的模型。", "When the main model cannot read images, this model describes them first and hands the description to the main model. Pick a model that accepts image input.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)

	})
}
