package settings

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/ui/i18n"
)

// ModelsAccess is the dedicated Models settings page (session/modelCatalog).
type ModelsAccess struct {
	ProviderDiscovery *ProviderDiscoveryAccess
	ProviderProfiles  *ProviderProfilesAccess
	Vision            *VisionAccess
	Labels            []string
	Selected          *string
	SessionID         string
	Err               string
	LastAction        string
	OnRefresh         func()
	OnApply           func()
}

func modelsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Models == nil {
		return
	}
	a := m.Models
	if m.Store != nil && m.OnSelectNamespace != nil {
		for _, namespace := range m.Store.Snapshot().Namespaces {
			if SectionForNamespace(namespace.NS) != SectionModels || namespace.NS == "llm-vision-fallback" {
				continue
			}
			ns := namespace.NS
			if ui.Button(c, providerCopy("编辑 Provider / 模型配置：", "Edit provider / model settings: ")+ns).Clicked() {
				m.OnSelectNamespace(ns)
			}
		}
	}
	referencedCredentialsPane(c, m.ReferencedCredentials, "", fg, muted)
	providerDiscoveryPane(c, a.ProviderDiscovery, fg, muted)
	visionPane(c, a.Vision, fg, muted)
	providerProfilesPane(c, a.ProviderProfiles, fg, muted)
	ui.Text(c, i18n.T(i18n.KeySettingsModelsTitle)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsModelsHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if a.SessionID != "" {
		ui.Text(c, i18n.T(i18n.KeySettingsModelsSession)+": "+a.SessionID).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	} else {
		ui.Text(c, i18n.T(i18n.KeySettingsModelsNoSession)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsModelsPicker), func() {
		if a.Selected == nil {
			return
		}
		opts := a.Labels
		if len(opts) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsModelsEmpty)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Select(c, a.Selected, opts).FillWidth()
	}, fg, muted)
	ui.Row(c).Gap(8).Children(func() {
		if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnApply != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
			a.OnApply()
		}
	})
}
