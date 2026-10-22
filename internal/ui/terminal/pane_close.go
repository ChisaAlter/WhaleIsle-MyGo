package terminal

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"whaleisle/internal/ui/i18n"
)

// requestClosePane closes a pane, asking first when its shell is still running:
// one click on a chip's close control should not end a live process.
func requestClosePane(m Model, id uint64) {
	p := m.Panel
	if p == nil {
		return
	}
	pane := p.pane(id)
	if pane == nil {
		return
	}
	if runningPane(pane) {
		p.pendingClose = id
		return
	}
	closePaneFromView(m, id)
}

// runningPane is the liveness probe, declared apart so the close flow can be
// exercised without spawning a shell. Production reads the terminal itself.
var runningPane = func(pane *terminalPane) bool { return paneRunning(pane) }

// paneRunning reports whether the pane still owns a live process.
func paneRunning(pane *terminalPane) bool {
	if pane == nil || pane.term == nil {
		return false
	}
	select {
	case <-pane.term.Done():
		return false
	default:
		return true
	}
}

// terminalClosePrompt asks before ending a running terminal. It closes itself
// when the pane is already gone, so a prompt can never outlive its pane.
func terminalClosePrompt(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	p := m.Panel
	if p == nil || p.pendingClose == 0 {
		return
	}
	pane := p.pane(p.pendingClose)
	if pane == nil || !runningPane(pane) {
		p.pendingClose = 0
		return
	}
	id := p.pendingClose
	open := true
	ui.DialogBase(c.Key("terminal-close-pane"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 360, 20).Gap(10).
			Label(i18n.T(i18n.KeySurfacesTerminalClose)).Children(func() {
			ui.Text(c, i18n.T(i18n.KeySurfacesTerminalClosePrompt)).FillWidth().
				FontSize(15).FontWeight(600).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeySurfacesTerminalCloseBody)).
				FillWidth().FontSize(12).TextColor(muted)
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Padding(6, 0, 0, 0).Children(func() {
				if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesTerminalCancel), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium}).Clicked() {
					open = false
				}
				if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesTerminalConfirmClose), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonMedium}).Clicked() {
					open = false
					closePaneFromView(m, id)
				}
			})
		})
	})
	if !open {
		p.pendingClose = 0
	}
}
