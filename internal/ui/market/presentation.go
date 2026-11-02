package market

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The desktop-owned MarketSection is an 800 DIP settings column. The
// container's measured width, rather than the window, controls its card grid.
func marketHeader(c *ui.Context, m Model, d marketstore.Discovery, fg, muted, border ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMarketTitle)).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
			// 0.3.13 wording: the catalog is curated and the Harness restarts after a change.
			ui.Text(c, marketText("从精选目录浏览、安装并更新社区插件；变更后 Harness 会自动重启。", "Browse, install and update community plugins from the curated catalog; the Harness restarts after a change.")).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted).FillWidth()
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			refreshLabel := i18n.T(i18n.KeyMarketRefresh)
			if d.Loading {
				refreshLabel = marketText("正在加载插件目录…", "Loading the plugin catalog…")
			}
			// A 28 DIP outlined icon button; it shows the same refresh glyph while loading.
			refresh := ui.Button(c, "").Label(refreshLabel).Size(28, 28).Shrink(0).Padding(6).Radius(8).Border(.667, border).Background(ui.Transparent).Tooltip(refreshLabel).Disabled(d.Loading || m.InstallBusy)
			refresh.Children(func() { ui.Icon(c, marketRefreshIcon).Size(16, 16).TextColor(fg) })
			if refresh.Clicked() {
				m.Store.RefreshCatalog(true, i18n.Locale())
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background())
				}
			}
			if !m.Embedded && m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyMarketBack)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
				m.Store.CloseDetails()
				m.OnClose()
			}
		})
	})
}

func marketTabs(c *ui.Context, m Model, tab string, fg, muted, field ui.Color) {
	ui.Row(c).Role(ui.RoleTabList).Label(i18n.T(i18n.KeyMarketTitle)).FillWidth().Wrap().Gap(4).Children(func() {
		for _, item := range []struct{ id, zh, en string }{
			{"discover", i18n.T(i18n.KeyW4MarketDiscover), "Discover"}, {"installed", i18n.T(i18n.KeyW4MarketInstalled2), "Installed"},
			{"favorites", i18n.T(i18n.KeyW4MarketSave), "Favorites"}, {"history", i18n.T(i18n.KeyW4MarketInstallHistory), "History"},
		} {
			active := tab == item.id || tab == "" && item.id == "discover"
			label := marketText(item.zh, item.en)
			if item.id == "installed" {
				count := len(desktopInstalledRows(m.Store.Snapshot().Rows))
				if count > 0 {
					label = fmt.Sprintf("%s (%d)", label, count)
				}
			}
			button := ui.Button(c.Key("market-tab-"+item.id), label).Role(ui.RoleTab).Height(28).Padding(0, 12).Radius(999).BorderWidth(0).FontSize(theme.InterfaceSize(12.5)).Background(ui.Transparent).TextColor(muted).Selected(active)
			if active {
				button.Background(field).TextColor(fg).FontWeight(500)
			}
			if button.Clicked() {
				m.Store.SetTab(item.id)
			}
		}
	})
}

func catalogCategories(c *ui.Context, m Model, d marketstore.Discovery, fg, muted, field, border ui.Color) {
	if len(d.Catalog.Categories) <= 1 {
		return
	}
	ui.Row(c).Role(ui.RoleRadioGroup).Label(marketText(i18n.T(i18n.KeyW4MarketCategories), "Categories")).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		for i, category := range d.Catalog.Categories {
			if i >= 5 && category.ID != d.Category && (m.categoriesExpanded == nil || !*m.categoriesExpanded) {
				continue
			}
			active := category.ID == d.Category
			button := ui.Button(c.Key("market-category-"+category.ID), fmt.Sprintf("%s %d", category.Label, category.Count)).Role(ui.RoleRadio).Checked(active).Height(28).Padding(0, 10).Radius(999).BorderWidth(0).FontSize(theme.InterfaceSize(12.5)).Background(ui.Transparent).TextColor(muted)
			if active {
				button.Background(field).TextColor(fg).FontWeight(500)
			}
			if button.Clicked() {
				m.Store.SetCategory(category.ID)
			}
		}
		if len(d.Catalog.Categories) > 5 && m.categoriesExpanded != nil {
			label := marketText(i18n.T(i18n.KeyW4MarketExpandCategories), "More categories")
			if *m.categoriesExpanded {
				label = marketText(i18n.T(i18n.KeyW4MarketCollapseCategories), "Fewer categories")
			}
			if ui.Button(c, label).Height(28).Padding(0, 10).Radius(999).Border(.5, border).FontSize(theme.InterfaceSize(12.5)).Clicked() {
				*m.categoriesExpanded = !*m.categoriesExpanded
			}
		}
	})
}

func catalogCards(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	if len(d.Rows) == 0 && !d.Loading && d.Err == "" {
		message := marketText(i18n.T(i18n.KeyW4MarketNoPluginsYet), "No plugins available.")
		if d.Query != "" || d.Category != "all" && d.Category != "" || d.Period != "all" || d.Tab == "favorites" {
			message = marketText(i18n.T(i18n.KeyW4MarketNoMatchingPlugins), "No matching plugins.")
		}
		ui.Text(c, message).FillWidth().Padding(24).Border(.5, border).Radius(12).TextColor(muted)
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
			card := ui.Column(c.Key("market-card-"+i.ID)).MinWidth(0).Border(.5, border).Radius(12).Padding(14, 16).Gap(8)
			cardBg := field
			if card.Hovered() {
				cardBg = c.Theme().SurfaceHover
			}
			chrome.GlassSurface(card, cardBg, chrome.MaterialPanel, 12)
			if columns == 2 {
				card.Width((width - 12) / 2)
			} else {
				card.FillWidth()
			}
			card.Children(func() {
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Start).Children(func() {
					marketIconTile(c, i, muted, border)
					ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
						ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, catalogTitle(i)).Grow(1).MinWidth(0).SingleLine().Tooltip(i.ID).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
							if i.Deprecated {
								ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketDeprecated), "Deprecated")).FontSize(theme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, accent).TextColor(accent)
							}
						})
						ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
							if i.Owner != "" && !PluginImage(c, "https://github.com/"+url.PathEscape(i.Owner)+".png?size=96", 16, 16, m.SVGImages) {
								ui.Text(c, strings.ToUpper(string([]rune(i.Owner)[0]))).Width(16).Height(16).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							}
							ui.Text(c, i.Owner).FontSize(theme.InterfaceSize(12)).TextColor(muted)
							ui.Text(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketSStars), "%s stars"), formatStars(i.Stars))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						})
					})
				})
				ui.Text(c, i.Description).FillWidth().MinWidth(0).MaxLines(2).MinHeight(36).FontSize(theme.InterfaceSize(12)).TextColor(muted).Tooltip(i.Description)
				ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, catalogCategory(d.Catalog, i.Category)).FontSize(theme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, border).TextColor(muted)
					favoriteButton(c, m, i.ID)
					ui.Box(c).Grow(1)
					if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), marketText(i18n.T(i18n.KeyW4MarketDetails), "Details"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
						m.Store.SelectDetails(i.ID, false)
					}
					if parsed, err := url.Parse(i.Homepage); err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil && m.OnOpenURL != nil {
						if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), marketText(i18n.T(i18n.KeyW4MarketHomepage), "Homepage"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							m.OnOpenURL(parsed.String())
						}
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
	other := installedGroup{label: marketText(i18n.T(i18n.KeyW4MarketUncategorized), "Ungrouped")}
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

func installedRows(c *ui.Context, m Model, d marketstore.Discovery, snap marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	var rows []marketstore.Entry
	installed := desktopInstalledRows(snap.Rows)
	for _, row := range installed {
		if showInstalledRow(m, row) {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 && len(installed) > 0 {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketNoMatchingPlugins), "No matching plugins.")).FillWidth().Padding(24).TextColor(muted)
	}
	for _, group := range installedGroups(d.Catalog, rows) {
		if len(group.rows) == 0 {
			continue
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, group.label).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(muted)
			for _, row := range group.rows {
				r := row
				card := ui.Column(c.Key("installed-"+r.Label)).FillWidth().MinWidth(0).Border(.5, border).Radius(12).Padding(12, 16)
				chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
				card.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, r.Label).FillWidth().SingleLine().Tooltip(r.Label).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
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
								if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
									m.OnSetBundleEnabled(r.Label, !*r.Enabled)
								}
							}
							if snap.ClientReady && !snap.Loading && snap.Err == "" && !m.InstallBusy && r.IdentityKnown && r.ReadOnlyReason == "" && r.Removable && m.OnRemoveBundle != nil && ui.Button(c, i18n.T(i18n.KeyMarketRemove)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
								m.OnRemoveBundle(r.Label)
							}
						})
					})
				})
			}
		})
	}
}

// The desktop Market's installed list is the profile's declared dependencies;
// optional packages merely shipped by the runtime belong to Plugin Manager.
func desktopInstalledRows(rows []marketstore.Entry) []marketstore.Entry {
	var installed []marketstore.Entry
	for _, row := range rows {
		if row.IdentityKnown && row.Installed {
			installed = append(installed, row)
		}
	}
	return installed
}

func formatStars(stars float64) string {
	if stars >= 1000 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", stars/1000), ".0") + "k"
	}
	return fmt.Sprintf("%.0f", stars)
}

// marketCloseIcon is the reference Modal's 14 DIP cross.
var marketCloseIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1" aria-hidden="true"><path d="m4 4 8 8M12 4l-8 8"/></svg>`))

func catalogDetailsDialog(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, accent ui.Color) {
	if d.Selected == nil || m.Store.Confirmation() != "" {
		return
	}
	item := *d.Selected
	open := true
	w, h := c.Size()
	ui.DialogBase(c.Key("market-details-dialog"), &open, func(backdrop, panel ui.Element) {
		box := chrome.DialogSurfaceBox{
			Width:   min(float32(680), max(float32(320), w-48)),
			Height:  min(float32(720), max(float32(320), h-48)),
			Padding: 0,
		}
		panel = chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), box)
		panel.Label(catalogTitle(item)).Gap(0).Children(func() {
			// The reference Modal header: 22/14/12/24 around a 16/500 title and a
			// 28 DIP close box; reload belongs to the body, as the header holds
			// only the title and the close control.
			ui.Row(c).FillWidth().Height(58).Shrink(0).Padding(22, 14, 12, 24).AlignItems(ui.Start).Children(func() {
				ui.Text(c, catalogTitle(item)).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).SingleLine()
				closeLabel := marketText(i18n.T(i18n.KeyW4MarketClose), "Close")
				closeBox := ui.Button(c, "").Label(closeLabel).Size(28, 28).Shrink(0).Margin(-2, 0, 0, 0).Padding(7).
					Radius(8).BorderWidth(0).Background(ui.Transparent).FocusRing(false)
				closeBox.Children(func() { ui.Icon(c, marketCloseIcon).Size(14, 14).TextColor(muted) })
				if closeBox.Hovered() {
					closeBox.Background(c.Theme().SurfaceHover)
				}
				if closeBox.Clicked() {
					open = false
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Padding(20, 24, 24, 24).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
					ui.Text(c, item.Description).FillWidth().TextColor(fg)
					ui.Text(c, item.ID+"\n"+item.Homepage+"\n"+item.InstallSpec).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if item.Deprecated {
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketDeprecatedInstallationIsNotOffered), "Deprecated; installation unavailable.")+" "+item.Replacement).FillWidth().TextColor(accent)
					}
					if d.DetailsLoading {
						ui.Text(c, i18n.T(i18n.KeyMarketLoading)).TextColor(muted)
					}
					if d.DetailsErr != "" {
						ui.Text(c, d.DetailsErr).FillWidth().TextColor(accent)
					}
					if d.Details.Partial {
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketSomeDetailsFailedToLoad), "Some details could not be loaded.")).FillWidth().TextColor(accent)
					}
					if d.Details.Version != "" {
						ui.Text(c, "Version: "+d.Details.Version).TextColor(fg)
					}
					if len(d.Details.Requirements) > 0 {
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketDeclaredRuntimeRequirementsNotACompatib), "Declared requirements (not a compatibility verdict)")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						for _, requirement := range d.Details.Requirements {
							ui.Text(c, requirement).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(fg)
						}
					}
					marketScreenshotGallery(c, m, item, fg, muted, field)
					marketReadme(c, m, d.Details.Readme, fg, muted, field, accent)
				})
			})
			ui.Row(c).FillWidth().Padding(10, 24, 16, 24).Gap(8).AlignItems(ui.Center).Children(func() {
				if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), marketText(i18n.T(i18n.KeyW4MarketReloadDetails), "Reload details"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: d.DetailsLoading}).Clicked() {
					m.Store.SelectDetails(item.ID, true)
				}
				favoriteButton(c, m, item.ID)
				ui.Box(c).Grow(1)
				if m.InstallBusy {
					ui.Text(c, i18n.T(i18n.KeyMarketInstallRunning)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				} else {
					catalogInstallButton(c, m, d, item, inventory, fg, muted)
				}
			})
		})
	})
	if !open {
		m.Store.CloseDetails()
	}
}

// marketIconTile is the card's 44px identity tile: a rounded frame with the
// plugin's initial, the same weight a row icon carries in the plugin list.
func marketIconTile(c *ui.Context, item desktopmarket.CatalogItem, muted, border ui.Color) {
	tile := ui.Box(c.Key("market-icon-"+item.ID)).Size(44, 44).Shrink(0).Radius(10).Border(.5, border).AlignItems(ui.Center).Justify(ui.Center)
	tile.Children(func() {
		initial := "?"
		if runes := []rune(strings.TrimSpace(catalogTitle(item))); len(runes) > 0 {
			initial = strings.ToUpper(string(runes[0]))
		}
		ui.Text(c, initial).FontSize(theme.InterfaceSize(17)).FontWeight(600).TextColor(muted)
	})
}

// marketRefreshIcon is the circular-arrow glyph of the header's refresh button.
var marketRefreshIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round"><path d="M13 6a5 5 0 1 0 0 5M13 2v4H9"/></svg>`))
