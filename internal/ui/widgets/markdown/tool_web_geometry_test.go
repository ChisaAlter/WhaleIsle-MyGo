package markdown

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestToolWebPresetGeometryEvidence(t *testing.T) {
	records := []map[string]any{}
	for _, tc := range []struct {
		name string
		data WebData
	}{
		{"empty", WebData{Kind: "search"}},
		{"empty-truncated", WebData{Kind: "search", Truncated: true}},
		{"whitespace-answer", WebData{Kind: "search", Answer: " "}},
		{"answer-only", WebData{Kind: "search", Answer: "Answer only"}},
		{"source-only", WebData{Kind: "search", Sources: []WebSource{{URL: "https://source.example.test/a", Title: "Source title"}}}},
		{"source-full", WebData{Kind: "search", Sources: []WebSource{{URL: "https://source.example.test/a", Title: "Source title", Snippet: "Snippet text", PublishedAt: "2026-10-24"}}}},
		{"source-space-details", WebData{Kind: "search", Sources: []WebSource{{URL: "https://source.example.test/a", Title: " ", Snippet: " ", PublishedAt: " "}}}},
		{"two-sources", WebData{Kind: "search", Sources: []WebSource{{URL: "https://source.example.test/a", Title: "Source title", Snippet: "Snippet text", PublishedAt: "2026-10-24"}, {URL: "https://source.example.test/b", Title: "Second title"}}, Truncated: true}},
		{"fetch", WebData{Kind: "fetch", URL: "https://final.example.test/b", StatusCode: 200}},
		{"fetch-truncated", WebData{Kind: "fetch", URL: "https://final.example.test/b", StatusCode: 200, Truncated: true}},
		{"fetch-unsafe", WebData{Kind: "fetch", URL: "file:///tmp/page", StatusCode: -404, Truncated: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt, _ := webTestView(t, tc.data, 616)
			card, _ := tt.Find("web-card")
			boxes := map[string]ui.Rect{"card": card}
			for _, label := range []string{"No results found", "Answer only", "1.", "2.", "Source title", "Second title", "Snippet text", "2026-10-24", "Source list truncated", "HTTP 200", "HTTP -404", "Content truncated", "https://final.example.test/b", "file:///tmp/page"} {
				if r, ok := tt.Find(label); ok {
					boxes[label] = r
				}
			}
			if card.W != 616 {
				t.Fatal(card)
			}
			// Installed DPR1 geometry: list marker's natural inline row20,
			// source with one snippet/date63 and fetch19+6+20, plus card24.
			want := map[string]float32{"empty": 44, "empty-truncated": 72, "whitespace-answer": 32, "answer-only": 56, "source-only": 44, "source-full": 87, "source-space-details": 46, "two-sources": 145, "fetch": 69, "fetch-truncated": 69, "fetch-unsafe": 69}[tc.name]
			if card.H != want {
				t.Fatalf("%s height%v expected%v", tc.name, card.H, want)
			}
			if link, ok := boxes["Source title"]; ok && tc.name == "source-only" && link.W >= 553 {
				t.Fatal("source anchor stretches through blank row")
			}
			records = append(records, map[string]any{"case": tc.name, "rects": boxes})
		})
	}
	if dir := os.Getenv("TOOL_WEB_EVIDENCE_DIR"); dir != "" {
		value, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-web-pass265-native-preset-geometry.json"), value, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
