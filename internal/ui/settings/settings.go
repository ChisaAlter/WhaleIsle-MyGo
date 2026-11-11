// Package settings is the M8 generic harness settings shell.
package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Model wires the settings panel to store state.
type Model struct {
	PublicContributions pluginui.PublicContributions
	// Optional feature-owned section body, composed by the application.
	ActiveSection       SectionID
	SectionContent      func(*ui.Context)
	ContributedSections []ContributedSection
	Pet                 *pet.Companion
	PetSettings         *PetSettingsAccess
	Store               *settings.Store
	ThemeID             string
	SystemDark          bool
	OnClose             func()
	// OnOpenConfig opens the configuration file in the host editor; nil renders
	// the header action disabled with an explanation.
	OnOpenConfig func()
	OnDraft      func(key, value string)
	// DraftPtr returns a stable *string for editing one field in the selected namespace.
	DraftPtr func(key, seed string) *string
	// BoolPtr returns a stable *bool wired to the same field drafts (bool kinds).
	BoolPtr func(key, seed string) *bool
	// ConfirmReplace is the legacy synchronous callback for existing callers.
	ConfirmReplace func() bool
	// OnRequestReplace presents a controlled confirmation and invokes confirm
	// only on acceptance. The candidate is captured before presentation.
	OnRequestReplace func(confirm func())

	DesktopActive         bool
	Desktop               *DesktopAccess
	AccountActive         bool
	Account               *AccountAccess
	CredentialsActive     bool
	Credentials           *CredentialsAccess
	ReferencedCredentials *ReferencedCredentialsAccess
	AboutActive           bool
	About                 *AboutAccess
	AppearanceActive      bool
	Appearance            *AppearanceAccess
	InterfaceActive       bool
	Interface             *InterfaceAccess
	ConversationActive    bool
	Conversation          *ConversationAccess
	FeaturesActive        bool
	Features              *FeaturesAccess
	ModelsActive          bool
	Models                *ModelsAccess
	MCPActive             bool
	MCP                   *MCPAccess
	PluginsActive         bool
	Plugins               *PluginsAccess
	SkillsActive          bool
	Skills                *SkillsAccess
	AgentPresetsActive    bool
	AgentPresets          *AgentPresetsAccess
	RemoteActive          bool
	Remote                *RemoteAccess
	OnSelectRemote        func()
	OnSelectDesktop       func()
	OnSelectAccount       func()
	OnSignIn              func()
	OnSelectCredentials   func()
	OnSelectAbout         func()
	OnSelectAppearance    func()
	OnSelectInterface     func()
	OnSelectConversation  func()
	OnSelectFeatures      func()
	OnSelectModels        func()
	OnSelectMCP           func()
	OnSelectPlugins       func()
	OnSelectSkills        func()
	OnSelectAgentPresets  func()
	OnSelectNamespace     func(ns string)
	// Overlay entries (plan §3.2) that leave the settings shell for dedicated panels.
	OnOpenMarket          func()
	OnOpenRemote          func()
	OnOpenRemoteWorkspace func()
	OnOpenUsage           func()
	// FilterPtr is the left-nav namespace search box (schema namespaces only).
	FilterPtr *string
	// RevealSecrets, when true, shows password-like string fields in the clear.
	RevealSecrets *bool
}

// View paints namespace list + schema-driven form.
func View(c *ui.Context, m Model) {
	if m.Store == nil && m.Desktop == nil && m.SectionContent == nil && m.Remote == nil {
		return
	}
	finishPreferenceErrors := beginBodyPreferenceErrorNotices(c)
	defer finishPreferenceErrors()
	if close := m.OnClose; close != nil {
		m.OnClose = func() { ClearPreferenceErrorNotices(c); close() }
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	// The modal owns the single bg-layer-2 glass surface; its layout children
	// are transparent so they cannot flatten the sampled backdrop.
	bg := ui.Transparent
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	var snap settings.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot()
	}

	// The rail and the options column scroll independently inside one panel.
	// SettingsRoot.module.css: the 188 DIP rail directly precedes content;
	// the options' own 24 DIP inset puts its copy at panel.X + 212.
	root := ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).Gap(0).Background(bg).AlignItems(ui.Stretch)
	query := ui.Local(root, "settings-search", func() string { return "" })
	root.Children(func() {
		nsList(c, m, snap, fg, muted, border, bg, chrome.Color(colors.Hover), query)
		ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			header(c, m, snap, fg, muted, border)
			formPane(c, m, snap, fg, muted, border, bg, query)
		})
	})
}

func header(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border ui.Color) {
	ui.Row(c).FillWidth().Height(54).Shrink(0).Padding(20, 14, 8, 10).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Grow(1).MinWidth(0)
		renderSettingsHeaderActions(c, m, func() {
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			configLabel := i18n.T(i18n.KeySurfacesSettingsOpenConfig)
			configButton := widgets.Button(c, colors, configLabel, widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: m.OnOpenConfig == nil})
			if m.OnOpenConfig == nil {
				configButton.Tooltip(i18n.T(KeyPass4AConfigUnavailable))
			} else if configButton.Clicked() {
				m.OnOpenConfig()
			}
		})
		if m.OnClose != nil {
			closeLabel := i18n.T(i18n.KeySettingsBack)
			if m.PublicContributions.Entries != nil {
				for _, entry := range m.PublicContributions.Entries("settings.close", "") {
					if entry.Label != "" {
						closeLabel = entry.Label
						break
					}
				}
			}
			close := ui.Button(c, "").Label(closeLabel).Size(28, 28).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent)
			close.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(14, 14).TextColor(fg) })
			if close.Clicked() {
				*settingsSearchJumpState(c) = settingsSearchJump{}
				m.OnClose()
			}
		}
	})
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
	case SectionConversation:
		return i18n.T(i18n.KeySettingsSectionConversation)
	case SectionFeatures:
		return i18n.T(i18n.KeySettingsSectionFeatures)
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
	case SectionAgentLoop:
		return providerCopy("智能体循环", "Agent loop")
	case SectionWebSearch:
		return providerCopy("网页搜索", "Web search")
	case SectionSubagent:
		return providerCopy("子智能体", "Subagents")
	case SectionShell:
		return providerCopy("命令执行", "Shell")
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
		"about", "usage-stats", "usagestats", "usage", "pet", "live2d":
		return true
	default:
		return false
	}
}

// settingsSelectedRailColors follows the source's active-accent wash and its
// 65% brand / 35% primary-label text. Explicit family token overrides win.
func settingsSelectedRailColors(m Model) (fill, ink ui.Color) {
	resolved := theme.ResolveWithSystem(m.ThemeID, m.SystemDark)
	colors := resolved.Active(m.SystemDark)
	preferences := theme.RenderPreferences()
	familyID := preferences.Light
	dark := resolved.Scheme == theme.SchemeDark
	if dark {
		familyID = preferences.Dark
	}
	family := preferences.Family(familyID)
	seed := family.Light
	if dark {
		seed = family.Dark
	}
	// design-platform.css's default half: deepseek-100 and
	// neutral-bluish-800; the brand-new dark token is deepseek-450.
	// 0.3.13: the active cell is a neutral wash (light rgba(38,49,72,.10), dark
	// rgba(255,255,255,.14)) and keeps the label colour.
	brand, fill := ui.RGB(65, 118, 230), ui.RGBA(38, 49, 72, .10)
	if dark {
		brand, fill = ui.RGB(86, 134, 254), ui.RGBA(255, 255, 255, .14)
	}
	if family.ID != "deepseek" {
		// ui-theme/derive.js: mixUntilContrast(background, accent,
		// max(washStrong, wash+.08), layer2, 1.4), in .04 steps.
		base, foreground, accent := chrome.Color(seed.Background), chrome.Color(seed.Foreground), chrome.Color(seed.Accent)
		factor := max(0, min(float32(seed.ContrastValue()/100), 1))
		wash, strong, layerFG, layerAccent := .10+factor*.08, .18+factor*.10, .02+factor*.02, .07+factor*.04
		if chrome.RelativeLuminance(base) < chrome.RelativeLuminance(foreground) {
			wash, strong, layerFG, layerAccent = .14+factor*.10, .26+factor*.12, .05+factor*.05, .12+factor*.06
		}
		layer := base.Mix(foreground, layerFG).Mix(accent, layerAccent)
		amount := max(strong, wash+.08)
		fill = base.Mix(accent, amount)
		for chrome.ContrastRatio(fill, layer) < 1.4 && amount < 1 {
			amount = min(1, amount+.04)
			fill = base.Mix(accent, amount)
		}
		brand = accent
	}
	if value := seed.Overrides["--dsw-alias-brand-primary-new-colorprimary-new-color"]; value != "" {
		brand = chrome.Color(value)
	}
	if value := seed.Overrides["--dsw-specific-sidebar-nav-item-active-accent"]; value != "" {
		fill = chrome.Color(value)
	}
	_ = brand
	return fill, chrome.Color(colors.FG)
}

func effectiveActiveSection(c *ui.Context, m Model, snap settings.ViewState) SectionID {
	base := activeSection(m, snap)
	if m.ActiveSection != "" && m.SectionContent != nil {
		return base
	}
	jump := settingsSearchJumpState(c)
	if jump.Section == "" {
		return base
	}
	if base != jump.BaseActive && base != jump.Section {
		*jump = settingsSearchJump{}
		return base
	}
	for _, entry := range primaryNavigation(m, snap) {
		if entry.id == jump.Section {
			return jump.Section
		}
	}
	return base
}

func nsList(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, navBG, hover ui.Color, query *string) {
	active := effectiveActiveSection(c, m, snap)
	searching := strings.TrimSpace(*query) != ""
	selectedBG, selectedFG := settingsSelectedRailColors(m)
	groups := navigationGroups(m, snap)
	ui.Column(c).Key("settings-rail").Width(188).Shrink(0).MinHeight(0).Padding(22, 12, 0, 12).Gap(18).Children(func() {
		if m.PublicContributions.Slot != nil {
			m.PublicContributions.Slot(c, "settings.header", "", func() {
				ui.Text(c, i18n.T(i18n.KeySettingsTitle)).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg).Padding(0, 12, 0, 12)
			})
		} else {
			ui.Text(c, i18n.T(i18n.KeySettingsTitle)).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg).Padding(0, 12, 0, 12)
		}
		settingsRailSearch(c, m, snap, query, fg, muted, border)
		// navGroup: a 20 DIP label, 2 DIP, then 36 DIP cells 2 DIP apart; groups
		// sit 12 DIP from each other.
		ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Gap(12).Children(func() {
			for _, group := range groups {
				ui.Column(c.Key("settings-group-" + group.label)).FillWidth().MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, group.label).FillWidth().Padding(0, 12, 2, 12).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
					for _, entry := range group.entries {
						// The rail keeps no selection while the search lists results.
						selected := entry.id == active && !searching
						fill, ink := ui.Transparent, fg
						if selected {
							fill, ink = selectedBG, selectedFG
						}
						label := navigationLabel(m, entry.id)
						button := ui.Button(c.Key(string(entry.id)), "").Label(label).FillWidth().Height(36).Shrink(0).Radius(12).BorderWidth(0).Background(fill).TextColor(fg).Padding(7, 16, 7, 12).Gap(8).AlignItems(ui.Center).FocusRing(false)
						button = button.Transition(ui.ElementTransition{Duration: chrome.PopoverDuration, Colors: true})
						button.Children(func() {
							ui.Icon(c, settingsIcons[entry.icon]).Size(16, 16).Shrink(0).TextColor(ink)
							weight := 400
							if selected {
								weight = 500
							}
							ui.Text(c, label).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(weight).TextColor(ink).Grow(1).MinWidth(0).SingleLine()
						})
						if button.Hovered() && !selected {
							button.Background(hover)
						}
						if button.Clicked() {
							*query = ""
							*settingsSearchJumpState(c) = settingsSearchJump{}
							if entry.onSelect != nil {
								entry.onSelect()
							}
						}
					}
				})
			}
		})
	})
}

// settingsRailSearch is the 164x32 search field above the groups: a 12 DIP
// pill with a hairline border, the magnifier at +10, the text from +32 and a
// clear button once there is text.
func settingsRailSearch(c *ui.Context, m Model, snap settings.ViewState, query *string, fg, muted, border ui.Color) {
	ui.Row(c.Key("settings-search")).FillWidth().Height(32).Shrink(0).Radius(12).Border(0.667, border).AlignItems(ui.Center).Padding(0, 6, 0, 10).Gap(6).Children(func() {
		ui.Icon(c, bodyIcon("search")).Size(16, 16).Shrink(0).TextColor(muted)
		input := ui.TextInput(c, query).Grow(1).MinWidth(0).Height(30).Padding(0).Radius(0).BorderWidth(0).Background(ui.Transparent).FontSize(theme.InterfaceSize(13)).TextColor(fg).Label(i18n.T(i18n.KeySettingsNavSearch)).Placeholder(i18n.T(i18n.KeySettingsNavSearch)).OnSubmit(func() {
			ids := map[SectionID]bool{}
			for _, entry := range primaryNavigation(m, snap) {
				ids[entry.id] = true
			}
			if matches := settingsSearchMatches(m, ids, *query); len(matches) > 0 {
				activateSettingsSearchEntry(c, m, snap, matches[0], func() { *query = "" })
			}
		})
		if input.Changed() {
			*settingsSearchJumpState(c) = settingsSearchJump{}
		}
		if *query != "" {
			clearLabel := i18n.T(i18n.KeySettingsNavSearchClear)
			clear := ui.Button(c, "").Label(clearLabel).Size(20, 20).Shrink(0).Padding(4).Radius(10).BorderWidth(0).Background(ui.Transparent).Tooltip(clearLabel)
			clear.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(12, 12).TextColor(muted) })
			if clear.Clicked() {
				*query = ""
			}
		}
	})
}
func formPane(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, paneBG ui.Color, query *string) {
	active := effectiveActiveSection(c, m, snap)
	jump := settingsSearchJumpState(c)
	jump.Active = active
	if m.FilterPtr != nil {
		jump.FilterQuery = strings.TrimSpace(*m.FilterPtr)
	} else {
		jump.FilterQuery = ""
	}
	jump.HighlightedKey = ""
	jump.HighlightedTitle = ""
	// General/Interface source lists have no inter-row gap: each Setting-Cell
	// owns its 16 DIP vertical padding and a half-pixel bottom border.
	gap := float32(10)
	if active == SectionGeneral || active == SectionInterface || active == SectionConversation || active == SectionFeatures {
		gap = 0
	} else if active == SectionAppearance {
		gap = 28
	}
	searching := strings.TrimSpace(*query) != ""
	scrollKey := "settings-options-" + string(active)
	if searching {
		scrollKey = "settings-options-search"
	}
	pane := settingsOptionsScroll(c, scrollKey, gap, paneBG)
	// A section switch fades the page in over the popover token. The pane is
	// keyed by its section, so each switch builds a new element whose lifetime is
	// one opening, and Reveal runs the enter by hand.
	pane.Opacity(chrome.Reveal(c, pane, chrome.PopoverDuration))
	pane.Children(func() {
		if searching {
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			settingsSearchResults(c, m, snap, *query, fg, muted, chrome.Color(colors.Hover), func() { *query = "" })
			return
		}
		finishItems := beginSettingsItemContributions(c, m.PublicContributions, active)
		defer finishItems()
		if m.SectionContent != nil && m.ActiveSection != "" {
			m.SectionContent(c)
			return
		}
		if active == SectionGeneral && m.CredentialsActive && m.Credentials != nil {
			credentialsPane(c, m, fg, muted)
			return
		}
		if active == SectionGeneral && m.Desktop != nil && !m.AccountActive {
			desktopPane(c, m, fg, muted)
			// The reference keeps its General rows (permission, language,
			// font, work detail, shortcuts, links, busy/close) at the top;
			// the account shortcut and the namespace browser sit last.
			generalNavigation(c, m, snap, fg, muted)
			return
		}
		if active == SectionGeneral && !m.AccountActive {
			generalNavigation(c, m, snap, fg, muted)
		}
		if active == SectionConversation && m.Conversation != nil {
			conversationPane(c, m, fg, muted)
			return
		}
		if active == SectionFeatures && m.Features != nil {
			featuresPane(c, m, fg, muted)
			return
		}
		if active == SectionAccount && m.Account != nil {
			accountPane(c, m, fg, muted)
			return
		}
		if active == SectionAppearance && m.Appearance != nil {
			appearancePane(c, m, fg, muted)
			return
		}
		if active == SectionInterface && m.Interface != nil {
			interfacePane(c, m, fg, muted)
			return
		}
		if active == SectionModels && m.Models != nil {
			modelsPane(c, m, fg, muted)
			return
		}
		if active == SectionPlugins && m.Plugins != nil {
			pluginsPane(c, m, fg, muted)
			return
		}
		if active == SectionSkills && m.Skills != nil {
			skillsPane(c, m, fg, muted)
			return
		}
		if active == SectionAgentPresets && m.AgentPresets != nil {
			agentPresetsPane(c, m, fg, muted)
			return
		}
		if active == SectionMCP && m.MCP != nil {
			mcpPane(c, m, fg, muted)
			return
		}
		if active == SectionAbout && m.About != nil {
			aboutPane(c, m, fg, muted)
			return
		}
		if active == SectionRemote && m.Remote != nil {
			remotePane(c, m, fg, muted)
			return
		}
		if (snap.SelectedNS == "pet" || snap.SelectedNS == "live2d") && m.Pet != nil {
			petPane(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), m.Pet, fg, muted, m.PetSettings)
			if m.PetSettings == nil {
				petLookPane(c, m.Pet, m.Store, fg, muted)
			}
			return
		}
		namespaceForm(c, m, snap, fg, muted)
	})
}

// RequestReplace confirms (optional) then posts a full settings/replace from drafts.
func RequestReplace(m Model) {
	if m.Store == nil {
		return
	}
	confirm := m.Store.PrepareReplaceFromDrafts(context.Background())
	if m.OnRequestReplace != nil {
		if confirm != nil {
			m.OnRequestReplace(confirm)
		}
		return
	}
	if m.ConfirmReplace != nil && !m.ConfirmReplace() {
		return
	}
	if confirm != nil {
		confirm()
	}
}

// namespaceForm paints the schema-driven form of the selected namespace: the
// title row, feature-owned hints, one row per field and the action bar.
func namespaceForm(c *ui.Context, m Model, snap settings.ViewState, fg, muted ui.Color) {
	var ns *settings.Namespace
	for i := range snap.Namespaces {
		if snap.Namespaces[i].NS == snap.SelectedNS {
			ns = &snap.Namespaces[i]
			break
		}
	}
	if ns == nil {
		ui.Text(c, i18n.T(i18n.KeySurfacesSettingsPick)).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		return
	}
	access := i18n.T(i18n.KeySurfacesSettingsWritable)
	accessColor := fg
	if !snap.Writable {
		access = i18n.T(i18n.KeySurfacesSettingsReadOnly)
		accessColor = chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
	}
	ui.Row(c).FillWidth().Gap(8).Padding(14, 0, 2, 0).AlignItems(ui.Center).Children(func() {
		ui.Text(c, ns.NS).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Grow(1)
		ui.Text(c, fmt.Sprintf("%s %g", i18n.T(i18n.KeySurfacesSettingsRevision), ns.Revision)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, access).FontSize(theme.InterfaceSize(11)).FontWeight(500).TextColor(accessColor)
	})
	if m.RevealSecrets != nil && hasSecretField(ns) {
		reveal := *m.RevealSecrets
		settingsFieldRow(c, "reveal-secrets", i18n.T(i18n.KeySurfacesSettingsReveal), "", fg, muted, nil, func() {
			if ui.Switch(c, &reveal).Label(i18n.T(i18n.KeySurfacesSettingsReveal)).Changed() {
				*m.RevealSecrets = reveal
			}
		})
	}
	if snap.SelectedNS == settings.VisualRepliesNamespace {
		ui.Text(c, i18n.T(i18n.KeySurfacesSettingsVisualHint)).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
	}
	if ns.NS == "llm-deepseek" || ns.NS == "llm-pi-ai" || ns.NS == "web-search-deepseek" {
		referencedCredentialsPane(c, m.ReferencedCredentials, ns.NS, fg, muted)
	}
	if snap.SelectedNS == "general" && m.OnSelectNamespace != nil {
		for _, candidate := range snap.Namespaces {
			switch candidate.NS {
			case "permission":
				if ui.Button(c, providerCopy("新任务默认权限", "Default permissions for new tasks")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
					m.OnSelectNamespace("permission")
				}
			case settings.CustomInstructionsNamespace:
				if ui.Button(c, i18n.T(i18n.KeyCustomInstructionsTitle)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
					m.OnSelectNamespace(settings.CustomInstructionsNamespace)
				}
			}
		}
	}
	for _, raw := range ns.Fields {
		field := raw
		if permissionDefaultField(c, m, snap, field, fg, muted) {
			continue
		}
		if configurationField(c, m, snap, *ns, field, fg, muted) {
			continue
		}
		fieldRow(c, m, snap, field, fg, muted)
	}
	settingsActions(c, m, snap, ns, fg, muted)
}

// settingsActions is the footer of a namespace page: the save state on the left,
// then discard, the rare full replace and the save action. Replace asks in the
// application's own dialog, so a destructive write never opens an operating
// system message box over the themed window.
func settingsActions(c *ui.Context, m Model, snap settings.ViewState, ns *settings.Namespace, fg, muted ui.Color) {
	blocked := snap.Loading || snap.Saving || !snap.Writable || snap.SelectedNS == "" ||
		customInstructionsInvalid(snap) || configurationInvalid(snap) || permissionDefaultUnavailable(snap)
	anchor := ui.Column(c.Key("settings-actions")).FillWidth().MinWidth(0).Gap(0)
	confirming := ui.Local(anchor, "confirm-replace", func() bool { return false })
	preparedReplace := ui.Local(anchor, "prepared-replace", func() func() { return nil })
	anchor.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Padding(16, 0, 6, 0).AlignItems(ui.Center).Children(func() {
			switch {
			case snap.Err != "":
				settingsStatusLine(c, snap.Err, c.Theme().Danger)
			case snap.Saving:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsSaving), muted)
			case snap.Dirty:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsUnsaved), muted)
			}
			ui.Box(c).Grow(1)
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			if ns != nil && !ns.HasSecrets {
				if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesSettingsReplace), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonMedium, Disabled: blocked}).Clicked() {
					submit := m.Store.PrepareReplaceFromDrafts(context.Background())
					if submit != nil {
						*preparedReplace = submit
						*confirming = true
					}
				}
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesSettingsDiscard), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium, Disabled: snap.Loading || snap.Saving}).Clicked() {
				m.Store.DiscardDrafts(context.Background())
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesSettingsSave), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: blocked || !snap.Dirty}).Clicked() {
				m.Store.SaveUpdate(context.Background())
			}
		})
		if ns != nil && ns.HasSecrets {
			settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsSecretsHint), muted)
		}
		settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsSaveHint), muted)
	})
	if !*confirming {
		return
	}
	ConfirmDialog(c, ConfirmOptions{
		Colors:       theme.Resolve(m.ThemeID).Active(m.SystemDark),
		Title:        i18n.T(i18n.KeySurfacesSettingsConfirmTitle),
		Body:         i18n.T(i18n.KeySurfacesSettingsConfirmBody),
		Detail:       i18n.T(i18n.KeySurfacesSettingsReplaceHint),
		ConfirmLabel: i18n.T(i18n.KeySurfacesSettingsConfirmOK),
		CancelLabel:  i18n.T(i18n.KeySurfacesSettingsCancel),
		Danger:       true,
		OnConfirm: func() {
			*confirming = false
			submit := *preparedReplace
			*preparedReplace = nil
			if submit != nil {
				submit()
			}
		},
		OnCancel: func() { *confirming = false; *preparedReplace = nil },
	})
}

// hasSecretField reports whether a namespace holds any password-like string
// field, which gates the full-replace action.
func hasSecretField(ns *settings.Namespace) bool {
	if ns == nil {
		return false
	}
	for _, f := range ns.Fields {
		if f.Kind == settings.FieldString && secretField(f) {
			return true
		}
	}
	return false
}

// fieldRow paints one schema field as a settings row: the switch, the select or
// the input sits at the row's end, and only long-form text takes a full block.
func fieldRow(c *ui.Context, m Model, snap settings.ViewState, f settings.Field, fg, muted ui.Color) {
	if snap.SelectedNS == settings.CustomInstructionsNamespace && f.Key == settings.CustomInstructionsField {
		customInstructionsRow(c, m, snap, fg, muted)
		return
	}
	if m.DraftPtr == nil {
		return
	}
	label := strings.TrimSpace(f.Label)
	if label == "" {
		label = f.Key
	}
	if f.Required {
		label += " *"
	}
	hint := f.Description
	if snap.Resets[f.Key] {
		hint = i18n.T(i18n.KeySurfacesSettingsInherit)
	}
	disabled := snap.Loading || snap.Saving || !snap.Writable
	var reset func()
	if m.Store != nil && !disabled {
		reset = func() { m.Store.ResetDraft(f.Key) }
	}
	seed := snap.Drafts[f.Key]
	key := "field-" + f.Key
	switch f.Kind {
	case settings.FieldBool:
		if m.BoolPtr == nil {
			return
		}
		checked := m.BoolPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			ui.Row(c).FillWidth().Justify(ui.End).AlignItems(ui.Center).Children(func() {
				if ui.Switch(c, checked).Label(label).Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, fmt.Sprint(*checked))
				}
			})
		})
	case settings.FieldEnum:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			if ui.Select(c, draft, f.EnumOptions).Label(label).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		})
	case settings.FieldJSON:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldBlock(c, key, label, hint, fg, muted, func() {
			if ui.TextArea(c, draft).Fill().Lines(4, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		})
	case settings.FieldNumber:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			if ui.TextInput(c, draft).Fill().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		})
	default:
		draft := m.DraftPtr(f.Key, seed)
		if secretField(f) {
			settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
				input := ui.TextInput(c, draft).Fill().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13))
				if m.RevealSecrets == nil || !*m.RevealSecrets {
					input = input.Password()
				}
				if input.Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
			})
			return
		}
		if MultilineStringField(f) || strings.Contains(seed, "\n") || strings.Contains(*draft, "\n") {
			settingsFieldBlock(c, key, label, hint, fg, muted, func() {
				if ui.TextArea(c, draft).Fill().Lines(3, 10).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
			})
			return
		}
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			if ui.TextInput(c, draft).Fill().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		})
	}
}

// settings.action is the content header's action list, immediately before the
// close button. The config action is the native open-document builtin, order 0.
func renderSettingsHeaderActions(c *ui.Context, m Model, drawConfig func()) {
	p := m.PublicContributions
	entries := []pluginui.PublicSlotEntry{}
	if p.Entries != nil && p.RenderEntry != nil {
		entries = p.Entries("settings.action", "")
	}
	hasDocument := false
	for _, entry := range entries {
		if entry.ID == "open-document" {
			hasDocument = true
			break
		}
	}
	drawn := false
	for _, entry := range entries {
		if entry.Native && entry.ID == "open-document" {
			if !drawn {
				drawConfig()
				drawn = true
			}
		} else if !entry.Native {
			// Existing native callers do not require a private renderer connection.
			// If it omits its optional document owner, retain the native action at 0.
			if !hasDocument && !drawn && entry.Order >= 0 && p.NativeWinner("settings.action", "", "open-document") {
				drawConfig()
				drawn = true
			}
			if entry.ID == "open-document" {
				drawn = true
			}
			p.RenderEntry(c, entry.Slot, entry.OwnerKey, entry.Handle)
		}
	}
	if !drawn && p.NativeWinner("settings.action", "", "open-document") {
		drawConfig()
	}
}
