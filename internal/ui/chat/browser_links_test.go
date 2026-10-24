package chat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/widgets/markdown"
)

func TestBrowserLinksFixture(t *testing.T) {
	links := markdown.Links("[docs](https://example.com/docs) https://example.com/plain <https://example.com/angle> [bad](javascript:alert) [private](https://user:pass@example.com) [missing](https://) https://example.com/plain")
	if len(links) != 3 {
		t.Fatalf("links: %+v", links)
	}
	body := strings.Repeat("prefix ", 45) + "[result](https://example.com/result)"
	raw, _ := json.Marshal(map[string]any{"data": map[string]any{"message": map[string]any{"content": []map[string]string{{"type": "text", "text": body}}}}})
	card := ParseToolCard("tool/result", raw)
	if len(card.Links) != 1 || card.Links[0].URL != "https://example.com/result" {
		t.Fatalf("truncated result lost target: %+v", card)
	}
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ToolCardView(c, "tool/result", raw, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{OnOpenURL: func(s string) { opened = s }})
	}, 800, 600)
	tt.Frame()
	if opened != "" {
		t.Fatal("passive render navigated")
	}
	// The row is a disclosure: its link actions mount with the body.
	if err := tt.Click(toolRowTitle(card.Name)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("↗ result"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://example.com/result" {
		t.Fatalf("tool click: %q", opened)
	}
	draft := "https://example.com/draft"
	opened = ""
	ct := ui.NewTester(func(c *ui.Context) {
		composer.View(c, composer.Model{Draft: &draft, OnOpenURL: func(s string) { opened = s }})
	}, 800, 600)
	ct.Frame()
	if opened != "" {
		t.Fatal("draft render navigated")
	}
	if err := ct.Click("↗ https://example.com/draft"); err != nil {
		t.Fatal(err)
	}
	if opened != draft || draft != "https://example.com/draft" {
		t.Fatal("draft click mutated input or target")
	}
}
