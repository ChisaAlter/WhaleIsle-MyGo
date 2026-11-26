// Package settings is the M8 generic harness settings shell.
package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/state"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the settings panel to store state.
type Model struct {
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
	OnSelectVoice         func()
	OnSelectDeepSeekAPIs  func()
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

	// version is the settings store's published version, read once by View
	// and shared by everything it builds.
	version *state.Version[settings.ViewState]
}

// withVersion reads the store's published version unless the page entry
// already did.
func (m Model) withVersion() Model {
	if m.version == nil && m.Store != nil {
		m.version = m.Store.Snapshot()
	}
	return m
}

// snapshot is the published view state View read, or the zero state.
func (m Model) snapshot() settings.ViewState {
	if m.version == nil {
		return settings.ViewState{}
	}
	return m.version.Value
}

// rev is the published version's revision, 0 without a store.
func (m Model) rev() uint64 {
	if m.version == nil {
		return 0
	}
	return m.version.Rev
}

// View paints the settings shell: the coss sidebar rail and the options pane
// of the active section. It reads the settings store's published version
// once and hands it down; nested views never take a snapshot of their own.
func View(c *ui.Context, m Model) {
	if m.Store == nil && m.Desktop == nil && m.SectionContent == nil && m.Remote == nil {
		return
	}
	finishPreferenceErrors := beginBodyPreferenceErrorNotices(c)
	defer finishPreferenceErrors()
	if close := m.OnClose; close != nil {
		m.OnClose = func() { ClearPreferenceErrorNotices(c); close() }
	}
	m = m.withVersion()
	snap := m.snapshot()
	t := tk(c)
	fg, muted, border := t.Foreground, t.MutedForeground, t.Border

	// The rail and the options column scroll independently inside one panel.
	root := ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).Gap(0).AlignItems(ui.Stretch)
	query := ui.Local(root, "settings-search", func() string { return "" })
	root.Children(func() {
		nsList(c, m, snap, fg, muted, border, query)
		ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			header(c, m, snap, fg, muted, border)
			formPane(c, m, snap, fg, muted, border, ui.Transparent, query)
		})
	})
}

// header is the options pane's top bar: the page title, the configuration
// file action and the close button, over a bottom border.
func header(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border ui.Color) {
	t := tk(c)
	bar := ui.Row(c).FillWidth().Height(design.Sp(14)).Shrink(0).PaddingX(design.Sp(6)).Gap(design.Sp(2)).AlignItems(ui.Center).
		BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border).Label(providerCopy("设置页头", "Settings header"))
	bar.Children(func() {
		kText(c, navigationLabel(m, effectiveActiveSection(c, m, snap)), design.TextBase, design.WeightSemibold, t.Foreground).
			Grow(1).MinWidth(0).SingleLine().Role(ui.RoleHeading)
		configButton := kButtonDisabled(c, i18n.T(i18n.KeySurfacesSettingsOpenConfig), m.OnOpenConfig == nil)
		if m.OnOpenConfig == nil {
			configButton.Tooltip(i18n.T(KeyPass4AConfigUnavailable))
		} else if configButton.Clicked() {
			m.OnOpenConfig()
		}
		if m.OnClose != nil {
			close := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIconSM, Icon: coss.IconX, Label: i18n.T(i18n.KeySettingsBack)})
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
	case SectionVoice:
		return i18n.T(i18n.KeySettingsSectionVoice)
	case SectionDeepSeekAPIs:
		return i18n.T(i18n.KeySettingsSectionDeepSeek)
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

// nsList is the settings rail as coss's Sidebar: the sidebar surface and
// border, a title, the search field, then labelled groups of menu buttons
// (32 DIPs, rounded-lg). The scroll container spans the rail's full inner width
// and reserves a right gutter so the overlay scrollbar never overlaps a button's
// pill edge. The groups are a memo region keyed by everything they show, so a
// frame that changes nothing in the rail does not rebuild it.
func nsList(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border ui.Color, query *string) {
	t := tk(c)
	active := effectiveActiveSection(c, m, snap)
	searching := strings.TrimSpace(*query) != ""
	// The search hit pulse fills with the same sidebar-accent wash.
	settingsSearchJumpState(c).HitFill = t.SidebarAccent
	groups := navigationGroups(m, snap)
	key := newMemoKey(m.rev()).u(design.Revision()).u(theme.Revision()).s(string(active)).b(searching)
	for _, group := range groups {
		key.s(group.label)
		for _, entry := range group.entries {
			key.s(string(entry.id), navigationLabel(m, entry.id), entry.icon)
		}
	}
	railBg := t.Sidebar
	if t.Glass || chrome.BackdropActive() {
		if t.IsDark() {
			railBg = design.TW.White.Alpha(0.04)
		} else {
			railBg = design.TW.Black.Alpha(0.025)
		}
	}
	rail := ui.Column(c).Key("settings-rail").Width(design.Sp(52)).Shrink(0).MinHeight(0).Padding(design.Sp(4), 0, 0, 0).Gap(design.Sp(3)).
		Background(railBg).BorderWidth(0, design.BorderWidth, 0, 0).BorderColor(t.SidebarBorder).
		Label(providerCopy("设置导航", "Settings navigation"))
	rail.Children(func() {
		kText(c, i18n.T(i18n.KeySettingsTitle), design.TextBase, design.WeightSemibold, t.SidebarAccentForeground).PaddingX(design.Sp(4)).Role(ui.RoleHeading)
		settingsRailSearch(c, m, snap, query, fg, muted, border)
		// Reserve a dedicated right gutter (16 DIPs) inside the full-width
		// scroll container so the vertical overlay scrollbar thumb and drag
		// track stay strictly right of every navigation pill.
		ui.Scroll(c.Key("settings-rail-scroll")).FillWidth().Grow(1).MinHeight(0).Padding(0, design.Sp(4), 0, design.Sp(2)).Memo(key.sum()).Children(func() {
			for _, group := range groups {
				ui.Column(c.Key("settings-group-"+group.label)).FillWidth().MinWidth(0).Gap(design.Sp(1)).Padding(0, 0, design.Sp(2), 0).Children(func() {
					ui.Row(c).FillWidth().Height(design.Sp(8)).Shrink(0).PaddingX(design.Sp(2)).AlignItems(ui.Center).Children(func() {
						kText(c, group.label, design.TextXS, design.WeightMedium, t.SidebarForeground).SingleLine()
					})
					for _, entry := range group.entries {
						// The rail keeps no selection while the search lists results.
						railButton(c, m, entry, entry.id == active && !searching, query)
					}
				})
			}
		})
	})
}

// railButton is one SidebarMenuButton. Selected paints coss's sidebar-accent
// pill with a hairline sidebar-border, a leading sidebar-primary indicator and
// semibold ink; hover uses a noticeably lighter wash without border or
// indicator so a hovered neighbor never reads as a second active section.
func railButton(c *ui.Context, m Model, entry navigationEntry, selected bool, query *string) {
	t := tk(c)
	label := navigationLabel(m, entry.id)
	borderColor := ui.Transparent
	if selected {
		borderColor = t.SidebarBorder
	}
	button := ui.ButtonBase(c.Key(string(entry.id))).Label(label).FillWidth().Height(design.Sp(8)).Shrink(0).
		Radius(design.RadiusLG).Border(design.BorderWidth, borderColor).Padding(0, design.Sp(2), 0, design.Sp(2.5)).Gap(design.Sp(2)).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
	if selected {
		button.Selected(true)
	}
	hovered, pressed := button.Hovered(), button.Pressed()
	iconInk, textInk, weight := t.SidebarForeground, t.SidebarForeground, design.WeightNormal
	switch {
	case selected:
		button.Background(t.SidebarAccent)
		iconInk = t.SidebarPrimary
		textInk = t.SidebarAccentForeground
		weight = design.WeightSemibold
		indicator := t.SidebarPrimary
		button.DrawOver(func(p *ui.Painter, r ui.Rect) {
			barH := min(design.Sp(4), max(0, r.H-design.Sp(3)))
			barW := design.Sp(0.75)
			p.Fill(ui.Rect{X: r.X + design.Sp(0.5), Y: r.Y + (r.H-barH)/2, W: barW, H: barH}, indicator, barW/2)
		})
	case pressed:
		button.Background(t.SidebarAccent.Alpha(0.75))
		iconInk = t.SidebarAccentForeground
		textInk = t.SidebarAccentForeground
	case hovered:
		button.Background(t.SidebarAccent.Alpha(0.45))
		mid := t.SidebarForeground.Mix(t.SidebarAccentForeground, 0.35)
		iconInk, textInk = mid, mid
	}
	if button.FocusVisible() {
		button.Shadow(0, 0, 0, design.RingWidth, t.SidebarRing)
	}
	button.Children(func() {
		ui.Icon(c, settingsIcons[entry.icon]).Size(design.IconSM, design.IconSM).Shrink(0).TextColor(iconInk).PassThrough()
		kText(c, label, design.TextSM, weight, textInk).Grow(1).MinWidth(0).SingleLine()
	})
	if button.Clicked() {
		*query = ""
		*settingsSearchJumpState(c) = settingsSearchJump{}
		if entry.onSelect != nil {
			entry.onSelect()
		}
	}
}

// settingsRailSearch is the rail's SidebarInput: a small coss input group
// with the search icon, and a clear button once there is text.
func settingsRailSearch(c *ui.Context, m Model, snap settings.ViewState, query *string, fg, muted, border ui.Color) {
	label := i18n.T(i18n.KeySettingsNavSearch)
	var end func()
	if *query != "" {
		end = func() {
			clearLabel := i18n.T(i18n.KeySettingsNavSearchClear)
			clear := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIconXS, Icon: coss.IconX, Label: clearLabel})
			clear.Tooltip(clearLabel)
			if clear.Clicked() {
				*query = ""
			}
		}
	}
	ui.Box(c.Key("settings-search")).FillWidth().Shrink(0).PaddingX(design.Sp(2)).Label(label).Children(func() {
		input := coss.InputGroup(c, query, coss.InputGroupOptions{
			InputOptions: coss.InputOptions{Size: coss.InputSM, Placeholder: label, Label: label},
			Start:        func() { coss.AddonIcon(c, coss.IconSearch) },
			End:          end,
		})
		input.OnSubmit(func() {
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
	// Row lists own their rhythm (each row its padding and rule); other pages
	// stack cards and groups 24 DIPs apart, coss's form gap.
	gap := design.Sp(6)
	if active == SectionGeneral || active == SectionInterface || active == SectionConversation || active == SectionFeatures {
		gap = 0
	}
	searching := strings.TrimSpace(*query) != ""
	scrollKey := "settings-options-" + string(active)
	if searching {
		scrollKey = "settings-options-search"
	}
	pane := settingsOptionsScroll(c, scrollKey, gap, paneBG)
	// A section switch fades the page in over the swap token — the original's
	// dsh-motion-swap-in keyframe is an opacity-only 100 ms ease-in-out. The
	// pane is keyed by its section, so each switch builds a new element whose
	// lifetime is one opening, and Reveal runs the enter by hand.
	pane.Opacity(chrome.Reveal(c, pane, chrome.SwapDuration))
	pane.Children(func() {
		if searching {
			settingsSearchResults(c, m, snap, *query, fg, muted, tk(c).Accent, func() { *query = "" })
			return
		}
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
		coss.Empty(c, coss.EmptyOptions{Icon: coss.IconFile, Description: i18n.T(i18n.KeySurfacesSettingsPick)}, nil)
		return
	}
	ui.Row(c).FillWidth().Gap(design.Sp(2)).Padding(design.Sp(4), 0, design.Sp(1), 0).AlignItems(ui.Center).Children(func() {
		kText(c, ns.NS, design.TextBase, design.WeightSemibold, tk(c).Foreground).Grow(1).MinWidth(0).Role(ui.RoleHeading)
		coss.Badge(c, fmt.Sprintf("%s %g", i18n.T(i18n.KeySurfacesSettingsRevision), ns.Revision), coss.BadgeOptions{Variant: coss.BadgeOutline})
		if snap.Writable {
			coss.Badge(c, i18n.T(i18n.KeySurfacesSettingsWritable), coss.BadgeOptions{Variant: coss.BadgeSecondary})
		} else {
			coss.Badge(c, i18n.T(i18n.KeySurfacesSettingsReadOnly), coss.BadgeOptions{Variant: coss.BadgeWarning})
		}
	})
	if m.RevealSecrets != nil && hasSecretField(ns) {
		reveal := *m.RevealSecrets
		settingsFieldRow(c, "reveal-secrets", i18n.T(i18n.KeySurfacesSettingsReveal), "", fg, muted, nil, func() {
			if coss.Switch(c, &reveal, "", coss.CheckOptions{Label: i18n.T(i18n.KeySurfacesSettingsReveal)}).Changed() {
				*m.RevealSecrets = reveal
			}
		})
	}
	if snap.SelectedNS == settings.VisualRepliesNamespace {
		kHint(c, i18n.T(i18n.KeySurfacesSettingsVisualHint))
	}
	if ns.NS == "llm-deepseek" || ns.NS == "llm-pi-ai" || ns.NS == "web-search-deepseek" {
		referencedCredentialsPane(c, m.ReferencedCredentials, ns.NS, fg, muted)
	}
	if snap.SelectedNS == "general" && m.OnSelectNamespace != nil {
		ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
			for _, candidate := range snap.Namespaces {
				switch candidate.NS {
				case "permission":
					if kButton(c, providerCopy("新任务默认权限", "Default permissions for new tasks")).Clicked() {
						m.OnSelectNamespace("permission")
					}
				case settings.CustomInstructionsNamespace:
					if kButton(c, i18n.T(i18n.KeyCustomInstructionsTitle)).Clicked() {
						m.OnSelectNamespace(settings.CustomInstructionsNamespace)
					}
				}
			}
		})
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
// system message box over the themed window. A failed save shows its sentence
// inline, as a coss error Alert above the actions.
func settingsActions(c *ui.Context, m Model, snap settings.ViewState, ns *settings.Namespace, fg, muted ui.Color) {
	blocked := snap.Loading || snap.Saving || !snap.Writable || snap.SelectedNS == "" ||
		customInstructionsInvalid(snap) || configurationInvalid(snap) || permissionDefaultUnavailable(snap)
	anchor := ui.Column(c.Key("settings-actions")).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(4), 0, design.Sp(1), 0)
	confirming := ui.Local(anchor, "confirm-replace", func() bool { return false })
	preparedReplace := ui.Local(anchor, "prepared-replace", func() func() { return nil })
	anchor.Children(func() {
		settingsAlert(c, "settings-actions", snap.Err != "", providerCopy("没能保存设置，请重试。", "The settings could not be saved. Try again."), snap.Err)
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			switch {
			case snap.Saving:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsSaving), muted)
			case snap.Dirty:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsUnsaved), muted)
			}
			ui.Box(c).Grow(1)
			if ns != nil && !ns.HasSecrets {
				if kDanger(c, i18n.T(i18n.KeySurfacesSettingsReplace), blocked).Clicked() {
					submit := m.Store.PrepareReplaceFromDrafts(context.Background())
					if submit != nil {
						*preparedReplace = submit
						*confirming = true
					}
				}
			}
			if kGhost(c, i18n.T(i18n.KeySurfacesSettingsDiscard), snap.Loading || snap.Saving).Clicked() {
				m.Store.DiscardDrafts(context.Background())
			}
			if coss.Button(c, i18n.T(i18n.KeySurfacesSettingsSave), coss.ButtonOptions{Size: coss.ButtonSM, Disabled: blocked || !snap.Dirty, Loading: snap.Saving}).Clicked() {
				m.Store.SaveUpdate(context.Background())
			}
		})
		if ns != nil && ns.HasSecrets {
			kHint(c, i18n.T(i18n.KeySurfacesSettingsSecretsHint))
		}
		kHint(c, i18n.T(i18n.KeySurfacesSettingsSaveHint))
	})
	if !*confirming {
		return
	}
	ConfirmDialog(c, ConfirmOptions{
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
	changed := func(e ui.Element, value *string) {
		if e.Changed() && m.OnDraft != nil {
			m.OnDraft(f.Key, *value)
		}
	}
	switch f.Kind {
	case settings.FieldBool:
		if m.BoolPtr == nil {
			return
		}
		checked := m.BoolPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			ui.Row(c).FillWidth().Justify(ui.End).AlignItems(ui.Center).Children(func() {
				if coss.Switch(c, checked, "", coss.CheckOptions{Label: label, Disabled: disabled}).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, fmt.Sprint(*checked))
				}
			})
		})
	case settings.FieldEnum:
		value := m.DraftPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			changed(kSelect(c, value, f.EnumOptions, label, disabled), value)
		})
	case settings.FieldJSON:
		value := m.DraftPtr(f.Key, seed)
		settingsFieldBlock(c, key, label, hint, fg, muted, func() {
			changed(coss.Textarea(c, value, coss.TextareaOptions{Label: label, Disabled: disabled, MaxLines: 12}), value)
		})
	case settings.FieldNumber:
		value := m.DraftPtr(f.Key, seed)
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			changed(kInput(c, value, coss.InputOptions{Label: label, Disabled: disabled}), value)
		})
	default:
		value := m.DraftPtr(f.Key, seed)
		if secretField(f) {
			settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
				hidden := m.RevealSecrets == nil || !*m.RevealSecrets
				changed(kInput(c, value, coss.InputOptions{Label: label, Disabled: disabled, Password: hidden}), value)
			})
			return
		}
		if MultilineStringField(f) || strings.Contains(seed, "\n") || strings.Contains(*value, "\n") {
			settingsFieldBlock(c, key, label, hint, fg, muted, func() {
				changed(coss.Textarea(c, value, coss.TextareaOptions{Label: label, Disabled: disabled, MaxLines: 10}), value)
			})
			return
		}
		settingsFieldRow(c, key, label, hint, fg, muted, reset, func() {
			changed(kInput(c, value, coss.InputOptions{Label: label, Disabled: disabled}), value)
		})
	}
}
