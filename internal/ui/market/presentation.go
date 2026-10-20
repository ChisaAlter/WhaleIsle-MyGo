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
)

// The desktop-owned MarketSection is an 800 DIP settings column. The
// container's measured width, rather than the window, controls its card grid.
func marketHeader(c *ui.Context, m Model, d marketstore.Discovery, fg, muted ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMarketTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
			ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketFindAndManagePluginsInTheCurrentConfi), "Discover and manage plugins in the current profile.")).FontSize(theme.InterfaceSize(14)).TextColor(muted).FillWidth()
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
			if !m.Embedded && m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyMarketBack)).Clicked() {
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
			button := ui.Button(c.Key("market-tab-"+item.id), label).Role(ui.RoleTab).Height(30).Padding(0, 12).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Selected(active)
			if active {
				button.Background(field).TextColor(fg)
			}
			if button.Clicked() {
				m.Store.SetTab(item.id)
			}
		}
	})
}

func catalogCategories(c *ui.Context, m Model, d marketstore.Discovery, fg, muted, field ui.Color) {
	if len(d.Catalog.Categories) <= 1 {
		return
	}
	ui.Row(c).Role(ui.RoleRadioGroup).Label(marketText(i18n.T(i18n.KeyW4MarketCategories), "Categories")).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
		for i, category := range d.Catalog.Categories {
			if i >= 5 && category.ID != d.Category && (m.categoriesExpanded == nil || !*m.categoriesExpanded) {
				continue
			}
			active := category.ID == d.Category
			button := ui.Button(c.Key("market-category-"+category.ID), fmt.Sprintf("%s %d", category.Label, category.Count)).Role(ui.RoleRadio).Checked(active).Height(28).Padding(0, 10).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
			if active {
				button.Background(field).TextColor(fg)
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
			if ui.Button(c, label).Clicked() {
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
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketDeprecated), "Deprecated")).FontSize(theme.InterfaceSize(11)).TextColor(accent)
					}
				})
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					if i.Owner != "" && !PluginImage(c, "https://github.com/"+url.PathEscape(i.Owner)+".png?size=96", 16, 16, m.SVGImages) {
						ui.Text(c, strings.ToUpper(string([]rune(i.Owner)[0]))).Width(16).Height(16).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					}
					ui.Text(c, i.Owner).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					ui.Text(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketSStars), "%s stars"), formatStars(i.Stars))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				})
				ui.Text(c, i.Description).FillWidth().MinWidth(0).MaxLines(2).MinHeight(36).FontSize(theme.InterfaceSize(12)).TextColor(muted).Tooltip(i.Description)
				ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, catalogCategory(d.Catalog, i.Category)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					favoriteButton(c, m, i.ID)
					if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketDetails), "Details")).Clicked() {
						m.Store.SelectDetails(i.ID, false)
					}
					if parsed, err := url.Parse(i.Homepage); err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil && m.OnOpenURL != nil {
						if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketHomepage), "Homepage")).Clicked() {
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

func installedRows(c *ui.Context, m Model, d marketstore.Discovery, snap marketstore.ViewState, fg, muted, border, accent ui.Color) {
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

func catalogDetailsDialog(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	if d.Selected == nil || m.Store.Confirmation() != "" {
		return
	}
	item := *d.Selected
	open := true
	w, h := c.Size()
	ui.DialogBase(c.Key("market-details-dialog"), &open, func(backdrop, panel ui.Element) {
		progress := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
		chrome.RevealMask(backdrop, ui.RGBA(0, 0, 0, .35), progress)
		backdrop.Padding(16)
		panel.Label(catalogTitle(item)).Width(max(0, min(float32(680), w-32))).Height(max(0, min(float32(720), h-48))).Border(.5, border).Radius(12).Clip().Opacity(progress).Draw(chrome.Glass(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).BG), chrome.MaterialPanel, 12)).Children(func() {
			ui.Row(c).FillWidth().Padding(16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, catalogTitle(item)).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketReloadDetails), "Reload details")).Disabled(d.DetailsLoading).Clicked() {
					m.Store.SelectDetails(item.ID, true)
				}
				if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketClose), "Close")).Clicked() {
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
		})
		// --dsw-elevation-stroke: the panel paints its hairline after the glass,
		// because the glass fill runs after the element's own border and would
		// cover it once no wallpaper makes the fill opaque (as MenuSurface does).
		panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if border.A == 0 || r.W <= 0 || r.H <= 0 {
				return
			}
			p.Stroke(r, border, 12, 0.5)
		})
	})
	if !open {
		m.Store.CloseDetails()
	}
}
