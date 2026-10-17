package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// ModelsAccess is the dedicated Models settings page (session/modelCatalog).
type ModelsAccess struct {
	Labels     []string
	Selected   *string
	SessionID  string
	Err        string
	LastAction string
	OnRefresh  func()
	OnDiscover func()
	OnApply    func()
}

func modelsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Models == nil {
		return
	}
	a := m.Models
	ui.Text(c, i18n.T(i18n.KeySettingsModelsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsModelsHint)).FontSize(12).TextColor(muted)
	if a.SessionID != "" {
		ui.Text(c, i18n.T(i18n.KeySettingsModelsSession)+": "+a.SessionID).FontSize(11).TextColor(muted)
	} else {
		ui.Text(c, i18n.T(i18n.KeySettingsModelsNoSession)).FontSize(11).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsModelsPicker), func() {
		if a.Selected == nil {
			return
		}
		opts := a.Labels
		if len(opts) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsModelsEmpty)).FontSize(11).TextColor(muted)
			return
		}
		ui.Select(c, a.Selected, opts).FillWidth()
	}, fg, muted)
	ui.Row(c).Gap(8).Children(func() {
		if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnDiscover != nil && ui.Button(c, i18n.T(i18n.KeySettingsModelsDiscover)).Clicked() {
			a.OnDiscover()
		}
		if a.OnApply != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
			a.OnApply()
		}
	})
}
