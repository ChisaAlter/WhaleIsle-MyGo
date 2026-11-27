// Package webview is the M10 native chrome for third-party plugin web UI.
package webview

import (
	"runtime"
	"strings"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// Model wires the plugin webview panel chrome (WebView2 host sits below).
type Model struct {
	Title         string
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
	OnOpenURL     func(url string)
}

// View paints the plugin web chrome strip; the WebView2 Host covers the body.
// It renders the native placeholder until the user explicitly opens the host.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	danger := design.For(c).DestructiveForeground

	root := chrome.FrostSurface(ui.Column(c.Key("plugin-web-panel")).Grow(1).MinHeight(0), colors, m.SystemDark, bg, 0)
	root.Children(func() {
		topPad, bottomPad := panelChromeVerticalPadding(runtime.GOOS)
		leftPad, rightPad := panelChromeHorizontalPadding(c.TitleBar())
		ui.Row(c.Key("plugin-web-chrome")).FillWidth().MinWidth(0).Height(float32(PanelChromeHeight-1)).Shrink(0).
			Padding(topPad, rightPad, bottomPad, leftPad).Gap(8).AlignItems(ui.Center).DragWindow().Children(func() {
			title := strings.TrimSpace(m.Title)
			if title == "" {
				title = i18n.T(i18n.KeyWebViewTitle)
			}
			ui.Text(c, title).FontSize(theme.InterfaceSize(15)).Bold().TextColor(fg).MinWidth(0).Shrink(1).SingleLine().Ellipsis("…")
			if m.HostActive && m.HostError != "" {
				ui.Text(c, errcopy.SentenceText(m.HostError)).FontSize(theme.InterfaceSize(11)).TextColor(danger).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
			} else {
				ui.Box(c).Grow(1).MinWidth(8)
			}
			if m.HostActive && m.HostURL != "" && m.OnCopyURL != nil {
				if coss.Button(c, i18n.T(i18n.KeyWebViewCopyURL), coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-copy-url",
				}).Shrink(0).Clicked() {
					m.OnCopyURL(m.HostURL)
				}
			}
			if m.HostActive && m.HostURL != "" && m.OnReload != nil {
				if coss.Button(c, i18n.T(i18n.KeyBrowserPreviewReload), coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-reload",
				}).Shrink(0).Clicked() {
					m.OnReload(m.HostURL)
				}
			}
			if m.HostActive && m.OnCloseHost != nil {
				if coss.Button(c, i18n.T(i18n.KeyWebViewCloseHost), coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-close-host",
				}).Shrink(0).Clicked() {
					m.OnCloseHost()
				}
			}
			if m.OnClose != nil {
				if coss.Button(c, i18n.T(i18n.KeyWebViewBack), coss.ButtonOptions{
					Variant: coss.ButtonSecondary,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-back",
				}).Shrink(0).Clicked() {
					m.OnClose()
				}
			}
		})
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(border)

		if m.HostActive {
			// Body is covered by the WebView2 Host; keep a minimal native underlay.
			chrome.FrostSurface(ui.Box(c).Fill().MinHeight(0), colors, m.SystemDark, bg, 0)
			return
		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(16, 20, 16, 20).Gap(12).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWebViewNotOpen)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
			if m.HarnessOrigin != "" {
				ui.Text(c, i18n.T(i18n.KeyWebViewOrigin)+": "+m.HarnessOrigin).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			ui.Row(c).Gap(8).Children(func() {
				if m.OnOpenHost != nil && coss.Button(c, i18n.T(i18n.KeyWebViewOpenHost), coss.ButtonOptions{
					Variant: coss.ButtonDefault,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-open-host",
				}).Clicked() {
					m.OnOpenHost()
				}
				if m.OnOpenBrowser != nil && coss.Button(c, i18n.T(i18n.KeyWebViewOpenBrowser), coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonSM,
					Key:     "plugin-web-open-browser",
				}).Clicked() {
					m.OnOpenBrowser()
				}
			})
			if m.HostError != "" {
				ui.Text(c, errcopy.SentenceText(m.HostError)).FontSize(theme.InterfaceSize(11)).TextColor(danger)
				if m.OnOpenURL != nil && strings.Contains(strings.ToLower(m.HostError), "webview2") {
					if coss.Button(c, i18n.T(keyWebViewDownloadRuntime), coss.ButtonOptions{
						Variant: coss.ButtonOutline,
						Size:    coss.ButtonSM,
						Key:     "plugin-web-download-runtime",
					}).Clicked() {
						m.OnOpenURL("https://go.microsoft.com/fwlink/p/?LinkId=2124703")
					}
				}
			}
		})
	})
}

func panelChromeVerticalPadding(goos string) (top, bottom float32) {
	if goos == "windows" {
		// Align the 32 DIP action strip (y=12..44) with WindowsCaptionButtons
		// (Top: 12, Height: 32) inside the 48 DIP panel chrome band.
		return 12, 4
	}
	return 8, 8
}

func panelChromeHorizontalPadding(bar ui.TitleBar) (left, right float32) {
	const base = float32(12)
	left, right = base, base
	if bar.Left > 0 {
		left += bar.Left
	}
	if bar.Right > 0 {
		right += bar.Right
	}
	return left, right
}

// PanelChromeHeight is the DIP height reserved above the WebView2 Host
// (main title bar is applied separately via HostOptions.TopInset on non-Windows).
const PanelChromeHeight = 48
