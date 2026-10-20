package settings

import (
	"context"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/accountlogin"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// navigationEntry is one feature's place in the original settings rail.
type navigationEntry struct {
	id       SectionID
	icon     string
	onSelect func()
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
		{m.InterfaceActive, SectionInterface}, {m.ModelsActive, SectionModels},
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

// primaryNavigation is the feature rail, in the original order. Implementation
// namespaces such as remote-workspace are configuration sections inside General,
// never top-level destinations.
func primaryNavigation(m Model, snap store.ViewState) []navigationEntry {
	var entries []navigationEntry
	add := func(id SectionID, icon string, enabled bool, selectPage func()) {
		if enabled {
			entries = append(entries, navigationEntry{id, icon, selectPage})
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
	general := m.OnSelectDesktop
	if m.Desktop == nil || general == nil {
		general = selectNamespace(generalNamespace(snap))
	}
	accountVisible := m.Account != nil && (m.Account.Status == "credential-stored" || m.AccountActive || m.Account.SignInBusy || accountlogin.Active(m.Account.AttemptPhase))
	add(SectionAccount, "SettingsOutline", accountVisible, m.OnSelectAccount)
	add(SectionGeneral, "SettingsOutline", true, general)
	add(SectionAppearance, "LightOutline", m.Appearance != nil, m.OnSelectAppearance)
	add(SectionInterface, "PanelLeftOutline", m.Interface != nil, m.OnSelectInterface)
	add(SectionModels, "DataOutline", m.Models != nil, m.OnSelectModels)
	add(SectionPlugins, "PersonalizationOutline", m.Plugins != nil, m.OnSelectPlugins)
	add(SectionSkills, "SkillOutline", m.Skills != nil, m.OnSelectSkills)
	add(SectionMarket, "BrowseOutline", m.OnOpenMarket != nil, m.OnOpenMarket)
	add(SectionMCP, "ServerOutline", m.MCP != nil, m.OnSelectMCP)
	add(SectionRemote, "DeviceOutline", m.OnOpenRemote != nil, m.OnOpenRemote)
	add(SectionAgentPresets, "AgentPresetOutline", m.AgentPresets != nil, m.OnSelectAgentPresets)
	add(SectionUsageStats, "ChartOutline", m.OnOpenUsage != nil, m.OnOpenUsage)
	add(SectionPet, "SettingsOutline", m.Pet != nil, selectNamespace("pet"))
	add(SectionAbout, "InfoOutline", m.About != nil, m.OnSelectAbout)
	return entries
}

// generalNavigation paints the General page's own rows: the account entry, then
// the configuration browser folded under Advanced. The browser used to lead the
// page as a button strip plus a namespace dropdown, which read as an internal
// tool rather than as settings.
func generalNavigation(c *ui.Context, m Model, snap store.ViewState, fg, muted ui.Color) {
	if m.Account != nil && m.Account.Status != "credential-stored" && m.OnSelectAccount != nil {
		bodyPreferenceRow(c, "general-account", i18n.T(i18n.KeySettingsSectionAccount), "", fg, muted, func() {
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountSignIn)).Clicked() {
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
		if m.Desktop != nil && m.OnSelectDesktop != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopNav)).Clicked() {
			m.OnSelectDesktop()
		}
		if m.OnSelectNamespace != nil && ui.Button(c, sectionLabel(SectionGeneral)).Clicked() {
			m.OnSelectNamespace(generalNamespace(snap))
		}
		if m.Credentials != nil && m.OnSelectCredentials != nil && ui.Button(c, i18n.T(i18n.KeySettingsCredNav)).Clicked() {
			m.OnSelectCredentials()
		}
	})
	if m.Store == nil {
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
		if m.FilterPtr != nil {
			ui.TextInput(c, m.FilterPtr).FillWidth().Placeholder(i18n.T(i18n.KeySurfacesSettingsFilter))
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
			ui.Select(c, &selected, labels).FillWidth().Label(i18n.T(i18n.KeySurfacesSettingsNamespace)).OnChange(func() {
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
		if ui.Button(c, i18n.T(i18n.KeySurfacesSettingsRefresh)).Clicked() {
			m.Store.RefreshDescribe(context.Background())
		}
	})
}
