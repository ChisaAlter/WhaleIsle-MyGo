//go:build windows && difffixture && !readfixture && !summaryfixture && !codefixture && !contextfixture && !terminalfixture

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

// This opt-in fixture displays a retained mutation receipt, with no Host,
// filesystem mutation or code execution. Probes avoid the OS clipboard button.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_DIFF_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	call := toolDiffViewCall(13)
	call.CallID = "native-diff"
	var event map[string]any
	if err := json.Unmarshal(call.Result.Raw, &event); err != nil {
		panic(err)
	}
	meta := event["data"].(map[string]any)["meta"].(map[string]any)
	change := meta["diffs"].([]any)[0].(map[string]any)
	change["newText"] = "recorded long added line " + strings.Repeat("column ", 17) + "\n" + strings.Join(strings.Split(change["newText"].(string), "\n")[1:], "\n")
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
				ui.Text(c, "Recorded diff fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			ui.Column(c).Width(520).MinWidth(0).ObserveLayout(func(r ui.Rect) { row = r }).Children(func() {
				ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{Cwd: "/work", Expanded: &expanded, OnOpenFile: func(path string) { opened = append(opened, path) }})
			})
			ui.Box(c).Height(1).Label("Following diff flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round119 native diff fixture", Width: 760, Height: 760, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
