package boot

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

func bootText(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}

// The shell remains mounted while a real supervisor transition opens/closes
// this scene. Collapsible and Spinner respect the native reduced-motion policy.
func startupScene(c *ui.Context, m Model) {
	active := m.Status.State == harness.Starting || m.Status.State == harness.Failed || m.Status.RecoveryActive || m.Status.RecoveryCancelled
	parts := ui.CollapsibleBase(c.Key("boot-state-scene"), &active)
	parts.Panel(func() {
		seat := ui.Column(c).FillWidth().Padding(12).Gap(8)
		drawer := ui.Local(seat, "all-logs", func() bool { return false })
		seat.Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				if m.Status.State == harness.Starting && !m.Status.RecoveryCancelled {
					ui.Spinner(c).Size(20, 20)
				}
				ui.Text(c, i18n.T(i18n.KeyAppName)+" · "+i18n.T(KeyTitle)).Bold()
			})
			ui.Text(c, statusLine(m.Status)).Selectable()
			if line := pluginProgressLine(m.Status); line != "" {
				ui.Text(c, line)
			}
			ui.Row(c).Gap(8).Children(func() {
				cancelRecoveryAction(c, m)
				if bootActionsVisible(m.Status) && m.OnRetry != nil && ui.Button(c, i18n.T(KeyRetry)).Disabled(m.Status.RecoveryActive && !m.Status.RecoveryScheduled && !m.Status.RecoveryCancelled).Clicked() {
					m.OnRetry()
				}
				if m.Status.State == harness.Failed && !m.Status.RecoveryActive && m.OnConfigure != nil && ui.Button(c, bootText("打开启动器", "Open launcher")).Clicked() {
					m.OnConfigure()
				}
				if bootActionsVisible(m.Status) && m.OnExport != nil && ui.Button(c, i18n.T(KeyExportLog)).Clicked() {
					m.OnExport()
				}

			})
			bootTicker(c, m.Logs, c.Theme().TextMuted, func() { *drawer = true })
			ui.DialogBase(c, drawer, func(backdrop, panel ui.Element) {
				chrome.RevealOverlay(c, backdrop, panel, ui.RGBA(0, 0, 0, .4))
				w, h := c.Size()
				panel.Width(max(0, min(900, w-32))).Height(max(0, min(600, h-32))).Background(c.Theme().Background).Radius(20).Clip().Padding(20).Children(func() {
					ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Text(c, fmt.Sprintf("%s · %d", bootText("启动日志", "Startup logs"), len(m.Logs))).Bold().Grow(1)
						if ui.Button(c, "×").Size(28, 28).Label(bootText("关闭日志", "Close logs")).AutoFocus().Clicked() {
							*drawer = false
						}
					})
					ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
						ui.Column(c).Gap(4).Children(func() {
							paintBootLogLines(c, m.Logs, c.Theme().TextMuted, ui.RGBA(220, 38, 38, 1))
							if len(m.Logs) == 0 {
								ui.Text(c, i18n.T(KeyWaiting))
							}
						})
					})
				})
			})
		})
	})
}
