package markdown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestToolSearchPresetGeometryEvidence(t *testing.T) {
	records := []map[string]any{}
	for _, tc := range []struct {
		name   string
		data   SearchData
		height int
	}{
		{"empty-paths", SearchData{Kind: "paths"}, 81},
		{"empty-matches", SearchData{Kind: "matches"}, 81},
		{"one-path", SearchData{Kind: "paths", Paths: []string{"src/a.go"}, Total: 1}, 80},
		{"zero-match-file", SearchData{Kind: "matches", Files: []SearchFile{{Path: "empty.go"}}}, 80},
		{"short-matches", SearchData{Kind: "matches", Files: []SearchFile{{Path: "src/a.go", Matches: []SearchMatch{{1, "first"}, {2, "second"}}}}, Total: 2}, 124},
		{"paths-eight", searchGeometryPaths(8), 234},
		{"paths-nine", searchGeometryPaths(9), 253},
		{"tail-restored", searchTestFiles(), 253},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := searchTestView(t, tc.data, 616)
			card, _ := tt.Find("search-card")
			copy, copyExists := tt.Find("Copy")
			if card.W != 616 || card.H != float32(tc.height) {
				t.Fatalf("%s geometry%v", tc.name, card)
			}
			boxes := map[string]ui.Rect{"card": card}
			if copyExists {
				boxes["copy"] = copy
				if copy.H != 20 || copy.Y != 9 {
					t.Fatalf("%s copy%v", tc.name, copy)
				}
			}
			for _, label := range []string{"src/a.go", "src/a.go 2", "src/a.go 4", "src/b.go 6", "1: first", "1: first-00", "path-00.go", "Expand 1 more result lines", "Expand 4 more result lines", "empty.go 0"} {
				if r, ok := tt.Find(label); ok {
					boxes[label] = r
				}
			}
			if tc.name == "short-matches" {
				path, _ := tt.Find("src/a.go")
				header, _ := tt.Find("src/a.go 2")
				match, _ := tt.Find("1: first")
				if path.X != 14 || header.Y != 46 || header.H != 22 || match.Y != 68 || match.H != 22 {
					t.Fatalf("left/header/row geometry%v%v%v", path, header, match)
				}
			}
			records = append(records, map[string]any{"case": tc.name, "rects": boxes})
		})
	}
	if dir := os.Getenv("TOOL_SEARCH_EVIDENCE_DIR"); dir != "" {
		value, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-search-pass264-native-preset-geometry.json"), value, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestToolSearchFontPreferencesPreserveMinimumAndBanner(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	for _, tc := range []struct{ codeSize, expectedHeight float32 }{{9, 80}, {15, 80}, {19, 83}} {
		prefs := theme.Defaults()
		prefs.Code.Size = tc.codeSize
		prefs.Interface.Size = 22
		theme.SetPreferences(prefs)
		tt := ui.NewTester(func(c *ui.Context) {
			theme.ApplyInterface(c)
			ui.Column(c).Fill().Children(func() {
				ToolSearchBlock(c, SearchData{Kind: "paths", Paths: []string{"one.go"}, Total: 1}, 8, Style{FG: ui.Hex("#0f1115"), Muted: ui.Hex("#61666b"), CodeBG: ui.Hex("#f9fafb")}, searchTestLabels()).Label("search-card")
			})
		}, 616, 300)
		tt.Frame()
		card, _ := tt.Find("search-card")
		copy, _ := tt.Find("Copy")
		line, _ := tt.Find("one.go")
		if card.H != tc.expectedHeight || copy.H != 20 || copy.Y != 9 || line.H != max(float32(22), tc.codeSize+6) {
			t.Fatalf("code%v interface22: card%v copy%v row%v", tc.codeSize, card, copy, line)
		}
	}
}

func searchGeometryPaths(n int) SearchData {
	data := SearchData{Kind: "paths", Total: float64(n)}
	for i := 0; i < n; i++ {
		data.Paths = append(data.Paths, fmt.Sprintf("path-%02d.go", i))
	}
	return data
}
