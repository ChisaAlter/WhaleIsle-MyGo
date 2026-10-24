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
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func toolSummaryViewFixture(t *testing.T, call *session.ToolCallView, width int, dark bool, textOnly bool) (*ui.Tester, *map[string]bool, *[]string) {
	t.Helper()
	previousLocale, previousPreferences := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(previousLocale); theme.SetPreferences(previousPreferences) })
	expanded := map[string]bool{}
	opened := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(palette.BG)).Children(func() {
			ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{ID: "summary", Cwd: "/work", Home: "/home/me", TextOnly: textOnly, Expanded: &expanded, OnOpenFile: func(path string) { opened = append(opened, path) }})
			ui.Box(c).Height(1).Label("Following summary flow")
		})
	}, width, 580)
	tt.SetTime(time.Unix(11700, 0))
	tt.Move(float32(width-5), 575)
	return tt, &expanded, &opened
}

func TestToolSummaryViewPathClickKeyboardAndDisclosure(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := fileToolFixture("read", `{"file_path":"/work/src/read.go","offset":12}`, map[string]any{"path": "/recorded/src/read.go"}, "retained result")
			tt, expanded, opened := toolSummaryViewFixture(t, call, 640, dark, false)
			path, ok := tt.Find("src/read.go")
			if !ok {
				t.Fatal(tt.Texts())
			}
			toolTreeNear(t, "path row height", path.H, 24)
			flow, _ := tt.Find("Following summary flow")
			toolTreeNear(t, "collapsed flow", flow.Y, 44)
			if len(*opened) > 0 {
				t.Fatal("render navigated")
			}
			if err := tt.Click("src/read.go"); err != nil {
				t.Fatal(err)
			}
			for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
				tt.Key(0, key)
			}
			if len(*opened) != 3 {
				t.Fatalf("path activation count%d", len(*opened))
			}
			for _, target := range *opened {
				if target != "/recorded/src/read.go:12" {
					t.Fatalf("wrong paired target%q", target)
				}
			}
			if (*expanded)["summary|tool:"+call.CallID] || tt.HasText("retained result") {
				t.Fatalf("path changed disclosure%v", *expanded)
			}
			// Parent retains independent keyboard focus and actions.
			tt.Key(ui.Shift, ui.KeyTab)
			tt.Key(0, ui.KeyEnter)
			if !(*expanded)["summary|tool:"+call.CallID] {
				t.Fatalf("parent Enter did not expand%v", *expanded)
			}
			tt.SetTime(time.Unix(11700, 0).Add(250 * time.Millisecond))
			if !tt.HasText("retained result") {
				t.Fatal("expanded result absent")
			}
			if err := tt.Click("src/read.go"); err != nil {
				t.Fatal(err)
			}
			if len(*opened) != 4 || !(*expanded)["summary|tool:"+call.CallID] {
				t.Fatal("open-row path action changed disclosure")
			}
			toolSummaryExport(t, tt, fmt.Sprintf("read-dark-%v", dark))
		})
	}
}

func TestToolSummaryViewDiffTotalsSizeAndTruncation(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := fileToolFixture("edit", `{"file_path":"/work/src/change.go","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "a", "oldText": "old\nshared", "newText": "new\nextra\nshared"}}}, "done")
			tt, expanded, _ := toolSummaryViewFixture(t, call, 640, dark, false)
			if !tt.HasText("1KB") || !tt.HasText("+2 -1") {
				t.Fatal(tt.Texts())
			}
			path, _ := tt.Find("src/change.go")
			size, _ := tt.Find("1KB")
			stat, _ := tt.Find("+2 -1")
			toolTreeNear(t, "content suffix gap", size.X-path.X-path.W, 4)
			toolTreeNear(t, "diff stat gap", stat.X-size.X-size.W, 10)
			toolTreeNear(t, "content height", size.H, 24)
			// Find reports viewport-clipped bounds: the24DIP line is translated
			// down.5DIP inside the fixed-height disclosure, as installed CSS does.
			toolTreeNear(t, "stat visible height", stat.H, 23.5)
			toolTreeNear(t, "stat downshift", stat.Y-size.Y, .5)
			if toolSummaryHasColor(tt, stat, ui.Hex("#22c55e")) {
				t.Fatal("resting totals showed success color")
			}
			toolSummaryExport(t, tt, fmt.Sprintf("mutation-rest-dark-%v", dark))
			tt.ClickAt(610, 32)
			tt.Move(635, 575)
			if !(*expanded)["summary|tool:"+call.CallID] {
				t.Fatal("empty header end failed to open")
			}
			stat, _ = tt.Find("+2 -1")
			if !toolSummaryHasColor(tt, stat, ui.Hex("#22c55e")) {
				t.Fatal("open totals did not use installed success alias")
			}
			tt.SetTime(time.Unix(11700, 0).Add(250 * time.Millisecond))
			toolSummaryExport(t, tt, fmt.Sprintf("mutation-open-dark-%v", dark))
			args, _ := json.Marshal(map[string]any{"file_path": "/work/" + strings.Repeat("long-directory/", 12) + "change.go", "old_string": "old", "new_string": "new"})
			call.Arguments = string(args)
			tt.SetSize(300, 580)
			stat, ok := tt.Find("+2 -1")
			if !ok || stat.X+stat.W > 280.1 {
				t.Fatalf("totals clipped after long path %+v text%q", stat, tt.Texts())
			}
			toolSummaryExport(t, tt, fmt.Sprintf("mutation-narrow-dark-%v", dark))
		})
	}
}

func TestToolSummaryViewCuesAndReadOnlyRemainPassive(t *testing.T) {
	for _, state := range []string{"error", "stopped", "ok"} {
		call := fileToolFixture("write", `{"file_path":"/work/a","content":"new"}`, nil, "failure first\nsecond")
		call.State = state
		readOnly := state == "ok"
		tt, _, opened := toolSummaryViewFixture(t, call, 640, false, readOnly)
		if tt.HasText("1KB") || tt.HasText("+1 -0") {
			t.Fatalf("%s/readOnly%v acquired stats", state, readOnly)
		}
		tt.ClickAt(170, 32)
		tt.Key(0, ui.KeyEnter)
		tt.Key(0, ui.KeySpace)
		if len(*opened) != 0 {
			t.Fatalf("%s/readOnly%v opened%q", state, readOnly, *opened)
		}
	}
}

func toolSummaryHasColor(tt *ui.Tester, r ui.Rect, ink ui.Color) bool {
	image := tt.Image()
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := image.RGBAAt(x, y)
			if absToolColor(int(p.R)-int(ink.R))+absToolColor(int(p.G)-int(ink.G))+absToolColor(int(p.B)-int(ink.B)) < 24 {
				return true
			}
		}
	}
	return false
}

func absToolColor(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func toolSummaryExport(t *testing.T, tt *ui.Tester, suffix string) {
	t.Helper()
	dir := os.Getenv("TOOL_SUMMARY_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "tool-summary-pass261-native-software-"+suffix)
	file, err := os.Create(base + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, text := range tt.Texts() {
		if r, ok := tt.Find(text); ok {
			boxes[text] = r
		}
	}
	value, _ := json.MarshalIndent(map[string]any{"scope": "MyGo software DPR1; recorded data, no Host or physical input", "boxes": boxes}, "", "  ")
	if err = os.WriteFile(base+".json", append(value, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
