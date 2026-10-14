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
	Status     harness.Status
	Logs       []string // newest last; view shows a tail
	ThemeID    string
	SystemDark bool
	OnRetry    func()
	OnExport   func()
}

const logTail = 40

// i18n keys for the boot page.
const (
	KeyTitle     = "boot.title"
	KeyStarting  = "boot.starting"
	KeyReady     = "boot.ready"
	KeyFailed    = "boot.failed"
	KeyStopped   = "boot.stopped"
	KeyRetry     = "boot.retry"
	KeyExportLog = "boot.export"
	KeyWaiting        = "boot.waiting"
	KeyPlugins             = "boot.plugins"
	KeyPluginsDetail       = "boot.plugins_detail"
	KeyPluginsIndeterminate = "boot.plugins_indeterminate"
)

// View paints the boot page inside the main window body (below the title bar).
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)

	ui.Column(c).Fill().Background(bg).Padding(24, 32, 24, 32).Gap(16).Children(func() {
		ui.Text(c, i18n.T(KeyTitle)).FontSize(22).Bold().TextColor(fg)
		ui.Text(c, statusLine(m.Status)).FontSize(14).TextColor(muted)
		if line := pluginProgressLine(m.Status); line != "" {
			ui.Text(c, line).FontSize(13).TextColor(fg)
		}

		ui.Row(c).Gap(8).Children(func() {
			if m.OnRetry != nil && ui.Button(c, i18n.T(KeyRetry)).Clicked() {
				m.OnRetry()
			}
			if m.OnExport != nil && ui.Button(c, i18n.T(KeyExportLog)).Clicked() {
				m.OnExport()
			}
		})

		ui.Column(c).Fill().Background(chrome.Color(colors.Field)).
			Border(1, border).Radius(8).
			Padding(12, 12, 12, 12).Gap(4).Children(func() {
			start := 0
			if n := len(m.Logs); n > logTail {
				start = n - logTail
			}
			for _, line := range m.Logs[start:] {
				ui.Text(c, line).FontSize(12).TextColor(muted)
			}
			if len(m.Logs) == 0 {
				ui.Text(c, i18n.T(KeyWaiting)).FontSize(12).TextColor(muted)
			}
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
