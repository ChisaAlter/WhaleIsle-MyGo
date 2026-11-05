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
	if a == nil || !a.Snapshot.Present {
		return
	}
	s := a.Snapshot
	// A state holder only: out of the flow, so it does not add a gap slot above the card.
	errorSeat := ui.Box(c.Key("vision-errors")).Absolute().Size(0, 0).PassThrough()
	seen := ui.Local(errorSeat, "error", func() string { return "" })
	failure := s.Err
	if failure == "" {
		failure = s.CatalogErr
	}
	if failure != "" && failure != *seen {
		c.AddToast(ui.Toast{ID: "vision-save-error", Description: failure})
	}
	*seen = failure
	dark := c.Theme().Dark
	colors := fonttheme.ResolveWithSystem(fonttheme.IDSystem, dark).Active(dark)
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
		ui.Select(c, &selected, labels).FillWidth().Height(36).Padding(0, 14).Radius(18).BorderWidth(0).Background(fonttheme.Color(colors.SelectorFill)).FontSize(fonttheme.InterfaceSize(14)).Disabled(!s.Writable || s.Saving || s.Loading).OnChange(func() {
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
