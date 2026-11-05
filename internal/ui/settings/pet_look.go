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
	legacy := petlook.Legacy(value.LookProvider, value.LookModel)
	if legacy {
		selected = providerCopy("本机 API（旧配置） · ", "Desktop API (legacy) · ") + value.LookModel
		choices = append(choices, petlook.Route{Model: value.LookModel, Label: selected})
		found = true
	}
	for _, route := range catalog.Routes {
		if !petlook.Resolve(route.Provider, route.Model, catalog) {
			continue
		}
		// Explicit indices keep labels unique even if provider display names collide.
		route.Label = fmt.Sprintf("%d · %s · %s", len(choices), route.Provider, route.Model)
		choices = append(choices, route)
		if route.Provider == value.LookProvider && route.Model == value.LookModel {
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
		if legacy {
			ui.Text(c, providerCopy("旧配置使用桌面 API 设置，不依赖 Host。选择目录中的提供方后，将改走该提供方。", "This legacy selection uses desktop API settings without a Host. Selecting a catalog provider changes the route explicitly.")).TextColor(muted)
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
