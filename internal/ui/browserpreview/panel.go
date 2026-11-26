// Package browserpreview is the M9 native chrome for arbitrary http(s) browsing
// via the shared WebView2 host (internal/desktop/webview).
package browserpreview

import (
	"net/url"
	"strings"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// PanelChromeHeight is the DIP height reserved above the WebView2 host
// (main title bar is applied separately via HostOptions.TopInset).
const PanelChromeHeight = 48

// Model wires the browser preview panel (URL bar + host status).
type Model struct {
	ThemeID        string
	SystemDark     bool
	URLDraft       *string
	HostActive     bool
	HostURL        string
	HostError      string
	NavigateError  string
	OnClose        func()
	OnNavigate     func(url string)
	OnOpenExternal func(url string)
	OnCopyURL      func(url string)
	OnReload       func(url string)
	OnCloseHost    func()
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

	paneBg := bg
	if design.For(c).Glass || chrome.BackdropActive() {
		paneBg = ui.Transparent
	}
	ui.Column(c).Grow(1).MinHeight(0).Background(paneBg).Children(func() {
		leftPad, rightPad := chrome.TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
		ui.Row(c.Key("browser-preview-header")).FillWidth().MinWidth(0).Height(float32(PanelChromeHeight-1)).Shrink(0).Padding(design.Sp(2), rightPad, design.Sp(2), leftPad).Gap(design.Sp(2)).AlignItems(ui.Center).DragWindow().Children(func() {
			ui.Text(c, i18n.T(i18n.KeyBrowserPreviewTitle)).FontSize(theme.InterfaceSize(15)).Bold().TextColor(fg).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
			if m.OnClose != nil && coss.Button(c, i18n.T(i18n.KeyBrowserPreviewBack), coss.ButtonOptions{Variant: coss.ButtonSecondary, Size: coss.ButtonSM, Key: "browser-preview-back"}).Shrink(0).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(border)

		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(design.Sp(2), design.Sp(3), design.Sp(2), design.Sp(3)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			draft := ""
			if m.URLDraft != nil {
				draft = *m.URLDraft
			}
			if m.URLDraft != nil {
				var inp ui.Element
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					inp = coss.Input(c, m.URLDraft, coss.InputOptions{
						Size:        coss.InputSM,
						Placeholder: i18n.T(i18n.KeyBrowserPreviewURLPlaceholder),
						Key:         "browser-preview-url",
					})
				})
				if inp.Submitted() && m.OnNavigate != nil {
					m.OnNavigate(NormalizeURL(*m.URLDraft))
				}
			} else {
				ui.Text(c, draft).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0)
			}
			goLabel := i18n.T(i18n.KeyBrowserPreviewGo)
			if coss.Button(c, goLabel, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Key: "browser-preview-go"}).Shrink(0).Clicked() && m.OnNavigate != nil {
				m.OnNavigate(NormalizeURL(draft))
			}
			extURL := strings.TrimSpace(m.HostURL)
			if extURL == "" {
				extURL = NormalizeURL(draft)
			}
			if m.OnOpenExternal != nil && extURL != "" {
				if coss.Button(c, i18n.T(i18n.KeyBrowserPreviewOpenExternal), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "browser-preview-external"}).Shrink(0).Clicked() {
					m.OnOpenExternal(extURL)
				}
			}
			if m.OnCopyURL != nil && extURL != "" {
				if coss.Button(c, i18n.T(i18n.KeyBrowserPreviewCopyURL), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "browser-preview-copy"}).Shrink(0).Clicked() {
					m.OnCopyURL(extURL)
				}
			}
			if m.OnReload != nil && m.HostActive && extURL != "" {
				if coss.Button(c, i18n.T(i18n.KeyBrowserPreviewReload), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Key: "browser-preview-reload"}).Shrink(0).Clicked() {
					m.OnReload(extURL)
				}
			}
			if m.HostActive && m.OnCloseHost != nil {
				if coss.Button(c, i18n.T(i18n.KeyBrowserPreviewCloseHost), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Key: "browser-preview-close-host"}).Shrink(0).Clicked() {
					m.OnCloseHost()
				}
			}
		})

		danger := design.For(c).Destructive
		if m.NavigateError != "" {
			ui.Text(c, errcopy.SentenceText(m.NavigateError)).FontSize(theme.InterfaceSize(11)).TextColor(danger).Padding(0, design.Sp(3), design.Sp(1), design.Sp(3))
		}
		if m.HostError != "" {
			ui.Text(c, errcopy.SentenceText(m.HostError)).FontSize(theme.InterfaceSize(11)).TextColor(danger).Padding(0, design.Sp(3), design.Sp(1), design.Sp(3))
		}
		if m.HostActive && m.HostURL != "" {
			ui.Text(c, m.HostURL).FontSize(theme.InterfaceSize(11)).TextColor(accent).Padding(0, design.Sp(3), design.Sp(2), design.Sp(3))
		}

		if m.HostActive {
			if design.For(c).Glass || chrome.BackdropActive() {
				ui.Box(c).Fill().MinHeight(0).Background(ui.Transparent)
			} else {
				ui.Box(c).Fill().MinHeight(0).Background(field)
			}
			return
		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(design.Sp(4), design.Sp(5), design.Sp(4), design.Sp(5)).Gap(design.Sp(2)).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyBrowserPreviewHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
	if u.Hostname() == "" || u.User != nil {
		return ""
	}
	return u.String()
}
