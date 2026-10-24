package chat

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func toolWebViewCall() *session.ToolCallView {
	return fileToolFixture("web_search", `{"queries":["recorded query"]}`, map[string]any{"answer": "**Recorded answer**\n\nSecond paragraph.", "sources": []map[string]any{{"url": "https://source.example.test/a", "title": "Recorded source", "snippet": "Full recorded snippet.", "publishedAt": "2026-10-24"}, {"url": "https://www.example.test/b"}}, "truncated": true}, "raw envelope must not display")
}
func toolWebViewFixture(t *testing.T, call *session.ToolCallView, dark, textOnly bool) (*ui.Tester, *map[string]bool, *[]string) {
	t.Helper()
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	expanded := map[string]bool{}
	opened := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(palette.BG)).Children(func() {
			ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{ID: "summary", Expanded: &expanded, TextOnly: textOnly, OnOpenURL: func(raw string) { opened = append(opened, raw) }})
			ui.Box(c).Height(1).Label("Following web flow")
		})
	}, 640, 700)
	tt.SetTime(time.Unix(11700, 0))
	tt.Move(635, 695)
	return tt, &expanded, &opened
}
func TestToolWebViewExactHeaderArtwork(t *testing.T) {
	if toolKindIconForName("web_search") != toolIconWeb || toolKindIconForName("web_fetch") != toolIconRead {
		t.Fatal("keyed WebRow artwork differs from installed caller")
	}
	if toolKindIconForName("Web_fetch") == toolIconRead {
		t.Fatal("near-name tool gained keyed web artwork")
	}
}

func TestToolWebViewSearchUsesSingleCardAndFullAnswer(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := toolWebViewCall()
			tt, expanded, opened := toolWebViewFixture(t, call, dark, false)
			openReadView(t, tt, expanded)
			for _, absent := range []string{"IN", "OUT", "Copy", `"queries"`, "raw envelope"} {
				if tt.HasText(absent) {
					t.Fatalf("duplicate%s %v", absent, tt.Texts())
				}
			}
			for _, want := range []string{"Web results card", "Recorded answer", "Second paragraph.", "Recorded source", "Full recorded snippet.", "2026-10-24", "www.example.test", "Source list truncated", "1.", "2."} {
				if !tt.HasText(want) {
					t.Fatalf("missing%s %v", want, tt.Texts())
				}
			}
			card, _ := tt.Find("Web results card")
			toolTreeNear(t, "web leftmargin", card.X, 24)
			toolWebExport(t, tt, fmt.Sprintf("search-dark-%v", dark))
			if err := tt.Click("Recorded source"); err != nil {
				t.Fatal(err)
			}
			if len(*opened) != 1 || (*opened)[0] != "https://source.example.test/a" || !(*expanded)["summary|tool:file-call"] {
				t.Fatal("source click changed outer row", *opened)
			}
			tt.Key(0, ui.KeyEnter)
			if len(*opened) != 2 {
				t.Fatal("source Enter")
			}
			tt.Key(0, ui.KeySpace)
			if len(*opened) != 2 || !(*expanded)["summary|tool:file-call"] {
				t.Fatal("source Space activated or toggled outerrow")
			}
		})
	}
}
func TestToolWebViewFetchHeaderOriginalAndBodyFinalURL(t *testing.T) {
	original, final := "https://original.example.test/a", "https://final.example.test/b"
	call := fileToolFixture("web_fetch", `{"url":"`+original+`"}`, map[string]any{"url": final, "statusCode": 200, "truncated": true}, "retrieved full raw content")
	tt, expanded, opened := toolWebViewFixture(t, call, false, false)
	tt.Frame()
	if err := tt.Click(original); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || (*opened)[0] != original || (*expanded)["summary|tool:file-call"] {
		t.Fatal("headerclick toggled orwrongtarget", *opened)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if len(*opened) != 2 || (*expanded)["summary|tool:file-call"] {
		t.Fatal("headerkeyboard toggled oropenedspace", *opened)
	}
	openReadView(t, tt, expanded)
	for _, want := range []string{final, "HTTP 200", "Content truncated"} {
		if !tt.HasText(want) {
			t.Fatal(tt.Texts())
		}
	}
	if tt.HasText("retrieved full raw content") || tt.HasText("Copy") {
		t.Fatal("rawfetch leaked")
	}
	if err := tt.Click(final); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 3 || (*opened)[2] != final {
		t.Fatal(*opened)
	}
	toolWebExport(t, tt, "fetch-light")
	call.State = "error"
	tt.Frame()
	if info := toolWebModelFor(call); info != nil {
		t.Fatal("failed card")
	}
}
func TestToolWebViewGenericFallbackReadOnlyAndMetadataAuthority(t *testing.T) {
	for _, name := range []string{"web_search", "web_fetch"} {
		fallback := fileToolFixture(name, `{"url":"x","queries":["x"]}`, nil, "fallback evidence")
		tt, expanded, _ := toolWebViewFixture(t, fallback, false, false)
		openReadView(t, tt, expanded)
		if tt.HasText("Web results card") || tt.HasText("IN") || !tt.HasText("OUT") || !tt.HasText("fallback evidence") {
			t.Fatal(tt.Texts())
		}
	}
	call := toolWebViewCall()
	call.Result.Web = &session.WebResult{Kind: "search", Answer: "projected forgery", Sources: []session.WebSource{{URL: "https://forgery.test", Title: "forgedsource"}}}
	tt, expanded, _ := toolWebViewFixture(t, call, false, false)
	openReadView(t, tt, expanded)
	if tt.HasText("projected forgery") || tt.HasText("forgedsource") || !tt.HasText("Recorded source") {
		t.Fatal(tt.Texts())
	}
	ro := toolWebViewCall()
	tt, expanded, _ = toolWebViewFixture(t, ro, false, true)
	openReadView(t, tt, expanded)
	if tt.HasText("Web results card") || !tt.HasText("raw envelope must not display") {
		t.Fatal("readonlyevidence changed", tt.Texts())
	}
	for _, unsafe := range []string{"file:///tmp/a", "javascript:alert(1)", "https://bad host/"} {
		unsafeCall := fileToolFixture("web_fetch", `{"url":"`+unsafe+`"}`, map[string]any{"url": unsafe, "statusCode": 200, "truncated": false}, "")
		if toolWebFetchHref(unsafeCall) != "" {
			t.Fatal("unsafeheaderlink")
		}
	}
}
func toolWebExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_WEB_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	prefix := filepath.Join(dir, "tool-web-pass265-native-software-"+name)
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
	for _, label := range []string{"Web results card", "Recorded answer", "Second paragraph.", "Recorded source", "Full recorded snippet.", "2026-10-24", "www.example.test", "Source list truncated", "1.", "2.", "HTTP 200", "Content truncated", "https://final.example.test/b", "Following web flow"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	value, _ := json.MarshalIndent(map[string]any{"name": name, "rects": boxes, "texts": tt.Texts()}, "", "  ")
	if err = os.WriteFile(prefix+".json", value, 0600); err != nil {
		t.Fatal(err)
	}
}
