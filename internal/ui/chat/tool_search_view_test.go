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
)

func toolSearchViewCall() *session.ToolCallView {
	files := []map[string]any{{"path": "src/a.go", "matches": []map[string]any{}}, {"path": "src/b.go", "matches": []map[string]any{}}}
	for i := 0; i < 4; i++ {
		files[0]["matches"] = append(files[0]["matches"].([]map[string]any), map[string]any{"lineNumber": i + 1, "line": fmt.Sprintf("first-%02d", i)})
	}
	for i := 0; i < 6; i++ {
		files[1]["matches"] = append(files[1]["matches"].([]map[string]any), map[string]any{"lineNumber": i + 20, "line": fmt.Sprintf("second-%02d", i)})
	}
	return fileToolFixture("grep", `{"pattern":"needle","path":"src","include":"*.go"}`, map[string]any{"shape": "matches", "files": files, "total": 30, "truncated": true}, "retained search receipt\nfull output: /recorded/spill.txt")
}
func TestToolSearchViewStructuredGroupsRecoveryAndFullCopy(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := toolSearchViewCall()
			tt, expanded, opened := toolSummaryViewFixture(t, call, 640, dark, false)
			openReadView(t, tt, expanded)
			for _, absent := range []string{"IN", "OUT", "Search matches", "Wrap lines", `"pattern"`} {
				if tt.HasText(absent) {
					t.Fatalf("duplicate%s%v", absent, tt.Texts())
				}
			}
			for _, want := range []string{"Search results card", "Showing 10 of 30 matches · 2 files", "src/a.go 4", "src/b.go 6", "Expand 4 more result lines", "23: second-03", "retained search receipt"} {
				if _, ok := tt.Find(want); !ok && !tt.HasText(want) {
					t.Fatalf("missing%s%v", want, tt.Texts())
				}
			}
			if tt.HasText("22: second-02") {
				t.Fatal("tail header did not consume tail slot")
			}
			card, _ := tt.Find("Search results card")
			toolTreeNear(t, "search left margin", card.X, 24)
			toolTreeNear(t, "search folded height", card.H, 253)
			toolSearchExport(t, tt, fmt.Sprintf("folded-dark-%v", dark))
			if err := tt.Click("src/b.go 6"); err != nil {
				t.Fatal(err)
			}
			if tt.HasText("second-03") || !tt.HasText("4: first-03") {
				t.Fatal(tt.Texts())
			}
			if err := tt.Click("Copy"); err != nil {
				t.Fatal(err)
			}
			want := "src/a.go"
			for i := 0; i < 4; i++ {
				want += fmt.Sprintf("\n%d: first-%02d", i+1, i)
			}
			want += "\n\nsrc/b.go"
			for i := 0; i < 6; i++ {
				want += fmt.Sprintf("\n%d: second-%02d", i+20, i)
			}
			if tt.Clipboard() != want {
				t.Fatalf("fullcopy%q", tt.Clipboard())
			}
			tt.SetClipboard("guard")
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != "guard" {
				t.Fatal("copyguard")
			}
			tt.SetTime(time.Unix(11702, 0))
			tt.Key(0, ui.KeySpace)
			if tt.Clipboard() != want {
				t.Fatal("copyexpiry")
			}
			if err := tt.Click("src/b.go 6"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Expand 4 more result lines"); err != nil {
				t.Fatal(err)
			}
			card, _ = tt.Find("Search results card")
			toolTreeNear(t, "expanded search height", card.H, 341)
			if !tt.HasText("22: second-02") {
				t.Fatal(tt.Texts())
			}
			toolSearchExport(t, tt, fmt.Sprintf("expanded-dark-%v", dark))
			tt.Key(0, ui.KeySpace)
			if tt.HasText("22: second-02") {
				t.Fatal("Space fold")
			}
			if len(*opened) > 0 {
				t.Fatal("search body opened file")
			}
		})
	}
}
func TestToolSearchViewGlobEmptyFallbackAndReadOnly(t *testing.T) {
	paths := []string{}
	for i := 0; i < 11; i++ {
		paths = append(paths, fmt.Sprintf("src/path-%02d.go", i))
	}
	call := fileToolFixture("glob", `{"pattern":"**/*.go"}`, map[string]any{"shape": "paths", "paths": paths, "total": 999, "truncated": false}, "legacy output ignored")
	tt, expanded, _ := toolSummaryViewFixture(t, call, 640, false, false)
	openReadView(t, tt, expanded)
	if !tt.HasText("11 paths") || tt.HasText("999") || tt.HasText("legacy output ignored") || !tt.HasText("Expand 3 more result lines") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != strings.Join(paths, "\n") {
		t.Fatal(tt.Clipboard())
	}
	for _, name := range []string{"grep", "glob"} {
		meta := map[string]any{"shape": "paths", "paths": []string{}, "total": 0, "truncated": false}
		if name == "grep" {
			meta = map[string]any{"shape": "matches", "files": []any{}, "total": 0, "truncated": false}
		}
		empty := fileToolFixture(name, `{"pattern":"x"}`, meta, "nonmetadata text")
		tt, expanded, _ = toolSummaryViewFixture(t, empty, 640, false, false)
		openReadView(t, tt, expanded)
		card, _ := tt.Find("Search results card")
		if !tt.HasText("No results") || tt.HasText("Copy") || card.H != 81 {
			t.Fatalf("empty%s%v%v", name, card, tt.Texts())
		}
	}
	for _, name := range []string{"grep", "glob"} {
		fallback := fileToolFixture(name, `{"pattern":"x"}`, nil, "fallback output")
		tt, expanded, _ = toolSummaryViewFixture(t, fallback, 640, false, false)
		openReadView(t, tt, expanded)
		if tt.HasText("Search results card") || tt.HasText("IN") || !tt.HasText("OUT") || !tt.HasText("fallback output") {
			t.Fatal(tt.Texts())
		}
	}
	ro := toolSearchViewCall()
	tt, expanded, _ = toolSummaryViewFixture(t, ro, 640, false, true)
	openReadView(t, tt, expanded)
	if tt.HasText("Search results card") || !tt.HasText("retained search receipt") {
		t.Fatal("readonly evidence replaced")
	}
}
func toolSearchExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_SEARCH_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	prefix := filepath.Join(dir, "tool-search-pass264-native-software-"+name)
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
	for _, label := range []string{"Search results card", "Showing 10 of 30 matches · 2 files", "src/a.go 4", "src/b.go 6", "Copy", "Copied", "Expand 4 more result lines", "Collapse results", "1: first-00", "23: second-03", "Following summary flow"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	value, _ := json.MarshalIndent(map[string]any{"name": name, "rects": boxes, "texts": tt.Texts()}, "", "  ")
	if err = os.WriteFile(prefix+".json", value, 0600); err != nil {
		t.Fatal(err)
	}
}
