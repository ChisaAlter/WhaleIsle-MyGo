package settings

import (
	"context"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// navigationEntry is one feature's place in the original settings rail.
type navigationEntry struct {
	id       SectionID
	icon     string
	onSelect func()
}

// navigationGroup is one labelled block of the rail. The installed 0.3.13 rail
// is grouped (账户 / 通用 / 智能体 / 功能 / 其他) rather than one flat list.
type navigationGroup struct {
	label   string
	entries []navigationEntry
}

func activeSection(m Model, snap store.ViewState) (active SectionID) {
	defer func() {
		entries := primaryNavigation(m, snap)
		for _, entry := range entries {
			if entry.id == active {
				return
			}
		}
		if len(entries) > 0 {
			active = entries[0].id
		} else {
			active = SectionGeneral
		}
	}()
	if m.ActiveSection != "" && m.SectionContent != nil {
		return m.ActiveSection
	}
	for _, item := range []struct {
		active bool
		id     SectionID
	}{
		{m.DesktopActive, SectionGeneral}, {m.AccountActive, SectionAccount},
		{m.CredentialsActive, SectionGeneral}, {m.AppearanceActive, SectionAppearance},
		{m.InterfaceActive, SectionInterface}, {m.ConversationActive, SectionConversation},
		{m.FeaturesActive, SectionFeatures}, {m.ModelsActive, SectionModels},
		{m.PluginsActive, SectionPlugins}, {m.SkillsActive, SectionSkills},
		{m.AgentPresetsActive, SectionAgentPresets}, {m.MCPActive, SectionMCP}, {m.AboutActive, SectionAbout},
	} {
		if item.active {
			return item.id
		}
	}
	switch snap.SelectedNS {
	case "pet", "live2d":
		return SectionPet
	}
	return SectionGeneral
}

// primaryNavigation is the feature rail flattened in reading order.
func primaryNavigation(m Model, snap store.ViewState) []navigationEntry {
	var entries []navigationEntry
	for _, group := range navigationGroups(m, snap) {
		entries = append(entries, group.entries...)
	}
	return entries
}

// navigationGroups is the feature rail as the installed build lays it out.
// Implementation namespaces such as remote-workspace are configuration sections
// inside a page, never top-level destinations.
func navigationGroups(m Model, snap store.ViewState) []navigationGroup {
	var account, general, agents, features, other []navigationEntry
	add := func(into *[]navigationEntry, id SectionID, icon string, enabled bool, selectPage func()) {
		if enabled {
			*into = append(*into, navigationEntry{id, icon, selectPage})
		}
	}
	selectNamespace := func(ns string) func() {
		return func() {
			if m.OnSelectNamespace != nil {
				m.OnSelectNamespace(ns)
			} else if m.Store != nil {
				m.Store.Select(ns)
			}
		}
	}
	selectGeneral := m.OnSelectDesktop
	if m.Desktop == nil || selectGeneral == nil {
		selectGeneral = selectNamespace(generalNamespace(snap))
	}
	// The installed account contribution registers only after credentials are
	// stored. An explicitly opened sign-in flow keeps its dedicated page, while
	// the existing account menu/General sign-in entry remains available unsigned.
	add(&account, SectionAccount, "SettingsOutline", m.Account != nil && (m.Account.Status == "credential-stored" || m.AccountActive), m.OnSelectAccount)
	add(&account, SectionUsageStats, "ChartOutline", m.OnOpenUsage != nil, m.OnOpenUsage)
	add(&general, SectionGeneral, "SettingsOutline", true, selectGeneral)
	add(&general, SectionAppearance, "LightOutline", m.Appearance != nil, m.OnSelectAppearance)
	add(&general, SectionInterface, "PanelLeftOutline", m.Interface != nil, m.OnSelectInterface)
	add(&general, SectionConversation, "BrowseOutline", m.Conversation != nil, m.OnSelectConversation)
	add(&agents, SectionModels, "DataOutline", m.Models != nil, m.OnSelectModels)
	add(&agents, SectionAgentPresets, "AgentPresetOutline", m.AgentPresets != nil, m.OnSelectAgentPresets)
	add(&agents, SectionSkills, "SkillOutline", m.Skills != nil, m.OnSelectSkills)
	add(&agents, SectionMCP, "ServerOutline", m.MCP != nil, m.OnSelectMCP)
	add(&agents, SectionPlugins, "PersonalizationOutline", m.Plugins != nil, m.OnSelectPlugins)
	add(&agents, SectionMarket, "BrowseOutline", m.OnOpenMarket != nil, m.OnOpenMarket)
	add(&features, SectionFeatures, "PersonalizationOutline", m.Features != nil, m.OnSelectFeatures)
	add(&features, SectionPet, "SettingsOutline", m.Pet != nil, selectNamespace("pet"))
	add(&features, SectionRemote, "DeviceOutline", m.OnOpenRemote != nil, m.OnOpenRemote)
	add(&other, SectionAbout, "InfoOutline", m.About != nil, m.OnSelectAbout)
	var groups []navigationGroup
	for _, group := range []navigationGroup{
		{i18n.T(i18n.KeySettingsNavGroupAccount), account},
		{i18n.T(i18n.KeySettingsNavGroupGeneral), general},
		{i18n.T(i18n.KeySettingsNavGroupAgents), agents},
		{i18n.T(i18n.KeySettingsNavGroupFeatures), features},
		{i18n.T(i18n.KeySettingsNavGroupOther), other},
	} {
		if len(group.entries) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// generalNavigation paints the General page's own rows: the account entry, then
// the configuration browser folded under Advanced. The browser used to lead the
// page as a button strip plus a namespace dropdown, which read as an internal
// tool rather than as settings.
func generalNavigation(c *ui.Context, m Model, snap store.ViewState, fg, muted ui.Color) {
	if m.Account != nil && m.Account.Status != "credential-stored" && m.OnSelectAccount != nil {
		bodyPreferenceRow(c, "general-account", i18n.T(i18n.KeySettingsSectionAccount), "", fg, muted, func() {
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountSignIn)).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Clicked() {
				m.OnSelectAccount()
			}
		})
	}
	bodyDisclosure(c, "settings-advanced", i18n.T(i18n.KeySurfacesSettingsAdvanced), func() {
		advancedNavigation(c, m, snap, fg, muted)
	})
}

// advancedNavigation is the namespace browser: the backed desktop and credential
// pages, the live filter, the namespace chooser and a describe refresh.
func advancedNavigation(c *ui.Context, m Model, snap store.ViewState, fg, muted ui.Color) {
	ui.Text(c, i18n.T(i18n.KeySurfacesSettingsDevHint)).FillWidth().FontSize(11).FixedLineHeight(16).TextColor(muted)
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if m.Desktop != nil && m.OnSelectDesktop != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopNav)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
			m.OnSelectDesktop()
		}
		if m.OnSelectNamespace != nil && ui.Button(c, sectionLabel(SectionGeneral)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
			m.OnSelectNamespace(generalNamespace(snap))
		}
		if m.Credentials != nil && m.OnSelectCredentials != nil && ui.Button(c, i18n.T(i18n.KeySettingsCredNav)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
			m.OnSelectCredentials()
		}
	})
	if m.Store == nil {
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
		if m.FilterPtr != nil {
			ui.TextInput(c, m.FilterPtr).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Placeholder(i18n.T(i18n.KeySurfacesSettingsFilter))
		}
		names := []string{}
		for _, ns := range snap.Namespaces {
			if !dedicatedNamespace(ns.NS) || ns.NS == "general" {
				names = append(names, ns.NS)
			}
		}
		query := ""
		if m.FilterPtr != nil {
			query = *m.FilterPtr
		}
		filtered := FilterNamespaces(names, query)
		if len(filtered) > 0 {
			labels := make([]string, len(filtered))
			selected := ""
			for index, ns := range filtered {
				labels[index] = sectionLabel(SectionForNamespace(ns)) + " · " + ns
				if ns == snap.SelectedNS {
					selected = labels[index]
				}
			}
			ui.Select(c, &selected, labels).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Label(i18n.T(i18n.KeySurfacesSettingsNamespace)).OnChange(func() {
				for index, label := range labels {
					if label != selected {
						continue
					}
					namespace := filtered[index]
					if m.OnSelectNamespace != nil {
						m.OnSelectNamespace(namespace)
					} else {
						m.Store.Select(namespace)
					}
					break
				}
			})
		} else if len(names) > 0 {
			ui.Text(c, i18n.T(i18n.KeySurfacesSettingsFilterEmpty)).FontSize(11).TextColor(muted)
		}
		if ui.Button(c, i18n.T(i18n.KeySurfacesSettingsRefresh)).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
			m.Store.RefreshDescribe(context.Background())
		}
	})
}
