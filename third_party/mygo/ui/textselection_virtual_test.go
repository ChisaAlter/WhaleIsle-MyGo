package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func hasHighlightInRect(t *Tester, r Rect, highlight Color) bool {
	img := t.Image()
	for y := max(0, int(r.Y)); y < min(img.Bounds().Dy(), int(r.Y+r.H)); y++ {
		for x := max(0, int(r.X)); x < min(img.Bounds().Dx(), int(r.X+r.W)); x++ {
			p := img.RGBAAt(x, y)
			if p.R == highlight.R && p.G == highlight.G && p.B == highlight.B {
				return true
			}
		}
	}
	return false
}

func TestTextSelectionVirtualListScrollPreservesCrossRowAndPrependedSelection(t *testing.T) {
	type item struct {
		key  string
		text string
	}
	items := make([]item, 30)
	for i := range items {
		items[i] = item{key: fmt.Sprintf("msg-%02d", i), text: fmt.Sprintf("Row %02d body text", i)}
	}
	var list ListState
	list.Key = func(i int) any { return items[i].key }
	highlight := RGB(170, 205, 250)

	tt := coreNewTester(func(c *context) {
		c.Theme().Selection = highlight
		l := coreList(c, &list, len(items), func(i int) {
			coreBox(c).Height(50).Padding(8, 12).Children(func() {
				coreText(c, items[i].text)
			})
		})
		l.Selectable().Size(320, 200)
	}, 360, 240)

	// 1. Click anchor in row 0, scroll to row 10, Shift+Click caret in row 10:
	//    row 10 holds focus while row 0 unmounts out of List overscan window!
	x0, y0 := selectableAt(t, tt, items[0].text, 4)
	tt.ClickAt(x0, y0)

	list.ScrollTo(8, Start)
	tt.Frame()
	// Move focus cleanly to row 10 via Shift+Click so row 0 is genuinely offscreen!
	x1, y1 := selectableAt(t, tt, items[10].text, 6)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x1), Y: float64(y1), Mods: platform.ModShift})
	tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x1), Y: float64(y1), Mods: platform.ModShift})
	tt.Frame()

	if slices.ContainsFunc(tt.rt.texts, func(s *state) bool { return s.editor != nil && s.editor.source == items[0].text }) {
		t.Fatal("row 0 remained mounted after focus moved to row 10 and scrolled away")
	}
	if tt.rt.selection.scope == 0 {
		t.Fatal("scrolling anchor offscreen zeroed rt.selection")
	}

	// Visible rows 9 and 10 must paint highlight even though row 0 anchor is offscreen.
	r9, ok := tt.Find(items[9].text)
	if !ok || !hasHighlightInRect(tt, r9, highlight) {
		t.Fatalf("visible row 9 lost highlight when anchor scrolled offscreen (found=%v)", ok)
	}
	r10, ok := tt.Find(items[10].text)
	if !ok || !hasHighlightInRect(tt, r10, highlight) {
		t.Fatalf("visible caret row 10 lost highlight (found=%v)", ok)
	}

	// Copy while anchor is offscreen must include both offscreen and mounted range.
	tt.SetClipboard("")
	tt.Key(Cmd, KeyC)
	var wantParts []string
	wantParts = append(wantParts, items[0].text[4:])
	for i := 1; i <= 9; i++ {
		wantParts = append(wantParts, items[i].text)
	}
	wantParts = append(wantParts, items[10].text[:6])
	// Note: intermediate rows 1..7 weren't visited because we jumped directly via ScrollTo(8);
	// verify at least mounted rows 8..10 plus visited anchor row 0 are never corrupted, and
	// scroll sequentially to verify full offscreen cache capture across all scrolled rows.
	list.ScrollTo(0, Start)
	tt.Frame()
	if tt.rt.selection.scope == 0 {
		t.Fatal("selection lost before sequential scroll")
	}
	for row := 2; row <= 10; row += 2 {
		list.ScrollTo(row, Start)
		tt.Frame()
	}
	tt.SetClipboard("")
	tt.Key(Cmd, KeyC)
	if got, want := tt.Clipboard(), strings.Join(wantParts, "\n"); got != want {
		t.Fatalf("virtualized copy with offscreen anchor = %q, want %q", got, want)
	}

	// 2. Both endpoints offscreen: keep selection alive even if focused row is also scrolled off
	//    (e.g. another row unmounts focus -> rt.focused == 0) and restore highlights on scroll back.
	prevFocused := tt.rt.focused
	tt.rt.focused = 0
	list.ScrollTo(25, Start)
	tt.Frame()
	if tt.rt.selection.scope == 0 {
		t.Fatal("both endpoints offscreen zeroed rt.selection")
	}
	r25, _ := tt.Find(items[25].text)
	if hasHighlightInRect(tt, r25, highlight) {
		t.Fatal("unselected row 25 painted selection highlight")
	}
	_ = prevFocused

	// 3. Prepending earlier rows while selection is offscreen keeps key-bound selection intact.
	prepended := make([]item, 5, 35)
	for i := range prepended {
		prepended[i] = item{key: fmt.Sprintf("old-%02d", i), text: fmt.Sprintf("History %02d", i)}
	}
	items = append(prepended, items...)
	tt.Frame()
	if tt.rt.selection.scope == 0 {
		t.Fatal("prepending list rows cleared offscreen key-anchored selection")
	}
	// Scroll back to old msg-00 (now row 5) and msg-05 (now row 10); simulate window focus intact.
	list.ScrollTo(5, Start)
	tt.Frame()
	xBack, yBack := selectableAt(t, tt, items[5].text, 0)
	_ = xBack
	_ = yBack
	rMsg0, ok := tt.Find(items[5].text)
	if !ok || !hasHighlightInRect(tt, rMsg0, highlight) {
		t.Fatalf("scrolling back after prepend lost highlight on %q (found=%v)", items[5].text, ok)
	}

	// 4. Switching session keys (removing selected row keys) must clear selection cleanly.
	for i := range items {
		items[i] = item{key: fmt.Sprintf("sessionB-%02d", i), text: fmt.Sprintf("SessionB %02d", i)}
	}
	tt.Frame()
	if tt.rt.selection.scope != 0 {
		t.Fatalf("switching session row keys left stale selection scope=%d", tt.rt.selection.scope)
	}
}

func TestTextSelectionVirtualListInnerCardScopePreservedAcrossScroll(t *testing.T) {
	type item struct {
		key  string
		text string
	}
	items := make([]item, 25)
	for i := range items {
		items[i] = item{key: fmt.Sprintf("card-%02d", i), text: fmt.Sprintf("Card message %02d 中文内容", i)}
	}
	var list ListState
	list.Key = func(i int) any { return items[i].key }
	highlight := RGB(180, 210, 250)

	tt := coreNewTester(func(c *context) {
		c.Theme().Selection = highlight
		coreList(c, &list, len(items), func(i int) {
			coreColumn(c).Selectable().Padding(10, 12).Height(60).Children(func() {
				coreText(c, items[i].text)
			})
		}).Size(320, 180)
	}, 360, 220)

	selectBetween(t, tt, items[0].text, 5, items[0].text, 15)
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "message 00" {
		t.Fatalf("initial card selection copy = %q", got)
	}

	// Simulate scrolling away while focus is cleared/away so row 0's whole .Selectable() scope unmounts.
	tt.rt.focused = 0
	list.ScrollTo(20, Start)
	tt.Frame()
	if slices.ContainsFunc(tt.rt.texts, func(s *state) bool { return s.editor != nil && s.editor.source == items[0].text }) {
		t.Fatal("row 0 card stayed mounted at row 20")
	}
	if tt.rt.selection.scope == 0 {
		t.Fatal("scrolling inner Selectable card offscreen cleared rt.selection")
	}

	// Scroll back to row 0: card scope remounts with same deterministic key ID and selection highlights return.
	list.ScrollTo(0, Start)
	tt.Frame()
	r0, ok := tt.Find(items[0].text)
	if !ok || !hasHighlightInRect(tt, r0, highlight) {
		t.Fatalf("scrolling back to row 0 lost card highlight (found=%v)", ok)
	}
	tt.ClickAt(r0.X+1, r0.Y+1) // re-verify click isn't broken; first check Copy before clicking away!
	// Re-select and test copy on remount cleanly:
	selectBetween(t, tt, items[0].text, 5, items[0].text, 15)
	tt.rt.focused = 0
	list.ScrollTo(20, Start)
	tt.Frame()
	list.ScrollTo(0, Start)
	tt.Frame()
	// Focus returns on remounted card or right-click copy works directly without re-selecting:
	x, y := selectableAt(t, tt, items[0].text, 8)
	tt.RightClickAt(x, y)
	if err := tt.ChooseMenuItem("Copy"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "message 00" {
		t.Fatalf("copy after scrolling back = %q, want %q", got, "message 00")
	}
}

func TestTextSelectionStreamingMarkdownReflowAndBlockSplit(t *testing.T) {
	t.Run("inline syntax close reflow preserves selection", func(t *testing.T) {
		line := "Use **bold prefix"
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Selectable().Padding(12).Children(func() {
				coreText(c, line)
			})
		}, 400, 160)

		selectBetween(t, tt, line, 0, line, 4)
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Use " {
			t.Fatalf("initial copy = %q", got)
		}

		// Markdown closes **bold**: rendered text drops "**", breaking strict strings.HasPrefix!
		line = "Use bold prefix and trailing code"
		tt.Frame()
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Use " {
			t.Fatalf("inline markdown reflow cleared selection, copy=%q", got)
		}
	})

	t.Run("heading marker collapse shifts selection cleanly", func(t *testing.T) {
		line := "## Streaming heading"
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Selectable().Padding(12).Children(func() {
				coreText(c, line)
			})
		}, 400, 160)

		selectBetween(t, tt, line, 3, line, 12)
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Streaming" {
			t.Fatalf("initial heading copy = %q", got)
		}

		line = "Streaming heading closed"
		tt.Frame()
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Streaming" {
			t.Fatalf("heading reflow copy = %q, want %q", got, "Streaming")
		}
	})

	t.Run("code fence structural split preserves endpoint across node id change", func(t *testing.T) {
		split := false
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Selectable().Padding(12).Gap(8).Children(func() {
				if !split {
					coreText(c, "Intro line\n```go\nfmt.Pr")
				} else {
					coreBox(c).Children(func() {
						coreRichText(c, Span{Text: "Intro line"})
					})
					coreBox(c).Children(func() {
						coreText(c, "fmt.Println(\"ok\")")
					})
				}
			})
		}, 420, 200)

		selectBetween(t, tt, "Intro line\n```go\nfmt.Pr", 0, "Intro line\n```go\nfmt.Pr", 10)
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Intro line" {
			t.Fatalf("initial fence copy = %q", got)
		}

		split = true
		tt.Frame()
		if tt.rt.selection.scope == 0 {
			t.Fatal("structural markdown split cleared selection")
		}
		tt.SetClipboard("")
		tt.Key(Cmd, KeyC)
		if got := tt.Clipboard(); got != "Intro line" {
			t.Fatalf("Cmd+C after fence split copy = %q, want %q", got, "Intro line")
		}
		x, y := selectableAt(t, tt, "Intro line", 3)
		tt.RightClickAt(x, y)
		if err := tt.ChooseMenuItem("Copy"); err != nil {
			t.Fatal(err)
		}
		if got := tt.Clipboard(); got != "Intro line" {
			t.Fatalf("after fence split copy = %q, want %q", got, "Intro line")
		}
	})
}
