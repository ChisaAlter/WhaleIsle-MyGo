// Package webview is the M10 native chrome for third-party plugin web UI.
package webview

import (
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// Model wires the plugin webview panel chrome (WebView2 host sits below).
type Model struct {
	HarnessOrigin string
	ThemeID       string
	SystemDark    bool
	HostActive    bool
	HostURL       string
	HostError     string
	OnClose       func()
	OnOpenBrowser func()
	OnOpenHost    func()
	OnCloseHost   func()
	OnCopyURL     func(url string)
	OnReload      func(url string)
}

// View paints the plugin web chrome strip; the WebView2 Host covers the body.
// It opens that host on the first frame it is shown, so the entry lands on the
// real plugin page instead of a placeholder describing what could be built.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)

	root := ui.Column(c.Key("plugin-web-panel")).Grow(1).MinHeight(0).Background(bg)
	autoOpen := ui.Local(root, "auto-open", func() bool { return false })
	if !m.HostActive && !*autoOpen && m.HostError == "" && m.HarnessOrigin != "" && m.OnOpenHost != nil {
		*autoOpen = true
		m.OnOpenHost()
	}
	root.Children(func() {
		ui.Row(c).Height(float32(PanelChromeHeight-1)).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWebViewTitle)).FontSize(theme.InterfaceSize(15)).Bold().TextColor(fg)
			if m.HostActive && m.HostError != "" {
				ui.Text(c, m.HostError).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color("rgb(220, 38, 38)")).Grow(1)
			} else {
				ui.Box(c).Grow(1)
			}
			if m.HostActive && m.HostURL != "" && m.OnCopyURL != nil {
				if ui.Button(c, i18n.T(i18n.KeyWebViewCopyURL)).FontSize(theme.InterfaceSize(11)).Clicked() {
					m.OnCopyURL(m.HostURL)
				}
			}
			if m.HostActive && m.HostURL != "" && m.OnReload != nil {
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewReload)).FontSize(theme.InterfaceSize(11)).Clicked() {
					m.OnReload(m.HostURL)
				}
			}
			if m.HostActive && m.OnCloseHost != nil {
				if ui.Button(c, i18n.T(i18n.KeyWebViewCloseHost)).Clicked() {
					m.OnCloseHost()
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyWebViewBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		if m.HostActive {
			// Body is covered by the WebView2 Host; keep a minimal native underlay.
			ui.Box(c).Fill().MinHeight(0).Background(bg)
			return
		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(16, 20, 16, 20).Gap(12).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWebViewNotOpen)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
			if m.HarnessOrigin != "" {
				ui.Text(c, i18n.T(i18n.KeyWebViewOrigin)+": "+m.HarnessOrigin).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			ui.Row(c).Gap(8).Children(func() {
				if m.OnOpenHost != nil && ui.Button(c, i18n.T(i18n.KeyWebViewOpenHost)).Clicked() {
					m.OnOpenHost()
				}
				if m.OnOpenBrowser != nil && ui.Button(c, i18n.T(i18n.KeyWebViewOpenBrowser)).Clicked() {
					m.OnOpenBrowser()
				}
			})
			if m.HostError != "" {
				ui.Text(c, m.HostError).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color("rgb(220, 38, 38)"))
			}
		})
	})
}

// PanelChromeHeight is the DIP height reserved above the WebView2 Host
// (main title bar is applied separately via HostOptions.TopInset).
const PanelChromeHeight = 48
