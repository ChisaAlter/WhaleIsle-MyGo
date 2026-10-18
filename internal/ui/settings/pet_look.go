package settings

import (
	"context"
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/petlook"
	settingsstore "whaleisle/internal/store/settings"
)

func petLookPane(c *ui.Context, p *pet.Companion, store *settingsstore.Store, fg, muted ui.Color) {
	if p == nil || p.Store == nil {
		return
	}
	ui.Text(c, "桌宠看看模型 / Pet Look model").FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, "独立于全局视觉模型；选择后立即保存。截图和发送仍需每次确认。 / Saved separately from global vision settings. Each screenshot and send still asks for confirmation.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	catalog := petlook.Catalog{}
	if store != nil {
		catalog = store.ImageCatalogSnapshot()
	}
	if ui.Button(c, "刷新图像模型目录 / Refresh image catalog").Disabled(store == nil || catalog.Loading).Clicked() {
		store.RefreshImageCatalog(context.Background())
	}
	if !catalog.Connected {
		ui.Text(c, "Host 未连接；可关闭看看，已存选择保留 / Host offline; saved selection retained, Off remains available").TextColor(muted)
	}
	if catalog.Err != "" {
		ui.Text(c, "模型目录暂不可用，请刷新；已存选择保留 / Image catalog unavailable; refresh to retry").TextColor(muted)
	}
	if catalog.Loading {
		ui.Text(c, "正在读取图像模型目录… / Loading image catalog…").TextColor(muted)
	}
	value := p.Store.DesktopPet().Behavior
	choices := []petlook.Route{{Label: "关闭看看 / Off"}}
	selected := choices[0].Label
	found := value.LookProvider == "" && value.LookModel == ""
	for _, route := range catalog.Routes {
		if !petlook.Resolve(route.Provider, route.Model, catalog) {
			continue
		}
		// Explicit indices keep labels unique even if provider display names collide.
		route.Label = fmt.Sprintf("%d · %s / %s", len(choices), route.Provider, route.Model)
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
		selected = "已存但当前不可用 / Saved, unavailable: " + value.LookProvider + " / " + value.LookModel
		labels = append(labels, selected) // display-only sentinel, never saved implicitly
	}
	ui.Select(c, &selected, labels).FillWidth().OnChange(func() {
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
}
