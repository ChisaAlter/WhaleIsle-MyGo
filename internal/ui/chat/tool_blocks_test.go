package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParseSearchHitsReadsPathAndLocation(t *testing.T) {
	output := strings.Join([]string{
		"internal/ui/chat/chat.go:145: if len(msgs) == 0 {",
		`C:\work\whale\main.go:7:3: func main() {`,
		"plain note without a location",
		"",
	}, "\n")
	hits := parseSearchHits(output)
	if len(hits) != 3 {
		t.Fatalf("hits = %d, want 3 (blank lines drop)", len(hits))
	}
	if hits[0].Path != "internal/ui/chat/chat.go" || hits[0].Line != "145" || hits[0].Snippet != "if len(msgs) == 0 {" {
		t.Fatalf("first hit = %+v", hits[0])
	}
	if hits[1].Path != `C:\work\whale\main.go` || hits[1].Line != "7:3" {
		t.Fatalf("windows path / column hit = %+v", hits[1])
	}
	if hits[2].Path != "" || hits[2].Snippet != "plain note without a location" {
		t.Fatalf("plain line lost: %+v", hits[2])
	}
}

func TestSearchBlockKeepsLocationSnippetAndOverflow(t *testing.T) {
	var lines []string
	for index := 1; index <= searchHitLimit+3; index++ {
		lines = append(lines, "internal/ui/chat/chat.go:"+itoaFast(index)+": row "+itoaFast(index)+" mentions the term")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			searchHitsBlock(c, toolSeatStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 246, 247)},
				"k", "term", strings.Join(lines, "\n"), func(string) {})
		})
	}, 760, 460)
	if _, ok := tt.Find("internal/ui/chat/chat.go:1"); !ok {
		t.Fatalf("hit location missing: %q", tt.Texts())
	}
	if !tt.HasText("row 1 mentions the term") {
		t.Fatalf("hit snippet missing: %q", tt.Texts())
	}
	if !tt.HasText(strings.Replace(i18n.T(i18n.KeyChatToolMoreHits), "%d", "3", 1)) {
		t.Fatalf("overflow count missing: %q", tt.Texts())
	}
	// Only the first searchHitLimit rows mount; the rest is a count.
	if _, ok := tt.Find("internal/ui/chat/chat.go:" + itoaFast(searchHitLimit+1)); ok {
		t.Fatal("rows past the limit still mounted")
	}
}

func TestWebBlockShowsCardsAndOpensTheURL(t *testing.T) {
	opened := ""
	web := &session.WebResult{Kind: "search", Truncated: true, Sources: []session.WebSource{
		{URL: "https://www.example.com/guide", Title: "Guide to file writes", Snippet: "Write to a temp file and rename it so a crash never truncates the target.", PublishedAt: "2026-01-02"},
		{URL: "https://docs.example.org/api", Title: "API reference", Snippet: "MkdirAll creates the parents."},
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			webResultBlock(c, toolSeatStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 246, 247), Accent: ui.RGB(40, 90, 200)},
				"w", web, ToolCardInfo{Output: "sources"}, func(rawURL string) { opened = rawURL }, func(string) {})
		})
	}, 760, 340)
	if !tt.HasText("Guide to file writes") || !tt.HasText("API reference") {
		t.Fatalf("web titles missing: %q", tt.Texts())
	}
	if !tt.HasText("example.com · 2026-01-02") {
		t.Fatalf("domain/date line missing: %q", tt.Texts())
	}
	if !tt.HasText("MkdirAll creates the parents.") {
		t.Fatalf("snippet missing: %q", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyChatWebTruncated)) {
		t.Fatalf("truncated footer missing: %q", tt.Texts())
	}
	if err := tt.Click("Guide to file writes"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://www.example.com/guide" {
		t.Fatalf("open callback = %q", opened)
	}
}

func TestWebBlockWithoutTypedMetaKeepsRawOutput(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			webResultBlock(c, toolSeatStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 246, 247)},
				"w", nil, ToolCardInfo{Output: "raw fetch body"}, nil, func(string) {})
		})
	}, 600, 200)
	if !tt.HasText("raw fetch body") {
		t.Fatalf("raw fallback lost: %q", tt.Texts())
	}
}

func TestTodoBlockShowsStatusWordsAndCompletedRows(t *testing.T) {
	output := strings.Join([]string{
		"- [x] write the failing test",
		"- [ ] fix the table widths",
		"[~] review the gallery",
		"plain follow-up",
	}, "\n")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			todoBlock(c, toolSeatStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 246, 247)},
				"t", output, func(string) {})
		})
	}, 640, 260)
	for _, want := range []string{
		"write the failing test", "fix the table widths", "review the gallery", "plain follow-up",
		i18n.T(i18n.KeyChatTodoDone), i18n.T(i18n.KeyChatTodoRunning), i18n.T(i18n.KeyChatTodoPending),
	} {
		if !tt.HasText(want) {
			t.Fatalf("todo row missing %q: %q", want, tt.Texts())
		}
	}
	if items := parseTodos(output); len(items) != 4 || items[0].Status != "completed" || items[2].Status != "in_progress" {
		t.Fatalf("parseTodos = %+v", items)
	}
}
