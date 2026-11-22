package launcher

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type MaintenancePage uint8

const (
	MaintenanceComponents MaintenancePage = iota
	MaintenanceImport
	MaintenanceDiagnostics
	MaintenancePreferences
)

// Maintenance renders one explicitly requested task under the workbench title
// bar. It has no home page, rail, or window ownership.
func Maintenance(c *ui.Context, page MaintenancePage, m Model) {
	if m.SystemDark {
		applyDarkTokens()
	} else {
		applyLightTokens()
	}
	var title, detail string
	var body func(*ui.Context, Model)
	switch page {
	case MaintenanceComponents:
		title, detail, body = i18n.T(i18n.KeyLauncherTabComponents), i18n.T(i18n.KeyLauncherComponentsLede), componentsPanel
	case MaintenanceImport:
		title, body = releaseText("导入数据", "Import data"), importPanel
	case MaintenanceDiagnostics:
		title, body = releaseText("启动诊断", "Startup diagnostics"), pluginsPanel
	case MaintenancePreferences:
		title, body = releaseText("更新偏好与卸载", "Update preferences and uninstall"), settingsPanel
	}
	chrome.FrostSurface(ui.Column(c).Fill().MinWidth(0), launcherColors, m.SystemDark, colBase, 0).Children(func() {
		ui.Row(c).FillWidth().Wrap().AlignItems(ui.Center).Gap(12).Padding(16, 24).
			BorderWidth(0, 0, 1, 0).BorderColor(colBorderL2).Children(func() {
			ui.Text(c, title).Grow(1).FontSize(theme.InterfaceSize(18)).FontWeight(600).TextColor(colPrimary)
			if m.OnClose != nil && ghostSmall(c, releaseText("返回工作台", "Back to workbench")).Clicked() {
				m.OnClose()
			}
		})
		ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(1040).MinWidth(0).Padding(20, 24).Gap(14).Children(func() {
				if detail != "" {
					ui.Text(c, detail).FontSize(theme.InterfaceSize(13)).TextColor(colSecondary)
				}
				if page == MaintenanceDiagnostics && m.StatusNote != "" {
					ui.Text(c, m.StatusNote).FillWidth().Selectable().TextColor(colSecondary)
				}
				body(c, m)
			})
		})
	})
}
