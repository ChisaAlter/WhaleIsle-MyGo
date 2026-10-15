package settings

import (
	"github.com/egoist/mygo/ui"
	"strings"

	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

// PluginRow is one installed plugin for the dedicated settings page.
type PluginRow struct {
	Name           string
	Status         string
	Disabled       bool // authoritative Host bundle state
	CanToggle      bool
	ReadOnlyReason string
	Error          string
}

// PluginsAccess is the dedicated Plugins settings page (pluginManager/listPlugins).
type PluginsAccess struct {
	Inventory        *PluginInventoryAccess
	Rows             []PluginRow
	HasListRPC       bool
	Err              string
	LastAction       string
	ClientReady      bool
	Loading          bool
	Busy             bool
	OperationMessage string
	Activation       marketstore.Activation
	CanRestart       bool
	OnRestart        func()
	OnRefresh        func()
	OnOpenMarket     func()
	// OnSetDisabled changes the Host bundle and surfaces its ChangeResult.
	OnSetDisabled func(name string, disabled bool)
	OnInspect     func(name string)
	OnOpenView    func(name string)
}

func pluginsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Plugins == nil {
		return
	}
	a := m.Plugins
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsHint)).FontSize(12).TextColor(muted)
	ui.Text(c, i18n.T(i18n.KeySettingsPluginsMarketHint)).FontSize(11).TextColor(muted)
	pluginInventoryPane(c, a.Inventory, fg, muted)
	ui.Text(c, pluginsText("已安装插件包", "Installed plugin bundles")).FontSize(14).Bold().TextColor(fg)
	if !a.ClientReady {
		ui.Text(c, pluginsText("运行时未连接，请连接后管理插件。", "Runtime disconnected. Connect to manage plugins.")).FontSize(12).TextColor(muted)
	}
	if a.Loading || a.Busy {
		ui.Text(c, i18n.T(i18n.KeyMarketLoading)).FontSize(12).TextColor(muted)
	}
	if a.OperationMessage != "" {
		ui.Text(c, a.OperationMessage).FontSize(12).TextColor(fg)
	}
	if a.Activation.Pending {
		message := pluginsText("插件变更已保存，等待重启生效。", "Plugin changes are saved and await a runtime restart.")
		if a.Activation.Failed {
			message = pluginsText("重启失败，已保存的变更仍待生效。", "Restart failed. Saved changes still await activation.")
		}
		if a.Activation.Restarting {
			message = pluginsText("正在重启运行时…", "Restarting the runtime…")
		}
		ui.Text(c, message).FontSize(12).TextColor(muted)
		if a.CanRestart && !a.Busy && !a.Activation.Restarting && a.OnRestart != nil {
			if ui.Button(c, pluginsText("重启运行时并应用", "Restart runtime and apply")).Clicked() {
				a.OnRestart()
			}
		}
	}
	if !a.HasListRPC {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsNoRPC)).FontSize(12).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	if len(a.Rows) == 0 && !a.Loading {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsEmpty)).FontSize(12).TextColor(muted)
	} else {
		for _, row := range a.Rows {
			r := row
			if r.ReadOnlyReason != "" {
				ui.Text(c, r.Name+": "+r.ReadOnlyReason).FontSize(11).TextColor(muted)
			}
			if r.Error != "" {
				ui.Text(c, r.Name+": "+r.Error).FontSize(11).TextColor(fg)
			}
			desktopRow(c, emptyDash(r.Name), func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if r.Status != "" {
						ui.Text(c, r.Status).FontSize(11).TextColor(muted).Grow(1)
					} else {
						ui.Box(c).Grow(1).MinWidth(0)
					}
					if a.OnOpenView != nil && a.ClientReady && !a.Loading && !a.Busy && r.Name != "" && !r.Disabled {
						if ui.Button(c, i18n.T(i18n.KeyWebViewOpenHost)).FontSize(11).Clicked() {
							a.OnOpenView(r.Name)
						}
					}
					if a.OnInspect != nil && a.ClientReady && !a.Loading && !a.Busy && r.Name != "" {
						if ui.Button(c, i18n.T(i18n.KeySettingsPluginsInspect)).FontSize(11).Clicked() {
							a.OnInspect(r.Name)
						}
					}
					if a.OnSetDisabled != nil && a.ClientReady && !a.Loading && !a.Busy && a.Err == "" && r.CanToggle {
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
		if !a.Loading && !a.Busy && a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnOpenMarket != nil && ui.Button(c, i18n.T(i18n.KeySettingsPluginsOpenMarket)).Clicked() {
			a.OnOpenMarket()
		}
	})
}

func pluginsText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}
