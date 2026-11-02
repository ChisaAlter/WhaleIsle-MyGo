// Package market is the native plugin market surface: catalog discovery with
// search, categories, details and install, plus installed-plugin management
// (enable, disable, batch update, remove).
package market

import (
	"fmt"
	"strings"
	desktopmarket "whaleisle/internal/desktop/market"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/api"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Model wires the market panel to store, install state, and harness status.
type Model struct {
	SVGImages *svgimage.Owner
	Store     *marketstore.Store
	// Embedded is used by the Settings market section, whose shell owns its header and insets.
	Embedded             bool
	categoriesExpanded   *bool
	ThemeID              string
	SystemDark           bool
	HarnessReady         bool
	MuxConnected         bool
	HarnessMarketOverlay bool
	PluginDisabled       bool
	InstallReady         bool
	InstallSpec          *string
	InstallBusy          bool
	InstallMessage       string
	Activation           marketstore.Activation
	CanRestart           bool
	OnRestart            func()
	// InstallLog is the trailing installctrl.Result.Log (phase output).
	InstallLog       string
	NeedsAllowBuilds bool
	AllowBuilds      []string
	// CanCancelInstall is true when a pluginManager/installBundle requestId is active.
	CanCancelInstall bool
	Incompatible     []api.ChangeResultErrorIncompatibleItem

	OnClose            func()
	OnRefresh          func()
	OnInstallSpec      func(spec string, allowBuilds []string)
	OnBatchUpdate      func(targets []marketstore.InstallTarget)
	OnStopBatch        func()
	OnInstallCatalog   func(target marketstore.InstallTarget)
	OnRemoveBundle     func(name string)
	OnSetBundleEnabled func(name string, enabled bool)
	OnCancelInstall    func()
	OnAcceptExemption  func(packageVersion, runtimeVersion string)
	OnCopyInstallLog   func(log string)
	OnExportHistory    func()
	// OnOpenURL activates an authored documentation link after an explicit click.
	OnOpenURL func(string)
	// OnOpenHarnessMarket opens the Electron ui-settings-market WebView when overlay is on.
	OnOpenHarnessMarket func()
}

// View is a native discovery and installed-plugin surface.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	m = withMarketConfirmations(m)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	bg, border, fg, muted, field, accent := chrome.Color(colors.BG), chrome.Color(colors.Border), chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Field), chrome.Color(colors.Accent)
	snap := m.Store.Snapshot()
	d := m.Store.Discovery()
	personal := m.Store.Personal()
	top, side, bottom := float32(28), float32(24), float32(48)
	if m.Embedded {
		top, side, bottom = 0, 0, 0
	}
	container := ui.Column(c).FillWidth().MinWidth(0).Background(bg)
	m.categoriesExpanded = ui.Local(container, "market-categories-expanded", func() bool { return false })
	if !m.Embedded {
		container.Grow(1).MinHeight(0)
	}
	container.Children(func() {
		var body ui.Element
		if m.Embedded {
			// Settings owns the options scroller. An inner growing viewport
			// has no intrinsic height under that scroller and hides the section.
			body = ui.Column(c).FillWidth().MinWidth(0)
		} else {
			body = ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Padding(top, side, bottom, side)
		}
		body.Children(func() {
			ui.Column(c).Gap(16).FillWidth().MaxWidth(800).MinWidth(0).Children(func() {
				marketHeader(c, m, d, fg, muted, border)
				marketTabs(c, m, d.Tab, fg, muted, field)
				if personal.Err != "" {
					ui.Text(c, personal.Err).TextColor(accent)
				}
				if d.Tab == "history" {
					if m.OnExportHistory != nil && ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketExportInstallHistory), "Export installation history")).AlignSelf(ui.Start).Height(28).Padding(0, 12).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(len(personal.History) == 0).Clicked() {
						m.OnExportHistory()
					}
					ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketTheLast200OperationsInterruptedOrUnkno), "Latest 200 operations. Interrupted or unknown means unconfirmed; refresh inventory before retrying.")).TextColor(muted)
					if len(personal.History) == 0 {
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketNoInstallHistoryYet), "No installation history yet.")).TextColor(muted)
					}
					managementFilters(c, m)
					danger := chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Danger)
					for _, h := range personal.History {
						if m.Store.ManagementFilter() == "failed" && !failedOutcome(h.Result) {
							continue
						}
						if m.Store.ManagementFilter() == "updates" && h.Action != "update" {
							continue
						}
						marketHistoryRow(c, h, fg, muted, field, danger, border)
					}
				} else if d.Tab != "installed" {
					discoveryView(c, m, d, snap, fg, muted, field, border, accent)
				} else {
					if snap.Err != "" {
						ui.Text(c, snap.Err).TextColor(accent)
					}
					if snap.Loading {
						ui.Text(c, i18n.T(i18n.KeyMarketLoading)).TextColor(muted)
					}
					if !snap.ClientReady {
						ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketTheRuntimeIsNotConnectedTheCatalogIs), "Runtime disconnected; the catalog remains available.")).TextColor(muted)
					}
					managementFilters(c, m)
					updatesView(c, m, snap, fg, muted, border, accent)
					if len(desktopInstalledRows(snap.Rows)) == 0 {
						ui.Text(c, i18n.T(i18n.KeyMarketListEmpty)).TextColor(muted)
					}
					installedRows(c, m, d, snap, fg, muted, field, border, accent)
					installSection(c, m, fg, muted, field, border, accent)
				}
				if m.InstallMessage != "" {
					ui.Text(c, m.InstallMessage).TextColor(accent)
				}
				if m.Activation.Pending {
					text := marketText(i18n.T(i18n.KeyW4MarketPluginChangesAreSavedAndTakeEffectAft), "Plugin changes are saved and await a runtime restart.")
					if m.Activation.Failed {
						text = marketText(i18n.T(i18n.KeyW4MarketTheRestartDidNotSucceedAndChangesStil), "Restart failed. Saved changes still await activation; do not install again.")
					}
					if m.Activation.Restarting {
						text = marketText(i18n.T(i18n.KeyW4MarketRestartingTheRuntimeAndWaitingForTheN), "Restarting; waiting for the new Host to become ready…")
					}
					ui.Text(c, text).TextColor(accent)
					if m.CanRestart && !m.InstallBusy && !m.Activation.Restarting && m.OnRestart != nil {
						if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketRestartRuntimeAndApply), "Restart runtime and apply")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							m.OnRestart()
						}
					}
				} else if m.Activation.Refreshed {
					ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketTheNewRuntimeIsReadyCheckTheLoadStat), "The new runtime is ready. Check the plugin list for loading status.")).TextColor(muted)
				}
				if m.InstallBusy {
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
						ui.Text(c, i18n.T(i18n.KeyMarketInstallRunning)).TextColor(muted)
						if m.CanCancelInstall && m.OnCancelInstall != nil && ui.Button(c, i18n.T(i18n.KeyMarketCancelInstall)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							m.OnCancelInstall()
						}
					})
				}
				if m.NeedsAllowBuilds && len(m.AllowBuilds) > 0 {
					ui.Text(c, i18n.T(i18n.KeyMarketAllowBuilds)+": "+strings.Join(m.AllowBuilds, ", ")).TextColor(fg)
					if d.Tab != "installed" && !m.InstallBusy && m.OnInstallSpec != nil && m.InstallSpec != nil && ui.Button(c, i18n.T(i18n.KeyMarketAllowBuildsButton)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
						m.OnInstallSpec(*m.InstallSpec, append([]string(nil), m.AllowBuilds...))
					}
				}
				for _, item := range m.Incompatible {
					it := item
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
						ui.Text(c, it.Name+"@"+it.Version+" · runtime "+it.RuntimeVersion).TextColor(accent)
						if !m.InstallBusy && m.OnAcceptExemption != nil && it.Name != "" && it.Version != "" && ui.Button(c, i18n.T(i18n.KeyMarketAcceptExemption)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							m.OnAcceptExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
						}
					})
				}
				if log := strings.TrimSpace(m.InstallLog); log != "" {
					logBox := ui.Column(c.Key("market-install-log")).FillWidth().MinWidth(0).Gap(6)
					logOpen := ui.Local(logBox, "open", func() bool { return false })
					logBox.Children(func() {
						if strings.Contains(strings.ToLower(log), "operation-error") {
							ui.Text(c, marketText("安装未完成。请检查网络或安装源，或展开日志排查。", "Installation did not finish. Check the network or source, or open the log to inspect it.")).TextColor(accent)
						}
						if ui.Button(c, marketText("安装日志", "Install log")).FontSize(theme.InterfaceSize(11)).Clicked() {
							*logOpen = !*logOpen
						}
						if !*logOpen {
							return
						}
						if m.OnCopyInstallLog != nil && ui.Button(c, i18n.T(i18n.KeyMarketCopyLog)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							m.OnCopyInstallLog(log)
						}
						lines := strings.Split(log, "\n")
						if len(lines) > 40 {
							lines = lines[len(lines)-40:]
						}
						ui.Scroll(c).FillWidth().MaxHeight(180).Children(func() {
							ui.Text(c, strings.Join(lines, "\n")).FontSize(theme.InterfaceSize(10)).TextColor(muted).Selectable()
						})
					})
				}
			})
		})
	})
	catalogDetailsDialog(c, m, d, snap, fg, muted, field, accent)
	confirmationView(c, m, fg, accent)
}

func marketText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

// marketHistoryRow is one install-history receipt: the plugin and result lead,
// the timestamp and raw receipt follow as secondary lines.
func marketHistoryRow(c *ui.Context, h marketstore.InstallHistory, fg, muted, field, danger, border ui.Color) {
	result, color := marketHistoryResult(h.Result, fg, danger)
	action := marketHistoryAction(h.Action)
	row := ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(12, 16).Margin(0, 0, 6, 0).Radius(12).Border(.5, border)
	chrome.GlassSurface(row, field, chrome.MaterialPanel, 12)
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, h.Plugin).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg).SingleLine()
			ui.Text(c, result).FontSize(theme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, border).TextColor(color)
		})
		ui.Text(c, h.At.Local().Format("2006-01-02 15:04")+" · "+action).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if h.Error != "" && h.Error != h.Result {
			ui.Text(c, h.Error).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(11)).TextColor(muted).SingleLine()
		}
	})
}

func marketHistoryResult(result string, fg, danger ui.Color) (string, ui.Color) {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "applied", "applied-restart", "applied_restart":
		return marketText("已应用", "Applied"), fg
	case "failed":
		return marketText("失败", "Failed"), danger
	case "interrupted":
		return marketText("已中断", "Interrupted"), danger
	case "not-started":
		return marketText("未启动", "Not started"), danger
	case "unknown":
		return marketText("结果未知", "Unknown"), danger
	default:
		return result, fg
	}
}

func marketHistoryAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "install":
		return marketText("安装", "Install")
	case "update":
		return marketText("更新", "Update")
	case "remove":
		return marketText("移除", "Remove")
	case "enable":
		return marketText("启用", "Enable")
	case "disable":
		return marketText("停用", "Disable")
	default:
		return action
	}
}

func discoveryView(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketCatalogSource), "Catalog source: ")+d.Catalog.Source+" · "+marketText(i18n.T(i18n.KeyW4MarketUpdated), "updated: ")+d.Catalog.Updated).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	if !d.Catalog.FetchedAt.IsZero() {
		ui.Text(c, d.Catalog.FetchedAt.Format("2006-01-02 15:04 MST")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if d.Catalog.Warning != "" {
		ui.Text(c, d.Catalog.Warning).FontSize(theme.InterfaceSize(11)).TextColor(accent)
	}
	if d.Err != "" {
		ui.Text(c, d.Err).TextColor(accent)
	}
	if d.Loading {
		ui.Text(c, i18n.T(i18n.KeyMarketLoading)).TextColor(muted)
	}
	query := d.Query
	ui.TextInput(c, &query).FillWidth().Height(32).Radius(8).Border(.5, border).Background(field).Padding(0, 10).FontSize(theme.InterfaceSize(13)).Placeholder(marketText(i18n.T(i18n.KeyW4MarketSearchNamePackageOrDescription), "Search name, package or description")).OnChange(func() { m.Store.SetQuery(query) })
	catalogCategories(c, m, d, fg, muted, field, border)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketSort), "Sort")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		sortIDs := []string{"catalog", "stars", "newest"}
		sortLabels := []string{marketText(i18n.T(i18n.KeyW4MarketCatalogOrder), "Catalog order"), marketText(i18n.T(i18n.KeyW4MarketMostStars), "Most stars"), marketText(i18n.T(i18n.KeyW4MarketNewest), "Newest")}
		sortLabel := sortLabels[0]
		for n, id := range sortIDs {
			if id == d.Sort {
				sortLabel = sortLabels[n]
			}
		}
		ui.Select(c, &sortLabel, sortLabels).Width(180).Height(32).Radius(8).Border(.5, border).MaxWidthPercent(100).OnChange(func() {
			for n, label := range sortLabels {
				if label == sortLabel {
					m.Store.SetSort(sortIDs[n])
					break
				}
			}
		})
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketPublished), "Listed")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		periodIDs := []string{"all", "7", "30"}
		periodLabels := []string{marketText(i18n.T(i18n.KeyW4MarketAnyTime), "All time"), marketText(i18n.T(i18n.KeyW4MarketLast7Days), "Last 7 days"), marketText(i18n.T(i18n.KeyW4MarketLast30Days), "Last 30 days")}
		periodLabel := periodLabels[0]
		for n, id := range periodIDs {
			if id == d.Period {
				periodLabel = periodLabels[n]
			}
		}
		ui.Select(c, &periodLabel, periodLabels).Width(180).Height(32).Radius(8).Border(.5, border).MaxWidthPercent(100).OnChange(func() {
			for n, label := range periodLabels {
				if label == periodLabel {
					m.Store.SetPeriod(periodIDs[n])
					break
				}
			}
		})
	})
	ui.Text(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketDPlugins), "%d plugins"), d.Total)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	catalogCards(c, m, d, inventory, fg, muted, field, border, accent)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		if d.Page > 0 && ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketPreviousPage), "Previous")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
			m.Store.SetPage(d.Page - 1)
		}
		if (d.Page+1)*60 < d.Total && ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketNextPage), "Next")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
			m.Store.SetPage(d.Page + 1)
		}
	})
}

func catalogInstallButton(c *ui.Context, m Model, d marketstore.Discovery, item desktopmarket.CatalogItem, inventory marketstore.ViewState, fg, muted ui.Color) {
	if name := marketstore.InstalledName(item, inventory.Rows); name != "" {
		// The installed row keeps the button slot, disabled, so a card does not
		// change height when its state flips.
		widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), marketText(i18n.T(i18n.KeyW4MarketInstalled), "Installed"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: true})
		return
	}
	if !desktopmarket.ValidInstallItem(item) {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketNoSupportedInstallTarget), "No supported install target.")).TextColor(muted)
		return
	}
	if !d.RuntimeReady {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketConnectTheRuntimeBeforeInstalling), "Connect the runtime to install.")).TextColor(muted)
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if !m.InstallBusy && !d.Loading && m.OnInstallCatalog != nil && widgets.Button(c, colors, i18n.T(i18n.KeyMarketInstallButton), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
		if target, e := m.Store.CaptureTarget(item.ID); e == nil {
			m.OnInstallCatalog(target)
		}
	}
}

func installSection(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	sec := ui.Column(c).Border(.5, border).Radius(12).Padding(14, 16).Gap(8)
	chrome.GlassSurface(sec, field, chrome.MaterialPanel, 12)
	sec.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMarketInstallSpecLabel)).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(fg)
		spec := ""
		if m.InstallSpec != nil {
			spec = strings.TrimSpace(*m.InstallSpec)
		}
		var allow []string
		if m.NeedsAllowBuilds {
			allow = append([]string(nil), m.AllowBuilds...)
		}
		btnLabel := i18n.T(i18n.KeyMarketInstallButton)
		if m.NeedsAllowBuilds {
			btnLabel = i18n.T(i18n.KeyMarketAllowBuildsButton)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			if m.InstallSpec != nil {
				ui.TextInput(c, m.InstallSpec).Grow(1).MinWidth(0).Height(32).Radius(8).Border(.5, border).Background(field).Padding(0, 10).Placeholder(i18n.T(i18n.KeyMarketInstallPlaceholder))
			}
			if !m.InstallBusy {
				install := ui.Button(c, btnLabel).Height(28).Padding(0, 12).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Shrink(0).Disabled(spec == "")
				if install.Clicked() && m.OnInstallSpec != nil && spec != "" {
					m.OnInstallSpec(spec, allow)
				}
			}
		})
		if !m.InstallReady {
			ui.Text(c, i18n.T(i18n.KeyMarketInstallNotReady)).FontSize(theme.InterfaceSize(11)).TextColor(accent)
		}
	})
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyMarketOn)
	}
	return i18n.T(i18n.KeyMarketOff)
}

func muxLabel(ready, connected bool) string {
	if !ready {
		return i18n.T(i18n.KeyMarketMuxWaiting)
	}
	if connected {
		return i18n.T(i18n.KeyMarketMuxConnected)
	}
	return i18n.T(i18n.KeyMarketMuxDisconnected)
}

func rpcLabel(hasRPC bool, count int) string {
	if hasRPC {
		return i18n.T(i18n.KeyMarketRPCYes)
	}
	if count > 0 {
		return i18n.T(i18n.KeyMarketRPCPartial)
	}
	return i18n.T(i18n.KeyMarketRPCNo)
}

func favoriteButton(c *ui.Context, m Model, id string) {
	label := marketText(i18n.T(i18n.KeyW4MarketSave), "Favorite")
	if m.Store.Personal().Favorites[id] {
		label = marketText(i18n.T(i18n.KeyW4MarketRemoveFromSaved), "Unfavorite")
	}
	if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
		_ = m.Store.ToggleFavorite(id)
	}
}

// updatesView reuses catalog install receipts, so updates share build approval,
// cancellation, runtime guards, and installation history with normal installs.
func updatesView(c *ui.Context, m Model, inventory marketstore.ViewState, fg, muted, border, accent ui.Color) {
	u := m.Store.Updates()
	ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketPluginVersionUpdates), "Plugin version updates")).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	if !m.InstallBusy && !inventory.Loading && inventory.Err == "" && inventory.ClientReady && !m.Store.Discovery().Loading && !u.Loading {
		if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), marketText(i18n.T(i18n.KeyW4MarketCheckPluginUpdates), "Check plugin updates"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
			m.Store.CheckUpdates()
		}
	}
	if u.Loading {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketCheckingVersionMetadata), "Checking version metadata…")).TextColor(muted)
	}
	if u.Err != "" {
		ui.Text(c, u.Err).TextColor(accent)
	}
	if !u.CheckedAt.IsZero() {
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketCheckedAt), "Checked: ")+u.CheckedAt.Local().Format("2006-01-02 15:04:05 MST")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketOnlyConfirmedNpmStableVersionsAreCompa), "Only verified npm stable versions are compared. Git, local sources and incomparable versions remain unknown. Checking never installs automatically.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	batchSelectionView(c, m, u, muted, border)
	for _, row := range u.Rows {
		if m.Store.ManagementFilter() == "updates" && !row.Available {
			continue
		}
		if m.Store.ManagementFilter() == "failed" && !failedPlugin(m, row.ID, row.Name) {
			continue
		}
		r := row
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			current, latest := r.Current, r.Latest
			if current == "" {
				current = marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
			}
			if latest == "" {
				latest = marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
			}
			status := marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
			if r.Known {
				status = marketText(i18n.T(i18n.KeyW4MarketNoNewerVersion), "No newer version")
			}
			if r.Available {
				status = marketText(i18n.T(i18n.KeyW4MarketUpdateAvailable), "Update available")
			}
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, r.Name+" · "+current+" → "+latest+" · "+status).FontSize(theme.InterfaceSize(13)).TextColor(fg)
				if r.Reason != "" {
					ui.Text(c, r.Reason).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			})
			if r.Available && !m.InstallBusy && !u.Loading && u.Err == "" && m.OnInstallCatalog != nil {
				label := marketText(i18n.T(i18n.KeyW4MarketSelect), "Select")
				if m.Store.UpdateSelected(r.ID) {
					label = marketText(i18n.T(i18n.KeyW4MarketClearSelection), "Deselect")
				}
				if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
					m.Store.ToggleUpdateSelected(r.ID)
				}
				if target, err := m.Store.CaptureUpdateTarget(r.ID); err == nil {
					if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketUpdateTo), "Update to ")+r.Latest).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
						m.OnInstallCatalog(target)
					}
				}
			}
		})
	}
}
