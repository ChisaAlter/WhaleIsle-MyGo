// Package settings is the M8 generic harness settings shell.
package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the settings panel to store state.
type Model struct {
	Store      *settings.Store
	ThemeID    string
	SystemDark bool
	OnClose    func()
	OnDraft    func(key, value string)
	// DraftPtr returns a stable *string for editing one field in the selected namespace.
	DraftPtr func(key, seed string) *string
	// BoolPtr returns a stable *bool wired to the same field drafts (bool kinds).
	BoolPtr func(key, seed string) *bool
	// ConfirmReplace, when set, must return true before SaveReplaceFromDrafts runs.
	ConfirmReplace func() bool

	DesktopActive        bool
	Desktop              *DesktopAccess
	AccountActive        bool
	Account              *AccountAccess
	CredentialsActive    bool
	Credentials          *CredentialsAccess
	AboutActive          bool
	About                *AboutAccess
	AppearanceActive     bool
	Appearance           *AppearanceAccess
	InterfaceActive      bool
	Interface            *InterfaceAccess
	ModelsActive         bool
	Models               *ModelsAccess
	MCPActive            bool
	MCP                  *MCPAccess
	PluginsActive        bool
	Plugins              *PluginsAccess
	SkillsActive         bool
	Skills               *SkillsAccess
	AgentPresetsActive   bool
	AgentPresets         *AgentPresetsAccess
	OnSelectDesktop      func()
	OnSelectAccount      func()
	OnSelectCredentials  func()
	OnSelectAbout        func()
	OnSelectAppearance   func()
	OnSelectInterface    func()
	OnSelectModels       func()
	OnSelectMCP          func()
	OnSelectPlugins      func()
	OnSelectSkills       func()
	OnSelectAgentPresets func()
	OnSelectNamespace    func(ns string)
	// Overlay entries (plan §3.2) that leave the settings shell for dedicated panels.
	OnOpenMarket func()
	OnOpenRemote func()
	OnOpenUsage  func()
	// FilterPtr is the left-nav namespace search box (schema namespaces only).
	FilterPtr *string
	// RevealSecrets, when true, shows password-like string fields in the clear.
	RevealSecrets *bool
}

// View paints namespace list + schema-driven form.
func View(c *ui.Context, m Model) {
	if m.Store == nil && m.Desktop == nil {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	var snap settings.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot()
	}

	// Grow under the window title bar. Do not Fill() the nav column — Fill sets
	// width to 100% and leaves a dead gray band beside a squeezed form pane.
	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		header(c, m, snap, fg, muted, border)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			sidebarFill := chrome.Color(colors.Field)
			if colors.Sidebar != "" {
				sidebarFill = chrome.Color(colors.Sidebar)
			}
			nsList(c, m, snap, fg, muted, border, sidebarFill, chrome.Color(colors.Hover))
			ui.Box(c).Width(1).Shrink(0).Background(border)
			formPane(c, m, snap, fg, muted, border, bg)
		})
	})
}

func header(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border ui.Color) {
	ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, i18n.T(i18n.KeySettingsTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
		if snap.Loading {
			ui.Text(c, i18n.T(i18n.KeySettingsLoading)).FontSize(12).TextColor(muted)
		}
		if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeySettingsBack)).Clicked() {
			m.OnClose()
		}
	})
	ui.Box(c).Fill().Height(1).Background(border)
}

func sectionLabel(id SectionID) string {
	switch id {
	case SectionAccount:
		return i18n.T(i18n.KeySettingsSectionAccount)
	case SectionGeneral:
		return i18n.T(i18n.KeySettingsSectionGeneral)
	case SectionAppearance:
		return i18n.T(i18n.KeySettingsSectionAppearance)
	case SectionInterface:
		return i18n.T(i18n.KeySettingsSectionInterface)
	case SectionModels:
		return i18n.T(i18n.KeySettingsSectionModels)
	case SectionPlugins:
		return i18n.T(i18n.KeySettingsSectionPlugins)
	case SectionSkills:
		return i18n.T(i18n.KeySettingsSectionSkills)
	case SectionMarket:
		return i18n.T(i18n.KeySettingsSectionMarket)
	case SectionRemote:
		return i18n.T(i18n.KeySettingsSectionRemote)
	case SectionMCP:
		return i18n.T(i18n.KeySettingsSectionMCP)
	case SectionAgentPresets:
		return i18n.T(i18n.KeySettingsSectionAgentPresets)
	case SectionPet:
		return i18n.T(i18n.KeySettingsSectionPet)
	case SectionAbout:
		return i18n.T(i18n.KeySettingsSectionAbout)
	case SectionUsageStats:
		return i18n.T(i18n.KeySettingsSectionUsageStats)
	case SectionRemoteWorkspace:
		return i18n.T(i18n.KeySettingsSectionRemoteWorkspace)
	case SectionOther:
		return i18n.T(i18n.KeySettingsSectionOther)
	default:
		return string(id)
	}
}

func dedicatedNamespace(ns string) bool {
	switch strings.ToLower(strings.TrimSpace(ns)) {
	case "account", "general", "appearance", "theme", "interface", "ui", "models", "model", "llm",
		"plugins", "plugin", "skills", "skill", "market", "ui-settings-market",
		"remote", "mcp", "agent-presets", "agentpresets", "agentpreset",
		"about", "usage-stats", "usagestats", "usage", "remote-workspace", "pet", "live2d":
		return true
	default:
		return false
	}
}

func settingsNavItem(c *ui.Context, label string, active bool, fg, hover ui.Color, onClick func()) {
	bg := ui.Transparent
	if active {
		bg = hover
		if bg.A == 0 {
			// Light fallback when theme hover has no alpha; dark themes always set Hover.
			bg = chrome.Color("rgb(232, 236, 240)")
		}
	}
	btn := ui.Button(c, label).FillWidth().Height(36).Radius(8).
		BorderWidth(0).Background(bg).TextColor(fg).FontSize(13).
		Padding(0, 10, 0, 10).AlignItems(ui.Center)
	if onClick != nil && btn.Clicked() {
		onClick()
	}
}

func nsList(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, navBG, hover ui.Color) {
	// Electron ui-settings nav column is ~240 DIP.
	ui.Column(c).Width(240).Shrink(0).MinHeight(0).Background(navBG).Padding(8, 8, 8, 8).Gap(2).Children(func() {
		if m.FilterPtr != nil {
			ui.TextInput(c, m.FilterPtr).FillWidth().Placeholder(i18n.T(i18n.KeySettingsFilter))
		}
		if m.Desktop != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsDesktopNav), m.DesktopActive, fg, hover, m.OnSelectDesktop)
		}
		if m.Account != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsAccountNav), m.AccountActive, fg, hover, m.OnSelectAccount)
		}
		if m.Credentials != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsCredNav), m.CredentialsActive, fg, hover, m.OnSelectCredentials)
		}
		// plan §3.2 general — open harness general namespace when present.
		if m.OnSelectNamespace != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionGeneral), false, fg, hover, func() {
				m.OnSelectNamespace("general")
			})
		}
		if m.Appearance != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsAppearanceNav), m.AppearanceActive, fg, hover, m.OnSelectAppearance)
		}
		if m.Interface != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsInterfaceNav), m.InterfaceActive, fg, hover, m.OnSelectInterface)
		}
		if m.Models != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsModelsNav), m.ModelsActive, fg, hover, m.OnSelectModels)
		}
		if m.Plugins != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsPluginsNav), m.PluginsActive, fg, hover, m.OnSelectPlugins)
		}
		if m.Skills != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSkillsNav), m.SkillsActive, fg, hover, m.OnSelectSkills)
		}
		if m.OnOpenMarket != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionMarket), false, fg, hover, m.OnOpenMarket)
		}
		if m.OnOpenRemote != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionRemote), false, fg, hover, m.OnOpenRemote)
		}
		if m.OnSelectNamespace != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionRemoteWorkspace),
				snap.SelectedNS == "remote-workspace", fg, hover, func() {
					m.OnSelectNamespace("remote-workspace")
				})
			petActive := snap.SelectedNS == "pet" || snap.SelectedNS == "live2d"
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionPet), petActive, fg, hover, func() {
				m.OnSelectNamespace("pet")
			})
		}
		if m.AgentPresets != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsAgentPresetsNav), m.AgentPresetsActive, fg, hover, m.OnSelectAgentPresets)
		}
		if m.MCP != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsMCPNav), m.MCPActive, fg, hover, m.OnSelectMCP)
		}
		if m.About != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsAboutNav), m.AboutActive, fg, hover, m.OnSelectAbout)
		}
		if m.OnOpenUsage != nil {
			settingsNavItem(c, i18n.T(i18n.KeySettingsSectionUsageStats), false, fg, hover, m.OnOpenUsage)
		}
		names := make([]string, 0, len(snap.Namespaces))
		for _, ns := range snap.Namespaces {
			// Skip namespaces already covered by dedicated nav / overlay entries.
			if dedicatedNamespace(ns.NS) {
				continue
			}
			names = append(names, ns.NS)
		}
		query := ""
		if m.FilterPtr != nil {
			query = *m.FilterPtr
		}
		filtered := FilterNamespaces(names, query)
		if len(names) == 0 && m.Desktop == nil {
			ui.Text(c, i18n.T(i18n.KeySettingsEmpty)).FontSize(12).TextColor(muted).Padding(4, 4, 4, 4)
		} else if len(names) > 0 && len(filtered) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsFilterEmpty)).FontSize(12).TextColor(muted).Padding(4, 4, 4, 4)
		}
		dedicated := m.DesktopActive || m.AccountActive || m.CredentialsActive || m.AboutActive ||
			m.AppearanceActive || m.InterfaceActive || m.ModelsActive || m.MCPActive ||
			m.PluginsActive || m.SkillsActive || m.AgentPresetsActive
		for _, group := range GroupNamespaces(filtered) {
			ui.Text(c, sectionLabel(group.Section)).FontSize(11).Bold().TextColor(muted).Padding(4, 2, 0, 2)
			for _, ns := range group.Items {
				active := !dedicated && ns == snap.SelectedNS
				nsID := ns
				settingsNavItem(c, ns, active, fg, hover, func() {
					if m.OnSelectNamespace != nil {
						m.OnSelectNamespace(nsID)
					} else if m.Store != nil {
						m.Store.Select(nsID)
					}
				})
			}
		}
		if m.Store != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			m.Store.RefreshDescribe(context.Background())
		}
	})
	_ = border
}

func formPane(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, paneBG ui.Color) {
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Background(paneBG).Padding(12, 16, 12, 16).Gap(10).Children(func() {
		if m.DesktopActive && m.Desktop != nil {
			desktopPane(c, m, fg, muted)
			return
		}
		if m.AccountActive && m.Account != nil {
			accountPane(c, m, fg, muted)
			return
		}
		if m.CredentialsActive && m.Credentials != nil {
			credentialsPane(c, m, fg, muted)
			return
		}
		if m.AppearanceActive && m.Appearance != nil {
			appearancePane(c, m, fg, muted)
			return
		}
		if m.InterfaceActive && m.Interface != nil {
			interfacePane(c, m, fg, muted)
			return
		}
		if m.ModelsActive && m.Models != nil {
			modelsPane(c, m, fg, muted)
			return
		}
		if m.PluginsActive && m.Plugins != nil {
			pluginsPane(c, m, fg, muted)
			return
		}
		if m.SkillsActive && m.Skills != nil {
			skillsPane(c, m, fg, muted)
			return
		}
		if m.AgentPresetsActive && m.AgentPresets != nil {
			agentPresetsPane(c, m, fg, muted)
			return
		}
		if m.MCPActive && m.MCP != nil {
			mcpPane(c, m, fg, muted)
			return
		}
		if m.AboutActive && m.About != nil {
			aboutPane(c, m, fg, muted)
			return
		}
		if snap.Err != "" {
			ui.Text(c, snap.Err).FontSize(12).TextColor(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent))
		}
		if snap.LastAction != "" {
			ui.Text(c, snap.LastAction).FontSize(11).TextColor(muted)
		}
		var ns *settings.Namespace
		for i := range snap.Namespaces {
			if snap.Namespaces[i].NS == snap.SelectedNS {
				ns = &snap.Namespaces[i]
				break
			}
		}
		if ns == nil {
			ui.Text(c, i18n.T(i18n.KeySettingsPickNS)).FontSize(12).TextColor(muted)
			return
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, ns.NS).FontSize(14).Bold().TextColor(fg).Grow(1)
			ui.Text(c, fmt.Sprintf("%s %g", i18n.T(i18n.KeySettingsRevision), ns.Revision)).FontSize(11).TextColor(muted)
			if snap.Writable {
				ui.Text(c, i18n.T(i18n.KeySettingsWritable)).FontSize(11).Bold().TextColor(fg)
			} else {
				accent := chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
				ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).FontSize(11).Bold().TextColor(accent)
			}
		})
		if m.RevealSecrets != nil {
			hasSecret := false
			for _, f := range ns.Fields {
				if f.Kind == settings.FieldString && (SecretLikeKey(f.Key) || SecretLikeKey(f.Label)) {
					hasSecret = true
					break
				}
			}
			if hasSecret {
				ui.Checkbox(c, m.RevealSecrets, i18n.T(i18n.KeySettingsRevealSecrets))
			}
		}
		for _, f := range ns.Fields {
			fieldRow(c, m, snap, f, fg, muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			save := ui.Button(c, i18n.T(i18n.KeySettingsSave))
			if snap.Saving || !snap.Writable || snap.SelectedNS == "" {
				save = save.Disabled(true)
			}
			if save.Clicked() {
				m.Store.SaveUpdate(context.Background())
			}
			replace := ui.Button(c, i18n.T(i18n.KeySettingsReplace))
			if snap.Saving || !snap.Writable || snap.SelectedNS == "" {
				replace = replace.Disabled(true)
			}
			if replace.Clicked() {
				RequestReplace(m)
			}
		})
		ui.Text(c, i18n.T(i18n.KeySettingsReplaceHint)).FontSize(11).TextColor(muted)
	})
	_ = border
}

// RequestReplace confirms (optional) then posts a full settings/replace from drafts.
func RequestReplace(m Model) {
	if m.Store == nil {
		return
	}
	if m.ConfirmReplace != nil && !m.ConfirmReplace() {
		return
	}
	m.Store.SaveReplaceFromDrafts(context.Background())
}

func fieldRow(c *ui.Context, m Model, snap settings.ViewState, f settings.Field, fg, muted ui.Color) {
	if m.DraftPtr == nil {
		return
	}
	ui.Column(c).Gap(4).Children(func() {
		label := f.Label
		if f.Required {
			label += " *"
		}
		ui.Text(c, label).FontSize(12).TextColor(fg)
		if f.Description != "" {
			ui.Text(c, f.Description).FontSize(11).TextColor(muted)
		}
		seed := snap.Drafts[f.Key]
		switch f.Kind {
		case settings.FieldBool:
			if m.BoolPtr == nil {
				return
			}
			checked := m.BoolPtr(f.Key, seed)
			box := ui.Checkbox(c, checked, "")
			if box.Changed() && m.OnDraft != nil {
				if *checked {
					m.OnDraft(f.Key, "true")
				} else {
					m.OnDraft(f.Key, "false")
				}
			}
		case settings.FieldEnum:
			draft := m.DraftPtr(f.Key, seed)
			sel := ui.Select(c, draft, f.EnumOptions)
			if sel.Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		case settings.FieldJSON:
			draft := m.DraftPtr(f.Key, seed)
			area := ui.TextArea(c, draft).Fill().Lines(4, 12)
			if area.Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		case settings.FieldNumber:
			draft := m.DraftPtr(f.Key, seed)
			inp := ui.TextInput(c, draft).Fill()
			if inp.Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		default:
			draft := m.DraftPtr(f.Key, seed)
			if SecretLikeKey(f.Key) || SecretLikeKey(f.Label) {
				inp := ui.TextInput(c, draft).Fill()
				reveal := m.RevealSecrets != nil && *m.RevealSecrets
				if !reveal {
					inp = inp.Password()
				}
				if inp.Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
				break
			}
			if MultilineStringField(f) || strings.Contains(seed, "\n") || strings.Contains(*draft, "\n") {
				area := ui.TextArea(c, draft).Fill().Lines(3, 10)
				if area.Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
				break
			}
			inp := ui.TextInput(c, draft).Fill()
			if inp.Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		}
	})
}
