package market

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/errcopy"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

func rowsCard(c *ui.Context, fn func()) {
	t := design.For(c)
	ui.Column(c).FillWidth().MinWidth(0).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().Children(fn)
}

func rowSeparator(c *ui.Context) {
	coss.Separator(c, coss.Horizontal)
}

func rowFrame(e ui.Element) ui.Element {
	return e.FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(3), design.Sp(4)).AlignItems(ui.Center)
}

// The desktop-owned MarketSection is an 800 DIP settings column. The
// container's measured width, rather than the window, controls its card grid.
func marketHeader(c *ui.Context, p *page) {
	m, d := p.m, p.d
	description := marketText("从精选目录浏览、安装并更新社区插件；变更后 Harness 会自动重启。", "Browse, install and update community plugins from the curated catalog; the Harness restarts after a change.")
	pagekit.Header(c, i18n.T(i18n.KeyMarketTitle), description, func() {
		refreshLabel := i18n.T(i18n.KeyMarketRefresh)
		if d.Loading {
			refreshLabel = marketText("正在加载插件目录…", "Loading the plugin catalog…")
		}
		if coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonOutline,
			Size:     coss.ButtonIconSM,
			Icon:     coss.IconRefresh,
			Label:    refreshLabel,
			Loading:  d.Loading,
			Disabled: d.Loading || m.InstallBusy,
		}).Tooltip(refreshLabel).Clicked() {
			m.Store.RefreshCatalog(true, i18n.Locale())
			if m.OnRefresh != nil {
				m.OnRefresh()
			} else {
				m.Store.Refresh(context.Background())
			}
		}
		if !m.Embedded && m.OnClose != nil && coss.Button(c, i18n.T(i18n.KeyMarketBack), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconArrowLeft}).Clicked() {
			m.Store.CloseDetails()
			m.OnClose()
		}
	})
}

func marketTabs(c *ui.Context, p *page) {
	m := p.m
	tabIDs := []string{"discover", "installed", "favorites", "history"}
	tabItems := []struct{ id, zh, en string }{
		{"discover", i18n.T(i18n.KeyW4MarketDiscover), "Discover"},
		{"installed", i18n.T(i18n.KeyW4MarketInstalled2), "Installed"},
		{"favorites", i18n.T(i18n.KeyW4MarketSave), "Favorites"},
		{"history", i18n.T(i18n.KeyW4MarketInstallHistory), "History"},
	}
	installedCount := len(desktopInstalledRows(m.Store.Snapshot().Value.Rows))
	labels := make([]string, len(tabItems))
	selected := 0
	for i, item := range tabItems {
		label := marketText(item.zh, item.en)
		if item.id == "installed" && installedCount > 0 {
			label = fmt.Sprintf("%s (%d)", label, installedCount)
		}
		labels[i] = label
		if p.d.Tab == item.id {
			selected = i
		}
	}
	if coss.Tabs(c, &selected, labels, coss.TabsOptions{Label: i18n.T(i18n.KeyMarketTitle), Key: "market-tabs"}).Changed() {
		m.Store.SetTab(tabIDs[selected])
	}
}

func catalogCategories(c *ui.Context, p *page) {
	m, d := p.m, p.d
	if len(d.Catalog.Categories) <= 1 {
		return
	}
	ui.Row(c).Label(marketText(i18n.T(i18n.KeyW4MarketCategories), "Categories")).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
		for i, category := range d.Catalog.Categories {
			if i >= 5 && category.ID != d.Category && (p.categoriesExpanded == nil || !*p.categoriesExpanded) {
				continue
			}
			on := category.ID == d.Category
			label := fmt.Sprintf("%s %d", category.Label, category.Count)
			if coss.Toggle(c, &on, label, coss.ToggleOptions{Variant: coss.ToggleOutline, Size: coss.ToggleSM, Key: "market-category-" + category.ID}).Changed() && on {
				m.Store.SetCategory(category.ID)
			}
		}
		if len(d.Catalog.Categories) > 5 && p.categoriesExpanded != nil {
			label := marketText(i18n.T(i18n.KeyW4MarketExpandCategories), "More categories")
			if *p.categoriesExpanded {
				label = marketText(i18n.T(i18n.KeyW4MarketCollapseCategories), "Fewer categories")
			}
			if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM}).Clicked() {
				*p.categoriesExpanded = !*p.categoriesExpanded
			}
		}
	})
}

func catalogCards(c *ui.Context, p *page) {
	m, d := p.m, p.d
	t := design.For(c)
	if len(d.Rows) == 0 && !d.Loading && d.Err == nil && !d.Catalog.FetchedAt.IsZero() {
		message := marketText(i18n.T(i18n.KeyW4MarketNoPluginsYet), "No plugins available.")
		if d.Query != "" || d.Category != "all" && d.Category != "" || d.Period != "all" || d.Tab == "favorites" {
			message = marketText(i18n.T(i18n.KeyW4MarketNoMatchingPlugins), "No matching plugins.")
		}
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconSearch, Title: message}, nil)
	}
	grid := ui.Row(c).Key("market-cards").FillWidth().MinWidth(0).Wrap().Gap(design.Sp(3)).AlignItems(ui.Start)
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
	version := p.version()
	cardVersion := pagekit.Version(version).Float(width).Key()
	grid.Children(func() {
		for _, item := range d.Rows {
			i := item
			card := ui.Column(c.Key("market-card-"+i.ID)).MinWidth(0).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Padding(design.Sp(3.5), design.Sp(4)).Gap(design.Sp(2))
			cardBg := t.Card
			if card.Hovered() {
				cardBg = t.Accent.Over(t.Card)
			}
			card.Background(cardBg)
			if columns == 2 {
				card.Width((width - design.Sp(3)) / 2)
			} else {
				card.FillWidth()
			}
			card.Memo(cardVersion).Children(func() {
				ui.Row(c).FillWidth().Gap(design.Sp(2.5)).AlignItems(ui.Start).Children(func() {
					marketIconTile(c, i)
					ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
						ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							pagekit.Strong(c, catalogTitle(i)).Grow(1).MinWidth(0).SingleLine().Tooltip(i.ID)
							if badge := capabilityBadgeLabel(i.CapabilityKind, i.NativeCompatibilityLabel); badge != "" {
								variant := coss.BadgeOutline
								if i.CapabilityKind == desktopmarket.CapabilityWebUIOnly {
									variant = coss.BadgeWarning
								}
								coss.Badge(c, badge, coss.BadgeOptions{Variant: variant}).Tooltip(capabilityGuidanceText(i.CapabilityKind, i.Capabilities.Guidance))
							}
							if i.Deprecated {
								coss.Badge(c, marketText(i18n.T(i18n.KeyW4MarketDeprecated), "Deprecated"), coss.BadgeOptions{Variant: coss.BadgeWarning})
							}
						})
						ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							if i.Owner != "" {
								shown, h := PluginImageHandle(c, "https://github.com/"+url.PathEscape(i.Owner)+".png?size=96", design.Sp(4), design.Sp(4), m.Media)
								if !shown {
									p.watch.Track(h)
									pagekit.Caption(c, strings.ToUpper(string([]rune(i.Owner)[0]))).Width(design.Sp(4)).Height(design.Sp(4))
								}
							}
							pagekit.Caption(c, i.Owner)
							pagekit.Caption(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketSStars), "%s stars"), formatStars(i.Stars)))
						})
					})
				})
				pagekit.Muted(c, i.Description).FillWidth().MinWidth(0).MaxLines(2).MinHeight(2 * design.TextSM.LineHeight).Tooltip(i.Description)
				if i.CapabilityKind == desktopmarket.CapabilityWebUIOnly {
					pagekit.Caption(c, capabilityGuidanceText(i.CapabilityKind, i.Capabilities.Guidance)).FillWidth().MinWidth(0).TextColor(t.WarningForeground)
				}
				ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					coss.Badge(c, catalogCategory(d.Catalog, i.Category), coss.BadgeOptions{Variant: coss.BadgeOutline})
					favoriteButton(c, p, i.ID)
					ui.Spacer(c)
					if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketDetails), "Details"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
						m.Store.SelectDetails(i.ID, false)
					}
					if parsed, err := url.Parse(i.Homepage); err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil && m.OnOpenURL != nil {
						if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketHomepage), "Homepage"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, TrailingIcon: coss.IconExternalLink}).Clicked() {
							m.OnOpenURL(parsed.String())
						}
					}
					catalogInstallButton(c, p, i)
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

func installedRows(c *ui.Context, p *page) {
	m, d, snap := p.m, p.d, p.snap
	t := design.For(c)
	var rows []marketstore.Entry
	installed := desktopInstalledRows(snap.Rows)
	for _, row := range installed {
		if showInstalledRow(m, row) {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 && len(installed) > 0 {
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconSearch, Title: marketText(i18n.T(i18n.KeyW4MarketNoMatchingPlugins), "No matching plugins.")}, nil)
	}
	version := p.version()
	ui.Column(c).Key("market-installed-rows").FillWidth().MinWidth(0).Gap(design.Sp(5)).Memo(version).Children(func() {
		for _, group := range installedGroups(d.Catalog, rows) {
			if len(group.rows) == 0 {
				continue
			}
			g := group
			pagekit.Section(c, g.label, nil, func() {
				rowsCard(c, func() {
					for i, row := range g.rows {
						r := row
						if i > 0 {
							rowSeparator(c)
						}
						el := rowFrame(ui.Row(c.Key("installed-" + r.Label))).Wrap()
						if el.Hovered() {
							el.Background(t.Accent)
						}
						el.Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
								ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
									pagekit.Strong(c, r.Label).Shrink(1).MinWidth(0).SingleLine().Tooltip(r.Label)
									if badge := capabilityBadgeLabel(r.CapabilityKind, r.NativeCompatibilityLabel); badge != "" {
										variant := coss.BadgeOutline
										if r.CapabilityKind == desktopmarket.CapabilityWebUIOnly {
											variant = coss.BadgeWarning
										}
										coss.Badge(c, badge, coss.BadgeOptions{Variant: variant}).Tooltip(capabilityGuidanceText(r.CapabilityKind, ""))
									}
								})
								pagekit.Caption(c, r.Detail).FillWidth().SingleLine().Tooltip(r.Detail)
								if r.CapabilityKind == desktopmarket.CapabilityWebUIOnly {
									pagekit.Caption(c, capabilityGuidanceText(r.CapabilityKind, "")).FillWidth().TextColor(t.WarningForeground)
								}
								if r.ReadOnlyReason != "" {
									pagekit.Caption(c, r.ReadOnlyReason).FillWidth()
								}
								if r.Error != "" {
									pagekit.Danger(c, errcopy.SentenceText(r.Error)).FillWidth()
								}
							})
							ui.Row(c).Shrink(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								if !m.Embedded && snap.ClientReady && !snap.Loading && snap.Err == nil && !m.InstallBusy && r.CanToggle() && m.OnSetBundleEnabled != nil {
									label := i18n.T(i18n.KeySettingsPluginsEnable)
									if *r.Enabled {
										label = i18n.T(i18n.KeySettingsPluginsDisable)
									}
									if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
										m.OnSetBundleEnabled(r.Label, !*r.Enabled)
									}
								}
								if snap.ClientReady && !snap.Loading && snap.Err == nil && !m.InstallBusy && r.IdentityKnown && r.ReadOnlyReason == "" && r.Removable && m.OnRemoveBundle != nil && coss.Button(c, i18n.T(i18n.KeyMarketRemove), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonSM, Icon: coss.IconTrash}).Clicked() {
									m.OnRemoveBundle(r.Label)
								}
							})
						})
					}
				})
			})
		}
	})
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

func catalogDetailsDialog(c *ui.Context, p *page) {
	m, d := p.m, p.d
	if d.Selected == nil || m.Store.Confirmation() != "" {
		return
	}
	t := design.For(c)
	item := *d.Selected
	open := true
	_, h := c.Size()
	refreshLabel := marketText("刷新详情", "Refresh details")
	closeLabel := marketText(i18n.T(i18n.KeyW4MarketClose), "Close")
	coss.Dialog(c, &open, coss.DialogOptions{
		Key:           "market-details-dialog",
		MaxWidth:      680,
		NoClose:       true,
		FooterVariant: coss.FooterBare,
		Footer: func() {
			favoriteButton(c, p, item.ID)
			ui.Spacer(c)
			if m.InstallBusy {
				pagekit.Caption(c, i18n.T(i18n.KeyMarketInstallRunning))
			} else {
				catalogInstallButton(c, p, item)
			}
		},
	}, func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			pagekit.Title(c, catalogTitle(item)).Grow(1).MinWidth(0).SingleLine()
			if coss.Button(c, "", coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonIconSM,
				Icon:     coss.IconRefresh,
				Label:    refreshLabel,
				Disabled: d.DetailsLoading,
			}).Tooltip(refreshLabel).Clicked() {
				m.Store.SelectDetails(item.ID, true)
			}
			if coss.Button(c, "", coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonIconSM,
				Icon:    coss.IconX,
				Label:   closeLabel,
			}).Tooltip(closeLabel).Clicked() {
				open = false
			}
		})
		ui.Column(c).FillWidth().MinWidth(0).MaxHeight(max(design.Sp(30), h-design.Sp(52))).Children(func() {
			coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(2), 0, design.Sp(1), 0).Children(func() {
					pagekit.Body(c, item.Description).FillWidth()
					catalogMetadata(c, marketText("作者", "Author"), item.Owner)
					catalogMetadata(c, marketText("来源", "Source"), item.InstallSpec)
					catalogMetadata(c, marketText("上架时间", "Added"), item.Added)
					compatKind, compatLabel, compatGuidance := resolveDetailsCapability(item, d.Details)
					compatValue := capabilityBadgeLabel(compatKind, compatLabel)
					if compatValue == "" {
						compatValue = marketText("未知", "Unknown")
					}
					catalogMetadata(c, marketText("兼容性", "Compatibility"), compatValue)
					if compatGuidance != "" {
						guidanceInk := t.MutedForeground
						if compatKind == desktopmarket.CapabilityWebUIOnly {
							guidanceInk = t.WarningForeground
						}
						pagekit.Caption(c, compatGuidance).FillWidth().TextColor(guidanceInk)
					}
					if item.Deprecated {
						pagekit.Note(c, coss.AlertWarning, marketText(i18n.T(i18n.KeyW4MarketDeprecatedInstallationIsNotOffered), "Deprecated; installation unavailable."), item.Replacement, nil)
					}
					if d.DetailsLoading {
						pagekit.Muted(c, i18n.T(i18n.KeyMarketLoading))
					}
					if d.DetailsErr != nil {
						pagekit.FaultSentence(c, marketText("详情加载失败", "Could not load details"), d.DetailsErr.SentenceText(), marketText("重试", "Retry"), func() {
							m.Store.SelectDetails(item.ID, true)
						})
					}
					if d.Details.Partial {
						pagekit.Note(c, coss.AlertWarning, marketText(i18n.T(i18n.KeyW4MarketSomeDetailsFailedToLoad), "Some details could not be loaded."), "", nil)
					}
					if d.Details.Version != "" {
						catalogMetadata(c, marketText("发布版本", "Published version"), d.Details.Version)
					}
					if len(d.Details.Requirements) > 0 {
						pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketDeclaredRuntimeRequirementsNotACompatib), "Declared requirements (not a compatibility verdict)")).FillWidth()
						for _, requirement := range d.Details.Requirements {
							pagekit.Code(c, requirement).FillWidth()
						}
					}
					marketScreenshotGallery(c, m, item)
					if parsed, err := url.Parse(item.Homepage); err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil && m.OnOpenURL != nil {
						if coss.Button(c, marketText("打开项目主页", "Open project homepage"), coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonSM, TrailingIcon: coss.IconExternalLink}).Label(item.Homepage).AlignSelf(ui.Start).MaxWidthPercent(100).MinWidth(0).Clicked() {
							m.OnOpenURL(parsed.String())
						}
					}
					marketReadme(c, m, d.Details.Readme)
				})
			})
		})
	}).Label(catalogTitle(item))
	if !open {
		m.Store.CloseDetails()
	}
}

func catalogMetadata(c *ui.Context, label, value string) {
	if value == "" {
		value = marketText("未知", "Unknown")
	}
	t := design.For(c)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).Children(func() {
		pagekit.Caption(c, label).Width(design.Sp(28)).MaxWidthPercent(100)
		pagekit.Caption(c, value).Grow(1).MinWidth(0).Selectable().TextColor(t.Foreground)
	})
}

// marketIconTile is the card's 44px identity tile: a rounded frame with the
// plugin's initial, the same weight a row icon carries in the plugin list.
func marketIconTile(c *ui.Context, item desktopmarket.CatalogItem) {
	t := design.For(c)
	tile := ui.Box(c.Key("market-icon-"+item.ID)).Size(design.Sp(11), design.Sp(11)).Shrink(0).Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Background(t.Card).AlignItems(ui.Center).Justify(ui.Center)
	tile.Children(func() {
		initial := "?"
		if runes := []rune(strings.TrimSpace(catalogTitle(item))); len(runes) > 0 {
			initial = strings.ToUpper(string(runes[0]))
		}
		ui.Text(c, initial).FontSize(design.TextLG.Size).FixedLineHeight(design.TextLG.LineHeight).FontWeight(design.WeightSemibold).TextColor(t.MutedForeground)
	})
}

func capabilityBadgeLabel(kind desktopmarket.CapabilityKind, rawLabel string) string {
	trimmed := strings.TrimSpace(rawLabel)
	var zhDefault, enDefault string
	switch kind {
	case desktopmarket.CapabilityPureHost:
		zhDefault, enDefault = "纯原生可用 (Pure Host)", "Pure native (Host)"
	case desktopmarket.CapabilityHybrid:
		zhDefault, enDefault = "原生兼容 + 可选 Web (Hybrid)", "Native + Web (Hybrid)"
	case desktopmarket.CapabilityMCP:
		zhDefault, enDefault = "MCP 服务 (原生支持)", "MCP server (Native)"
	case desktopmarket.CapabilitySkill:
		zhDefault, enDefault = "Skill 技能包 (原生支持)", "Skill package (Native)"
	case desktopmarket.CapabilityWebUIOnly:
		zhDefault, enDefault = "仅 Web 界面 (需兼容工作台)", "Web UI only"
	case desktopmarket.CapabilityUnknown, "":
		if trimmed != "" && trimmed != "未探测" {
			return trimmed
		}
		return ""
	default:
		if trimmed != "" {
			return trimmed
		}
		return string(kind)
	}
	if i18n.Locale() == i18n.LocaleZH {
		if trimmed != "" {
			return trimmed
		}
		return zhDefault
	}
	if trimmed != "" && trimmed != zhDefault && trimmed != "未探测" {
		return trimmed
	}
	return enDefault
}

func capabilityGuidanceText(kind desktopmarket.CapabilityKind, rawGuidance string) string {
	switch kind {
	case desktopmarket.CapabilityWebUIOnly:
		return marketText(
			"清单声明 Web UI 面板；在兼容工作台中打开后确认实际页面。",
			"Manifest declares a Web UI; open the compatibility workbench to resolve its live panels.",
		)
	case desktopmarket.CapabilityHybrid:
		return marketText(
			"清单声明 Host 与 Web 能力；后端状态以 Host 为准，页面在兼容工作台中解析。",
			"Host features run natively when enabled; the manifest's optional panels resolve in the compatibility workbench.",
		)
	case desktopmarket.CapabilityMCP:
		return marketText(
			"清单声明 MCP 服务；配置、连接与工具状态在 MCP 设置中确认。",
			"Declared MCP server; confirm configuration, connection and tools in MCP Settings.",
		)
	case desktopmarket.CapabilitySkill:
		return marketText(
			"清单声明 Skill 技能；安装后在技能列表确认内容和调用权限。",
			"Declared Skill package; confirm installed content and invocation settings in Skills.",
		)
	case desktopmarket.CapabilityPureHost:
		return marketText(
			"清单声明后端能力，无需 WebView2；实际启用与健康状态以 Host 为准。",
			"Manifest declares Host capabilities without WebView2; the Host reports enabled and health state.",
		)
	default:
		return strings.TrimSpace(rawGuidance)
	}
}

func resolveDetailsCapability(item desktopmarket.CatalogItem, details desktopmarket.Details) (desktopmarket.CapabilityKind, string, string) {
	kind := details.CapabilityKind
	label := details.NativeCompatibilityLabel
	guidance := details.Capabilities.Guidance
	if kind == "" || kind == desktopmarket.CapabilityUnknown {
		if item.CapabilityKind != "" && item.CapabilityKind != desktopmarket.CapabilityUnknown {
			kind = item.CapabilityKind
		}
	}
	if label == "" {
		label = details.Capabilities.CompatibilityLabel
	}
	if label == "" {
		label = item.NativeCompatibilityLabel
	}
	if guidance == "" {
		guidance = item.Capabilities.Guidance
	}
	return kind, label, capabilityGuidanceText(kind, guidance)
}
