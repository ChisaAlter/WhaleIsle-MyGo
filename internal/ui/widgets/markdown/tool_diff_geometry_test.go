package markdown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestToolDiffPresetGeometryEvidence(t *testing.T) {
	records := []map[string]any{}
	for _, tc := range []struct {
		name          string
		count, height int
	}{{"unchanged", 1, 87}, {"blank-added", 2, 106}, {"short", 4, 144}, {"nine", 9, 239}, {"ten", 10, 258}, {"fourteen", 14, 258}} {
		t.Run(tc.name, func(t *testing.T) {
			rows := []DiffLine{{"path", "src/a.go"}}
			for i := 1; i < tc.count; i++ {
				rows = append(rows, DiffLine{"add", fmt.Sprintf("line-%d", i)})
			}
			if tc.name == "blank-added" {
				rows[1].Text = ""
			}
			tt := diffTestView(t, rows, "entire patch", 616)
			card, _ := tt.Find("diff-card")
			wrap, _ := tt.Find("Wrap lines")
			copy, _ := tt.Find("Copy code")
			first, _ := tt.Find("src/a.go")
			if card.W != 616 || card.H != float32(tc.height) || wrap.W != 24 || wrap.H != 24 || wrap.Y-card.Y != 10 || copy.X-wrap.X != 28 || first.Y-card.Y != 48 || first.H != 19 || first.W != 616 {
				t.Fatalf("%s card%v wrap%v copy%v first%v", tc.name, card, wrap, copy, first)
			}
			records = append(records, map[string]any{"case": tc.name, "rects": map[string]ui.Rect{"card": card, "wrap": wrap, "copy": copy, "first": first}})
		})
	}
	if dir := os.Getenv("TOOL_DIFF_EVIDENCE_DIR"); dir != "" {
		value, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-diff-pass263-native-preset-geometry.json"), value, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
