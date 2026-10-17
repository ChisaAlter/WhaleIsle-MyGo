// Package browserpreview is the M9 native chrome for arbitrary http(s) browsing
// via the shared WebView2 host (internal/desktop/webview).
package browserpreview

import (
	"net/url"
	"strings"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// PanelChromeHeight is the DIP height reserved above the WebView2 host
// (main title bar is applied separately via HostOptions.TopInset).
const PanelChromeHeight = 48

// Model wires the browser preview panel (URL bar + host status).
type Model struct {
	ThemeID    string
	SystemDark bool
	URLDraft   *string
	HostActive bool
	HostURL    string
	HostError  string
	NavigateError string
	OnClose    func()
	OnNavigate func(url string)
	OnOpenExternal func(url string)
	OnCopyURL  func(url string)
	OnReload   func(url string)
	OnCloseHost func()
}

// View paints the browser preview chrome strip; WebView2 covers the body when active.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Height(float32(PanelChromeHeight - 1)).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyBrowserPreviewTitle)).FontSize(15).Bold().TextColor(fg)
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyBrowserPreviewBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).Fill().AlignItems(ui.Center).Children(func() {
			draft := ""
			if m.URLDraft != nil {
				draft = *m.URLDraft
			}
			if m.URLDraft != nil {
				ui.TextInput(c, m.URLDraft).Placeholder(i18n.T(i18n.KeyBrowserPreviewURLPlaceholder)).Grow(1)
			} else {
				ui.Text(c, draft).FontSize(12).TextColor(muted).Grow(1)
			}
			goLabel := i18n.T(i18n.KeyBrowserPreviewGo)
			if ui.Button(c, goLabel).Clicked() && m.OnNavigate != nil {
				m.OnNavigate(NormalizeURL(draft))
			}
			extURL := strings.TrimSpace(m.HostURL)
			if extURL == "" {
				extURL = NormalizeURL(draft)
			}
			if m.OnOpenExternal != nil && extURL != "" {
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewOpenExternal)).Clicked() {
					m.OnOpenExternal(extURL)
				}
			}
			if m.OnCopyURL != nil && extURL != "" {
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewCopyURL)).Clicked() {
					m.OnCopyURL(extURL)
				}
			}
			if m.OnReload != nil && m.HostActive && extURL != "" {
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewReload)).Clicked() {
					m.OnReload(extURL)
				}
			}
			if m.HostActive && m.OnCloseHost != nil {
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewCloseHost)).Clicked() {
					m.OnCloseHost()
				}
			}
		})

		if m.NavigateError != "" {
			ui.Text(c, m.NavigateError).FontSize(11).TextColor(chrome.Color("rgb(220, 38, 38)")).Padding(0, 12, 4, 12)
		}
		if m.HostError != "" {
			ui.Text(c, m.HostError).FontSize(11).TextColor(chrome.Color("rgb(220, 38, 38)")).Padding(0, 12, 4, 12)
		}
		if m.HostActive && m.HostURL != "" {
			ui.Text(c, m.HostURL).FontSize(11).TextColor(accent).Padding(0, 12, 8, 12)
		}

		if m.HostActive {
			ui.Box(c).Fill().MinHeight(0).Background(field)
			return
		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(16, 20, 16, 20).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyBrowserPreviewHint)).FontSize(12).TextColor(muted)
		})
	})
}

// NormalizeURL trims input and ensures an http(s) scheme when missing.
// Returns "" when the result is not a valid http(s) URL.
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	if u.Host == "" {
		return ""
	}
	return u.String()
}
