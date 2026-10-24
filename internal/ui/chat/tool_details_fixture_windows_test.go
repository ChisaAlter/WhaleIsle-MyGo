//go:build windows && detailsfixture && !imagefixture && !webfixture && !searchfixture && !difffixture && !readfixture && !summaryfixture && !codefixture && !contextfixture && !terminalfixture

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

// The fixture only displays recorded local JSON and captures navigation payloads.
// It does not attach to a Host or dispatch goal changes or inspector RPCs.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_DETAILS_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	makeCall := func(id, name, args string, value any) *session.ToolCallView {
		text, _ := json.Marshal(value)
		head, _ := json.Marshal(map[string]any{"type": "tool/call", "data": map[string]any{"callId": id, "name": name, "arguments": args}})
		receipt, _ := json.Marshal(map[string]any{"type": "tool/result", "data": map[string]any{"message": map[string]any{"toolCallId": id, "content": []any{map[string]any{"type": "text", "text": string(text)}}}}})
		return &session.ToolCallView{CallID: id, Name: name, State: "ok", Arguments: args, Call: &session.Message{EventType: "tool/call", Raw: head}, Result: &session.Message{EventType: "tool/result", Raw: receipt}}
	}
	goal := makeCall("native-goal", "get_goal", `{}`, map[string]any{"goal": map[string]any{"id": "recorded-goal", "objective": "Recorded objective", "phase": "active", "revision": 7, "roundsStarted": 123, "maxGoalRounds": 256}, "activation": "disarmed"})
	empty := makeCall("native-empty", "get_goal", `{}`, map[string]any{"goal": nil})
	generic := makeCall("native-generic", "unknown_tool", `{"query":"recorded"}`, map[string]any{"recorded": "generic result"})
	expanded := map[string]bool{}
	dark, narrow := false, false
	type navigation struct {
		CallID    string
		MessageID string
		Seq       float64
	}
	inspected := []navigation{}
	var row, emptyRow, genericRow, following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		if c.Shortcut(0, ui.KeyF9) {
			raw, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "narrow": narrow, "row": row, "empty": emptyRow, "generic": genericRow, "following": following, "inspected": inspected}, "", "  ")
			if err := os.WriteFile(prefix+"-metrics.json", raw, 0600); err != nil {
				panic(err)
			}
		}
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(palette.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded goal details fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
				if ui.Button(c, "Toggle width").Clicked() {
					narrow = !narrow
				}
			})
			st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
			width := float32(620)
			if narrow {
				width = 320
			}
			opts := func(messageID string, seq float64) ToolImageGalleryOptions {
				return ToolImageGalleryOptions{ID: messageID, Expanded: &expanded, OnInspectCall: inspectCallHandler(func(callID, msg string, n float64) { inspected = append(inspected, navigation{callID, msg, n}) }, messageID, seq)}
			}
			ui.Column(c).Width(width).ObserveLayout(func(r ui.Rect) { row = r }).Children(func() {
				ToolMessageView(c, session.Message{ID: "goal-message", Seq: 207, Tool: goal}, st, opts("goal-message", 207))
			})
			ui.Column(c).Width(width).ObserveLayout(func(r ui.Rect) { emptyRow = r }).Children(func() {
				ToolMessageView(c, session.Message{ID: "empty-message", Seq: 208, Tool: empty}, st, opts("empty-message", 208))
			})
			ui.Column(c).Width(width).ObserveLayout(func(r ui.Rect) { genericRow = r }).Children(func() {
				ToolMessageView(c, session.Message{ID: "generic-message", Seq: 209, Tool: generic}, st, opts("generic-message", 209))
			})
			ui.Box(c).Height(1).Label("Following details flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round123 native goal details fixture", Width: 780, Height: 900, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
