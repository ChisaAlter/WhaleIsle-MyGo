//go:build windows && summaryfixture && !codefixture && !contextfixture && !terminalfixture

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

// This opt-in fixture records callbacks only: it never opens files, connects to
// Host or executes the read/edit calls that supply its retained example rows.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_SUMMARY_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	read := fileToolFixture("read", `{"file_path":"/work/src/read.go","offset":37}`, map[string]any{"path": "/recorded/src/read.go"}, "Recorded read output")
	read.CallID = "read-summary"
	edit := fileToolFixture("edit", `{"file_path":"/work/src/change.go","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "src/change.go", "oldText": "old\nshared", "newText": "new\nextra\nshared"}}}, "Recorded edit output")
	edit.CallID = "edit-summary"
	expanded := map[string]bool{}
	opened := []string{}
	dark := false
	rows := map[string]ui.Rect{}
	var following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		if c.Shortcut(0, ui.KeyF9) {
			value, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "rows": rows, "following": following, "opened": opened}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", value, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(palette.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded file summary fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			for _, call := range []*session.ToolCallView{read, edit} {
				id := call.CallID
				ui.Column(c.Key(id)).Width(560).MinWidth(0).ObserveLayout(func(r ui.Rect) { rows[id] = r }).Children(func() {
					ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{Cwd: "/work", Expanded: &expanded, OnOpenFile: func(path string) { opened = append(opened, path) }})
				})
			}
			ui.Box(c).Height(1).Label("Following summary flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round117 native file summary fixture", Width: 760, Height: 760, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
