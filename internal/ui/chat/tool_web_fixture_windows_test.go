//go:build windows && webfixture && !searchfixture && !difffixture && !readfixture && !summaryfixture && !codefixture && !contextfixture && !terminalfixture

package chat

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The recorded receipt fixture captures URL callbacks; it never opens a browser,
// contacts a Host, executes a web tool or changes the operating-system clipboard.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_WEB_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	search := toolWebViewCall()
	search.CallID = "native-web-search"
	original, final := "https://original.example.test/a", "https://final.example.test/b"
	fetch := fileToolFixture("web_fetch", `{"url":"`+original+`"}`, map[string]any{"url": final, "statusCode": 200, "truncated": true}, "recorded content omitted")
	fetch.CallID = "native-web-fetch"
	dark := false
	expanded := map[string]bool{}
	opened := []string{}
	var searchRect, fetchRect, following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		if c.Shortcut(0, ui.KeyF9) {
			value, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "search": searchRect, "fetch": fetchRect, "following": following, "opened": opened}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", value, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(palette.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded web fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
			options := ToolImageGalleryOptions{Expanded: &expanded, OnOpenURL: func(raw string) { opened = append(opened, raw) }}
			ui.Column(c).Width(520).MinWidth(0).ObserveLayout(func(r ui.Rect) { searchRect = r }).Children(func() { ToolMessageView(c, session.Message{Tool: search}, st, options) })
			ui.Column(c).Width(520).MinWidth(0).ObserveLayout(func(r ui.Rect) { fetchRect = r }).Children(func() { ToolMessageView(c, session.Message{Tool: fetch}, st, options) })
			ui.Box(c).Height(1).Label("Following web flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round121 native web fixture", Width: 760, Height: 900, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
