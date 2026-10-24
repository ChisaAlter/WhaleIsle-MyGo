//go:build windows && codefixture && !terminalfixture && !contextfixture

package chat

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The opt-in recorded fixture never executes displayed code or connects to a
// Host. F9 exports the previous frame's layout in DIPs for owned-window probes.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_CODE_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	args, _ := json.Marshal(map[string]any{"description": "Inspect recorded program", "code": "const sentence = '" + strings.Repeat("wide ", 25) + "';\n"})
	info := ToolCardInfo{Name: "run_code", Arguments: string(args), Output: "recorded output", Status: "ok"}
	expanded := map[string]bool{}
	dark := false
	var row, following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		colors := theme.Resolve(theme.DefaultID).Active(dark)
		st := toolCardStyle(colors, chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border))
		if c.Shortcut(0, ui.KeyF9) {
			value, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "row": row, "following": following}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", value, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(colors.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded run_code fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			ui.Column(c).Width(320).MinWidth(0).ObserveLayout(func(r ui.Rect) { row = r }).Children(func() { toolCardInfoView(c, info, st, ToolImageGalleryOptions{ID: "code", Expanded: &expanded}) })
			ui.Box(c).Height(1).Label("Following code flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round116 native run_code fixture", Width: 760, Height: 760, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
