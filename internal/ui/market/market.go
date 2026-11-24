// Package market is the native plugin market surface: catalog discovery with
// search, categories, details and install, plus installed-plugin management
// (enable, disable, batch update, remove).
package market

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/errcopy"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
	"whaleisle/internal/ui/theme"
)

// Model wires the market panel to store, install state, and harness status.
type Model struct {
	Media *media.Owner
	Store *marketstore.Store
	// Embedded is used by the Settings market section, whose shell owns its header and insets.
	Embedded           bool
	categoriesExpanded *bool
	ThemeID            string
	SystemDark         bool
	HarnessReady       bool
	MuxConnected       bool
	PluginDisabled     bool
	InstallReady       bool
	InstallSpec        *string
	InstallBusy        bool
	InstallMessage     string
	Activation         marketstore.Activation
	CanRestart         bool
	OnRestart          func()
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
	// OnOpenManager opens package installation/configuration in Plugin Manager.
	OnOpenManager func()
	// OnOpenURL activates an authored documentation link after an explicit click.
	OnOpenURL func(string)
}

// page is one frame's reading of the market store and host model, with its
// media watch and store/media epochs for memoized subtrees.
type page struct {
	m                  Model
	snap               marketstore.ViewState
	snapRev            uint64
	d                  marketstore.Discovery
	discRev            uint64
	personal           marketstore.Personal
	updates            marketstore.Updates
	filter             string
	watch              *pagekit.MediaWatch
	categoriesExpanded *bool
	epoch              uint64
	mediaEpoch         uint64
}

type storeEpoch struct {
	last *marketstore.Store
	n    uint64
}

type mediaEpoch struct {
	last *media.Owner
	n    uint64
}

func testPage(m Model, d marketstore.Discovery, snap marketstore.ViewState) *page {
	p := &page{
		m:                  m,
		snap:               snap,
		d:                  d,
		categoriesExpanded: m.categoriesExpanded,
	}
	if m.Store != nil {
		p.personal = m.Store.Personal()
		p.updates = m.Store.Updates()
		p.filter = m.Store.ManagementFilter()
	}
	return p
}

// version is the memo version of regions rendering the store's catalog,
// installed list, updates or history.
func (p *page) version() uint64 {
	v := pagekit.NewVersion().
		Uint(p.snapRev).
		Uint(p.discRev).
		Uint(p.epoch).
		Uint(p.mediaEpoch).
		Uint(design.Revision()).
		Uint(theme.Revision()).
		Uint(p.watch.Epoch()).
		String(p.filter).
		String(p.personal.Err).
		Uint(uint64(len(p.personal.History))).
		Bool(p.m.Embedded).
		Bool(p.m.InstallBusy).
		Bool(p.m.InstallReady).
		Bool(p.m.OnOpenURL != nil).
		Bool(p.m.OnInstallCatalog != nil).
		Bool(p.m.OnRemoveBundle != nil).
		Bool(p.m.OnSetBundleEnabled != nil).
		Bool(p.m.OnBatchUpdate != nil).
		Bool(p.m.OnStopBatch != nil).
		Bool(p.m.OnExportHistory != nil).
		Bool(p.m.OnOpenManager != nil)
	if p.m.InstallSpec != nil {
		v = v.String(strings.TrimSpace(*p.m.InstallSpec))
	}
	v = v.Bool(p.updates.Loading).Uint(uint64(p.updates.CheckedAt.UnixNano())).String(p.updates.Err)
	for _, r := range p.updates.Rows {
		v = v.String(r.ID).String(r.Name).String(r.Current).String(r.Latest).String(r.Spec).String(r.Reason).String(r.Kind).Bool(r.Known).Bool(r.Available)
		if p.m.Store != nil {
			v = v.Bool(p.m.Store.UpdateSelected(r.ID))
		}
	}
	return v.Key()
}

// View is a native discovery and installed-plugin surface.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	m = withMarketConfirmations(m)
	snapVer := m.Store.Snapshot()
	discVer := m.Store.DiscoveryVersion()
	personal := m.Store.Personal()
	updates := m.Store.Updates()
	filter := m.Store.ManagementFilter()

	root := ui.Column(c).Key("market-page").FillWidth().MinWidth(0)
	if !m.Embedded {
		root.Grow(1).MinHeight(0)
	}
	expanded := ui.Local(root, "market-categories-expanded", func() bool { return false })
	m.categoriesExpanded = expanded
	epoch := ui.Local(root, "store-epoch", func() storeEpoch { return storeEpoch{} })
	if epoch.last != m.Store {
		epoch.last, epoch.n = m.Store, epoch.n+1
	}
	watch := ui.Local(root, "media-watch", func() *pagekit.MediaWatch { return &pagekit.MediaWatch{} })
	lease := ui.Local(root, "media-epoch", func() mediaEpoch { return mediaEpoch{} })
	if lease.last != m.Media {
		lease.last, lease.n = m.Media, lease.n+1
		// Handles the released owner was waiting on stay Pending forever;
		// a fresh watch stops counting the dead ones.
		*watch = &pagekit.MediaWatch{}
	}
	scroll := ui.Local(root, "scroll", func() *ui.ScrollState { return &ui.ScrollState{} })

	p := &page{
		m:                  m,
		snap:               snapVer.Value,
		snapRev:            snapVer.Rev,
		d:                  discVer.Value,
		discRev:            discVer.Rev,
		personal:           personal,
		updates:            updates,
		filter:             filter,
		watch:              *watch,
		categoriesExpanded: expanded,
		epoch:              epoch.n,
		mediaEpoch:         lease.n,
	}

	root.Children(func() {
		var pageScroll *ui.ScrollState
		if !m.Embedded {
			pageScroll = *scroll
		}
		pagekit.Page(c, pagekit.PageOptions{MaxWidth: 800, Embedded: m.Embedded, Scroll: pageScroll}, func() {
			marketHeader(c, p)
			marketTabs(c, p)
			if personal.Err != "" {
				renderPersonalError(c, personal.Err)
			}
			switch p.d.Tab {
			case "history":
				historyView(c, p)
			case "installed":
				installedTabView(c, p)
			default:
				discoveryView(c, p)
			}
			if m.InstallMessage != "" {
				renderInstallMessage(c, m.InstallMessage)
			}
			activationNotice(c, m)
			allowBuildsNotice(c, p)
			incompatibleNotices(c, m)
			installLogPanel(c, m)
		})
	})
	catalogDetailsDialog(c, p)
	confirmationView(c, p)
}

func marketText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func renderPersonalError(c *ui.Context, raw string) {
	switch raw {
	case "Could not read local favorites/history; existing data was not overwritten.":
		pagekit.Note(c, coss.AlertWarning, marketText("无法读取本地收藏与操作记录；未覆盖现有数据。", raw), "", nil)
	case "Could not save local favorites/history; this change will not survive restart.":
		pagekit.Note(c, coss.AlertWarning, marketText("无法保存本地收藏与操作记录；本次变更在重启后不会保留。", raw), "", nil)
	default:
		pagekit.Fault(c, "", raw, "", nil)
	}
}

func renderInstallMessage(c *ui.Context, raw string) {
	if text, variant, ok := authoredMarketMessage(raw); ok {
		pagekit.Note(c, variant, text, "", nil)
		return
	}
	pagekit.Fault(c, "", raw, "", nil)
}

// authoredMarketMessage recognizes authored outcomes from internal/app/market
// so raw errors never reach the screen.
func authoredMarketMessage(raw string) (string, coss.AlertVariant, bool) {
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return "", 0, false
	case strings.HasPrefix(trimmed, "Applied;") || strings.HasPrefix(trimmed, "已应用；") ||
		strings.HasPrefix(trimmed, "Saved; a runtime restart") || strings.HasPrefix(trimmed, "已保存，需重启运行时"):
		return trimmed, coss.AlertSuccess, true
	case strings.HasPrefix(trimmed, "Installation cancelled.") || strings.HasPrefix(trimmed, "安装已取消。") ||
		strings.HasPrefix(trimmed, "Cleared queued batch items") || strings.HasPrefix(trimmed, "已取消待执行的批量项目") ||
		strings.HasPrefix(trimmed, "Cancelled; waiting for installation") || strings.HasPrefix(trimmed, "已取消，等待安装请求"):
		return trimmed, coss.AlertInfo, true
	case strings.HasPrefix(trimmed, "Operation failed.") || strings.HasPrefix(trimmed, "操作失败。") ||
		strings.HasPrefix(trimmed, "The runtime is not connected") || strings.HasPrefix(trimmed, "运行时未连接") ||
		strings.HasPrefix(trimmed, "The installer is busy") || strings.HasPrefix(trimmed, "安装器正忙") ||
		strings.HasPrefix(trimmed, "The catalog is refreshing") || strings.HasPrefix(trimmed, "正在刷新插件目录") ||
		strings.HasPrefix(trimmed, "Build approval expired") || strings.HasPrefix(trimmed, "构建授权已过期") ||
		strings.HasPrefix(trimmed, "Cannot save operation receipt") || strings.HasPrefix(trimmed, "无法保存操作记录") ||
		strings.HasPrefix(trimmed, "Cannot save batch receipts") || strings.HasPrefix(trimmed, "无法保存批量操作记录") ||
		strings.HasPrefix(trimmed, "Select updates with verified versions") || strings.HasPrefix(trimmed, "请选择已确认版本的更新") ||
		strings.HasPrefix(trimmed, "This change is overridden") || strings.HasPrefix(trimmed, "此变更被更高优先级配置覆盖") ||
		strings.HasPrefix(trimmed, "Result unconfirmed") || strings.HasPrefix(trimmed, "结果未确认") ||
		strings.HasPrefix(trimmed, "Already applying") || strings.HasPrefix(trimmed, "已进入应用阶段") ||
		strings.HasPrefix(trimmed, "No active installation") || strings.HasPrefix(trimmed, "无活动安装请求") ||
		strings.HasPrefix(trimmed, "中断记录保存失败"):
		return trimmed, coss.AlertWarning, true
	case strings.Contains(trimmed, "Explicit build-script approval") || strings.Contains(trimmed, "需要明确授权以下构建脚本") ||
		strings.Contains(trimmed, "Batch stopped") || strings.Contains(trimmed, "批量更新已停止") ||
		strings.Contains(trimmed, "Batch paused") || strings.Contains(trimmed, "批量更新暂停") ||
		strings.Contains(trimmed, "Result could not be saved") || strings.Contains(trimmed, "结果保存失败"):
		return trimmed, coss.AlertWarning, true
	}
	return "", 0, false
}

func activationNotice(c *ui.Context, m Model) {
	if m.Activation.Pending {
		variant := coss.AlertInfo
		text := marketText(i18n.T(i18n.KeyW4MarketPluginChangesAreSavedAndTakeEffectAft), "Plugin changes are saved and await a runtime restart.")
		if m.Activation.Failed {
			variant = coss.AlertWarning
			text = marketText(i18n.T(i18n.KeyW4MarketTheRestartDidNotSucceedAndChangesStil), "Restart failed. Saved changes still await activation; do not install again.")
		}
		if m.Activation.Restarting {
			text = marketText(i18n.T(i18n.KeyW4MarketRestartingTheRuntimeAndWaitingForTheN), "Restarting; waiting for the new Host to become ready…")
		}
		var action func()
		if m.CanRestart && !m.InstallBusy && !m.Activation.Restarting && m.OnRestart != nil {
			action = func() {
				if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketRestartRuntimeAndApply), "Restart runtime and apply"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconRefresh}).Clicked() {
					m.OnRestart()
				}
			}
		}
		pagekit.Note(c, variant, text, "", action)
	} else if m.Activation.Refreshed {
		pagekit.Note(c, coss.AlertSuccess, marketText(i18n.T(i18n.KeyW4MarketTheNewRuntimeIsReadyCheckTheLoadStat), "The new runtime is ready. Check the plugin list for loading status."), "", nil)
	}
}

func allowBuildsNotice(c *ui.Context, p *page) {
	m := p.m
	if !m.NeedsAllowBuilds || len(m.AllowBuilds) == 0 {
		return
	}
	var action func()
	if (p.d.Tab != "installed" || m.Embedded) && !m.InstallBusy && m.InstallReady && m.OnInstallSpec != nil && m.InstallSpec != nil {
		action = func() {
			if coss.Button(c, i18n.T(i18n.KeyMarketAllowBuildsButton), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				m.OnInstallSpec(*m.InstallSpec, append([]string(nil), m.AllowBuilds...))
			}
		}
	}
	pagekit.Note(c, coss.AlertWarning, i18n.T(i18n.KeyMarketAllowBuilds), strings.Join(m.AllowBuilds, ", "), action)
}

func incompatibleNotices(c *ui.Context, m Model) {
	for _, item := range m.Incompatible {
		it := item
		var action func()
		if !m.InstallBusy && m.OnAcceptExemption != nil && it.Name != "" && it.Version != "" {
			action = func() {
				if coss.Button(c, i18n.T(i18n.KeyMarketAcceptExemption), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
					m.OnAcceptExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
				}
			}
		}
		pagekit.Note(c, coss.AlertWarning, it.Name+"@"+it.Version, fmt.Sprintf(marketText("运行时 %s", "runtime %s"), it.RuntimeVersion), action)
	}
}

func installLogPanel(c *ui.Context, m Model) {
	log := strings.TrimSpace(m.InstallLog)
	if log == "" {
		return
	}
	t := design.For(c)
	logBox := ui.Column(c.Key("market-install-log")).FillWidth().MinWidth(0).Gap(design.Sp(2))
	logOpen := ui.Local(logBox, "open", func() bool { return false })
	logBox.Children(func() {
		if strings.Contains(strings.ToLower(log), "operation-error") {
			pagekit.Note(c, coss.AlertWarning, marketText("安装未完成。请检查网络或安装源，或展开日志排查。", "Installation did not finish. Check the network or source, or open the log to inspect it."), "", nil)
		}
		chevron := coss.IconChevronDown
		if *logOpen {
			chevron = coss.IconChevronUp
		}
		if coss.Button(c, marketText("安装日志", "Install log"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, TrailingIcon: chevron}).AlignSelf(ui.Start).Margin(0, 0, 0, -design.Sp(2)).Clicked() {
			*logOpen = !*logOpen
		}
		if !*logOpen {
			return
		}
		if m.OnCopyInstallLog != nil && coss.Button(c, i18n.T(i18n.KeyMarketCopyLog), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconCopy}).AlignSelf(ui.Start).Clicked() {
			m.OnCopyInstallLog(log)
		}
		lines := strings.Split(log, "\n")
		if len(lines) > 40 {
			lines = lines[len(lines)-40:]
		}
		ui.Column(c).FillWidth().MaxHeight(design.Sp(45)).Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Background(t.Card).Padding(design.Sp(3)).Children(func() {
			coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
				pagekit.Code(c, strings.Join(lines, "\n")).FillWidth().Selectable() // verbatim-diagnostics
			})
		})
	})
}

func historyView(c *ui.Context, p *page) {
	m := p.m
	personal := p.personal
	if m.OnExportHistory != nil {
		if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketExportInstallHistory), "Export installation history"), coss.ButtonOptions{
			Variant:  coss.ButtonOutline,
			Size:     coss.ButtonSM,
			Icon:     coss.IconDownload,
			Disabled: len(personal.History) == 0,
		}).AlignSelf(ui.Start).Clicked() {
			m.OnExportHistory()
		}
	}
	pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketTheLast200OperationsInterruptedOrUnkno), "Latest 200 operations. Interrupted or unknown means unconfirmed; refresh inventory before retrying."))
	if len(personal.History) == 0 {
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconHistory, Title: marketText(i18n.T(i18n.KeyW4MarketNoInstallHistoryYet), "No installation history yet.")}, nil)
	}
	managementFilters(c, p)
	version := pagekit.Version(p.version())
	for _, h := range personal.History {
		version = version.Uint(uint64(h.At.UnixNano())).String(h.Plugin).String(h.Result).String(h.Action).String(h.Error).String(h.Diagnostic).Uint(uint64(len(h.Log)))
	}
	ui.Column(c).Key("market-history-list").FillWidth().MinWidth(0).Gap(design.Sp(2)).Memo(version.Key()).Children(func() {
		for _, h := range personal.History {
			if p.filter == "failed" && !failedOutcome(h.Result) {
				continue
			}
			if p.filter == "updates" && h.Action != "update" {
				continue
			}
			marketHistoryRow(c, h)
		}
	})
}

func installedTabView(c *ui.Context, p *page) {
	m, snap := p.m, p.snap
	if snap.Err != nil {
		pagekit.FaultSentence(c, marketText("已安装插件列表加载失败", "Could not load installed plugins"), snap.Err.SentenceText(), marketText("重试", "Retry"), func() {
			if m.OnRefresh != nil {
				m.OnRefresh()
			} else {
				m.Store.Refresh(context.Background())
			}
		})
	}
	if !snap.ClientReady {
		pagekit.Note(c, coss.AlertWarning, marketText("运行时未连接", "Runtime disconnected"), marketText(i18n.T(i18n.KeyW4MarketTheRuntimeIsNotConnectedTheCatalogIs), "Runtime disconnected; the catalog remains available."), nil)
	}
	managementFilters(c, p)
	updatesView(c, p)
	if len(desktopInstalledRows(snap.Rows)) == 0 {
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconPuzzle, Title: i18n.T(i18n.KeyMarketListEmpty)}, nil)
	}
	installedRows(c, p)
	if m.Embedded {
		if coss.Button(c, marketText("管理插件包", "Manage plugin packages"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: m.OnOpenManager == nil}).AlignSelf(ui.Start).Clicked() && m.OnOpenManager != nil {
			m.OnOpenManager()
		}
	} else {
		installSection(c, p)
	}
}

// marketHistoryRow is one install-history receipt: the plugin and result lead,
// the timestamp and action follow as secondary lines.
func marketHistoryRow(c *ui.Context, h marketstore.InstallHistory) {
	t := design.For(c)
	result, badgeVariant := marketHistoryResult(h.Result)
	action := marketHistoryAction(h.Action)
	row := ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Padding(design.Sp(3), design.Sp(4)).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card)
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			pagekit.Strong(c, h.Plugin).Grow(1).MinWidth(0).SingleLine()
			coss.Badge(c, result, coss.BadgeOptions{Variant: badgeVariant})
		})
		pagekit.Caption(c, h.At.Local().Format("2006-01-02 15:04")+" · "+action)
		if h.Error != "" && h.Error != h.Result {
			pagekit.Caption(c, historyErrorText(h.Error)).FillWidth().MinWidth(0).SingleLine()
		}
		if h.Diagnostic != "" {
			if msg, _, ok := authoredMarketMessage(h.Diagnostic); ok {
				pagekit.Caption(c, msg).FillWidth().MinWidth(0)
			} else {
				pagekit.Caption(c, errcopy.SentenceText(h.Diagnostic)).FillWidth().MinWidth(0)
			}
		}
		if h.Log != "" {
			open := ui.Local(row, "history-log-open", func() bool { return false })
			chevron := coss.IconChevronDown
			if *open {
				chevron = coss.IconChevronUp
			}
			if coss.Button(c, marketText("操作日志", "Operation log"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, TrailingIcon: chevron}).AlignSelf(ui.Start).Margin(0, 0, 0, -design.Sp(2)).Clicked() {
				*open = !*open
			}
			if *open {
				if coss.Button(c, marketText("复制日志", "Copy log"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconCopy}).AlignSelf(ui.Start).Clicked() {
					c.WriteClipboard(h.Log)
				}
				pagekit.Code(c, h.Log).FillWidth().MinWidth(0).Selectable() // verbatim-diagnostics
			}
		}
	})
}

func historyErrorText(code string) string {
	switch code {
	case "operation-failed":
		return marketText("操作失败", "Operation failed")
	case "result-unconfirmed":
		return marketText("结果未确认", "Result unconfirmed")
	default:
		return errcopy.SentenceText(code)
	}
}

func marketHistoryResult(result string) (string, coss.BadgeVariant) {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "applied", "applied-restart", "applied_restart":
		return marketText("已应用", "Applied"), coss.BadgeSuccess
	case "failed":
		return marketText("失败", "Failed"), coss.BadgeError
	case "interrupted":
		return marketText("已中断", "Interrupted"), coss.BadgeError
	case "not-started":
		return marketText("未启动", "Not started"), coss.BadgeError
	case "unknown":
		return marketText("结果未知", "Unknown"), coss.BadgeError
	default:
		return result, coss.BadgeOutline
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

func discoveryView(c *ui.Context, p *page) {
	m, d := p.m, p.d
	// A catalog that has never been fetched is loading, not empty: printing the
	// count and the two source fields from a zero value reads as "0 plugins from
	// nowhere", which is a different statement from "we are still reading".
	loaded := !d.Catalog.FetchedAt.IsZero()
	if !loaded && d.Loading {
		pagekit.Caption(c, marketText("正在加载插件目录…", "Loading the plugin catalog…"))
		pagekit.SkeletonRows(c, 3, marketText("正在加载插件目录…", "Loading the plugin catalog…"))
	} else {
		source := strings.TrimSpace(d.Catalog.Source)
		if source == "" {
			source = marketText("未知", "unknown")
		}
		updated := strings.TrimSpace(d.Catalog.Updated)
		if updated == "" {
			updated = "—"
		}
		pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketCatalogSource), "Catalog source: ")+source+" · "+marketText(i18n.T(i18n.KeyW4MarketUpdated), "updated: ")+updated)
		if !d.Catalog.FetchedAt.IsZero() {
			pagekit.Caption(c, d.Catalog.FetchedAt.Format("2006-01-02 15:04 MST"))
		}
	}
	if d.Catalog.Warning != "" {
		renderCatalogWarning(c, d.Catalog.Warning)
	}
	if d.Err != nil {
		pagekit.FaultSentence(c, marketText("插件目录加载失败", "Could not load the plugin catalog"), d.Err.SentenceText(), marketText("重试", "Retry"), func() {
			m.Store.RefreshCatalog(true, i18n.Locale())
		})
	}
	query := d.Query
	placeholder := marketText(i18n.T(i18n.KeyW4MarketSearchNamePackageOrDescription), "Search name, package or description")
	if coss.InputGroup(c, &query, coss.InputGroupOptions{
		InputOptions: coss.InputOptions{
			Label:       placeholder,
			Placeholder: placeholder,
			Key:         "market-search",
		},
		Start: func() { coss.AddonIcon(c, coss.IconSearch) },
	}).Changed() {
		m.Store.SetQuery(query)
	}
	catalogCategories(c, p)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketSort), "Sort"))
		sortOptions := []coss.SelectOption{
			{Value: "catalog", Label: marketText(i18n.T(i18n.KeyW4MarketCatalogOrder), "Catalog order")},
			{Value: "stars", Label: marketText(i18n.T(i18n.KeyW4MarketMostStars), "Most stars")},
			{Value: "newest", Label: marketText(i18n.T(i18n.KeyW4MarketNewest), "Newest")},
		}
		sortVal := d.Sort
		if sortVal == "" {
			sortVal = "catalog"
		}
		if coss.Select(c, &sortVal, sortOptions, coss.SelectOptions{
			Size:  coss.InputSM,
			Label: marketText(i18n.T(i18n.KeyW4MarketSort), "Sort"),
			Key:   "market-sort",
		}).Width(180).MaxWidthPercent(100).Changed() {
			m.Store.SetSort(sortVal)
		}

		pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketPublished), "Listed"))
		periodOptions := []coss.SelectOption{
			{Value: "all", Label: marketText(i18n.T(i18n.KeyW4MarketAnyTime), "All time")},
			{Value: "7", Label: marketText(i18n.T(i18n.KeyW4MarketLast7Days), "Last 7 days")},
			{Value: "30", Label: marketText(i18n.T(i18n.KeyW4MarketLast30Days), "Last 30 days")},
		}
		periodVal := d.Period
		if periodVal == "" {
			periodVal = "all"
		}
		if coss.Select(c, &periodVal, periodOptions, coss.SelectOptions{
			Size:  coss.InputSM,
			Label: marketText(i18n.T(i18n.KeyW4MarketPublished), "Listed"),
			Key:   "market-period",
		}).Width(180).MaxWidthPercent(100).Changed() {
			m.Store.SetPeriod(periodVal)
		}

		compatLabel := marketText("仅原生可用", "Native compatible only")
		allLabel := marketText("全部插件", "All plugins")
		compatTooltip := marketText("默认隐藏仅支持 Web UI 的插件；切换到“全部插件”可查看全部条目。", "Hide Web UI-only plugins by default; switch to All plugins to browse every entry.")
		compatIdx := 1
		if d.NativeCompatibleOnly {
			compatIdx = 0
		}
		if coss.SegmentedControl(c, &compatIdx, []string{compatLabel, allLabel}, coss.SegmentedOptions{
			Size:  coss.SegmentedSM,
			Label: marketText("兼容性筛选", "Compatibility filter"),
			Key:   "market-compat",
		}).Tooltip(compatTooltip).Changed() {
			m.Store.SetNativeCompatibleOnly(compatIdx == 0)
		}
	})
	if loaded || d.Total > 0 {
		pagekit.Caption(c, fmt.Sprintf(marketText(i18n.T(i18n.KeyW4MarketDPlugins), "%d plugins"), d.Total))
	}
	catalogCards(c, p)
	if len(d.Rows) < d.Total {
		ui.Row(c).FillWidth().MinWidth(0).Justify(ui.Center).Children(func() {
			if coss.Button(c, marketText("显示更多", "Show more"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				m.Store.ShowMore()
			}
		})
	}
}

func renderCatalogWarning(c *ui.Context, warning string) {
	switch {
	case warning == "Using a cached catalog (less than one hour old).":
		pagekit.Note(c, coss.AlertInfo, marketText("正在使用 1 小时内的缓存目录。", warning), "", nil)
	case warning == "Catalog loaded; local cache could not be saved.":
		pagekit.Note(c, coss.AlertWarning, marketText("目录已加载，但本地缓存保存失败。", warning), "", nil)
	case strings.HasPrefix(warning, "Catalog refresh failed: "):
		raw := strings.TrimPrefix(warning, "Catalog refresh failed: ")
		pagekit.Fault(c, marketText("目录刷新失败，正在使用缓存或离线快照", "Catalog refresh failed; using cached or offline catalog"), raw, "", nil)
	default:
		pagekit.Fault(c, "", warning, "", nil)
	}
}

func catalogInstallButton(c *ui.Context, p *page, item desktopmarket.CatalogItem) {
	m, d, inventory := p.m, p.d, p.snap
	if name := marketstore.InstalledName(item, inventory.Rows); name != "" {
		coss.Badge(c, marketText(i18n.T(i18n.KeyW4MarketInstalled), "Installed"), coss.BadgeOptions{Variant: coss.BadgeSuccess})
		ready := inventory.ClientReady && !inventory.Loading && inventory.Err == nil && !m.InstallBusy && !d.Loading
		for _, update := range p.updates.Rows {
			if update.ID == item.ID && update.Available && m.OnInstallCatalog != nil {
				current, latest := update.Current, update.Latest
				if update.Kind == "github" {
					current, latest = current[:min(7, len(current))], latest[:min(7, len(latest))]
				}
				pagekit.Caption(c, current+" → "+latest).MaxWidthPercent(100).MinWidth(0).SingleLine().Tooltip(update.Current + " → " + update.Latest)
				label := marketText("更新", "Update")
				if m.InstallBusy && m.InstallSpec != nil && strings.TrimSpace(*m.InstallSpec) == update.Spec {
					label = marketText("正在更新…", "Updating…")
				}
				if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Icon: coss.IconDownload, Disabled: !ready}).Clicked() {
					if target, err := m.Store.CaptureUpdateTarget(item.ID); err == nil {
						m.OnInstallCatalog(target)
					}
				}
				break
			}
		}
		for _, row := range inventory.Rows {
			if row.Label == name && row.IdentityKnown && row.Removable && row.ReadOnlyReason == "" && m.OnRemoveBundle != nil {
				if coss.Button(c, marketText("卸载", "Uninstall"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: !ready}).Clicked() {
					m.OnRemoveBundle(name)
				}
				break
			}
		}
		return
	}
	if !desktopmarket.ValidInstallItem(item) {
		pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketNoSupportedInstallTarget), "No supported install target."))
		return
	}
	if !d.RuntimeReady {
		pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketConnectTheRuntimeBeforeInstalling), "Connect the runtime to install."))
		return
	}
	if !m.InstallBusy && !d.Loading && m.OnInstallCatalog != nil && coss.Button(c, i18n.T(i18n.KeyMarketInstallButton), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Icon: coss.IconDownload}).Clicked() {
		if target, e := m.Store.CaptureTarget(item.ID); e == nil {
			m.OnInstallCatalog(target)
		}
	}
}

func installSection(c *ui.Context, p *page) {
	m := p.m
	t := design.For(c)
	var allow []string
	if m.NeedsAllowBuilds {
		allow = append([]string(nil), m.AllowBuilds...)
	}
	btnLabel := i18n.T(i18n.KeyMarketInstallButton)
	if m.NeedsAllowBuilds {
		btnLabel = i18n.T(i18n.KeyMarketAllowBuildsButton)
	}
	var notReady string
	if !m.InstallReady {
		notReady = i18n.T(i18n.KeyMarketInstallNotReady)
	}
	ui.Column(c).FillWidth().MinWidth(0).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Padding(design.Sp(4)).Gap(design.Sp(2)).Children(func() {
		coss.Field(c, coss.FieldOptions{
			Label:       i18n.T(i18n.KeyMarketInstallSpecLabel),
			Description: notReady,
		}, func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				if m.InstallSpec != nil {
					var in ui.Element
					ui.Row(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
						in = coss.Input(c, m.InstallSpec, coss.InputOptions{
							Placeholder: i18n.T(i18n.KeyMarketInstallPlaceholder),
							Disabled:    m.InstallBusy || m.NeedsAllowBuilds,
							Key:         "market-install-spec",
						})
					})
					if in.Changed() {
						// *m.InstallSpec is applied by Changed().
					}
					if !m.InstallBusy {
						spec := strings.TrimSpace(*m.InstallSpec)
						if coss.Button(c, btnLabel, coss.ButtonOptions{
							Variant:  coss.ButtonOutline,
							Size:     coss.ButtonSM,
							Disabled: spec == "" || !m.InstallReady || m.OnInstallSpec == nil,
						}).Shrink(0).Clicked() && m.OnInstallSpec != nil && spec != "" {
							m.OnInstallSpec(spec, allow)
						}
					}
				} else if !m.InstallBusy {
					coss.Button(c, btnLabel, coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Size:     coss.ButtonSM,
						Disabled: true,
					}).Shrink(0)
				}
			})
		})
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

func favoriteButton(c *ui.Context, p *page, id string) {
	label := marketText(i18n.T(i18n.KeyW4MarketSave), "Favorite")
	if p.personal.Favorites[id] {
		label = marketText(i18n.T(i18n.KeyW4MarketRemoveFromSaved), "Unfavorite")
	}
	if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM}).Clicked() {
		_ = p.m.Store.ToggleFavorite(id)
	}
}

// updatesView reuses catalog install receipts, so updates share build approval,
// cancellation, runtime guards, and installation history with normal installs.
func updatesView(c *ui.Context, p *page) {
	m, inventory, u := p.m, p.snap, p.updates
	var checkAction func()
	if !m.InstallBusy && !inventory.Loading && inventory.Err == nil && inventory.ClientReady && !p.d.Loading && !u.Loading {
		checkAction = func() {
			if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketCheckPluginUpdates), "Check plugin updates"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconRefresh}).Clicked() {
				m.Store.CheckUpdates()
			}
		}
	}
	pagekit.Section(c, marketText(i18n.T(i18n.KeyW4MarketPluginVersionUpdates), "Plugin version updates"), checkAction, func() {
		if u.Err != "" {
			if u.Err == "Update check interrupted; check again" {
				pagekit.Note(c, coss.AlertWarning, marketText("检查更新已中断，请重试。", u.Err), "", nil)
			} else {
				pagekit.Fault(c, "", u.Err, "", nil)
			}
		}
		if !u.CheckedAt.IsZero() {
			pagekit.Caption(c, marketText(i18n.T(i18n.KeyW4MarketCheckedAt), "Checked: ")+u.CheckedAt.Local().Format("2006-01-02 15:04:05 MST"))
		}
		pagekit.Caption(c, marketText("比较 npm 语义版本与已确认 GitHub 提交。检查不会自动安装。", "Compare npm semantic versions and verified GitHub commits. Checking never installs automatically."))
		batchSelectionView(c, p)

		var visible []marketstore.UpdateRow
		for _, row := range u.Rows {
			if p.filter == "updates" && !row.Available {
				continue
			}
			if p.filter == "failed" && !failedPlugin(m, row.ID, row.Name) {
				continue
			}
			visible = append(visible, row)
		}
		if len(visible) == 0 {
			return
		}
		version := p.version()
		ui.Column(c).Key("market-updates-list").FillWidth().MinWidth(0).Memo(version).Children(func() {
			rowsCard(c, func() {
				for i, row := range visible {
					r := row
					if i > 0 {
						rowSeparator(c)
					}
					rowFrame(ui.Row(c.Key("market-update-" + r.Name))).Wrap().Children(func() {
						current, latest := r.Current, r.Latest
						if r.Kind == "github" {
							if len(current) > 7 {
								current = current[:7]
							}
							if len(latest) > 7 {
								latest = latest[:7]
							}
						}
						if current == "" {
							current = marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
						}
						if latest == "" {
							latest = marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
						}
						status := marketText(i18n.T(i18n.KeyW4MarketUnknown), "unknown")
						badgeVariant := coss.BadgeOutline
						if r.Known {
							status = marketText(i18n.T(i18n.KeyW4MarketNoNewerVersion), "No newer version")
							badgeVariant = coss.BadgeSecondary
						}
						if r.Available {
							status = marketText(i18n.T(i18n.KeyW4MarketUpdateAvailable), "Update available")
							badgeVariant = coss.BadgeInfo
						}
						ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
							ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								pagekit.Strong(c, r.Name+" · "+current+" → "+latest)
								coss.Badge(c, status, coss.BadgeOptions{Variant: badgeVariant})
							})
							if r.Reason != "" {
								pagekit.Caption(c, updateReasonText(r.Reason))
							}
						})
						if r.Available && !m.InstallBusy && !u.Loading && u.Err == "" && m.OnInstallCatalog != nil {
							ui.Row(c).Shrink(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								label := marketText(i18n.T(i18n.KeyW4MarketSelect), "Select")
								if m.Store.UpdateSelected(r.ID) {
									label = marketText(i18n.T(i18n.KeyW4MarketClearSelection), "Deselect")
								}
								if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
									m.Store.ToggleUpdateSelected(r.ID)
								}
								if target, err := m.Store.CaptureUpdateTarget(r.ID); err == nil {
									if coss.Button(c, marketText(i18n.T(i18n.KeyW4MarketUpdateTo), "Update to ")+r.Latest, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Icon: coss.IconDownload}).Clicked() {
										m.OnInstallCatalog(target)
									}
								}
							})
						}
					})
				}
			})
		})
	})
}

func updateReasonText(raw string) string {
	switch raw {
	case "No verified npm update source":
		return marketText("没有已验证的 npm 更新源", raw)
	case "Version check failed; retry":
		return marketText("版本检查失败，请重试", raw)
	case "Version comparison unavailable":
		return marketText("无法比较版本", raw)
	case "Commit check failed; retry":
		return marketText("提交检查失败，请重试", raw)
	case "Installed commit is unknown or ambiguous":
		return marketText("已安装提交未知或存在歧义", raw)
	default:
		return raw
	}
}
