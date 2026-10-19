package market

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The desktop-owned MarketSection is an 800 DIP settings column. The
// container's measured width, rather than the window, controls its card grid.
func marketHeader(c *ui.Context, m Model, d marketstore.Discovery, fg, muted ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMarketTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
			ui.Text(c, marketText("发现并管理当前配置中的插件。", "Discover and manage plugins in the current profile.")).FontSize(theme.InterfaceSize(14)).TextColor(muted).FillWidth()
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, i18n.T(i18n.KeyMarketRefresh)).Disabled(d.Loading || m.InstallBusy).Clicked() {
				m.Store.RefreshCatalog(true, i18n.Locale())
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background())
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyMarketBack)).Clicked() {
				m.Store.CloseDetails()
				m.OnClose()
			}
		})
	})
}

func marketTabs(c *ui.Context, m Model, tab string, fg, muted, field ui.Color) {
	ui.Row(c).Role(ui.RoleTabList).Label(i18n.T(i18n.KeyMarketTitle)).FillWidth().Wrap().Gap(4).Children(func() {
		for _, item := range []struct{ id, zh, en string }{
			{"discover", "发现", "Discover"}, {"installed", "已安装", "Installed"},
			{"favorites", "收藏", "Favorites"}, {"history", "安装历史", "History"},
		} {
			active := tab == item.id || tab == "" && item.id == "discover"
			button := ui.Button(c.Key("market-tab-"+item.id), marketText(item.zh, item.en)).Role(ui.RoleTab).Height(30).Padding(0, 12).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Selected(active)
			if active {
				button.Background(field).TextColor(fg)
			}
			if button.Clicked() {
				m.Store.SetTab(item.id)
			}
		}
		if m.Embedded && ui.Button(c, i18n.T(i18n.KeyMarketRefresh)).Disabled(m.Store.Discovery().Loading || m.InstallBusy).Clicked() {
			m.Store.RefreshCatalog(true, i18n.Locale())
			if m.OnRefresh != nil {
				m.OnRefresh()
			} else {
				m.Store.Refresh(context.Background())
			}
		}
	})
}

func catalogCategories(c *ui.Context, m Model, d marketstore.Discovery, fg, muted, field ui.Color) {
	ui.Row(c).Role(ui.RoleRadioGroup).Label(marketText("分类", "Categories")).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		for _, category := range d.Catalog.Categories {
			active := category.ID == d.Category
			button := ui.Button(c.Key("market-category-"+category.ID), fmt.Sprintf("%s %d", category.Label, category.Count)).Role(ui.RoleRadio).Checked(active).Height(28).Padding(0, 10).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
			if active {
				button.Background(field).TextColor(fg)
			}
			if button.Clicked() {
				m.Store.SetCategory(category.ID)
			}
		}
	})
}

func catalogCards(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	if len(d.Rows) == 0 && !d.Loading && d.Err == "" {
		message := marketText("暂无插件。", "No plugins available.")
		if d.Query != "" || d.Category != "all" && d.Category != "" || d.Period != "all" || d.Tab == "favorites" {
			message = marketText("没有匹配的插件。", "No matching plugins.")
		}
		ui.Text(c, message).FillWidth().Padding(24).Border(1, border).Radius(8).TextColor(muted)
	}
	grid := ui.Row(c).Key("market-cards").FillWidth().MinWidth(0).Wrap().Gap(12).AlignItems(ui.Start)
	width := grid.Bounds().W
	grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	columns := 1
	if width >= 572 {
		columns = 2
	}
	grid.Children(func() {
		for _, item := range d.Rows {
			i := item
			card := ui.Column(c.Key("market-card-"+i.ID)).MinWidth(0).Background(field).Border(.5, border).Radius(8).Padding(12, 14).Gap(8)
			if columns == 2 {
				card.Width((width - 12) / 2)
			} else {
				card.FillWidth()
			}
			card.Children(func() {
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, catalogTitle(i)).Grow(1).MinWidth(0).SingleLine().Tooltip(i.ID).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
					if i.Deprecated {
						ui.Text(c, marketText("已弃用", "Deprecated")).FontSize(theme.InterfaceSize(11)).TextColor(accent)
					}
				})
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					ui.Text(c, i.Owner).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					ui.Text(c, fmt.Sprintf(marketText("%s 星标", "%s stars"), formatStars(i.Stars))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				})
				ui.Text(c, i.Description).FillWidth().MinWidth(0).MaxLines(2).MinHeight(36).FontSize(theme.InterfaceSize(12)).TextColor(muted).Tooltip(i.Description)
				ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, catalogCategory(d.Catalog, i.Category)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					favoriteButton(c, m, i.ID)
					if ui.Button(c, marketText("详情", "Details")).Clicked() {
						m.Store.SelectDetails(i.ID, false)
					}
					catalogInstallButton(c, m, d, i, inventory, fg, muted)
				})
			})
		}
	})
}

func catalogTitle(item desktopmarket.CatalogItem) string {
	if item.Repo != "" {
		return item.Repo
	}
	return item.ID
}

func catalogCategory(catalog desktopmarket.Catalog, id string) string {
	for _, category := range catalog.Categories {
		if category.ID == id {
			return category.Label
		}
	}
	return id
}

type installedGroup struct {
	label string
	rows  []marketstore.Entry
}

func installedGroups(catalog desktopmarket.Catalog, rows []marketstore.Entry) []installedGroup {
	groups := make([]installedGroup, len(catalog.Categories))
	for index, category := range catalog.Categories {
		groups[index].label = category.Label
	}
	other := installedGroup{label: marketText("未分类", "Ungrouped")}
	for _, row := range rows {
		category := ""
		for _, item := range catalog.Items {
			if marketstore.InstalledName(item, []marketstore.Entry{row}) != "" {
				category = item.Category
				break
			}
		}
		grouped := false
		for index, entry := range catalog.Categories {
			if entry.ID == category && entry.ID != "all" {
				groups[index].rows = append(groups[index].rows, row)
				grouped = true
				break
			}
		}
		if !grouped {
			other.rows = append(other.rows, row)
		}
	}
	return append(groups, other)
}

func installedRows(c *ui.Context, m Model, d marketstore.Discovery, snap marketstore.ViewState, fg, muted, border, accent ui.Color) {
	var rows []marketstore.Entry
	for _, row := range snap.Rows {
		if showInstalledRow(m, row) {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 && len(snap.Rows) > 0 {
		ui.Text(c, marketText("没有匹配的插件。", "No matching plugins.")).FillWidth().Padding(24).TextColor(muted)
	}
	for _, group := range installedGroups(d.Catalog, rows) {
		if len(group.rows) == 0 {
			continue
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, group.label).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(muted)
			for _, row := range group.rows {
				r := row
				ui.Column(c.Key("installed-" + r.Label)).FillWidth().MinWidth(0).Children(func() {
					ui.Box(c).FillWidth().Height(.5).Background(border)
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(12).Padding(8).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, r.Label).FillWidth().SingleLine().Tooltip(r.Label).FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
							ui.Text(c, r.Detail).FillWidth().SingleLine().Tooltip(r.Detail).FontSize(theme.InterfaceSize(12)).TextColor(muted)
							if r.ReadOnlyReason != "" {
								ui.Text(c, r.ReadOnlyReason).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(muted)
							}
							if r.Error != "" {
								ui.Text(c, r.Error).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(accent)
							}
						})
						ui.Row(c).Wrap().Gap(8).Children(func() {
							if snap.ClientReady && !snap.Loading && snap.Err == "" && !m.InstallBusy && r.CanToggle() && m.OnSetBundleEnabled != nil {
								label := i18n.T(i18n.KeySettingsPluginsEnable)
								if *r.Enabled {
									label = i18n.T(i18n.KeySettingsPluginsDisable)
								}
								if ui.Button(c, label).Clicked() {
									m.OnSetBundleEnabled(r.Label, !*r.Enabled)
								}
							}
							if snap.ClientReady && !snap.Loading && snap.Err == "" && !m.InstallBusy && r.IdentityKnown && r.ReadOnlyReason == "" && r.Removable && m.OnRemoveBundle != nil && ui.Button(c, i18n.T(i18n.KeyMarketRemove)).Clicked() {
								m.OnRemoveBundle(r.Label)
							}
						})
					})
				})
			}
		})
	}
}

func formatStars(stars float64) string {
	if stars >= 1000 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", stars/1000), ".0") + "k"
	}
	return fmt.Sprintf("%.0f", stars)
}

func catalogDetailsDialog(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	if d.Selected == nil || m.Store.Confirmation() != "" {
		return
	}
	item := *d.Selected
	open := true
	w, h := c.Size()
	ui.DialogBase(c.Key("market-details-dialog"), &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .35)).Padding(16)
		panel.Label(catalogTitle(item)).Width(max(0, min(float32(680), w-32))).Height(max(0, min(float32(720), h-48))).Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).BG)).Border(.5, border).Radius(12).Clip().Children(func() {
			ui.Row(c).FillWidth().Padding(16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, catalogTitle(item)).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				if ui.Button(c, marketText("重新加载详情", "Reload details")).Disabled(d.DetailsLoading).Clicked() {
					m.Store.SelectDetails(item.ID, true)
				}
				if ui.Button(c, marketText("关闭", "Close")).Clicked() {
					open = false
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Padding(0, 16, 16, 16).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
					ui.Text(c, item.Description).FillWidth().TextColor(fg)
					ui.Text(c, item.ID+"\n"+item.Homepage+"\n"+item.InstallSpec).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					favoriteButton(c, m, item.ID)
					catalogInstallButton(c, m, d, item, inventory, fg, muted)
					if item.Deprecated {
						ui.Text(c, marketText("已弃用，不提供安装。", "Deprecated; installation unavailable.")+" "+item.Replacement).FillWidth().TextColor(accent)
					}
					if d.DetailsLoading {
						ui.Text(c, i18n.T(i18n.KeyMarketLoading)).TextColor(muted)
					}
					if d.DetailsErr != "" {
						ui.Text(c, d.DetailsErr).FillWidth().TextColor(accent)
					}
					if d.Details.Partial {
						ui.Text(c, marketText("部分详情加载失败。", "Some details could not be loaded.")).FillWidth().TextColor(accent)
					}
					if d.Details.Version != "" {
						ui.Text(c, "Version: "+d.Details.Version).TextColor(fg)
					}
					if len(d.Details.Requirements) > 0 {
						ui.Text(c, marketText("声明的运行要求（不代表兼容性验证）", "Declared requirements (not a compatibility verdict)")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						for _, requirement := range d.Details.Requirements {
							ui.Text(c, requirement).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg)
						}
					}
					marketScreenshotGallery(c, m, item, fg, muted, field)
					marketReadme(c, m, d.Details.Readme, fg, muted, field, accent)
				})
			})
		})
	})
	if !open {
		m.Store.CloseDetails()
	}
}
