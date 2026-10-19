// Package market is the M9 native plugin marketplace stub panel.
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
	"whaleisle/internal/ui/theme"
)

// Model wires the market panel to store, install state, and harness status.
type Model struct {
	Store *marketstore.Store
	// Embedded is used by the Settings market section, whose shell owns its header and insets.
	Embedded             bool
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
				if !m.Embedded {
					marketHeader(c, m, d, fg, muted)
				}
				marketTabs(c, m, d.Tab, fg, muted, field)
				if personal.Err != "" {
					ui.Text(c, personal.Err).TextColor(accent)
				}
				if d.Tab == "history" {
					if m.OnExportHistory != nil && ui.Button(c, marketText("导出安装历史", "Export installation history")).Disabled(len(personal.History) == 0).Clicked() {
						m.OnExportHistory()
					}
					ui.Text(c, marketText("最近 200 次操作；interrupted 或 unknown 表示结果未确认，重试前请刷新库存。", "Latest 200 operations. Interrupted or unknown means unconfirmed; refresh inventory before retrying.")).TextColor(muted)
					if len(personal.History) == 0 {
						ui.Text(c, marketText("暂无安装历史。", "No installation history yet.")).TextColor(muted)
					}
					managementFilters(c, m)
					for _, h := range personal.History {
						if m.Store.ManagementFilter() == "failed" && !failedOutcome(h.Result) {
							continue
						}
						if m.Store.ManagementFilter() == "updates" && h.Action != "update" {
							continue
						}
						ui.Text(c, h.At.Local().Format("2006-01-02 15:04:05 MST")+" · "+h.Plugin+" · "+h.Action+" · "+h.Result+" "+h.Error).TextColor(fg)
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
						ui.Text(c, marketText("运行时未连接；目录仍可浏览。", "Runtime disconnected; the catalog remains available.")).TextColor(muted)
					}
					managementFilters(c, m)
					updatesView(c, m, snap, fg, muted, accent)
					if len(snap.Rows) == 0 {
						ui.Text(c, i18n.T(i18n.KeyMarketListEmpty)).TextColor(muted)
					}
					installedRows(c, m, d, snap, fg, muted, border, accent)
					installSection(c, m, fg, muted, field, border, accent)
				}
				if m.InstallMessage != "" {
					ui.Text(c, m.InstallMessage).TextColor(accent)
				}
				if m.Activation.Pending {
					text := marketText("插件变更已保存，等待重启生效。", "Plugin changes are saved and await a runtime restart.")
					if m.Activation.Failed {
						text = marketText("重启未成功，变更仍待生效；无需再次安装。", "Restart failed. Saved changes still await activation; do not install again.")
					}
					if m.Activation.Restarting {
						text = marketText("正在重启运行时，等待新的 Host 就绪…", "Restarting; waiting for the new Host to become ready…")
					}
					ui.Text(c, text).TextColor(accent)
					if m.CanRestart && !m.InstallBusy && !m.Activation.Restarting && m.OnRestart != nil {
						if ui.Button(c, marketText("重启运行时并应用", "Restart runtime and apply")).Clicked() {
							m.OnRestart()
						}
					}
				} else if m.Activation.Refreshed {
					ui.Text(c, marketText("新的运行时已就绪，请在插件列表中查看加载状态。", "The new runtime is ready. Check the plugin list for loading status.")).TextColor(muted)
				}
				if m.InstallBusy {
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
						ui.Text(c, i18n.T(i18n.KeyMarketInstallRunning)).TextColor(muted)
						if m.CanCancelInstall && m.OnCancelInstall != nil && ui.Button(c, i18n.T(i18n.KeyMarketCancelInstall)).Clicked() {
							m.OnCancelInstall()
						}
					})
				}
				if m.NeedsAllowBuilds && len(m.AllowBuilds) > 0 {
					ui.Text(c, i18n.T(i18n.KeyMarketAllowBuilds)+": "+strings.Join(m.AllowBuilds, ", ")).TextColor(fg)
					if d.Tab != "installed" && !m.InstallBusy && m.OnInstallSpec != nil && m.InstallSpec != nil && ui.Button(c, i18n.T(i18n.KeyMarketAllowBuildsButton)).Clicked() {
						m.OnInstallSpec(*m.InstallSpec, append([]string(nil), m.AllowBuilds...))
					}
				}
				for _, item := range m.Incompatible {
					it := item
					ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
						ui.Text(c, it.Name+"@"+it.Version+" · runtime "+it.RuntimeVersion).TextColor(accent)
						if !m.InstallBusy && m.OnAcceptExemption != nil && it.Name != "" && it.Version != "" && ui.Button(c, i18n.T(i18n.KeyMarketAcceptExemption)).Clicked() {
							m.OnAcceptExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
						}
					})
				}
				if log := strings.TrimSpace(m.InstallLog); log != "" {
					if m.OnCopyInstallLog != nil && ui.Button(c, i18n.T(i18n.KeyMarketCopyLog)).Clicked() {
						m.OnCopyInstallLog(log)
					}
					lines := strings.Split(log, "\n")
					if len(lines) > 40 {
						lines = lines[len(lines)-40:]
					}
					ui.Text(c, strings.Join(lines, "\n")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			})
		})
	})
	catalogDetailsDialog(c, m, d, snap, fg, muted, field, border, accent)
	confirmationView(c, m, fg, accent)
}

func marketText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

func discoveryView(c *ui.Context, m Model, d marketstore.Discovery, inventory marketstore.ViewState, fg, muted, field, border, accent ui.Color) {
	ui.Text(c, marketText("目录来源：", "Catalog source: ")+d.Catalog.Source+" · "+marketText("更新日期：", "updated: ")+d.Catalog.Updated).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
	ui.TextInput(c, &query).FillWidth().Placeholder(marketText("搜索名称、包名或描述", "Search name, package or description")).OnChange(func() { m.Store.SetQuery(query) })
	catalogCategories(c, m, d, fg, muted, field)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, marketText("排序", "Sort")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		sortIDs := []string{"catalog", "stars", "newest"}
		sortLabels := []string{marketText("目录顺序", "Catalog order"), marketText("星标最多", "Most stars"), marketText("最新上架", "Newest")}
		sortLabel := sortLabels[0]
		for n, id := range sortIDs {
			if id == d.Sort {
				sortLabel = sortLabels[n]
			}
		}
		ui.Select(c, &sortLabel, sortLabels).Width(180).MaxWidthPercent(100).OnChange(func() {
			for n, label := range sortLabels {
				if label == sortLabel {
					m.Store.SetSort(sortIDs[n])
					break
				}
			}
		})
		ui.Text(c, marketText("上架时间", "Listed")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		periodIDs := []string{"all", "7", "30"}
		periodLabels := []string{marketText("全部时间", "All time"), marketText("最近 7 天", "Last 7 days"), marketText("最近 30 天", "Last 30 days")}
		periodLabel := periodLabels[0]
		for n, id := range periodIDs {
			if id == d.Period {
				periodLabel = periodLabels[n]
			}
		}
		ui.Select(c, &periodLabel, periodLabels).Width(180).MaxWidthPercent(100).OnChange(func() {
			for n, label := range periodLabels {
				if label == periodLabel {
					m.Store.SetPeriod(periodIDs[n])
					break
				}
			}
		})
	})
	ui.Text(c, fmt.Sprintf(marketText("%d 项插件", "%d plugins"), d.Total)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	catalogCards(c, m, d, inventory, fg, muted, field, border, accent)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		if d.Page > 0 && ui.Button(c, marketText("上一页", "Previous")).Clicked() {
			m.Store.SetPage(d.Page - 1)
		}
		if (d.Page+1)*60 < d.Total && ui.Button(c, marketText("下一页", "Next")).Clicked() {
			m.Store.SetPage(d.Page + 1)
		}
	})
}
func catalogInstallButton(c *ui.Context, m Model, d marketstore.Discovery, item desktopmarket.CatalogItem, inventory marketstore.ViewState, fg, muted ui.Color) {
	if name := marketstore.InstalledName(item, inventory.Rows); name != "" {
		ui.Text(c, marketText("已安装：", "Installed: ")+name).TextColor(fg)
		return
	}
	if !desktopmarket.ValidInstallItem(item) {
		ui.Text(c, marketText("无受支持的安装目标。", "No supported install target.")).TextColor(muted)
		return
	}
	if !d.RuntimeReady {
		ui.Text(c, marketText("请连接运行时后安装。", "Connect the runtime to install.")).TextColor(muted)
		return
	}
	if !m.InstallBusy && !d.Loading && m.OnInstallCatalog != nil && ui.Button(c, i18n.T(i18n.KeyMarketInstallButton)).Clicked() {
		if target, e := m.Store.CaptureTarget(item.ID); e == nil {
			m.OnInstallCatalog(target)
		}
	}
}

func installSection(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMarketInstallSpecLabel)).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		spec := ""
		if m.InstallSpec != nil {
			ui.TextInput(c, m.InstallSpec).FillWidth().Placeholder(i18n.T(i18n.KeyMarketInstallPlaceholder))
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
		if !m.InstallBusy && ui.Button(c, btnLabel).Clicked() && m.OnInstallSpec != nil && spec != "" {
			m.OnInstallSpec(spec, allow)
		}
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
	label := marketText("收藏", "Favorite")
	if m.Store.Personal().Favorites[id] {
		label = marketText("取消收藏", "Unfavorite")
	}
	if ui.Button(c, label).Clicked() {
		_ = m.Store.ToggleFavorite(id)
	}
}

// updatesView reuses catalog install receipts, so updates share build approval,
// cancellation, runtime guards, and installation history with normal installs.
func updatesView(c *ui.Context, m Model, inventory marketstore.ViewState, fg, muted, accent ui.Color) {
	u := m.Store.Updates()
	ui.Text(c, marketText("插件版本更新", "Plugin version updates")).Bold().TextColor(fg)
	if !m.InstallBusy && !inventory.Loading && inventory.Err == "" && inventory.ClientReady && !m.Store.Discovery().Loading && !u.Loading {
		if ui.Button(c, marketText("检查插件更新", "Check plugin updates")).Clicked() {
			m.Store.CheckUpdates()
		}
	}
	if u.Loading {
		ui.Text(c, marketText("正在检查版本元数据…", "Checking version metadata…")).TextColor(muted)
	}
	if u.Err != "" {
		ui.Text(c, u.Err).TextColor(accent)
	}
	if !u.CheckedAt.IsZero() {
		ui.Text(c, marketText("检查时间：", "Checked: ")+u.CheckedAt.Local().Format("2006-01-02 15:04:05 MST")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Text(c, marketText("仅比较已确认的 npm 稳定版本；Git、本地来源及不可比较版本显示未知。检查不会自动安装。", "Only verified npm stable versions are compared. Git, local sources and incomparable versions remain unknown. Checking never installs automatically.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	batchSelectionView(c, m, u, muted)
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
				current = marketText("未知", "unknown")
			}
			if latest == "" {
				latest = marketText("未知", "unknown")
			}
			status := marketText("未知", "unknown")
			if r.Known {
				status = marketText("没有较新版本", "No newer version")
			}
			if r.Available {
				status = marketText("可更新", "Update available")
			}
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, r.Name+" · "+current+" → "+latest+" · "+status).TextColor(fg)
				if r.Reason != "" {
					ui.Text(c, r.Reason).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			})
			if r.Available && !m.InstallBusy && !u.Loading && u.Err == "" && m.OnInstallCatalog != nil {
				label := marketText("选择", "Select")
				if m.Store.UpdateSelected(r.ID) {
					label = marketText("取消选择", "Deselect")
				}
				if ui.Button(c, label).Clicked() {
					m.Store.ToggleUpdateSelected(r.ID)
				}
				if target, err := m.Store.CaptureUpdateTarget(r.ID); err == nil {
					if ui.Button(c, marketText("更新至 ", "Update to ")+r.Latest).Clicked() {
						m.OnInstallCatalog(target)
					}
				}
			}
		})
	}
}
