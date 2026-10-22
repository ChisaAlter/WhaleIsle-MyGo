// Package boot draws the native harness startup page: status, recent log
// lines, and retry / export actions. M2 wires this into cmd/whaleisle while
// the supervisor starts dsh web.
package boot

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model is the boot page's observable state (filled by the shell).
type Model struct {
	Status           harness.Status
	Logs             []string // newest last; view shows a tail
	ThemeID          string
	SystemDark       bool
	OnCancelRecovery func()
	OnRetry          func()
	OnExport         func()
	OnConfigure      func()
}

const logTail = 40

// i18n keys for the boot page.
const (
	KeyTitle                = "boot.title"
	KeyStarting             = "boot.starting"
	KeyReady                = "boot.ready"
	KeyFailed               = "boot.failed"
	KeyStopped              = "boot.stopped"
	KeyRetry                = "boot.retry"
	KeyExportLog            = "boot.export"
	KeyWaiting              = "boot.waiting"
	KeyPlugins              = "boot.plugins"
	KeyPluginsDetail        = "boot.plugins_detail"
	KeyPluginsIndeterminate = "boot.plugins_indeterminate"
)

// View paints the boot page inside the main window body (below the title bar):
// the reference page's layered scene with a centred brand block, status line and
// the tail of the startup log.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)

	root := ui.Column(c).Key("boot-root").Fill().Background(bg)
	root.Draw(bootBackdrop(bg, chrome.Color(colors.Accent), m.SystemDark))
	drawer := ui.Local(root, "boot-log-drawer", func() bool { return false })
	root.Children(func() {
		ui.Column(c).Key("boot-center").Fill().AlignItems(ui.Center).Justify(ui.Center).Padding(24, 32, 28, 32).Children(func() {
			ui.Column(c).Width(460).MaxWidthPercent(92).Gap(10).AlignItems(ui.Center).Children(func() {
				if bmp := chrome.BrandAvatar(); bmp != nil {
					ui.Image(c, bmp).Size(64, 64).Fit(ui.Contain)
				}
				ui.Text(c, i18n.T(KeyTitle)).FontSize(theme.InterfaceSize(24)).FixedLineHeight(theme.InterfaceSize(30)).
					FontWeight(600).TextColor(fg)
				ui.Text(c, "BASED ON DEEPSEEK HARNESS").FontSize(theme.InterfaceSize(8)).FixedLineHeight(theme.InterfaceSize(10)).
					FontWeight(500).LetterSpacing(.32).TextColor(muted)
				ui.Box(c).Height(10)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if m.Status.State == harness.Starting && !m.Status.RecoveryCancelled {
						ui.Spinner(c).Size(14, 14)
					}
					ui.Text(c, statusLine(m.Status)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(fg)
				})
				if line := pluginProgressLine(m.Status); line != "" {
					ui.Text(c, line).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				if m.Status.State == harness.Failed {
					hint := "请检查启动器中的工作区、Node.js 与 Host 路径；修正后重试，或导出日志排查。"
					if i18n.Locale() == "en" {
						hint = "Check the workspace, Node.js and Host paths in the launcher, then retry or export the logs."
					}
					ui.Text(c, hint).FontSize(theme.InterfaceSize(12)).TextColor(muted).MaxLines(3)
				}

				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 0, 0, 0).Children(func() {
					if m.OnConfigure != nil {
						label := "打开启动器"
						if i18n.Locale() == "en" {
							label = "Open launcher"
						}
						if ui.Button(c, label).Clicked() {
							m.OnConfigure()
						}
					}
					cancelRecoveryAction(c, m)
					if m.OnRetry != nil && ui.Button(c, i18n.T(KeyRetry)).Clicked() {
						m.OnRetry()
					}
					if m.OnExport != nil && ui.Button(c, i18n.T(KeyExportLog)).Clicked() {
						m.OnExport()
					}
					if ui.Button(c, bootText("全部启动日志", "All startup logs")).Clicked() {
						*drawer = true
					}
				})

				// Ticker: the newest line, or the waiting placeholder before logs.
				last := i18n.T(KeyWaiting)
				if len(m.Logs) > 0 {
					last = m.Logs[len(m.Logs)-1]
				}
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(10, 0, 0, 0).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(muted.Alpha(.5))
					ui.Text(c, last).FontSize(theme.InterfaceSize(11)).TextColor(muted).MinWidth(0).Grow(1).SingleLine().Ellipsis("…")
				})
			})
		})
		ui.DialogBase(c, drawer, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, .4))
			w, h := c.Size()
			panel.Width(max(0, min(900, w-32))).Height(max(0, min(600, h-32))).Background(bg).Border(1, border).
				Radius(20).Clip().Padding(20).Children(func() {
				ui.Text(c, fmt.Sprintf("%s · %d", bootText("启动日志", "Startup logs"), len(m.Logs))).Bold().TextColor(fg)
				if ui.Button(c, bootText("关闭日志", "Close logs")).Clicked() {
					*drawer = false
				}
				ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
					ui.Column(c).Gap(4).Children(func() {
						for _, line := range m.Logs {
							ui.Text(c, line).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
						}
						if len(m.Logs) == 0 {
							ui.Text(c, i18n.T(KeyWaiting)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
					})
				})
			})
		})
	})
}


func pluginProgressLine(st harness.Status) string {
	if st.State != harness.Starting {
		return ""
	}
	prog := st.PluginLoad
	if prog.Total > 0 {
		line := fmt.Sprintf(i18n.T(KeyPlugins), prog.Ready, prog.Total)
		if prog.Detail != "" {
			line += " — " + fmt.Sprintf(i18n.T(KeyPluginsDetail), prog.Detail)
		}
		return line
	}
	if prog.Phase == "runtime" {
		return i18n.T(KeyPluginsIndeterminate)
	}
	return ""
}

func statusLine(st harness.Status) string {
	if st.RecoveryCancelled {
		if i18n.Locale() == "en" {
			return "Automatic recovery cancelled. You can retry manually."
		}
		return "已取消自动恢复，可手动重试。"
	}
	switch st.State {
	case harness.Starting:
		return i18n.T(KeyStarting)
	case harness.Ready:
		return i18n.T(KeyReady)
	case harness.Failed:
		if st.Err != nil {
			return i18n.T(KeyFailed) + ": " + st.Err.Error()
		}
		return i18n.T(KeyFailed)
	default:
		return i18n.T(KeyStopped)
	}
}

// RecoveryControl exposes boot recovery in shells that keep their navigation
// and dialogue mounted while the Host is starting or unavailable.
func RecoveryControl(c *ui.Context, m Model) {
	bootOverlay(c, m)
}

func cancelRecoveryAction(c *ui.Context, m Model) {
	if !m.Status.RecoveryActive || m.Status.RecoveryCancelled || m.OnCancelRecovery == nil {
		return
	}
	label := "取消自动恢复"
	if i18n.Locale() == "en" {
		label = "Cancel automatic recovery"
	}
	if ui.Button(c, label).Clicked() {
		m.OnCancelRecovery()
	}
}
