package main

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	marketctl "whaleisle/internal/app/market"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	manager "whaleisle/internal/pluginmanager"
	marketstore "whaleisle/internal/store/market"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	managerui "whaleisle/internal/ui/pluginmanager"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) invalidateMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}

func (s *mainState) marketInstallReadyFor(nodeBin string) bool {
	return s.marketCoordinator().InstallReadyFor(nodeBin)
}

func (s *mainState) nativePluginController() *manager.Controller {
	return s.marketCoordinator().NativeController()
}

func (s *mainState) bindNativePluginManager(client *dsh.Client) {
	s.marketCoordinator().BindNativePluginManager(client)
}

func (s *mainState) nativePluginEventHandler(client *dsh.Client) func(dsh.EventsEmit) {
	return s.marketCoordinator().NativeEventHandler(client)
}

func (s *mainState) attachPluginInventory(client *dsh.Client) {
	s.marketCoordinator().AttachInventory(client)
}

func (s *mainState) refreshPluginInventory() {
	s.marketCoordinator().RefreshInventory()
}

func (s *mainState) parkPluginViewForSettings(v bool) {
	s.marketCoordinator().ParkPluginViewForSettings(v)
}

func (s *mainState) inspectInstalledPlugin(name string) {
	s.marketCoordinator().InspectInstalledPlugin(name)
}

func (s *mainState) restartMarketPlugins() {
	s.marketCoordinator().RestartPlugins()
}

func (s *mainState) observeMarketActivation(st harness.Status) {
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	var actual *harness.Status
	if sup != nil {
		a := sup.Status()
		actual = &a
	}
	s.marketCoordinator().ObserveActivation(st, actual)
}

func (s *mainState) runMarketBatchUpdates(targets []marketstore.InstallTarget) {
	s.marketCoordinator().RunBatchUpdates(targets)
}

func (s *mainState) stopMarketBatch() {
	s.marketCoordinator().StopBatch()
}

func (s *mainState) runMarketCatalogInstall(target marketstore.InstallTarget) {
	s.marketCoordinator().RunCatalogInstall(target)
}

func (s *mainState) runMarketInstallForOwner(spec string, allowBuilds []string, owner *marketctl.Owner) {
	s.marketCoordinator().RunInstallForOwner(spec, allowBuilds, owner)
}

func (s *mainState) runMarketChange(method string, args map[string]any) {
	s.marketCoordinator().RunChange(method, args)
}

func (s *mainState) runMarketChangeForOwner(method string, args map[string]any, owner *marketctl.Owner) {
	s.marketCoordinator().RunChangeForOwner(method, args, owner)
}

func (s *mainState) acceptMarketExemption(packageVersion, runtimeVersion string) {
	s.marketCoordinator().AcceptExemption(packageVersion, runtimeVersion)
}

func (s *mainState) cancelMarketInstall() {
	s.marketCoordinator().CancelInstall()
}

func (s *mainState) pluginInventoryAccess() *settings.PluginInventoryAccess {
	inv := s.marketCoordinator().Inventory()
	snap := inv.Snapshot().Value
	return &settings.PluginInventoryAccess{
		Snapshot:       snap,
		OnRefresh:      s.refreshPluginInventory,
		OnQuery:        func(query string) { inv.SetQuery(snap.Host, query); s.invalidateMainWindow() },
		OnSelectPreset: func(id string) { inv.SelectPreset(snap.Host, id); s.invalidateMainWindow() },
		OnToggleRow: func(key string) {
			inv.ToggleExpanded(snap.Host, key)
			s.invalidateMainWindow()
		},
		OnToggleGlobal: func() { inv.ToggleGlobal(snap.Host); s.invalidateMainWindow() },
	}
}

type pluginContributionsCache struct {
	store      *settingsstore.Store
	locale     string
	themeID    string
	systemDark bool
	nsMask     uint8
	items      []managerui.Contribution
}

func pluginContributionNamespaceMask(store *settingsstore.Store) uint8 {
	if store == nil {
		return 0
	}
	var mask uint8
	for _, ns := range store.Snapshot().Value.Namespaces {
		switch ns.NS {
		case "subagent":
			mask |= 1 << 0
		case "subagent-model-selection-settings":
			mask |= 1 << 1
		case "web-search-deepseek":
			mask |= 1 << 2
		case "web":
			mask |= 1 << 3
		case "web-deepseek-api":
			mask |= 1 << 4
		case "agent-loop":
			mask |= 1 << 5
		case "pwsh-sandbox":
			mask |= 1 << 6
		case "bash-sandbox":
			mask |= 1 << 7
		}
	}
	return mask
}

func (s *mainState) cachedPluginContributions(store *settingsstore.Store, themeID string, systemDark bool) []managerui.Contribution {
	if store == nil {
		return nil
	}
	locale := i18n.Locale()
	mask := pluginContributionNamespaceMask(store)
	s.mu.Lock()
	cached := s.pluginContribCache
	s.mu.Unlock()
	if cached.store == store && cached.locale == locale && cached.themeID == themeID && cached.systemDark == systemDark && cached.nsMask == mask {
		return cached.items
	}
	served := map[string]bool{
		"subagent":                          mask&(1<<0) != 0,
		"subagent-model-selection-settings": mask&(1<<1) != 0,
		"web-search-deepseek":               mask&(1<<2) != 0,
		"web":                               mask&(1<<3) != 0,
		"web-deepseek-api":                  mask&(1<<4) != 0,
		"agent-loop":                        mask&(1<<5) != 0,
		"pwsh-sandbox":                      mask&(1<<6) != 0,
		"bash-sandbox":                      mask&(1<<7) != 0,
	}
	var contributions []managerui.Contribution
	// Titles and summaries are the original plugins.item registrations'
	// label and the card's view:'summary' text (ui-settings-*/locales.ts).
	add := func(key, zh, en, summaryZh, summaryEn string, order int, namespaces ...string) {
		active := []string{}
		for _, ns := range namespaces {
			if served[ns] {
				active = append(active, ns)
			}
		}
		if len(active) == 0 {
			return
		}
		title, summary := en, summaryEn
		if i18n.Speaks("zh") {
			title, summary = zh, summaryZh
		}
		contributions = append(contributions, managerui.Contribution{Kind: "item", Key: key, Title: title, Summary: summary, Order: order, Render: func(c *ui.Context, _ manager.Subject) {
			if settings.ServedPluginBody(c, settings.Model{Store: store, ThemeID: themeID, SystemDark: systemDark}, key) {
				return
			}
			for _, ns := range active {
				settings.NamespaceBody(c, settings.Model{Store: store, ThemeID: themeID, SystemDark: systemDark}, ns)
			}
		}})
	}
	add("subagent", "子智能体", "Subagent", "设置子智能体的递归层级、数量和模型。", "Set Subagent recursion depth, count, and models.", 10, "subagent", "subagent-model-selection-settings")
	if served["web-search-deepseek"] {
		add("web-search", "网页搜索与获取", "Web search and fetch", "选择网页搜索与网页获取使用的提供方。", "Choose the providers web search and web fetch use.", 20, "web-search-deepseek", "web", "web-deepseek-api")
	}
	add("agent-loop", "Agent 循环", "Agent loop", "控制 Agent 派发工具调用的方式。", "Control how the Agent dispatches tool calls.", 30, "agent-loop")
	if served["pwsh-sandbox"] {
		add("shell", "终端", "Shell", "限制每条命令最多能跑多久、最多输出多少内容。", "Limit how long each command may run and how much it may output.", 40, "pwsh-sandbox")
	} else {
		add("shell", "终端", "Shell", "限制每条命令最多能跑多久、最多输出多少内容。", "Limit how long each command may run and how much it may output.", 40, "bash-sandbox")
	}
	s.mu.Lock()
	s.pluginContribCache = pluginContributionsCache{
		store:      store,
		locale:     locale,
		themeID:    themeID,
		systemDark: systemDark,
		nsMask:     mask,
		items:      contributions,
	}
	s.mu.Unlock()
	return contributions
}

func (s *mainState) pluginManagerView(c *ui.Context, themeID string) {
	creatorReady, creatorDescription, createPlugin := s.pluginCreatorAccess()
	s.mu.Lock()
	store := s.settings
	sessions, win := s.sessions, s.win
	s.mu.Unlock()
	ctrl := s.nativePluginController()
	snap := ctrl.Snapshot().Value
	if snap.Connected && snap.Discover.Probe == "" {
		// Gates the Installed/Discover SegmentedTabs the way the original
		// gates on a plugins.tab slot contribution; the catalog itself
		// still lazy-loads when the Discover tab opens.
		ctrl.ProbeDiscover(context.Background(), snap.Owner)
	}
	view := snap.View
	if s.marketCoordinator().ObserveFormSubject(view) && store != nil {
		store.LeaveServedForms()
	}
	systemDark := mygo.Theme.IsDark()
	contributions := s.cachedPluginContributions(store, themeID, systemDark)
	managerui.View(c, managerui.Model{
		Media:                   s.mediaOwner(win, sessions),
		Controller:              ctrl,
		ThemeID:                 themeID,
		SystemDark:              systemDark,
		Contributions:           contributions,
		OnCreatePlugin:          createPlugin,
		CreatePluginReady:       creatorReady,
		CreatePluginDescription: creatorDescription,
		OnOpenAddMenu:           s.refreshPluginCreatorRoster,
		CreatorNotice:           s.renderPluginCreatorNotice,
		RenderConfiguration: func(c *ui.Context, subject manager.Subject) bool {
			return subject.Kind == "bundle" && s.renderVoiceDeviceSettings(c, subject.Name)
		},
		OnOpenURL: func(raw string) { _ = mygo.Shell.OpenExternal(raw) },
	})
}
