// Package settings is the M8 generic harness settings shell.
package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the settings panel to store state.
type Model struct {
	// Optional feature-owned section body, composed by the application.
	ActiveSection  SectionID
	SectionContent func(*ui.Context)
	Pet            *pet.Companion
	PetSettings    *PetSettingsAccess
	Store          *settings.Store
	ThemeID        string
	SystemDark     bool
	OnClose        func()
	OnDraft        func(key, value string)
	// DraftPtr returns a stable *string for editing one field in the selected namespace.
	DraftPtr func(key, seed string) *string
	// BoolPtr returns a stable *bool wired to the same field drafts (bool kinds).
	BoolPtr func(key, seed string) *bool
	// ConfirmReplace, when set, must return true before SaveReplaceFromDrafts runs.
	ConfirmReplace func() bool

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
	OnSelectDesktop       func()
	OnSelectAccount       func()
	OnSelectCredentials   func()
	OnSelectAbout         func()
	OnSelectAppearance    func()
	OnSelectInterface     func()
	OnSelectModels        func()
	OnSelectMCP           func()
	OnSelectPlugins       func()
	OnSelectSkills        func()
	OnSelectAgentPresets  func()
	OnSelectNamespace     func(ns string)
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
	if m.Store == nil && m.Desktop == nil && m.SectionContent == nil {
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

	// The rail and the options column scroll independently inside one panel.
	ui.Row(c).Grow(1).MinWidth(0).MinHeight(0).Background(bg).AlignItems(ui.Stretch).Children(func() {
		nsList(c, m, snap, fg, muted, border, bg, chrome.Color(colors.Hover))
		ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			header(c, m, snap, fg, muted, border)
			formPane(c, m, snap, fg, muted, border, bg)
		})
	})
}

func header(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border ui.Color) {
	ui.Row(c).FillWidth().Height(54).Shrink(0).Padding(20, 14, 8, 10).Gap(8).AlignItems(ui.Center).Children(func() {
		if snap.Loading {
			ui.Text(c, i18n.T(i18n.KeySettingsLoading)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		ui.Box(c).Grow(1).MinWidth(0)
		if m.OnClose != nil {
			close := ui.Button(c, "").Label(i18n.T(i18n.KeySettingsBack)).Size(28, 28).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent)
			close.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(14, 14).TextColor(fg) })
			if close.Clicked() {
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

func nsList(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, navBG, hover ui.Color) {
	active := activeSection(m, snap)
	ui.Column(c).Width(188).Shrink(0).MinHeight(0).Padding(22, 12, 0, 12).Gap(18).Children(func() {
		ui.Text(c, i18n.T(i18n.KeySettingsTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Padding(0, 12, 0, 12)
		ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Gap(4).Children(func() {
			for _, entry := range primaryNavigation(m, snap) {
				selected := entry.id == active
				fill := ui.Transparent
				if selected {
					fill = hover
				}
				label := sectionLabel(entry.id)
				button := ui.Button(c.Key(string(entry.id)), "").Label(label).FillWidth().Height(40).Shrink(0).Radius(12).BorderWidth(0).Background(fill).TextColor(fg).Padding(9, 16, 9, 12).Gap(8).AlignItems(ui.Center)
				button.Children(func() {
					ui.Icon(c, settingsIcons[entry.icon]).Size(16, 16).Shrink(0).TextColor(fg)
					weight := 400
					if selected {
						weight = 500
					}
					ui.Text(c, label).FontSize(theme.InterfaceSize(14)).FontWeight(weight).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
				})
				if button.Hovered() && !selected {
					button.Background(hover)
				}
				if button.Clicked() && entry.onSelect != nil {
					entry.onSelect()
				}
			}
		})
	})
}
func formPane(c *ui.Context, m Model, snap settings.ViewState, fg, muted, border, paneBG ui.Color) {
	active := activeSection(m, snap)
	ui.Scroll(c.Key("settings-options-"+string(active))).Grow(1).MinWidth(0).MinHeight(0).Background(paneBG).Padding(0, 24, 24, 24).Gap(10).Children(func() {
		if snap.Err != "" {
			ui.Text(c, snap.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		if snap.LastAction != "" {
			ui.Text(c, snap.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if m.SectionContent != nil && m.ActiveSection != "" {
			m.SectionContent(c)
			return
		}
		if active == SectionGeneral {
			generalNavigation(c, m, snap, fg, muted)
		}
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
		if (snap.SelectedNS == "pet" || snap.SelectedNS == "live2d") && m.Pet != nil {
			petPane(c, m.Pet, fg, muted, m.PetSettings)
			if m.PetSettings == nil { petLookPane(c, m.Pet, m.Store, fg, muted) }
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
	if m.ConfirmReplace != nil && !m.ConfirmReplace() {
		return
	}
	m.Store.SaveReplaceFromDrafts(context.Background())
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
				if ui.Button(c, providerCopy("新任务默认权限", "Default permissions for new tasks")).Clicked() {
					m.OnSelectNamespace("permission")
				}
			case settings.CustomInstructionsNamespace:
				if ui.Button(c, i18n.T(i18n.KeyCustomInstructionsTitle)).Clicked() {
					m.OnSelectNamespace(settings.CustomInstructionsNamespace)
				}
			}
		}
	}
	for index, raw := range ns.Fields {
		field := raw
		last := index == len(ns.Fields)-1
		if permissionDefaultField(c, m, snap, field, fg, muted) {
			continue
		}
		if configurationField(c, m, snap, *ns, field, fg, muted) {
			continue
		}
		fieldRow(c, m, snap, field, fg, muted, last)
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
	anchor.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Padding(16, 0, 6, 0).AlignItems(ui.Center).Children(func() {
			switch {
			case snap.Saving:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsSaving), muted)
			case snap.Dirty:
				settingsStatusLine(c, i18n.T(i18n.KeySurfacesSettingsUnsaved), muted)
			}
			ui.Box(c).Grow(1)
			if ns != nil && !ns.HasSecrets {
				replace := ui.Button(c, i18n.T(i18n.KeySurfacesSettingsReplace)).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger).Disabled(blocked)
				if replace.Hovered() && !blocked {
					replace.Background(c.Theme().SurfaceHover)
				}
				if replace.Clicked() {
					*confirming = true
				}
			}
			if ui.Button(c, i18n.T(i18n.KeySurfacesSettingsDiscard)).FontSize(theme.InterfaceSize(12)).Disabled(snap.Loading || snap.Saving).Clicked() {
				m.Store.DiscardDrafts(context.Background())
			}
			if ui.PrimaryButton(c, i18n.T(i18n.KeySurfacesSettingsSave)).Disabled(blocked || !snap.Dirty).Clicked() {
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
		Title:        i18n.T(i18n.KeySurfacesSettingsConfirmTitle),
		Body:         i18n.T(i18n.KeySurfacesSettingsConfirmBody),
		Detail:       i18n.T(i18n.KeySurfacesSettingsReplaceHint),
		ConfirmLabel: i18n.T(i18n.KeySurfacesSettingsConfirmOK),
		CancelLabel:  i18n.T(i18n.KeySurfacesSettingsCancel),
		Danger:       true,
		OnConfirm: func() {
			*confirming = false
			m.Store.SaveReplaceFromDrafts(context.Background())
		},
		OnCancel: func() { *confirming = false },
	})
}

// hasSecretField reports whether a namespace holds any password-like string
// field, which gates the full-replace action.
func hasSecretField(ns *settings.Namespace) bool {
	if ns == nil {
		return false
	}
	for _, f := range ns.Fields {
		if f.Kind == settings.FieldString && (SecretLikeKey(f.Key) || SecretLikeKey(f.Label)) {
			return true
		}
	}
	return false
}

// fieldRow paints one schema field as a settings row: the switch, the select or
// the input sits at the row's end, and only long-form text takes a full block.
func fieldRow(c *ui.Context, m Model, snap settings.ViewState, f settings.Field, fg, muted ui.Color, last bool) {
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
	divider := !last
	seed := snap.Drafts[f.Key]
	key := "field-" + f.Key
	switch f.Kind {
	case settings.FieldBool:
		if m.BoolPtr == nil {
			return
		}
		checked := m.BoolPtr(f.Key, seed)
		settingsFieldRowDivider(c, key, label, hint, fg, muted, reset, func() {
			ui.Row(c).FillWidth().Justify(ui.End).AlignItems(ui.Center).Children(func() {
				if ui.Switch(c, checked).Label(label).Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, fmt.Sprint(*checked))
				}
			})
		}, divider)
	case settings.FieldEnum:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldRowDivider(c, key, label, hint, fg, muted, reset, func() {
			if ui.Select(c, draft, f.EnumOptions).Label(label).FillWidth().Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		}, divider)
	case settings.FieldJSON:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldBlockDivider(c, key, label, hint, fg, muted, func() {
			if ui.TextArea(c, draft).Fill().Lines(4, 12).Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		}, divider)
	case settings.FieldNumber:
		draft := m.DraftPtr(f.Key, seed)
		settingsFieldRowDivider(c, key, label, hint, fg, muted, reset, func() {
			if ui.TextInput(c, draft).Fill().Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		}, divider)
	default:
		draft := m.DraftPtr(f.Key, seed)
		if SecretLikeKey(f.Key) || SecretLikeKey(f.Label) {
			settingsFieldRowDivider(c, key, label, hint, fg, muted, reset, func() {
				input := ui.TextInput(c, draft).Fill()
				if m.RevealSecrets == nil || !*m.RevealSecrets {
					input = input.Password()
				}
				if input.Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
			}, divider)
			return
		}
		if MultilineStringField(f) || strings.Contains(seed, "\n") || strings.Contains(*draft, "\n") {
			settingsFieldBlockDivider(c, key, label, hint, fg, muted, func() {
				if ui.TextArea(c, draft).Fill().Lines(3, 10).Disabled(disabled).Changed() && m.OnDraft != nil {
					m.OnDraft(f.Key, *draft)
				}
			}, divider)
			return
		}
		settingsFieldRowDivider(c, key, label, hint, fg, muted, reset, func() {
			if ui.TextInput(c, draft).Fill().Disabled(disabled).Changed() && m.OnDraft != nil {
				m.OnDraft(f.Key, *draft)
			}
		}, divider)
	}
}