package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// PluginRow is one installed plugin for the dedicated settings page.
type PluginRow struct {
	Name     string
	Status   string
	Disabled bool // config.DisabledPlugins
}

// PluginsAccess is the dedicated Plugins settings page (pluginManager/listPlugins).
type PluginsAccess struct {
	Rows         []PluginRow
	HasListRPC   bool
	Err          string
	LastAction   string
	OnRefresh    func()
	OnOpenMarket func()
	// OnSetDisabled toggles go DisabledPlugins for a plugin name (restart may be required).
	OnSetDisabled func(name string, disabled bool)
	OnInspect     func(name string)
}

func pluginsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Plugins == nil {
		return
	}
	a := m.Plugins
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsHint)).FontSize(12).TextColor(muted)
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsMarketHint)).FontSize(11).TextColor(muted)
	if !a.HasListRPC {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsNoRPC)).FontSize(12).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	if len(a.Rows) == 0 {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsEmpty)).FontSize(12).TextColor(muted)
	} else {
		for _, row := range a.Rows {
			r := row
			desktopRow(c, emptyDash(r.Name), func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if r.Status != "" {
						ui.Text(c, r.Status).FontSize(11).TextColor(muted).Grow(1)
					} else {
						ui.Box(c).Grow(1).MinWidth(0)
					}
					if a.OnInspect != nil && r.Name != "" {
						if ui.Button(c, i18n.T(i18n.KeySettingsPluginsInspect)).FontSize(11).Clicked() {
							a.OnInspect(r.Name)
						}
					}
					if a.OnSetDisabled != nil {
						label := i18n.T(i18n.KeySettingsPluginsEnable)
						if !r.Disabled {
							label = i18n.T(i18n.KeySettingsPluginsDisable)
						}
						if ui.Button(c, label).FontSize(11).Clicked() {
							a.OnSetDisabled(r.Name, !r.Disabled)
						}
					}
				})
			}, fg, muted)
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnOpenMarket != nil && ui.Button(c, i18n.T(i18n.KeySettingsPluginsOpenMarket)).Clicked() {
			a.OnOpenMarket()
		}
	})
}
