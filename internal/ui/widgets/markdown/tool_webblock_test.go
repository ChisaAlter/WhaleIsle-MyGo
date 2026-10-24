package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func webTestLabels() WebLabels {
	return WebLabels{NoResults: "No results found", SourcesTruncated: "Source list truncated", HTTP: "HTTP", ContentTruncated: "Content truncated"}
}
func webTestView(t *testing.T, data WebData, width int) (*ui.Tester, *[]string) {
	t.Helper()
	previous := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(previous) })
	opened := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		ui.Column(c).Fill().Children(func() {
			ToolWebBlock(c, data, Style{FG: ui.Hex("#0f1115"), Muted: ui.Hex("#61666b"), ToolCodeLanguage: ui.Hex("#81858c"), CodeBG: ui.Hex("#f9fafb"), Accent: ui.Hex("#4d6bfe"), OnOpenURL: func(raw string) { opened = append(opened, raw) }}, webTestLabels()).Label("web-card")
		})
	}, width, 900)
	tt.Frame()
	return tt, &opened
}
func TestToolWebSearchAnswerFullSnippetDateAndNoToolbar(t *testing.T) {
	data := WebData{Kind: "search", Answer: "**Provider answer**\n\nA second paragraph.", Sources: []WebSource{{URL: "https://www.example.test/a", Title: "Source title", Snippet: strings.Repeat("Full source snippet and evidence. ", 9) + "snippet-tail", PublishedAt: "2026-10-24"}, {URL: "javascript:alert(1)", Title: "Unsafe plain label", Snippet: "second"}}, Truncated: true}
	tt, opened := webTestView(t, data, 420)
	for _, want := range []string{"Provider answer", "A second paragraph.", "Source title", "snippet-tail", "2026-10-24", "Unsafe plain label", "Source list truncated", "1.", "2."} {
		if !tt.HasText(want) {
			t.Fatalf("missing%s %v", want, tt.Texts())
		}
	}
	for _, absent := range []string{"Copy", "Collapse", "Wrap lines", "www.example.test ·"} {
		if tt.HasText(absent) {
			t.Fatalf("unexpected%s", absent)
		}
	}
	snippet, _ := tt.Find(webInlineText(data.Sources[0].Snippet))
	if snippet.H <= 38 {
		t.Fatalf("snippet still clamped%v", snippet)
	}
	if err := tt.Click("Source title"); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || (*opened)[0] != data.Sources[0].URL {
		t.Fatal(*opened)
	}
	tt.Key(0, ui.KeyEnter)
	if len(*opened) != 2 {
		t.Fatal("Enter did not open")
	}
	tt.Key(0, ui.KeySpace)
	if len(*opened) != 2 {
		t.Fatal("Space activated anchor")
	}
	if err := tt.Click("Unsafe plain label"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if len(*opened) != 2 {
		t.Fatal("unsafe link became active")
	}
}
func TestToolWebSearchEmptyAnswerAndSourceScroll(t *testing.T) {
	for _, tc := range []struct {
		answer string
		height float32
		empty  bool
	}{{"", 44, true}, {" ", 32, false}, {"Answer only", 56, false}} {
		tt, _ := webTestView(t, WebData{Kind: "search", Answer: tc.answer}, 616)
		card, _ := tt.Find("web-card")
		if card.H != tc.height || tt.HasText("No results found") != tc.empty {
			t.Fatalf("answer%q card%v texts%v", tc.answer, card, tt.Texts())
		}
	}
	data := WebData{Kind: "search", Truncated: true}
	for i := 0; i < 120; i++ {
		data.Sources = append(data.Sources, WebSource{URL: fmt.Sprintf("https://example.test/%d", i), Title: fmt.Sprintf("source-%03d", i)})
	}
	tt, opened := webTestView(t, data, 420)
	card, _ := tt.Find("web-card")
	if card.H != 372 {
		t.Fatalf("scroll cap%v", card)
	}
	before, _ := tt.Find("Source list truncated")
	tt.Scroll(200, 180, 0, 4000)
	tt.Frame()
	after, _ := tt.Find("Source list truncated")
	if before != after {
		t.Fatal("source scroll moved truncation note")
	}
	last, ok := tt.Find("source-119")
	if !ok || last.H == 0 {
		t.Fatal("full source list did not scroll")
	}
	if err := tt.Click("source-119"); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || (*opened)[0] != "https://example.test/119" {
		t.Fatal(*opened)
	}
}
func TestToolWebFetchRedirectStatusAndNoCopy(t *testing.T) {
	data := WebData{Kind: "fetch", URL: "https://final.example.test/page", StatusCode: -1e300, Truncated: true}
	tt, opened := webTestView(t, data, 616)
	if !tt.HasText("HTTP -1e+300") || !tt.HasText("Content truncated") || tt.HasText("Copy") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click(data.URL); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || (*opened)[0] != data.URL {
		t.Fatal(*opened)
	}
	for _, number := range []float64{0, -0.0, -1, 199, 1e20, 1e21, 1e300, -1e300} {
		if strings.Contains(webStatusNumber(number), "e+0") {
			t.Fatal("exponent zero padding")
		}
	}
	unsafe, _ := webTestView(t, WebData{Kind: "fetch", URL: "file:///tmp/page", StatusCode: 200}, 616)
	if err := unsafe.Click("file:///tmp/page"); err != nil {
		t.Fatal(err)
	}
}
func TestToolWebAnswerURLUsesAnchorKeyboard(t *testing.T) {
	tt, opened := webTestView(t, WebData{Kind: "search", Answer: "Read [answer link](https://answer.example.test/a) for more."}, 616)
	if err := tt.Click("answer link"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if len(*opened) != 2 || (*opened)[0] != "https://answer.example.test/a" || (*opened)[1] != "https://answer.example.test/a" {
		t.Fatal("Markdown answer anchor activation", *opened)
	}
}

func TestToolWebSiteGlyphLongestSuffix(t *testing.T) {
	for _, tc := range []struct{ raw, symbol string }{{"https://github.com/a", "Y0"}, {"https://gist.github.com/a", "Y0"}, {"https://weixin.qq.com/a", "JA"}, {"https://a.weixin.qq.com/a", "JA"}, {"https://qq.com/a", "FA"}, {"https://github.com.evil.test/a", ""}, {"https://github.com./a", ""}, {"mailto:a@github.com", ""}} {
		want := webGlobe
		if tc.symbol != "" {
			want = webSiteMarks[tc.symbol]
		}
		if got := webResultGlyph(tc.raw); got != want {
			t.Fatalf("%s site mark", tc.raw)
		}
	}
	if len(webSiteHosts) != 40 || len(webSiteMarks) != 34 {
		t.Fatal("installed glyph table incomplete")
	}
}
