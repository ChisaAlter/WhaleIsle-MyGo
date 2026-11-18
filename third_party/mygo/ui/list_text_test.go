package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestListTextCopyClipsOffscreenEndpointParagraphs(t *testing.T) {
	rows := make([][]string, 24)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("old-%02d-中文", i), fmt.Sprintf("new-%02d-中文", i)}
	}
	var list ListState
	tt := NewTester(func(c *Context) {
		List(c, &list, len(rows)).Size(360, 200).Selectable().
			SelectionText(1, func(i int) []string { return rows[i] }).
			Rows(func(row ListRow) {
				Column(row.Context).Height(60).Padding(6, 10).Children(func() {
					Text(row.Context, rows[row.Index][0])
					Text(row.Context, rows[row.Index][1])
				})
			})
	}, 400, 240)

	x, y := selectableAt(t, tt, rows[0][1], 4)
	tt.ClickAt(x, y)
	// Keep row 10 inside the viewport rather than at an overscan edge: the
	// focused anchor can remain mounted outside the visible scroll region.
	list.ScrollTo(8, Start)
	tt.Frame()
	x, y = selectableAt(t, tt, rows[10][0], 8)
	if y <= 0 || y >= 200 {
		t.Fatalf("target row 10 is outside the viewport: y=%v", y)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Mods: platform.ModShift})
	tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y), Mods: platform.ModShift})
	tt.Frame()
	tt.Key(Cmd, KeyC)
	want := []string{string([]rune(rows[0][1])[4:])}
	for i := 1; i < 10; i++ {
		want = append(want, rows[i]...)
	}
	want = append(want, string([]rune(rows[10][0])[:8]))
	if got := tt.Clipboard(); got != strings.Join(want, "\n") {
		t.Fatalf("offscreen multi-paragraph range = %q, want %q", got, strings.Join(want, "\n"))
	}
}

func TestListTextSelectAllReadsUnbuiltRowsAndUnicode(t *testing.T) {
	rows := make([]string, 10000)
	for i := range rows {
		rows[i] = fmt.Sprintf("line %05d · 中文🙂", i)
	}
	var list ListState
	builds := 0
	tt := NewTester(func(c *Context) {
		List(c, &list, len(rows)).Size(360, 200).Selectable().
			SelectionText(1, func(i int) []string { return rows[i : i+1] }).
			Rows(func(row ListRow) {
				builds++
				Text(row.Context, rows[row.Index]).Height(40).Padding(6, 10)
			})
	}, 400, 240)
	x, y := selectableAt(t, tt, rows[0], 4)
	tt.ClickAt(x, y)
	before := builds
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd, KeyC)
	if got, want := tt.Clipboard(), strings.Join(rows, "\n"); got != want {
		t.Fatalf("Select All copied %d bytes, want %d", len(got), len(want))
	}
	if builds-before > 100 {
		t.Fatalf("logical copy built offscreen UI: %d rows", builds-before)
	}
}

func TestListTextOffscreenRevisionPreservesAppendAndRejectsReplacement(t *testing.T) {
	rows := make([]string, 30)
	for i := range rows {
		rows[i] = fmt.Sprintf("row-%02d original", i)
	}
	var list ListState
	revision := uint64(1)
	tt := NewTester(func(c *Context) {
		List(c, &list, len(rows)).Size(360, 200).Selectable().
			SelectionText(revision, func(i int) []string { return rows[i : i+1] }).
			Rows(func(row ListRow) { Text(row.Context, rows[row.Index]).Height(40).Padding(6, 10) })
	}, 400, 240)
	selectBetween(t, tt, rows[0], 1, rows[0], 8)
	tt.rt.focused = 0
	list.ScrollTo(20, Start)
	tt.Frame()
	rows[0] += " appended"
	revision++
	tt.Frame()
	if tt.rt.selection.scope == 0 || tt.rt.selection.anchor.at != 1 || tt.rt.selection.caret.at != 8 {
		t.Fatal("append-only source update changed offscreen selection")
	}
	rows[0] = "replacement text"
	revision++
	tt.Frame()
	if tt.rt.selection.scope != 0 {
		t.Fatal("replacement reused stale offscreen text offsets")
	}
}

func TestSliceTextRunes(t *testing.T) {
	for _, tc := range []struct {
		text     string
		from, to int
		want     string
	}{
		{"a中文🙂z", 1, 4, "中文🙂"},
		{"a中文🙂z", 0, 5, "a中文🙂z"},
		{"a中文🙂z", 4, 5, "z"},
		{"", 0, 0, ""},
		{"a", 1, 1, ""},
	} {
		if got := sliceTextRunes(tc.text, tc.from, tc.to); got != tc.want {
			t.Fatalf("sliceTextRunes(%q, %d, %d) = %q, want %q", tc.text, tc.from, tc.to, got, tc.want)
		}
	}
}
