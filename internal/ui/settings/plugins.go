package settings

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
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
	Tabs             []PluginSettingsTab
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
	// OnOpenManager opens the Host plugin manager, not the curated marketplace.
	OnOpenManager func()
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
	// The source's .intro copy is 13px on the normal line box (~17.3 DIP).
	ui.Text(c, i18n.T(KeyPass4ABuiltinHint)).FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(17.3)).TextColor(muted)
	pluginSettingsTabs(c, a, fg, muted)
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if kButtonDisabled(c, pluginsText("管理插件包", "Manage plugin packages"), a.OnOpenManager == nil).Clicked() && a.OnOpenManager != nil {
			a.OnOpenManager()
		}
		if kButtonDisabled(c, i18n.T(i18n.KeySettingsPluginsOpenMarket), a.OnOpenMarket == nil).Clicked() && a.OnOpenMarket != nil {
			a.OnOpenMarket()
		}
	})
}

func pluginPackages(c *ui.Context, a *PluginsAccess, fg, muted ui.Color) {
	builtinPluginGrid(c, a, fg, muted)
	ui.Text(c, pluginsText("已安装插件包", "Installed plugin bundles")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	if !a.ClientReady {
		ui.Text(c, pluginsText("运行时未连接，请连接后管理插件。", "Runtime disconnected. Connect to manage plugins.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	opFailed := a.OperationMessage != "" && isPluginOperationFailure(a.OperationMessage)
	settingsAlert(c, "plugins:79-error", opFailed, errcopy.SentenceText(a.OperationMessage), a.OperationMessage)
	settingsNotice(c, "plugins:79", a.OperationMessage != "" && !opFailed, func() string { return a.OperationMessage })
	if a.Activation.Pending {
		message := pluginsText("插件变更已保存，等待重启生效。", "Plugin changes are saved and await a runtime restart.")
		if a.Activation.Failed {
			message = pluginsText("重启失败，已保存的变更仍待生效。", "Restart failed. Saved changes still await activation.")
		}
		if a.Activation.Restarting {
			message = pluginsText("正在重启运行时…", "Restarting the runtime…")
		}
		ui.Text(c, message).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if a.CanRestart && !a.Busy && !a.Activation.Restarting && a.OnRestart != nil {
			if kPrimary(c, pluginsText("重启运行时并应用", "Restart runtime and apply"), false).AlignSelf(ui.Start).Clicked() {
				a.OnRestart()
			}
		}
	}
	if !a.HasListRPC {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsNoRPC)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	settingsAlert(c, "plugins:100", a.Err != "", providerCopy("没能读取插件列表，请重试。", "Plugins could not be read. Try again."), a.Err)
	settingsNotice(c, "plugins:103", a.LastAction != "", func() string { return settingsActionText(a.LastAction) })
	if len(a.Rows) == 0 && !a.Loading {
		ui.Text(c, i18n.T(i18n.KeySettingsPluginsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	} else {
		for _, row := range a.Rows {
			r := row
			if r.ReadOnlyReason != "" {
				ui.Text(c, r.Name+": "+pluginReasonLabel(r.ReadOnlyReason)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			if r.Error != "" {
				ui.Text(c, r.Name+": "+r.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
			}
			desktopRow(c, emptyDash(r.Name), func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if r.Status != "" {
						ui.Text(c, r.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Grow(1)
					} else {
						ui.Box(c).Grow(1).MinWidth(0)
					}
					if a.OnOpenView != nil && a.ClientReady && !a.Loading && !a.Busy && r.Name != "" && !r.Disabled {
						if kButton(c, i18n.T(i18n.KeyWebViewOpenHost)).Clicked() {
							a.OnOpenView(r.Name)
						}
					}
					if a.OnInspect != nil && a.ClientReady && !a.Loading && !a.Busy && r.Name != "" {
						if kButton(c, i18n.T(i18n.KeySettingsPluginsInspect)).Clicked() {
							a.OnInspect(r.Name)
						}
					}
					if a.OnSetDisabled != nil && a.ClientReady && !a.Loading && !a.Busy && a.Err == "" && r.CanToggle {
						label := i18n.T(i18n.KeySettingsPluginsEnable)
						if !r.Disabled {
							label = i18n.T(i18n.KeySettingsPluginsDisable)
						}
						if kButton(c, label).Clicked() {
							a.OnSetDisabled(r.Name, !r.Disabled)
						}
					}
				})
			}, fg, muted)
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		if !a.Loading && !a.Busy && a.OnRefresh != nil && kButton(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.OnOpenMarket != nil && kButton(c, i18n.T(i18n.KeySettingsPluginsOpenMarket)).Clicked() {
			a.OnOpenMarket()
		}
	})
}

func pluginsText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func isPluginOperationFailure(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return false
	}
	for _, token := range []string{
		"失败", "无法", "不可用", "错误", "异常", "过期", "覆盖", "未确认", "未连接", "中断",
		"fail", "error", "cannot", "could not", "unable", "unavailable",
		"expired", "overridden", "unconfirmed", "unknown", "denied", "refused",
		"timeout", "timed out", "exit status", "too late", "not found", "not connected",
		"invalid", "enoent", "eacces", "eperm", "econn", "broken pipe", "eof",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// builtinPluginGrid is the built-in plugin list as the installed build shows it: a
// two column grid of 60 DIP cards, or a 2×2 skeleton while the list loads.
func builtinPluginGrid(c *ui.Context, a *PluginsAccess, fg, muted ui.Color) {
	if a.Loading && len(a.Rows) == 0 {
		ui.Text(c, i18n.T(KeyPass4ABuiltinLoading)).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		grid := ui.Grid(c.Key("builtin-plugins-loading")).FillWidth().MinWidth(0).Gap(12)
		grid.Columns(2).Children(func() {
			for index := 0; index < 4; index++ {
				ui.Column(c.Key(fmt.Sprintf("builtin-skeleton-%d", index))).MinWidth(0).Height(60).Gap(8).Padding(12, 16).
					Radius(12).Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .2)).Justify(ui.Center).Children(func() {
					ui.Box(c).Width(120).Height(10).Shrink(0).Radius(4).Background(ui.RGBA(muted.R, muted.G, muted.B, .1))
					ui.Box(c).Width(180).Height(8).Shrink(0).Radius(4).Background(ui.RGBA(muted.R, muted.G, muted.B, .1))
				})
			}
		})
		return
	}
	if len(a.Rows) == 0 {
		return
	}
	grid := ui.Grid(c.Key("builtin-plugins")).FillWidth().MinWidth(0).Gap(12)
	grid.Columns(2).Children(func() {
		for _, row := range a.Rows {
			plugin := row
			detail := pluginStatusLabel(plugin)
			if plugin.ReadOnlyReason != "" {
				detail = pluginReasonLabel(plugin.ReadOnlyReason)
			}
			card := chrome.FrostSurface(ui.Column(c.Key("builtin-plugin/"+plugin.Name)).MinWidth(0).Height(60).Gap(2).Padding(12, 16),
				fonttheme.Default().Active(c.Theme().Dark), c.Theme().Dark, c.Theme().Surface, 12).Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .2)).Justify(ui.Center)
			card.Children(func() {
				ui.Text(c, plugin.Name).FillWidth().FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg).SingleLine().Ellipsis("…")
				if detail != "" {
					ui.Text(c, detail).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Ellipsis("…")
				}
			})
		}
	})
}

// pluginStatusLabel turns a Host status value into its localized wording; the raw
// value never reaches the card.
// pluginReasonLabel localizes the Host's read-only reason token. Unknown
// reasons stay as-is so a new Host reason is never hidden.
func pluginReasonLabel(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "management-required":
		return pluginsText("由桌面宿主管理", "Managed by the desktop host")
	case "read-only":
		return pluginsText("只读", "Read-only")
	case "":
		return ""
	default:
		return reason
	}
}

func pluginStatusLabel(row PluginRow) string {
	if row.Disabled {
		return pluginsText("已停用", "Disabled")
	}
	switch strings.ToLower(strings.TrimSpace(row.Status)) {
	case "enabled", "active", "running":
		return pluginsText("已启用", "Enabled")
	case "disabled", "inactive", "stopped":
		return pluginsText("已停用", "Disabled")
	case "failed", "error", "broken":
		return pluginsText("加载失败", "Failed to load")
	case "conditional", "on-demand":
		return pluginsText("按条件启用", "Conditional")
	case "":
		return ""
	default:
		return pluginsText("状态未知", "Unknown state")
	}
}
