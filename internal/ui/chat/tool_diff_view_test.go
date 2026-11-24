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
	"whaleisle/internal/ui/design"
)

func toolDiffViewCall(n int) *session.ToolCallView {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("created-%02d", i)
	}
	args, _ := json.Marshal(map[string]any{"file_path": "/work/src/create.go", "content": strings.Join(parts, "\n")})
	return fileToolFixture("write", string(args), map[string]any{"path": "/work/src/create.go", "diffs": []any{map[string]any{"path": "/work/src/create.go", "oldText": nil, "newText": strings.Join(parts, "\n")}}}, "Recorded write complete")
}
func TestToolDiffViewSingleCardFoldPaletteAndFullCopy(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := toolDiffViewCall(13)
			tt, expanded, opened := toolSummaryViewFixture(t, call, 640, dark, false)
			openReadView(t, tt, expanded)
			for _, absent := range []string{"IN", "OUT", "Applied diff", "Proposed diff", `"file_path"`, "Recorded write complete", "Expand file"} {
				if tt.HasText(absent) {
					t.Fatalf("duplicate%s%v", absent, tt.Texts())
				}
			}
			for _, want := range []string{"Diff content card", "go", "+13 -0", "Expand 5 more diff lines", "+ created-03", "+ created-09"} {
				if _, ok := tt.Find(want); !ok {
					t.Fatalf("missing%s%v", want, tt.Texts())
				}
			}
			card, _ := tt.Find("Diff content card")
			toolTreeNear(t, "diff left inset", card.X, 24)
			toolTreeNear(t, "diff folded height", card.H, 258)
			scheme := design.Light
			if dark {
				scheme = design.Dark
			}
			wantStripe := design.Base(scheme).SuccessForeground
			add, _ := tt.Find("+ created-00")
			pixel := tt.Image().RGBAAt(int(card.X+1), int(add.Y+4))
			if absToolColor(int(pixel.R)-int(wantStripe.R))+absToolColor(int(pixel.G)-int(wantStripe.G))+absToolColor(int(pixel.B)-int(wantStripe.B)) > 4 {
				t.Fatalf("diff stripe%v want%v", pixel, wantStripe)
			}
			if tt.HasText("created-04") {
				t.Fatal("folded middle visible")
			}
			toolDiffExport(t, tt, fmt.Sprintf("folded-dark-%v", dark))
			if err := tt.Click("Copy"); err != nil {
				t.Fatal(err)
			}
			want := "/work/src/create.go"
			for i := 0; i < 13; i++ {
				want += "\n+ " + fmt.Sprintf("created-%02d", i)
			}
			if tt.Clipboard() != want {
				t.Fatalf("diff fullcopy%q", tt.Clipboard())
			}
			tt.SetClipboard("guard")
			tt.Key(0, ui.KeySpace)
			if tt.Clipboard() != "guard" {
				t.Fatal("copyguard")
			}
			tt.SetTime(time.Unix(11702, 0))
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != want {
				t.Fatal("copytimeout")
			}
			if err := tt.Click("Expand 5 more diff lines"); err != nil {
				t.Fatal(err)
			}
			card, _ = tt.Find("Diff content card")
			toolTreeNear(t, "expanded diff height", card.H, 353)
			if !tt.HasText("created-04") {
				t.Fatal("expand middle")
			}
			toolDiffExport(t, tt, fmt.Sprintf("expanded-dark-%v", dark))
			tt.Key(0, ui.KeySpace)
			if tt.HasText("created-04") {
				t.Fatal("Space collapse")
			}
			if len(*opened) > 0 {
				t.Fatal("body navigation")
			}
		})
	}
}
func TestToolDiffViewAppliedRowsAndPassiveBranches(t *testing.T) {
	old := "same\nold\nend"
	next := "same\nnew\nend"
	call := fileToolFixture("edit", `{"file_path":"src/change.go","old_string":"intended-old","new_string":"intended-new"}`, map[string]any{"path": "/recorded/change.go", "diffs": []any{map[string]any{"path": "/recorded/change.go", "oldText": old, "newText": next}}}, "Recorded edit complete")
	tt, expanded, _ := toolSummaryViewFixture(t, call, 640, false, false)
	openReadView(t, tt, expanded)
	for _, want := range []string{"src/change.go", "  same", "- old", "+ new", "  end"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("missing%s%v", want, tt.Texts())
		}
	}
	if tt.HasText("/recorded/change.go") || tt.HasText("intended-old") || tt.HasText("Recorded edit complete") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "src/change.go\n  same\n- old\n+ new\n  end" {
		t.Fatal(tt.Clipboard())
	}
	call.Result = nil
	call.State = "running"
	tt, expanded, _ = toolSummaryViewFixture(t, call, 640, false, false)
	openReadView(t, tt, expanded)
	if !tt.HasText("- intended-old") || !tt.HasText("+ intended-new") {
		t.Fatal("running intent missing")
	}
	invalid := fileToolFixture("edit", `{"file_path":"src/a.go","old_string":"a","new_string":"b"}`, nil, "fallback edit output")
	tt, expanded, _ = toolSummaryViewFixture(t, invalid, 640, false, false)
	openReadView(t, tt, expanded)
	if tt.HasText("Diff content card") || tt.HasText("IN") || !tt.HasText("OUT") || !tt.HasText("fallback edit output") {
		t.Fatal(tt.Texts())
	}
	for _, state := range []string{"error", "stopped"} {
		invalid.State = state
		tt, expanded, _ = toolSummaryViewFixture(t, invalid, 640, false, false)
		openReadView(t, tt, expanded)
		if tt.HasText("Diff content card") || tt.HasText("IN") {
			t.Fatalf("%s%v", state, tt.Texts())
		}
	}
	ro := toolDiffViewCall(2)
	tt, expanded, _ = toolSummaryViewFixture(t, ro, 640, false, true)
	openReadView(t, tt, expanded)
	if tt.HasText("Diff content card") || !tt.HasText("Recorded write complete") {
		t.Fatal("readonly evidence replaced")
	}
}
func TestToolDiffViewSemanticRowsBothThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := fileToolFixture("edit", `{"file_path":"a.go","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "a.go", "oldText": "before\nold\nafter", "newText": "before\nnew\nafter"}}}, "done")
			tt, expanded, _ := toolSummaryViewFixture(t, call, 640, dark, false)
			openReadView(t, tt, expanded)
			card, _ := tt.Find("Diff content card")
			scheme := design.Light
			bgAlpha := float32(20.0 / 255)
			if dark {
				scheme = design.Dark
				bgAlpha = 31.0 / 255
			}
			tok := design.Base(scheme)
			for _, sample := range []struct {
				label  string
				stripe ui.Color
				bg     ui.Color
			}{
				{"- old", tok.DestructiveForeground, tok.Destructive.Alpha(bgAlpha).Over(tok.Code)},
				{"+ new", tok.SuccessForeground, tok.Success.Alpha(bgAlpha).Over(tok.Code)},
			} {
				r, _ := tt.Find(sample.label)
				stripe := tt.Image().RGBAAt(int(card.X+1), int(r.Y+4))
				fill := tt.Image().RGBAAt(int(card.X+card.W-5), int(r.Y+4))
				if absToolColor(int(stripe.R)-int(sample.stripe.R)) > 1 || absToolColor(int(stripe.G)-int(sample.stripe.G)) > 1 || absToolColor(int(stripe.B)-int(sample.stripe.B)) > 1 {
					t.Fatalf("%s stripe %v want %v", sample.label, stripe, sample.stripe)
				}
				if absToolColor(int(fill.R)-int(sample.bg.R)) > 2 || absToolColor(int(fill.G)-int(sample.bg.G)) > 2 || absToolColor(int(fill.B)-int(sample.bg.B)) > 2 {
					t.Fatalf("%s full row background %v want %v", sample.label, fill, sample.bg)
				}
			}
			toolDiffExport(t, tt, fmt.Sprintf("semantic-dark-%v", dark))
		})
	}
}

func toolDiffExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_DIFF_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	prefix := filepath.Join(dir, "tool-diff-pass263-native-software-"+name)
	file, err := os.Create(prefix + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, tt.Image()); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"Diff content card", "go", "+13 -0", "Copy", "Wrap lines", "Expand 5 more diff lines", "Collapse diff", "+ created-00", "+ created-12", "Following summary flow"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	value, _ := json.MarshalIndent(map[string]any{"name": name, "rects": boxes, "texts": tt.Texts()}, "", "  ")
	if err = os.WriteFile(prefix+".json", value, 0600); err != nil {
		t.Fatal(err)
	}
}
