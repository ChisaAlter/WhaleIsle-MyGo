// Package boot draws the native harness startup page: status, recent log
// lines, and retry / export actions. M2 wires this into cmd/whaleisle while
// the supervisor starts dsh web.
package boot

import (
	"fmt"
	"regexp"

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
// the reference page's layered scene, the brand block lifted over the horizon,
// the status line with its actions and the newest log line at the foot.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)
	alert := chrome.Color(colors.Danger)
	failed := m.Status.State == harness.Failed

	root := ui.Column(c).Key("boot-root").Fill().Background(bg)
	root.Draw(bootScene(sceneSpec{
		background: bg, accent: accent, ink: fg, alert: alert,
		dark: m.SystemDark, failed: failed,
	}, !c.Preferences().ReduceMotion))
	drawer := ui.Local(root, "boot-log-drawer", func() bool { return false })
	root.Children(func() {
		bootCore(c, m, fg, muted, alert, func() { *drawer = true })
		// The foot of the page keeps the ticker's light while the sea darkens
		// under it: --boot-under, the accent lifted toward white.
		tint := float32(.85)
		if m.SystemDark {
			tint = .45
		}
		bootTicker(c, m.Logs, accent.Mix(ui.RGBA(255, 255, 255, 1), tint), func() { *drawer = true })
		ui.DialogBase(c, drawer, func(backdrop, panel ui.Element) {
			// boot.css:596-604 fades the log drawer in (drawer-fade) and rises it
			// 14 DIP (drawer-rise); the fade is the part the hand-run overlay
			// token can play here.
			chrome.RevealOverlay(c, backdrop, panel, ui.RGBA(0, 0, 0, .4))
			w, h := c.Size()
			panel.Width(max(0, min(900, w-32))).Height(max(0, min(600, h-32))).Background(bg).Border(1, border).
				Radius(20).Clip().Padding(20).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, fmt.Sprintf("%s · %d", bootText("启动日志", "Startup logs"), len(m.Logs))).Bold().TextColor(fg).Grow(1)
					if ui.Button(c, "×").Size(28, 28).Label(bootText("关闭日志", "Close logs")).AutoFocus().Clicked() {
						*drawer = false
					}
				})
				ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
					ui.Column(c).Gap(4).Children(func() {
						paintBootLogLines(c, m.Logs, muted, alert)
						if len(m.Logs) == 0 {
							ui.Text(c, i18n.T(KeyWaiting)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
					})
				})
			})
		})
	})
}

// bootCore is .core: the brand block, the status line, the failure copy and the
// actions. Its foot is padded by 26vh so the block sits over the sky and the
// horizon crosses below it instead of through it (boot.css:207-217); a short
// window keeps 20vh and 34 DIP letters (boot.css:453-459).
func bootCore(c *ui.Context, m Model, fg, muted, alert ui.Color, openLog func()) {
	_, height := c.Size()
	short := height < 700
	lift := height * .26
	if short {
		lift = height * .20
	}
	ui.Column(c).Key("boot-core").Grow(1).MinWidth(0).MinHeight(0).FillWidth().
		AlignItems(ui.Center).Justify(ui.Center).Padding(24, 32, lift, 32).Children(func() {
		ui.Column(c).Width(460).MaxWidthPercent(92).AlignItems(ui.Center).Children(func() {
			bootBrand(c, short, fg, muted, m.SystemDark)

			// .status: muted while work runs, the ink when ready, the alert color
			// on failure (boot.css:301-320), with its dots while starting.
			statusColor := muted
			if m.Status.State == harness.Ready {
				statusColor = fg
			}
			if m.Status.State == harness.Failed || m.Status.RecoveryCancelled {
				statusColor = alert
			}
			ui.Row(c).Key("boot-status").Gap(7).AlignItems(ui.Center).Margin(26, 0, 0, 0).Children(func() {
				ui.Text(c, statusLine(m.Status)).FontSize(theme.InterfaceSize(14)).
					FixedLineHeight(theme.InterfaceSize(22)).TextColor(statusColor).TextAlign(ui.Center).MinWidth(0)
				if m.Status.State == harness.Starting && !m.Status.RecoveryCancelled {
					statusDots(c, statusColor)
				}
			})
			if line := pluginProgressLine(m.Status); line != "" {
				ui.Text(c, line).FontSize(theme.InterfaceSize(12)).TextColor(muted).TextAlign(ui.Center).Margin(4, 0, 0, 0)
			}
			if m.Status.State == harness.Failed {
				// .failure: 12/18 in the code face and the alert color, the copy
				// the log drawer explains (boot.css:344-357).
				ui.Text(c, failureCopy()).Font(theme.RenderPreferences().Code.Family).
					FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
					TextColor(alert).TextAlign(ui.Center).MaxLines(3).Margin(6, 0, 0, 0)
			}

			ui.Row(c).Key("boot-actions").Gap(10).AlignItems(ui.Center).Justify(ui.Center).
				Margin(26, 0, 0, 0).Wrap().Children(func() {
				if m.Status.State == harness.Failed && !m.Status.RecoveryActive && m.OnConfigure != nil {
					label := "打开启动器"
					if i18n.Locale() == "en" {
						label = "Open launcher"
					}
					if ui.Button(c, label).Clicked() {
						m.OnConfigure()
					}
				}
				cancelRecoveryAction(c, m)
				if bootActionsVisible(m.Status) && m.OnRetry != nil && ui.Button(c, i18n.T(KeyRetry)).Disabled(m.Status.RecoveryActive && !m.Status.RecoveryScheduled && !m.Status.RecoveryCancelled).Clicked() {
					m.OnRetry()
				}
				if bootActionsVisible(m.Status) && m.OnExport != nil && ui.Button(c, i18n.T(KeyExportLog)).Clicked() {
					m.OnExport()
				}
			})
		})
	})
}

// failureCopy is .failure's text: what to check next, in the log drawer's
// technical voice (boot.css:344-357).
func failureCopy() string {
	if i18n.Locale() == "en" {
		return "Check the workspace, Node.js and Host paths in the launcher, then retry or export the logs."
	}
	return "请检查启动器中的工作区、Node.js 与 Host 路径；修正后重试，或导出日志排查。"
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
	if !m.Status.RecoveryScheduled || m.Status.RecoveryCancelled || m.OnCancelRecovery == nil {
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

func paintBootLogLines(c *ui.Context, logs []string, muted, alert ui.Color) {
	for i, line := range logs {
		ink := muted
		if importantBootLog.MatchString(line) {
			ink = alert
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
			ui.Text(c, fmt.Sprintf("%d.", i+1)).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(10)).TextColor(muted.Alpha(.6)).Shrink(0)
			ui.Text(c, line).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(11.5)).FixedLineHeight(theme.InterfaceSize(19)).TextColor(ink).Selectable().MinWidth(0).Grow(1)
		})
	}
}

// Matches installed renderer/boot-recovery.js LOG_ERROR_PATTERN.
var importantBootLog = regexp.MustCompile(`ERR_[A-Z0-9_]+|Cannot find (?:package|module)|Error \[|plugin tree failed to load|cannot get property|cannot resolve profile bundle`)

func bootActionsVisible(st harness.Status) bool {
	return st.State == harness.Failed || st.RecoveryActive || st.RecoveryCancelled
}
