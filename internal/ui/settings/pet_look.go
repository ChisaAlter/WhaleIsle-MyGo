package settings

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/petlook"
	settingsstore "whaleisle/internal/store/settings"
)

func petLookPane(c *ui.Context, p *pet.Companion, store *settingsstore.Store, fg, muted ui.Color, blocked ...bool) {
	if p == nil || p.Store == nil {
		return
	}
	catalog := petlook.Catalog{}
	if store != nil {
		catalog = store.ImageCatalogSnapshot()
	}
	value := p.Store.DesktopPet().Behavior
	disabled := len(blocked) > 0 && blocked[0]
	choices := []petlook.Route{{Label: providerCopy("关闭看看", "Off")}}
	selected := choices[0].Label
	found := value.LookProvider == "" && value.LookModel == ""
	// The original picker displays a legacy model-only selection as its route
	// only when exactly one image-capable provider offers that model. Displaying
	// it does not rewrite config; a deliberate selection saves the whole pair.
	provider := value.LookProvider
	if provider == "" && value.LookModel != "" {
		matches := 0
		for _, route := range catalog.Routes {
			if route.Model == value.LookModel && petlook.Resolve(route.Provider, route.Model, catalog) {
				provider = route.Provider
				matches++
			}
		}
		if matches != 1 {
			provider = ""
		}
	}
	for _, route := range catalog.Routes {
		if !petlook.Resolve(route.Provider, route.Model, catalog) {
			continue
		}
		// Explicit indices keep labels unique even if provider display names collide.
		route.Label = fmt.Sprintf("%d · %s · %s", len(choices), route.Provider, route.Model)
		choices = append(choices, route)
		if route.Provider == provider && route.Model == value.LookModel {
			selected, found = route.Label, true
		}
	}
	labels := make([]string, len(choices))
	for i, route := range choices {
		labels[i] = route.Label
	}
	if !found {
		selected = providerCopy("已存但当前不可用：", "Saved, unavailable: ") + value.LookProvider + " · " + value.LookModel
		labels = append(labels, selected) // display-only sentinel, never saved implicitly
	}
	bodyPreferenceRow(c, "pet-look-model", providerCopy("看看模型", "Look model"), providerCopy("独立于全局视觉模型保存。每次截图和发送前仍需确认。", "Saved separately from the global vision model. Each screenshot and send still asks for confirmation."), fg, muted, func() {
		if ui.Button(c, providerCopy("刷新模型目录", "Refresh models")).Disabled(disabled || store == nil || catalog.Loading).Clicked() {
			store.RefreshImageCatalog(context.Background())
		}
		if !catalog.Connected {
			ui.Text(c, providerCopy("Host 未连接；保留已存选择，仍可关闭。", "Host disconnected; the saved selection is retained, and Off remains available.")).TextColor(muted)
		}
		if catalog.Err != "" {
			ui.Text(c, providerCopy("模型目录暂不可用；保留已存选择，请刷新重试。", "Model catalog unavailable; saved selection is retained. Refresh to retry.")).TextColor(muted)
		}
		if catalog.Loading {
			ui.Text(c, providerCopy("正在读取模型目录…", "Loading model catalog…")).TextColor(muted)
		}
		ui.Select(c, &selected, labels).Label("pet-look-model-route").FillWidth().Disabled(disabled || catalog.Loading).OnChange(func() {
			for _, route := range choices {
				if route.Label == selected {
					fresh := petlook.Catalog{}
					if store != nil {
						fresh = store.ImageCatalogSnapshot()
					}
					p.SaveLookRoute(route.Provider, route.Model, fresh)
					return
				}
			}
		})

	})
}
