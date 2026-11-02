package settings

import (
	"github.com/egoist/mygo/ui"
	"strings"
	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/ui/i18n"
)

// ModelsAccess is the dedicated Models settings page (session/modelCatalog).
type ModelsAccess struct {
	configurationBinding *store.NamespaceBinding
	ProviderDiscovery    *ProviderDiscoveryAccess
	ProviderProfiles     *ProviderProfilesAccess
	Vision               *VisionAccess
	// ProviderEnv reports the environment variables a provider namespace reads
	// its API key from, and whether this process has one set: the orange hint on
	// a card that still needs a key. Nil leaves the page saying nothing about the
	// environment, as before the Host is described.
	ProviderEnv func(namespace string) (names []string, present bool)
	Labels      []string
	Selected    *string
	SessionID   string
	Err         string
	LastAction  string
	OnConfigure func(namespace string)
	OnRefresh   func()
	OnApply     func()
}

func modelsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Models == nil {
		return
	}
	a := m.Models
	ui.Text(c, providerCopy("模型", "Models")).FontSize(fonttheme.InterfaceSize(16)).FixedLineHeight(fonttheme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("填入各提供商的 API 密钥即可使用其模型。", "Enter your API keys to use models from the following providers.")).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(muted)
	if a.Err != "" {
		ui.Text(c, a.Err).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, settingsActionText(a.LastAction)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	providerProfilesPane(c, a.ProviderProfiles, fg, muted)
	if m.Store != nil {
		state := m.Store.Snapshot()
		if state.Loading {
			ui.Text(c, i18n.T(i18n.KeySettingsLoading)).TextColor(muted)
		}
		count := 0
		for _, namespace := range state.Namespaces {
			if SectionForNamespace(namespace.NS) != SectionModels || namespace.NS == "llm-vision-fallback" {
				continue
			}
			count++
			providerCard := ui.Column(c.Key("model-provider-" + namespace.NS)).FillWidth().MinWidth(0).Gap(8)
			editing := ui.Local(providerCard, "editing", func() bool { return false })
			providerCard.Children(func() {
				modelRowCard(c, "model-row-"+namespace.NS, func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, modelProviderTitle(namespace.NS)).Grow(1).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).TextColor(fg)
						if modelPill(c, providerCopy("编辑", "Edit"), false, true, (m.OnSelectNamespace == nil && a.OnConfigure == nil) || state.Loading || state.Saving).Label(providerCopy("编辑 ", "Edit ") + modelProviderTitle(namespace.NS)).Clicked() {
							if a.OnConfigure != nil {
								*editing = !*editing
								a.OnConfigure(namespace.NS)
							} else if m.OnSelectNamespace != nil {
								m.OnSelectNamespace(namespace.NS)
							}
						}
					})
				})
				if *editing && a.OnConfigure != nil {
					modelSetupCard(c, "model-setup-"+namespace.NS, func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, modelProviderTitle(namespace.NS)).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).TextColor(fg)
							ui.Text(c, namespace.NS).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
						})
						modelNamespaceBody(c, m, namespace.NS)
					})
				}
			})
		}
		if count == 0 && !state.Loading {
			ui.Text(c, providerCopy("当前 Host 尚未提供模型供应商配置。", "This Host exposes no provider configuration.")).TextColor(muted)
		}
		modelAddProviderCard(c, m, a, fg, muted)
	} else {
		modelAddProviderCard(c, m, a, fg, muted)
	}
	visionPane(c, a.Vision, fg, muted)
	advanced := ui.Column(c.Key("model-provider-tools")).FillWidth().MinWidth(0).Gap(12)
	open := ui.Local(advanced, "open", func() bool { return false })
	advanced.Children(func() {
		label := providerCopy("供应商密钥与模型探测", "Provider keys and model discovery")
		if ui.Button(c, label).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Clicked() {
			*open = !*open
		}
		if *open {
			referencedCredentialsPane(c, m.ReferencedCredentials, "", fg, muted)
			providerDiscoveryPane(c, a.ProviderDiscovery, fg, muted)
		}
	})
	// Session choice belongs to the current conversation and stays secondary to
	// provider configuration, preserving the existing apply contract.
	if a.SessionID != "" && a.Selected != nil {
		bodyRosterRow(c, providerCopy("当前任务模型", "Current task model"), func() {
			if len(a.Labels) > 0 {
				ui.Select(c, a.Selected, a.Labels).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13))
			} else {
				ui.Text(c, i18n.T(i18n.KeySettingsModelsEmpty)).TextColor(muted)
			}
			if a.OnApply != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(len(a.Labels) == 0).Clicked() {
				a.OnApply()
			}
		}, fg, muted)
	}
}
func modelProviderTitle(namespace string) string {
	switch namespace {
	case "llm-deepseek", "llm-deepseek-api-key":
		return "DeepSeek API"
	case "llm-deepseek-account":
		return providerCopy("DeepSeek 账号", "DeepSeek Account")
	case "llm-pi-ai":
		return providerCopy("自定义供应商", "Custom providers")
	case "subagent-model-selection-settings":
		return providerCopy("子代理模型", "Subagent models")
	default:
		return namespace
	}
}

// modelAddProviderCard is the reference's dashed full-width add row. Providers
// still come from the Host, so without a selection target the card explains
// itself instead of implying a local form.
func modelAddProviderCard(c *ui.Context, m Model, a *ModelsAccess, fg, muted ui.Color) {
	// The catalog text already carries a leading plus; the button draws its own icon.
	label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(i18n.T(i18n.KeySettingsModelsAddProvider)), "+"))
	target := ""
	if m.OnSelectNamespace != nil {
		target = "llm-pi-ai"
	}
	disabled := target == ""
	button := modelPill(c.Key("model-add-provider"), "", false, false, disabled).FillWidth().Label(label)
	if disabled {
		button.Tooltip(i18n.T(i18n.KeySettingsModelsAddUnavailable))
	}
	button.Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Icon(c, bodyIcon("plus")).Size(16, 16)
		ui.Text(c, label).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22))
	})
	if !disabled && button.Clicked() {
		m.OnSelectNamespace(target)
	}
}
