//go:build windows && terminalfixture && !contextfixture

package chat

import (
	"encoding/json"
	"fmt"
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

// Opt-in native fixture. Its recorded tool data never connects to a Host or runs
// the displayed commands; F9 exports layout/model facts from the previous frame.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TERMINAL_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	command, output := []string{}, []string{}
	for n := 1; n <= 35; n++ {
		command = append(command, fmt.Sprintf("fixture-command-%02d", n))
		output = append(output, fmt.Sprintf("fixture-output-%02d", n))
	}
	makeCall := func(id, args, text string) *session.ToolCallView {
		raw, _ := json.Marshal(map[string]any{"type": "tool/result", "data": map[string]any{"message": map[string]any{"isError": false, "content": []any{map[string]any{"type": "text", "text": text}}}}})
		return &session.ToolCallView{CallID: id, Name: "pwsh", Arguments: args, State: "ok", Result: &session.Message{EventType: "tool/result", Raw: raw}}
	}
	args, _ := json.Marshal(map[string]any{"command": strings.Join(command, "\n"), "description": "Inspect recorded terminal", "workdir": "C:\\fixture\\repo"})
	foreground := makeCall("foreground", string(args), "\x1b[31m"+strings.Join(output, "\n")+"\x1b[0m\n[exit code: 7]")
	background := makeCall("background", `{"command":"fixture background job","description":"Recorded background acknowledgement","run_in_background":true}`, "background acknowledgement")
	running := &session.ToolCallView{CallID: "running", Name: "pwsh", Arguments: `{"command":"fixture pending command","description":"Recorded running command"}`, State: "running"}
	expanded := map[string]bool{}
	dark := false
	var cardRect, bodyRect, flowRect ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		colors := theme.Resolve(theme.DefaultID).Active(dark)
		st := toolCardStyle(colors, chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border))
		if c.Shortcut(0, ui.KeyF9) {
			value, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "foreground": terminalModelFor(foreground, "C:\\session"), "row": cardRect, "body": bodyRect, "following": flowRect}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", value, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(colors.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded native terminal fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			ui.Column(c).Width(600).MinWidth(0).ObserveLayout(func(r ui.Rect) { cardRect = r }).Children(func() {
				info := ToolLifecycleCardInfo(foreground)
				info.Arguments, info.Output, info.terminal = foreground.Arguments, toolLifecycleOutput(foreground), terminalModelFor(foreground, "C:\\session")
				if terminalModelFailed(info.terminal) {
					info.Status = "error"
				}
				toolCardInfoViewExtra(c, info, st, func() {
					ui.Box(c).Height(1).Label("Recorded terminal body end").ObserveLayout(func(r ui.Rect) { bodyRect = r })
				}, ToolImageGalleryOptions{ID: "foreground", Expanded: &expanded})
			})
			ui.Box(c).Height(1).Label("Following terminal flow").ObserveLayout(func(r ui.Rect) { flowRect = r })
			ui.Column(c).Width(600).Children(func() {
				ToolMessageView(c, session.Message{ID: "background", Tool: background}, st, ToolImageGalleryOptions{ID: "background", Expanded: &expanded})
			})
			ui.Column(c).Width(600).Children(func() {
				ToolMessageView(c, session.Message{ID: "running", Tool: running}, st, ToolImageGalleryOptions{ID: "running", Expanded: &expanded})
			})
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round115 native terminal fixture", Width: 760, Height: 760, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
