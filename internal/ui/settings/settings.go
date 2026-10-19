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
		"about", "usage-stats", "usagestats", "usage", "remote-workspace", "pet", "live2d":
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
			generalNavigation(c, m, snap)
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
			petPane(c, m.Pet, fg, muted)
			petLookPane(c, m.Pet, m.Store, fg, muted)
			return
		}
		var ns *settings.Namespace
		for i := range snap.Namespaces {
			if snap.Namespaces[i].NS == snap.SelectedNS {
				ns = &snap.Namespaces[i]
				break
			}
		}
		if ns == nil {
			ui.Text(c, i18n.T(i18n.KeySettingsPickNS)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			return
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, ns.NS).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			ui.Text(c, fmt.Sprintf("%s %g", i18n.T(i18n.KeySettingsRevision), ns.Revision)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			if snap.Writable {
				ui.Text(c, i18n.T(i18n.KeySettingsWritable)).FontSize(theme.InterfaceSize(11)).Bold().TextColor(fg)
			} else {
				accent := chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
				ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).FontSize(theme.InterfaceSize(11)).Bold().TextColor(accent)
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
		if snap.SelectedNS == "general" && m.OnSelectNamespace != nil {
			for _, candidate := range snap.Namespaces {
				if candidate.NS == "permission" && ui.Button(c, providerCopy("新任务默认权限", "Default permissions for new tasks")).Clicked() {
					m.OnSelectNamespace("permission")
				}
				if candidate.NS == settings.CustomInstructionsNamespace && ui.Button(c, i18n.T(i18n.KeyCustomInstructionsTitle)).Clicked() {
					m.OnSelectNamespace(settings.CustomInstructionsNamespace)
				}
			}
		}
		if snap.SelectedNS == settings.VisualRepliesNamespace {
			ui.Text(c, "启用 HTML 视觉回复前会检查本机隔离渲染；失败时保持关闭并显示原因。 / Enabling HTML visual replies first checks isolated rendering; failure keeps the setting off.").FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if ns.NS == "llm-deepseek" || ns.NS == "llm-pi-ai" || ns.NS == "web-search-deepseek" {
			referencedCredentialsPane(c, m.ReferencedCredentials, ns.NS, fg, muted)
		}
		for _, f := range ns.Fields {
			if permissionDefaultField(c, m, snap, f, fg, muted) {
				continue
			}
			if configurationField(c, m, snap, *ns, f, fg, muted) {
				continue
			}
			fieldRow(c, m, snap, f, fg, muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			save := ui.Button(c, i18n.T(i18n.KeySettingsSave))
			if snap.Loading || snap.Saving || !snap.Writable || !snap.Dirty || snap.SelectedNS == "" || customInstructionsInvalid(snap) || configurationInvalid(snap) || permissionDefaultUnavailable(snap) {
				save = save.Disabled(true)
			}
			if save.Clicked() {
				m.Store.SaveUpdate(context.Background())
			}
			replace := ui.Button(c, i18n.T(i18n.KeySettingsReplace))
			if snap.Loading || snap.Saving || !snap.Writable || ns.HasSecrets || snap.SelectedNS == "" || customInstructionsInvalid(snap) || configurationInvalid(snap) || permissionDefaultUnavailable(snap) {
				replace = replace.Disabled(true)
			}
			if replace.Clicked() {
				RequestReplace(m)
			}
			if ui.Button(c, "放弃草稿 / Discard drafts").Disabled(snap.Loading || snap.Saving).Clicked() {
				m.Store.DiscardDrafts(context.Background())
			}
		})
		ui.Text(c, providerCopy("保存只提交已修改字段；重置会删除覆盖值并继承默认设置。", "Save submits changed fields only. Reset removes an override and inherits its default.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if ns.HasSecrets {
			ui.Text(c, providerCopy("此配置包含隐藏密钥，完整替换不可用；普通保存会保留未编辑的密钥。", "This section contains hidden secrets. Use Save to preserve unedited secrets; full replacement is unavailable.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Text(c, i18n.T(i18n.KeySettingsReplaceHint)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
	if snap.SelectedNS == settings.CustomInstructionsNamespace && f.Key == settings.CustomInstructionsField {
		customInstructionsRow(c, m, snap, fg, muted)
		return
	}
	if m.DraftPtr == nil {
		return
	}
	ui.Column(c).Gap(4).Children(func() {
		label := f.Label
		if f.Required {
			label += " *"
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1)
			if m.Store != nil && ui.Button(c, providerCopy("重置为继承值", "Reset to inherited")).FontSize(theme.InterfaceSize(11)).Disabled(snap.Loading || snap.Saving || !snap.Writable).Clicked() {
				m.Store.ResetDraft(f.Key)
			}
		})
		if snap.Resets[f.Key] {
			ui.Text(c, providerCopy("保存后使用继承值；再次编辑可取消重置。", "Will inherit after Save. Edit again to cancel the reset.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if f.Description != "" {
			ui.Text(c, f.Description).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
