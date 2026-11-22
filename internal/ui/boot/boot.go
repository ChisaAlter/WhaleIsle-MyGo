// Package boot draws the native startup page within the main window.
package boot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/hoststate"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model is the boot page's observable state (filled by the shell).
type Model struct {
	Status           harness.Status
	Logs             []string // newest last; view shows a tail
	Dark             bool
	OnCancelRecovery func()
	OnRetry          func()
	OnExport         func()
	OnConfigure      func()
	StatusText       string
	// Declined marks a cold launch whose automatic start was declined. The page
	// stays the window's content and offers the retry and diagnostics actions,
	// where a running start would show only its status.
	Declined bool
}

const bootCodeFont = "Consolas, Menlo, monospace"

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

// WorkbenchStatus handles an explicitly stopped Host or a failure after the
// workbench has opened. Cold startup owns the full-page View instead.
func WorkbenchStatus(c *ui.Context, m Model, message, configureLabel string) {
	report := hoststate.Of(m.Status)
	if message == "" {
		message = report.Line
	}
	ui.Row(c).Key("workbench-host-status").FillWidth().Padding(8, 16).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
		ui.Text(c, message).FontSize(theme.InterfaceSize(12)).Grow(1).MinWidth(0)
		cancelRecoveryAction(c, m)
		if m.OnRetry != nil {
			// A stopped service never reads as a plain retry: the strip's action
			// is the one the report names for this state.
			label := i18n.T(KeyRetry)
			if report.Action != "" {
				label = report.Action
			}
			if ui.Button(c, label).Clicked() {
				m.OnRetry()
			}
		}
		if m.OnConfigure != nil && ui.Button(c, configureLabel).Clicked() {
			m.OnConfigure()
		}
		if m.OnExport != nil && ui.Button(c, i18n.T(KeyExportLog)).Clicked() {
			m.OnExport()
		}
	})
}

// View paints the full main window, including behind the native title bar:
// the reference page's layered scene, the brand block lifted over the horizon,
// the status line with its actions and the newest log line at the foot.
func View(c *ui.Context, m Model) {
	palette := bootPalette(m.Dark)
	failed := m.Status.State == harness.Failed

	root := ui.Column(c).Key("boot-root").Label(bootText("启动加载页", "Startup loading page")).Fill().Radius(bootRadius(c)).Clip()
	drawer := ui.Local(root, "boot-log-drawer", func() bool { return false })
	lines := bootDrawerLines(m)
	root.Children(func() {
		// .scene's silhouette: an element's own Draw is painted before its clip,
		// so the sky has to be a child for the radius to cut it (boot.css:83-97).
		ui.Box(c).Key("boot-scene").Fill().Draw(bootScene(m.Dark, failed, !c.Preferences().ReduceMotion))
		bootCore(c, m, palette)
		bootTicker(c, m, palette, func() { *drawer = true })
		ui.Row(c).Absolute().Top(0).Left(c.TitleBar().Left).Right(c.TitleBar().Right).Height(chrome.EffectiveTitleBarHeight()).DragWindow()
		ui.DialogBase(c, drawer, func(backdrop, panel ui.Element) {
			// .logdrawer is a full-window scrim that fades in over .18s
			// (drawer-fade) and seats the panel bottom-centred; the backdrop is a
			// column, so Justify's End is the bottom and AlignItems' Center is
			// horizontal (boot.css:462-475).
			scrim := chrome.Reveal(c, backdrop, 180*time.Millisecond)
			chrome.RevealMask(backdrop, ui.RGBA(0, 0, 0, .32), scrim)
			backdrop.Justify(ui.End).AlignItems(ui.Center)
			// .logdrawer-panel: min(760px, w-96px) wide, at most 46% of the
			// window tall, the hairline on every edge but the bottom, the 14 DIP
			// top corners and the --boot-panel fill frosted by a 12 DIP scene
			// blur; it rises 14 DIP over .24s (drawer-rise, boot.css:477-489,
			// 596-604). A negative bottom margin paints it lower, as
			// translateY does.
			rise := chrome.Reveal(c, panel, 240*time.Millisecond)
			w, h := c.Size()
			// A Scroll has no intrinsic height: the panel's own height is the
			// head plus the lines it lists, capped at the sheet's 46% — the
			// panel's content height and its cap in one expression.
			linesH := float32(max(1, len(lines))) * theme.InterfaceSize(19)
			panel.Width(max(0, min(760, w-96))).Height(min(h*.46, 68+linesH)).
				Margin(0, 0, (rise-1)*14, 0).Opacity(.4+.6*rise).
				Radius(14, 14, 0, 0).Clip().BorderWidth(1, 1, 0, 1).BorderColor(palette.hairline).
				Background(palette.panel).Material(glass.Blur{Radius: 12}).
				Children(func() {
					// .logdrawer-head: title, count and close on one row, with the
					// hairline under it (boot.css:491-495, boot.html:45-48).
					ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Padding(12, 18).
						BorderWidth(0, 0, 1, 0).BorderColor(palette.hairline).Children(func() {
						ui.Text(c, bootText("启动日志", "Startup logs")).FontSize(theme.InterfaceSize(12)).LetterSpacing(12 * .14).TextColor(palette.ink)
						ui.Text(c, bootText(fmt.Sprintf("%d 行", len(lines)), fmt.Sprintf("%d lines", len(lines)))).
							Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(10.5)).TextColor(palette.muted)
						ui.Box(c).Grow(1)
						close := ui.ButtonBase(c).Label(bootText("关闭", "Close")).BorderWidth(0).Background(ui.Transparent).Padding(0, 4).AutoFocus()
						closeInk := palette.muted
						if close.Hovered() {
							closeInk = palette.ink
						}
						close.Children(func() {
							ui.Text(c, "×").FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(closeInk)
						})
						if close.Clicked() {
							*drawer = false
						}
					})
					// .logdrawer-lines: the ordered list in the code face, 11.5/19,
					// muted, the markers at 10px and important lines alerting; it is
					// the part that scrolls (boot.css:497-511).
					ui.Scroll(c).Grow(1).MinHeight(0).Padding(10, 18, 16, 36).Children(func() {
						ui.Column(c).Gap(4).Selectable().Children(func() {
							paintBootLogLines(c, lines, palette.muted, palette.alert)
							if len(lines) == 0 {
								ui.Text(c, i18n.T(KeyWaiting)).FontSize(12).TextColor(palette.muted)
							}
						})
					})
				})
		})
	})
}

// bootCore is .core: the brand block, the status line, the failure detail and
// the actions. Its foot is padded by 26vh so the block sits over the sky and the
// horizon crosses below it instead of through it (boot.css:207-217); a short
// window keeps 20vh and 34 DIP letters (boot.css:453-459).
func bootCore(c *ui.Context, m Model, palette *scenePalette) {
	_, height := c.Size()
	short := height < 700
	lift := height * .26
	if short {
		lift = height * .20
	}
	ui.Column(c).Key("boot-core").Absolute().Top(0).Left(0).Right(0).Bottom(0).
		AlignItems(ui.Center).Justify(ui.Center).Padding(0, 24, lift, 24).Children(func() {
		ui.Column(c).Width(460).MaxWidthPercent(92).AlignItems(ui.Center).Children(func() {
			bootBrand(c, short, palette)

			// .status: muted while work runs, the ink when ready, the alert color
			// on failure (boot.css:301-320), with its dots while starting.
			statusColor := palette.muted
			if m.Status.State == harness.Ready {
				statusColor = palette.ink
			}
			if m.Status.State == harness.Failed || m.Status.RecoveryCancelled {
				statusColor = palette.alert
			}
			ui.Row(c).Key("boot-status").Gap(7).AlignItems(ui.Center).Margin(26, 0, 0, 0).Children(func() {
				line := m.StatusText
				if line == "" {
					line = statusTitle(m.Status)
					if m.Status.State == harness.Starting && !m.Status.RecoveryCancelled {
						line = bootText("启动中", "Starting")
					}
				}
				ui.Text(c, line).FontSize(14).FixedLineHeight(22).TextColor(statusColor).TextAlign(ui.Center).MinWidth(0)
				if m.Status.State == harness.Starting && !m.Status.RecoveryCancelled {
					statusDots(c, statusColor)
				}
			})

			if !bootActionsVisible(m.Status) && !m.Status.RecoveryScheduled && !m.Declined {
				return
			}
			ui.Row(c).Key("boot-actions").Gap(10).AlignItems(ui.Center).Justify(ui.Center).
				Margin(26, 0, 0, 0).Wrap().Children(func() {
				// .actions: 重试 leads, filled with the alert color and white
				// letters; the rest are ghost pills in the reference's order
				// (boot.html:27-32, boot.css:373-413).
				if bootActionsVisible(m.Status) || m.Declined {
					if m.OnRetry != nil {
						disabled := m.Status.RecoveryActive && !m.Status.RecoveryScheduled && !m.Status.RecoveryCancelled
						label := retryLabel(m.Status)
						bootAction(c, label, true, disabled, palette, m.OnRetry)
					}
				}
				if m.Status.RecoveryScheduled && !m.Status.RecoveryCancelled && m.OnCancelRecovery != nil {
					bootAction(c, cancelRecoveryLabel(), false, false, palette, m.OnCancelRecovery)
				}
				if (m.Status.State == harness.Failed || m.Declined) && !m.Status.RecoveryActive && m.OnConfigure != nil {
					bootAction(c, bootText("启动诊断", "Startup diagnostics"), false, false, palette, m.OnConfigure)
				}
				if (bootActionsVisible(m.Status) || m.Declined) && m.OnExport != nil {
					bootAction(c, bootText("下载日志", "Download log"), false, false, palette, m.OnExport)
				}
			})

			bootFailureDock(c, m, palette)
		})
	})
}

// bootFailureDock is the shell's failure detail. The installed page takes
// .failure out of .core and mounts it in a feedback progress surface titled
// 启动详情 (native boot.js:16-18), whose placement pass seats it in the free band
// below the actions rather than under the status line (feedback.js:133-190).
// That surface is the toast layer's: dark in both schemes (dsh-webui-tokens.css:
// 54-55,112), 16 DIP corners and padding, and the page's 14/22 copy rather than
// .failure's code face (feedback.css:72-79,137-156,180). Its slot hangs off the
// core column's foot, so the brand block above keeps the page's own seat instead
// of being pushed up by the pane.
func bootFailureDock(c *ui.Context, m Model, palette *scenePalette) {
	if m.Status.State != harness.Failed || m.StatusText != "" {
		return
	}
	cause := errcopy.Sentence(m.Status.Err)
	if cause == "" {
		return
	}
	window, _ := c.Size()
	ui.Column(c).Key("boot-failure-dock-slot").Absolute().TopPercent(100).Left(0).Right(0).
		AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
		// The pane spans min(640, main - 48) of the window's column; the
		// native page floats it outside .core at that width.
		ui.Column(c).Key("boot-failure-dock").Label(bootText("启动详情", "Startup details")).
			Width(min(640, max(0, window-48))).Background(palette.toast).Radius(theme.ContextRadii(c).LG).
			Padding(16).Children(func() {
			// .failure keeps its cause-and-next-step copy; the raw message
			// stayed in the log drawer for both pages (native boot.js:220-240).
			ui.Text(c, cause).FontSize(14).FixedLineHeight(22).TextColor(palette.toastInk).MinWidth(0)
		})
	})
}

// bootDrawerLines is the log the drawer lists: the failure message joins it
// when no line carries the message already, so the raw error is still readable
// after the status line stopped repeating it (boot.js comment at FAILURE_TEXT_PATTERNS).
func bootDrawerLines(m Model) []string {
	if m.Status.Err == nil {
		return m.Logs
	}
	raw := strings.TrimSpace(m.Status.Err.Error())
	if raw == "" {
		return m.Logs
	}
	for _, line := range m.Logs {
		if strings.Contains(line, raw) {
			return m.Logs
		}
	}
	return append(m.Logs[:len(m.Logs):len(m.Logs)], raw)
}

// bootAction is .actions button: a 34 DIP pill with a hairline edge, the ink
// label and, for the leading action, the alert fill and white letters
// (boot.css:373-413). .actions only renders over a failed or recovering Host,
// so the leading pill always takes the body[data-state="error"] colours. The
// sheet's 999px radius is the pill's half height; this painter takes the radius
// literally, so it is the 17 DIP a 34 DIP box clamps to.
func bootAction(c *ui.Context, label string, primary, disabled bool, palette *scenePalette, onClick func()) {
	ink, fill, edge := palette.ink, ui.Transparent, palette.hairline
	if primary {
		ink, fill, edge = ui.RGB(255, 255, 255), palette.alert, palette.alert
	}
	if disabled {
		ink, fill, edge = ink.Alpha(.4), fill.Alpha(.4), edge.Alpha(.4)
	}
	button := ui.ButtonBase(c).Key("boot-action-"+label).Label(label).Disabled(disabled).
		MinHeight(34).Padding(5, 18).Radius(17).Background(fill).Border(1, edge)
	if button.Hovered() && !disabled {
		if primary {
			button.Background(palette.alert.Alpha(.85))
		} else {
			button.Background(palette.ink.Alpha(.08))
		}
	}
	button.Children(func() {
		ui.Text(c, label).FontSize(12).LetterSpacing(12 * .06).FixedLineHeight(20).TextColor(ink)
	})
	if button.Clicked() && onClick != nil {
		onClick()
	}
}

// bootRadius is .scene's 20 DIP silhouette, square once the window has none of
// its own, in the maximized and full screen states (boot.css:613-621).
func bootRadius(c *ui.Context) float32 {
	if win := frameWindow(c); win != nil && (win.IsMaximized() || win.IsFullScreen()) {
		return 0
	}
	return 20
}

// frameWindow returns the window this frame paints. The boot page is built from
// the window's own view, which the ui engine does not name to it, so take the
// focused window when its content is this frame's size, else the one window
// that is.
func frameWindow(c *ui.Context) *mygo.Window {
	w, h := c.Size()
	sized := func(win *mygo.Window) bool {
		bounds := win.ContentBounds()
		return float32(bounds.Width) == w && float32(bounds.Height) == h
	}
	if win := mygo.FocusedWindow(); win != nil && sized(win) {
		return win
	}
	for _, win := range mygo.Windows() {
		if sized(win) {
			return win
		}
	}
	return nil
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

// retryLabel is .actions' leading pill: 重试 for a start failure, 立即重启 for a
// Host that exited after it was ready (native boot.js:257-259).
func retryLabel(st harness.Status) string {
	if st.RuntimeFailure {
		return bootText("立即重启", "Restart now")
	}
	return i18n.T(KeyRetry)
}

// statusTitle is .status's own text: the state's label, or the recovery
// sentence the native shell adds. The message behind a failure is not part of
// it (boot.js:245-262).
func statusTitle(st harness.Status) string {
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
		// A Host that exited after it was ready is 意外退出; a cold start that
		// never became ready is a start failure (native boot.js:253-254).
		if st.RuntimeFailure {
			return bootText("桌面端意外退出", "Desktop exited unexpectedly")
		}
		return bootText("桌面端启动失败", "Desktop startup failed")
	default:
		return i18n.T(KeyStopped)
	}
}

// StatusLine is the sentence the workbench's host strip shows: the state's
// sentence from internal/hoststate, which names what happened and the next step.
// The raw error stays in the logs and never becomes the strip's primary
// sentence. Exported for the launcher's own banner over the workbench.
func StatusLine(st harness.Status) string {
	if line := hoststate.Of(st).Line; line != "" {
		return line
	}
	return statusTitle(st)
}

func bootText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func cancelRecoveryAction(c *ui.Context, m Model) {
	if !m.Status.RecoveryScheduled || m.Status.RecoveryCancelled || m.OnCancelRecovery == nil {
		return
	}
	if ui.Button(c, cancelRecoveryLabel()).Clicked() {
		m.OnCancelRecovery()
	}
}

// cancelRecoveryLabel names the action that stops a scheduled restart; the
// reference has no such state and no copy for it (boot.html:29).
func cancelRecoveryLabel() string {
	if i18n.Locale() == "en" {
		return "Cancel automatic recovery"
	}
	return "取消自动恢复"
}

func paintBootLogLines(c *ui.Context, logs []string, muted, alert ui.Color) {
	for i, line := range logs {
		ink := muted
		if importantBootLog.MatchString(line) {
			ink = alert
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
			ui.Text(c, fmt.Sprintf("%d.", i+1)).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(10)).TextColor(muted.Alpha(.6)).Shrink(0).Unselectable()
			ui.Text(c, line).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(11.5)).FixedLineHeight(theme.InterfaceSize(19)).TextColor(ink).Selectable().MinWidth(0).Grow(1)
		})
	}
}

// Matches installed renderer/boot-recovery.js LOG_ERROR_PATTERN.
var importantBootLog = regexp.MustCompile(`ERR_[A-Z0-9_]+|Cannot find (?:package|module)|Error \[|plugin tree failed to load|cannot get property|cannot resolve profile bundle`)

func bootActionsVisible(st harness.Status) bool {
	return st.State == harness.Failed || st.RecoveryActive || st.RecoveryCancelled
}
