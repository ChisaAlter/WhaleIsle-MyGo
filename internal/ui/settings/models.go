package settings

import (
	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"whaleisle/internal/ui/chrome"
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
	Labels               []string
	Selected             *string
	SessionID            string
	Err                  string
	LastAction           string
	OnConfigure          func(namespace string)
	OnRefresh            func()
	OnApply              func()
}

func modelsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Models == nil {
		return
	}
	a := m.Models
	ui.Text(c, providerCopy("模型", "Models")).FontSize(fonttheme.InterfaceSize(16)).Bold().TextColor(fg)
	ui.Text(c, providerCopy("管理模型供应商、模型目录与视觉模型。", "Manage providers, model catalogs and vision models.")).FontSize(fonttheme.InterfaceSize(14)).TextColor(muted)
	if a.Err != "" {
		ui.Text(c, a.Err).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
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
			providerCard := ui.Column(c.Key("model-provider-" + namespace.NS)).FillWidth().MinWidth(0)
			editing := ui.Local(providerCard, "editing", func() bool { return false })
			providerCard.Children(func() {
				bodyCard(c, m, func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
							ui.Text(c, modelProviderTitle(namespace.NS)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
							ui.Text(c, namespace.NS).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
						})
						if widgets.Button(c, fonttheme.Resolve(m.ThemeID).Active(m.SystemDark), providerCopy("编辑", "Edit"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: (m.OnSelectNamespace == nil && a.OnConfigure == nil) || state.Loading || state.Saving}).Clicked() {
							if a.OnConfigure != nil {
								*editing = !*editing
								a.OnConfigure(namespace.NS)
							} else if m.OnSelectNamespace != nil {
								m.OnSelectNamespace(namespace.NS)
							}
						}
					})
					if *editing && a.OnConfigure != nil {
						modelNamespaceBody(c, m, namespace.NS)
					}
				})
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
		if ui.Button(c, label).Clicked() {
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
				ui.Select(c, a.Selected, a.Labels).FillWidth()
			} else {
				ui.Text(c, i18n.T(i18n.KeySettingsModelsEmpty)).TextColor(muted)
			}
			if a.OnApply != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Disabled(len(a.Labels) == 0).Clicked() {
				a.OnApply()
			}
		}, fg, muted)
	}
	if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
		a.OnRefresh()
	}
}
func modelProviderTitle(namespace string) string {
	switch namespace {
	case "llm-deepseek", "llm-deepseek-api-key":
		return "DeepSeek API"
	case "llm-deepseek-account":
		return providerCopy("DeepSeek 账户", "DeepSeek Account")
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
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	label := i18n.T(i18n.KeySettingsModelsAddProvider)
	card := ui.Column(c.Key("model-add-provider")).FillWidth().MinWidth(0).Height(52).
		Border(1, chrome.Color(colors.Border)).Radius(12).Justify(ui.Center).AlignItems(ui.Center)
	target := ""
	if m.OnSelectNamespace != nil {
		target = "llm-pi-ai"
	}
	disabled := target == ""
	if disabled {
		card.Tooltip(i18n.T(i18n.KeySettingsModelsAddUnavailable))
	}
	card.Children(func() {
		button := ui.Button(c, "").Label(label).Height(24).Padding(0, 10).BorderWidth(0).
			Background(ui.Transparent).TextColor(muted).Disabled(disabled)
		button.Children(func() {
			ui.Text(c, label).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
		})
		if !disabled && button.Clicked() {
			m.OnSelectNamespace(target)
		}
	})
}
