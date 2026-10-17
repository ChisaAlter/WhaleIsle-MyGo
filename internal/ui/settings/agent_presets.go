package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// AgentPresetsAccess is the dedicated Agent Presets settings page (agentPresets/list + select).
type AgentPresetsAccess struct {
	Labels     []string
	Selected   *string
	SessionID  string
	Err        string
	LastAction string
	DocPreview string
	OnRefresh  func()
	OnApply    func()
	OnRead     func()
}

func agentPresetsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.AgentPresets == nil {
		return
	}
	a := m.AgentPresets
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsHint)).FontSize(12).TextColor(muted)
	if a.SessionID != "" {
		ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsSession)+": "+a.SessionID).FontSize(11).TextColor(muted)
	} else {
		ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsNoSession)).FontSize(11).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsAgentPresetsPicker), func() {
		if a.Selected == nil {
			return
		}
		opts := a.Labels
		if len(opts) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsEmpty)).FontSize(11).TextColor(muted)
			return
		}
		ui.Select(c, a.Selected, opts).FillWidth()
	}, fg, muted)
	ui.Row(c).Gap(8).Children(func() {
		if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnRead != nil && ui.Button(c, i18n.T(i18n.KeySettingsAgentPresetsRead)).Clicked() {
			a.OnRead()
		}
		if a.OnApply != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
			a.OnApply()
		}
	})
	if a.DocPreview != "" {
		ui.Text(c, a.DocPreview).FontSize(11).TextColor(muted)
	}
}
