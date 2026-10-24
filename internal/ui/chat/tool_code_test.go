package chat

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestToolCodeProgramUsesAuthoredCodeOrFormattedArguments(t *testing.T) {
	for _, tc := range []struct {
		name, args, want string
		present          bool
	}{
		{"absent", "", "", false},
		{"code", `{"code":"  const n = 1;\n\n","description":"Run"}`, "  const n = 1;\n\n", true},
		{"literal", "{\"code\":", "{\"code\":", true},
		{"empty code field", `{"code":""}`, "{\n  \"code\": \"\"\n}", true},
		{"nonstring field", `{"code":42}`, "{\n  \"code\": 42\n}", true},
		{"array", `[1,"x"]`, "[\n  1,\n  \"x\"\n]", true},
		{"null", `null`, "null", true},
		{"whitespace code", `{"code":" \t\n"}`, " \t\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, present := toolCodeProgram(tc.args)
			if got != tc.want || present != tc.present {
				t.Fatalf("program=%q/%v want%q/%v", got, present, tc.want, tc.present)
			}
		})
	}
}

func TestToolCodeBodyCompositionGeometryAndCopy(t *testing.T) {
	previous := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(previous) })
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			code := "const answer = 42;\n"
			args, _ := json.Marshal(map[string]any{"code": code, "description": "Compute answer"})
			info := ToolCardInfo{Name: "run_code", Arguments: string(args), Output: "42", Status: "ok"}
			tt, expanded := toolRowViewFixture(t, &info, 640, dark)
			(*expanded)["row"] = true
			tt.Frame()
			tt.SetTime(time.Unix(7800, 0).Add(250 * time.Millisecond))
			if tt.HasText("IN") || !tt.HasText("OUT") || !tt.HasText("typescript") || tt.HasText(`"description"`) {
				t.Fatal(tt.Texts())
			}
			viewport, ok := tt.Find("Tool program")
			if !ok {
				t.Fatal("missing program scrollport")
			}
			toolTreeNear(t, "code viewport height", viewport.H, 95)
			card, _ := tt.Find("Tool code card")
			toolTreeNear(t, "code indent", card.X, 24)
			toolTreeNear(t, "code card height", card.H, 87)
			copyButton, _ := tt.Find("Copy")
			toolTreeNear(t, "toolbar action width", copyButton.W, 24)
			toolTreeNear(t, "toolbar action height", copyButton.H, 24)
			label, _ := tt.Find("typescript")
			toolTreeNear(t, "toolbar label inset", label.X-card.X, 22)
			out, _ := tt.Find("OUT")
			toolTreeNear(t, "separate output y", out.Y, 144)
			if err := tt.Click("Copy"); err != nil {
				t.Fatal(err)
			}
			if tt.Clipboard() != strings.TrimSuffix(code, "\n") || !tt.HasText("Copied") {
				t.Fatalf("clipboard=%q text=%q", tt.Clipboard(), tt.Texts())
			}
			tt.SetClipboard("guarded")
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != "guarded" {
				t.Fatal("copied feedback allowed another write")
			}
			tt.SetTime(time.Unix(7800, 0).Add(1400 * time.Millisecond))
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != strings.TrimSuffix(code, "\n") {
				t.Fatal("focused code copy did not resume")
			}
			toolCodeExport(t, tt, fmt.Sprintf("short-dark-%v", dark))
		})
	}
}

func TestToolCodeBodyWrapAndIndependentScroll(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"code": strings.Repeat("const fixture = '"+strings.Repeat("wide", 30)+"';\n", 35), "description": "Inspect program"})
	info := ToolCardInfo{Name: "run_code", Arguments: string(args), Output: strings.Repeat("recorded output\n", 40), Status: "ok"}
	tt, expanded := toolRowViewFixture(t, &info, 640, false)
	(*expanded)["row"] = true
	tt.Frame()
	tt.SetTime(time.Unix(7800, 0).Add(250 * time.Millisecond))
	program, _ := tt.Find("Tool program")
	output, _ := tt.Find("OUT")
	copyBefore, _ := tt.Find("Copy")
	toolTreeNear(t, "code scroll cap", program.H, 260)
	toolTreeNear(t, "output own cap", output.H, 174)
	before := image.NewRGBA(tt.Image().Bounds())
	copy(before.Pix, tt.Image().Pix)
	tt.Scroll(program.X+100, program.Y+program.H/2, 0, 120)
	copyAfter, _ := tt.Find("Copy")
	outputAfter, _ := tt.Find("OUT")
	toolTreeNear(t, "sticky copy clamp", copyAfter.Y, program.Y+10)
	if outputAfter != output || copyAfter.Y > copyBefore.Y {
		t.Fatal("code wheel moved independent output or toolbar")
	}
	if !toolCodePixelsChanged(before, tt.Image(), program) {
		t.Fatal("code wheel did not change painted source")
	}
	// aria-pressed semantics use the stable Wrap lines name, while the tooltip
	// describes the action. Focused Enter toggles without bubbling to the row.
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if !(*expanded)["row"] {
		t.Fatal("code toolbar Enter toggled disclosure")
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	code, _ := toolCodeProgram(info.Arguments)
	if tt.Clipboard() != strings.TrimSuffix(code, "\n") {
		t.Fatal("scrolled code copy dropped unmounted lines")
	}
	toolCodeExport(t, tt, "independent-scroll")
}

func TestToolCodeBodyFallbackReadOnlyAndExactName(t *testing.T) {
	for _, tc := range []struct {
		name, args, output string
		code, io           bool
	}{
		{"run_code", `{"code":""}`, "", true, false},
		{"run_code", `{"code":`, "error", true, true},
		{"run_code", "", "result", false, true},
		{"Run_Code", `{"code":"literal"}`, "result", false, true},
		{"functions.run_code", `{"code":"literal"}`, "result", false, true},
	} {
		t.Run(tc.name+tc.args, func(t *testing.T) {
			info := ToolCardInfo{Name: tc.name, Arguments: tc.args, Output: tc.output, Status: "ok"}
			tt, expanded := toolRowViewFixture(t, &info, 640, false)
			(*expanded)["row"] = true
			tt.Frame()
			if tt.HasText("Tool program") != tc.code || tt.HasText("OUT") != tc.io {
				t.Fatal(tt.Texts())
			}
			if tc.code && tt.HasText("IN") {
				t.Fatal("code duplicated input JSON")
			}
		})
	}
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	info := ToolCardInfo{Name: "run_code", Arguments: `{"code":"recorded();"}`, Output: "result", Status: "ok"}
	expanded := map[string]bool{"history": true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			toolCardInfoView(c, info, ToolCardStyle{}, ToolImageGalleryOptions{ID: "history", Expanded: &expanded, TextOnly: true})
		})
	}, 640, 500)
	if tt.HasText("typescript") || !tt.HasText("IN") || !tt.HasText("OUT") {
		t.Fatal("read-only call evidence replaced", tt.Texts())
	}
}

func toolCodePixelsChanged(a, b *image.RGBA, r ui.Rect) bool {
	for y := int(r.Y + 50); y < int(r.Y+r.H-8); y++ {
		for x := int(r.X + 22); x < int(r.X+180); x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return true
			}
		}
	}
	return false
}

func toolCodeExport(t *testing.T, tt *ui.Tester, suffix string) {
	t.Helper()
	dir := os.Getenv("TOOL_CODE_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "tool-code-pass260-native-software-"+suffix)
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
	for _, name := range []string{"Tool program", "Tool code card", "Code", "typescript", "Copy", "Copied", "Wrap lines", "OUT", "following flow"} {
		if r, ok := tt.Find(name); ok {
			boxes[name] = r
		}
	}
	data, _ := json.MarshalIndent(map[string]any{"scope": "MyGo software rendering, no OS hardware input or original app claim", "boxes": boxes, "texts": tt.Texts()}, "", "  ")
	if err = os.WriteFile(base+".json", append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
