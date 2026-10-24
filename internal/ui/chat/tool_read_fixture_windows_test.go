//go:build windows && readfixture && !summaryfixture && !codefixture && !contextfixture && !terminalfixture

package chat

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The opt-in fixture renders a recorded receipt, with no Host, file opening or
// code execution. The owned probe avoids the actual OS clipboard action.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_READ_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	call := toolReadViewCall(14)
	call.CallID = "native-read"
	var event map[string]any
	if err := json.Unmarshal(call.Result.Raw, &event); err != nil {
		panic(err)
	}
	event["data"].(map[string]any)["meta"].(map[string]any)["lines"].([]any)[0].(map[string]any)["text"] = "recorded long line " + strings.Repeat("column ", 17)
	call.Result.Raw, _ = json.Marshal(event)
	dark := false
	expanded := map[string]bool{}
	opened := []string{}
	var row, following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		if c.Shortcut(0, ui.KeyF9) {
			value, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "row": row, "following": following, "opened": opened}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", value, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(palette.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded read fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			ui.Column(c).Width(520).MinWidth(0).ObserveLayout(func(r ui.Rect) { row = r }).Children(func() {
				ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{Cwd: "/work", Expanded: &expanded, OnOpenFile: func(path string) { opened = append(opened, path) }})
			})
			ui.Box(c).Height(1).Label("Following read flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round118 native read fixture", Width: 760, Height: 760, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
