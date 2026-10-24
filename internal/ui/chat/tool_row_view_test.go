package chat

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func toolRowViewFixture(t *testing.T, info *ToolCardInfo, width int, dark bool) (*ui.Tester, *map[string]bool) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	expanded := map[string]bool{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(palette.BG)).Children(func() {
			toolCardInfoView(c, *info, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{ID: "row", Expanded: &expanded})
			ui.Box(c).Height(1).Label("following flow")
		})
	}, width, 500)
	tt.SetTime(time.Unix(7800, 0))
	tt.Move(float32(width-5), 495)
	return tt, &expanded
}

func TestToolRowViewGenericIOGeometryKeyboardAndCopyOwnership(t *testing.T) {
	info := ToolCardInfo{Name: "custom", Arguments: "argument", Output: "result", Status: "ok"}
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			tt, expanded := toolRowViewFixture(t, &info, 640, dark)
			title := toolRowTitle(info.Name)
			row, ok := tt.Find(title)
			if !ok {
				t.Fatal("missing title")
			}
			toolTreeNear(t, "title inset", row.X, 42)
			toolTreeNear(t, "row height", row.H, 24)
			if tt.HasText("IN") || tt.HasText("OUT") {
				t.Fatal("collapsed body mounted")
			}
			toolRowExport(t, tt, fmt.Sprintf("idle-dark-%v", dark))
			tt.ClickAt(610, 32)
			tt.SetTime(time.Unix(7800, 0).Add(250 * time.Millisecond))
			if !(*expanded)["row"] {
				t.Fatal("empty end of header did not toggle")
			}
			input, inOK := tt.Find("IN")
			output, outOK := tt.Find("OUT")
			if !inOK || !outOK {
				t.Fatalf("generic sections missing: %q", tt.Texts())
			}
			toolTreeNear(t, "input x", input.X, 25)
			toolTreeNear(t, "input y", input.Y, 49)
			toolTreeNear(t, "input height", input.H, 40)
			toolTreeNear(t, "output y", output.Y, 89.5)
			toolTreeNear(t, "output height", output.H, 40)
			flow, _ := tt.Find("following flow")
			toolTreeNear(t, "expanded own height", flow.Y-20, 114.5)
			if tt.HasText(i18n.T(i18n.KeyChatToolCopy)) {
				t.Fatal("generic IO gained an extra clipboard control")
			}
			toolRowExport(t, tt, fmt.Sprintf("open-dark-%v", dark))
			tt.Key(0, ui.KeySpace)
			if (*expanded)["row"] || tt.HasText("argument") && tt.HasText("result") {
				t.Fatal("Space failed to close the body")
			}
			flow, _ = tt.Find("following flow")
			toolTreeNear(t, "closing hold keeps flow", flow.Y-20, 114.5)
			tt.SetTime(time.Unix(7800, 0).Add(500 * time.Millisecond))
			flow, _ = tt.Find("following flow")
			toolTreeNear(t, "after closing hold", flow.Y-20, 24)
			tt.Key(0, ui.KeyEnter)
			if !(*expanded)["row"] {
				t.Fatal("Enter failed to reopen")
			}
		})
	}
}

func TestToolRowViewPreparingAndEmptyStayPlainThenRunCanExpand(t *testing.T) {
	info := ToolCardInfo{Name: "read", Arguments: `{"file_path":"document.txt"}`, Status: "preparing"}
	tt, expanded := toolRowViewFixture(t, &info, 640, false)
	tt.ClickAt(100, 32)
	tt.Key(0, ui.KeyEnter)
	if len(*expanded) != 0 || tt.HasText("IN") || tt.Cursor() == ui.CursorPointer {
		t.Fatal("preparing row became a disclosure")
	}
	info.Status = "running"
	tt.Frame()
	tt.ClickAt(600, 32)
	if !(*expanded)["row"] {
		t.Fatal("dispatched row did not become expandable")
	}
	info = ToolCardInfo{Name: "custom", Status: "ok"}
	tt.Frame()
	tt.SetTime(time.Unix(7800, 0).Add(time.Second))
	flow, _ := tt.Find("following flow")
	toolTreeNear(t, "empty call body", flow.Y-20, 24)
	tt.ClickAt(100, 32)
	flow, _ = tt.Find("following flow")
	toolTreeNear(t, "empty call click", flow.Y-20, 24)
}

func TestToolRowViewIndependentScrollAndFontAxis(t *testing.T) {
	info := ToolCardInfo{Name: "custom", Arguments: strings.Repeat("input long line\n", 50), Output: "short output", Status: "ok"}
	tt, expanded := toolRowViewFixture(t, &info, 640, false)
	(*expanded)["row"] = true
	tt.Frame()
	input, _ := tt.Find("IN")
	output, _ := tt.Find("OUT")
	toolTreeNear(t, "input capped outer", input.H, 174)
	toolTreeNear(t, "short output independent", output.H, 40)
	tt.SetTime(time.Unix(7800, 0).Add(250 * time.Millisecond))
	before, _ := tt.Find("short output")
	textBefore, _ := tt.Find(info.Arguments)
	tt.Move(input.X+100, input.Y+50)
	tt.Scroll(input.X+100, input.Y+50, 0, 100)
	after, _ := tt.Find("short output")
	textAfter, _ := tt.Find(info.Arguments)
	if after != before || textAfter.Y >= textBefore.Y {
		t.Fatal("input scroll failed or moved the output section")
	}
	toolRowExport(t, tt, "independent-input-scroll")
	prior := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(prior) })
	next := prior.Clone()
	next.Chat.Size = 18
	theme.SetPreferences(next)
	tt.Frame()
	title, _ := tt.Find(toolRowTitle("custom"))
	toolTreeNear(t, "font18 title inset", title.X, 46)
	toolTreeNear(t, "font18 row", title.H, 28)
	toolRowExport(t, tt, "font18")
}

func TestToolRowViewRunningRetainsSelectedChatFamily(t *testing.T) {
	prior := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(prior) })
	next := prior.Clone()
	next.Chat.Family, next.Interface.Family = "Consolas", "Arial"
	theme.SetPreferences(next)
	info := ToolCardInfo{Name: "custom", Arguments: `{"description":"summary"}`, Status: "ok"}
	tt, _ := toolRowViewFixture(t, &info, 640, false)
	title := toolRowTitle(info.Name)
	settled, ok := tt.Find(title)
	if !ok {
		t.Fatal("settled title missing")
	}
	info.Status = "running"
	tt.Frame()
	running, ok := tt.Find(title)
	if !ok {
		t.Fatal("running title missing")
	}
	toolTreeNear(t, "selected family width across lifecycle", running.W, settled.W)
}

func toolRowExport(t *testing.T, tt *ui.Tester, suffix string) {
	t.Helper()
	dir := os.Getenv("TOOL_ROW_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "tool-row-pass258-native-software-"+suffix)
	f, err := os.Create(base + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"IN", "OUT", "Tool input/output", "following flow", toolRowTitle("custom")} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	payload, err := json.MarshalIndent(map[string]any{"scope": "MyGo software renderer only; not OS input, hardware hover or live Electron", "boxes": boxes, "texts": tt.Texts()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".json", payload, 0644); err != nil {
		t.Fatal(err)
	}
}
