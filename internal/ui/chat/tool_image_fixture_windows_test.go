//go:build windows && imagefixture && !webfixture && !searchfixture && !difffixture && !readfixture && !summaryfixture && !codefixture && !contextfixture && !terminalfixture

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Recorded native fixture: only the owned raster and injected loader are used.
// No Host, network request, OS clipboard or real tool execution occurs.
func TestMain(m *testing.M) {
	prefix := os.Getenv("TOOL_IMAGE_NATIVE_FIXTURE_OUTPUT")
	if prefix == "" {
		os.Exit(m.Run())
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	local := filepath.Join(filepath.Dir(prefix), "tool-image-pass266-fixture-raster.png")
	file, err := os.Create(local)
	if err != nil {
		panic(err)
	}
	raster := image.NewRGBA(image.Rect(0, 0, 640, 360))
	for y := 0; y < 360; y++ {
		for x := 0; x < 640; x++ {
			ink := color.RGBA{32, 132, 180, 255}
			if x >= 320 {
				ink = color.RGBA{240, 80, 40, 255}
			}
			raster.SetRGBA(x, y, ink)
		}
	}
	if err := png.Encode(file, raster); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	var loads atomic.Int32
	SetAttachmentFetcher(func(_ context.Context, id string) (string, error) {
		if id == "retry-image" && loads.Add(1) == 1 {
			return "", fmt.Errorf("recorded image load failure")
		}
		return local, nil
	})
	attachmentLoads.mu.Lock()
	attachmentLoads.entries["loaded-image"] = &attachmentLoad{path: local}
	attachmentLoads.mu.Unlock()
	newCall := func(id, attachment, label string) *session.ToolCallView {
		args := `{"file_path":"/work/argument.png"}`
		head, _ := json.Marshal(map[string]any{"type": "tool/call", "data": map[string]any{"callId": id, "name": "read_image", "arguments": args}})
		result, _ := json.Marshal(map[string]any{"type": "tool/result", "data": map[string]any{"meta": map[string]any{"path": "/work/recorded.png"}, "message": map[string]any{"toolCallId": id, "content": []any{map[string]any{"type": "text", "text": toolImageModelEnvelopeForTest}, map[string]any{"type": "image", "attachment": map[string]any{"attachmentId": attachment, "mediaType": "image/png", "bytes": 4096, "width": 640, "height": 360, "name": label}}, map[string]any{"type": "text", "text": "  Recorded extra text\nSecond raw line"}}}}})
		return &session.ToolCallView{CallID: id, Name: "read_image", Arguments: args, State: "ok", Call: &session.Message{EventType: "tool/call", Raw: head}, Result: &session.Message{EventType: "tool/result", Raw: result}}
	}
	first, second := newCall("native-image", "loaded-image", "Loaded sample"), newCall("native-image-retry", "retry-image", "Retry sample")
	expanded := map[string]bool{}
	opened, files := []string{}, []string{}
	dark := false
	displayed := ""
	var row, retry, following ui.Rect
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Gap(10).Background(chrome.Color(palette.BG)).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, "Recorded image fixture").FontSize(16)
				if ui.Button(c, "Toggle theme").Clicked() {
					dark = !dark
				}
			})
			st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
			opts := ToolImageGalleryOptions{Cwd: "/work", Expanded: &expanded, OnOpenFile: func(path string) { files = append(files, path) }, OnOpen: func(path string) { opened = append(opened, path); displayed = path }}
			ui.Column(c).Width(520).ObserveLayout(func(r ui.Rect) { row = r }).Children(func() { ToolMessageView(c, session.Message{Tool: first}, st, opts) })
			ui.Column(c).Width(520).ObserveLayout(func(r ui.Rect) { retry = r }).Children(func() { ToolMessageView(c, session.Message{Tool: second}, st, opts) })
			ui.Box(c).Height(1).Label("Following image flow").ObserveLayout(func(r ui.Rect) { following = r })
		})
		ImageLightboxOverlay(c, displayed, theme.DefaultID, dark, func() { displayed = "" })
		// A fixture-only overlay shortcut records state even while the preview
		// owns the modal focus scope; production has no such global diagnostic.
		ui.Overlay(c, func() {
			seat := ui.Box(c.Key("fixture-metrics")).Size(0, 0).PassThrough()
			if seat.OverlayShortcut(0, ui.KeyF9) {
				raw, _ := json.MarshalIndent(map[string]any{"time": time.Now().Format(time.RFC3339Nano), "scale": c.DeviceScale(), "expanded": expanded, "dark": dark, "row": row, "retry": retry, "following": following, "opened": opened, "files": files, "lightbox": displayed != "", "loads": loads.Load()}, "", "  ")
				if err := os.WriteFile(prefix+"-metrics.json", raw, 0600); err != nil {
					panic(err)
				}
			}
		})
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Round122 native image fixture", Width: 760, Height: 900, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
