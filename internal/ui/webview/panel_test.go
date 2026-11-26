package webview

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"
)

func webviewTestLocale(t *testing.T) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

// TestViewDoesNotAutoOpenHostAndRequiresExplicitClick proves the panel entry renders
// the WebView2 launch controls without implicitly launching a host window until clicked.
func TestViewDoesNotAutoOpenHostAndRequiresExplicitClick(t *testing.T) {
	webviewTestLocale(t)
	opened, browser := 0, 0
	m := Model{
		HarnessOrigin: "http://127.0.0.1:3080",
		ThemeID:       "deepseek",
		OnOpenHost:    func() { opened++ },
		OnOpenBrowser: func() { browser++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if opened != 0 {
		t.Fatalf("implicit auto-open fired: host opens = %d, want 0", opened)
	}
	if browser != 0 {
		t.Fatalf("browser opens = %d, want 0", browser)
	}
	if !tt.HasText(i18n.T(i18n.KeyWebViewNotOpen)) {
		t.Fatalf("missing not-open state: %v", tt.Texts())
	}
	texts := strings.Join(tt.Texts(), "\n")
	for _, banned := range []string{"stub", "无边框", "desktopPlugin", "第三方插件需"} {
		if strings.Contains(texts, banned) {
			t.Fatalf("developer placeholder %q is still rendered:\n%s", banned, texts)
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewOpenHost)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("explicit click host opens = %d, want 1", opened)
	}
}

// TestViewRetriesTheHostAfterAnError leaves the one-shot auto-open alone on
// failure, shows the real error and lets the user retry explicitly.
func TestViewRetriesTheHostAfterAnError(t *testing.T) {
	webviewTestLocale(t)
	opened := 0
	m := Model{
		HarnessOrigin: "http://127.0.0.1:3080",
		ThemeID:       "deepseek",
		HostError:     "WebView2 runtime unavailable",
		OnOpenHost:    func() { opened++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if opened != 0 {
		t.Fatalf("auto-opened despite an error: %d", opened)
	}
	if !tt.HasText(errcopy.SentenceText(m.HostError)) {
		t.Fatalf("missing sanitized error sentence %q: %v", errcopy.SentenceText(m.HostError), tt.Texts())
	}
	if tt.HasText(m.HostError) {
		t.Fatalf("raw HostError leaked into ZH UI: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewOpenHost)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("retry opens = %d, want 1", opened)
	}
}

func TestViewReservesCaptionControlSafeAreaAcrossWidthsAndThemes(t *testing.T) {
	webviewTestLocale(t)
	labels := []string{
		i18n.T(i18n.KeyWebViewCopyURL),
		i18n.T(i18n.KeyBrowserPreviewReload),
		i18n.T(i18n.KeyWebViewCloseHost),
		i18n.T(i18n.KeyWebViewBack),
	}
	for _, dark := range []bool{false, true} {
		m := Model{
			Title:       "very-long-third-party-plugin-workspace-panel-title",
			ThemeID:     "deepseek",
			SystemDark:  dark,
			HostActive:  true,
			HostURL:     "http://127.0.0.1:3080/?desktopPlugin=qa-panel",
			OnCopyURL:   func(string) {},
			OnReload:    func(string) {},
			OnCloseHost: func() {},
			OnClose:     func() {},
		}
		for _, tc := range []struct {
			name  string
			width int
			bar   ui.TitleBar
		}{
			{name: "narrow-windows", width: 640, bar: ui.TitleBar{Height: 12, Right: 104}},
			{name: "standard-windows", width: 1024, bar: ui.TitleBar{Height: 12, Right: 104}},
			{name: "maximized-windows", width: 1920, bar: ui.TitleBar{Height: 12, Right: 104}},
			{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}},
			{name: "fullscreen-no-caption", width: 1280, bar: ui.TitleBar{}},
		} {
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, tc.width, 600)
			tt.SetDark(dark)
			tt.SetTitleBar(tc.bar)
			maxRight := float32(tc.width) - tc.bar.Right
			for _, label := range labels {
				rect, ok := tt.Find(label)
				if !ok {
					t.Fatalf("dark=%v %s: missing action %q", dark, tc.name, label)
				}
				if rect.X+rect.W > maxRight {
					t.Fatalf("dark=%v %s: action %q overlaps window caption controls: rect=%+v maxRight=%.1f (bar.Right=%.1f)",
						dark, tc.name, label, rect, maxRight, tc.bar.Right)
				}
				if rect.Y+rect.H > float32(PanelChromeHeight) {
					t.Fatalf("dark=%v %s: action %q overflows panel chrome height: rect=%+v height=%d",
						dark, tc.name, label, rect, PanelChromeHeight)
				}
			}
		}
	}
}

func TestViewReservesLeadingTrafficLightInset(t *testing.T) {
	webviewTestLocale(t)
	m := Model{
		Title:      "QA Panel",
		ThemeID:    "deepseek",
		HostActive: true,
		HostURL:    "http://127.0.0.1:3080/?desktopPlugin=qa-panel",
		OnClose:    func() {},
	}
	bar := ui.TitleBar{Height: 28, Left: 76}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	tt.SetTitleBar(bar)
	rect, ok := tt.Find("QA Panel")
	if !ok {
		t.Fatalf("missing panel title: %v", tt.Texts())
	}
	if rect.X < bar.Left {
		t.Fatalf("title overlaps macOS traffic lights: rect=%+v bar.Left=%.1f", rect, bar.Left)
	}
}

func TestPanelChromeVerticalPaddingByOS(t *testing.T) {
	if top, bottom := panelChromeVerticalPadding("windows"); top != 12 || bottom != 4 {
		t.Fatalf("windows padding = (%v, %v), want (12, 4)", top, bottom)
	}
	if top, bottom := panelChromeVerticalPadding("darwin"); top != 8 || bottom != 8 {
		t.Fatalf("darwin padding = (%v, %v), want (8, 8)", top, bottom)
	}
}
