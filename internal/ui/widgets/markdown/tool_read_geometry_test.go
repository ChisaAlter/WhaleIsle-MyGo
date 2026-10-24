package markdown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestToolReadPresetGeometryEvidence(t *testing.T) {
	records := []map[string]any{}
	for _, tc := range []struct {
		name                 string
		count, start, height int
	}{
		{"short", 2, 1, 106}, {"empty", 0, 1, 68}, {"empty-text", 1, 1, 87}, {"eight", 8, 1, 220}, {"nine", 9, 1, 239}, {"noncontiguous", 4, 1000, 144},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines := make([]ReadLine, tc.count)
			for i := range lines {
				lines[i] = ReadLine{Number: tc.start + i*2, Text: fmt.Sprintf("line-%d", i)}
			}
			if tc.name == "empty-text" {
				lines[0].Text = ""
			}
			tt := readTestView(t, lines, tc.start+tc.count*2, 616, "")
			card, _ := tt.Find("read-card")
			wrap, _ := tt.Find("Wrap lines")
			if card.W != 616 || card.H != float32(tc.height) || wrap.W != 24 || wrap.H != 24 || wrap.Y-card.Y != 10 {
				t.Fatalf("%s card%v action%v", tc.name, card, wrap)
			}
			boxes := map[string]ui.Rect{"card": card, "wrap": wrap}
			if tc.count > 0 {
				gutter, _ := tt.Find(fmt.Sprint(lines[0].Number))
				boxes["firstGutter"] = gutter
				if gutter.Y-card.Y != 48 || gutter.H != 19 {
					t.Fatalf("%s row geometry %v", tc.name, gutter)
				}
			}
			records = append(records, map[string]any{"case": tc.name, "rects": boxes})
		})
	}
	if dir := os.Getenv("TOOL_READ_EVIDENCE_DIR"); dir != "" {
		value, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-read-pass262-native-preset-geometry.json"), value, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
