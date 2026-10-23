package frame

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func inspectorLongRows(before int) []session.InspectorRecord {
	var rows []session.InspectorRecord
	for i := 0; i < before; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("before:%d", i), Type: "before"})
	}
	rows = append(rows, session.InspectorRecord{Key: "turn", Type: "turn/start", Disclosure: "open"}, session.InspectorRecord{Key: "step", Parent: "turn", Type: "step/start", Disclosure: "open"}, session.InspectorRecord{Key: "assistant", Parent: "step", Type: "assistant/live"}, session.InspectorRecord{Key: "block", Parent: "assistant", Type: "block-start", Disclosure: "open"})
	for i := 0; i < 200; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("delta:%d", i), Parent: "block", Type: "text-delta"})
	}
	return append(rows, session.InspectorRecord{Key: "next", Parent: "step", Type: "next-event"})
}
func nearInspector(a, b float32) bool { return math.Abs(float64(a-b)) < .1 }
func TestInspectorManualFoldKeepsPinnedHeaderAndTrailingExtent(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	rows := inspectorLongRows(20)
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	row, ok := tt.Find("Inspector record turn")
	if !ok {
		t.Fatal("missing pinned Turn")
	}
	oldTop := row.Y
	if err := tt.Click("Collapse children turn"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	folded, ok := tt.Find("Inspector record turn")
	if !ok || !nearInspector(folded.Y, oldTop) || !nearInspector(p.tableY.Y, 600) {
		t.Fatalf("fold shifted header: old=%+v new=%+v Y=%+v scroll=%+v", row, folded, p.tableY, p.scroll)
	}
	padding := p.scroll.layout.bottomPadding
	if padding <= 0 {
		t.Fatal("final folded header could be clamped without trailing space")
	}
	rows = append(rows, session.InspectorRecord{Key: "after", Type: "after"})
	tt.Frame()
	after, ok := tt.Find("Inspector record after")
	folded, _ = tt.Find("Inspector record turn")
	if !ok || !nearInspector(after.Y, folded.Y+30) || !nearInspector(p.tableY.Y, 600) || !nearInspector(p.scroll.layout.bottomPadding, padding-30) {
		t.Fatalf("new row did not consume trailing blank: folded=%+v after=%+v scroll=%+v layout=%+v", folded, after, p.tableY, p.scroll.layout)
	}
	p.list.ScrollToEnd()
	tt.Frame()
	if p.scroll.mode != "manual" {
		t.Fatal("scrolling to bottom changed manual mode")
	}
	rows = append(rows, session.InspectorRecord{Key: "after2", Type: "after2"})
	tt.Frame()
	if !nearInspector(p.tableY.Y, 600) {
		t.Fatal("manual tail followed appended event")
	}
	p.followInspectorLatest()
	tt.Frame()
	if p.scroll.layout.bottomPadding != 0 || !nearInspector(p.tableY.Y, p.tableY.MaxY) {
		t.Fatalf("Latest did not release spacing: Y=%+v layout=%+v", p.tableY, p.scroll.layout)
	}
}
func TestInspectorAutomaticClosureRetainsParentAndAppendsContiguously(t *testing.T) {
	rows := inspectorLongRows(0)
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	for i := range rows {
		if rows[i].Key == "block" {
			rows[i].Disclosure = "closed"
		}
	}
	tt.Frame()
	if p.scroll.mode != "tail" || p.tableY.Y != 0 || p.scroll.layout.bottomPadding <= 0 || tt.HasText("Inspector record delta:199") {
		t.Fatalf("closure failed: Y=%+v layout=%+v", p.tableY, p.scroll.layout)
	}
	padding := p.scroll.layout.bottomPadding
	rows = append(rows, session.InspectorRecord{Key: "after", Type: "after"})
	tt.Frame()
	previous, _ := tt.Find("Inspector record next")
	after, _ := tt.Find("Inspector record after")
	if !nearInspector(after.Y, previous.Y+30) || !nearInspector(p.scroll.layout.bottomPadding, padding-30) || p.tableY.Y != 0 {
		t.Fatalf("postclosure append shifted: previous=%+v after=%+v Y=%+v padding=%v", previous, after, p.tableY, p.scroll.layout.bottomPadding)
	}
}
func TestInspectorClosureResumesFollowingAfterAllTrailingSpaceIsConsumed(t *testing.T) {
	rows := inspectorLongRows(20)
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	for i := range rows {
		if rows[i].Key == "block" {
			rows[i].Disclosure = "closed"
		}
	}
	tt.Frame()
	if p.scroll.mode != "tail" {
		t.Fatalf("programmatic anchor lost follow: %+v", p.scroll)
	}
	for i := 0; i < 40; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("append:%d", i), Type: "after"})
		tt.Frame()
	}
	if p.scroll.layout.bottomPadding != 0 || !nearInspector(p.tableY.Y, p.tableY.MaxY) {
		t.Fatalf("did not resume after blank: Y=%+v mode=%s layout=%+v", p.tableY, p.scroll.mode, p.scroll.layout)
	}
}

func TestInspectorProgrammaticEndAndRevealCommitCurrentStickyBand(t *testing.T) {
	var rows []session.InspectorRecord
	for _, group := range []string{"a", "b"} {
		rows = append(rows, session.InspectorRecord{Key: group, Type: "turn/" + group, Disclosure: "open"})
		for i := 0; i < 80; i++ {
			rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("%s:%d", group, i), Parent: group, Type: "delta"})
		}
	}
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	check := func(key string) {
		t.Helper()
		row, ok := tt.Find("Inspector record " + key)
		header, _ := tt.Find("Inspector table header")
		if !ok || !nearInspector(row.Y, header.Y+30) {
			t.Fatalf("sticky %s missing after programmatic scroll: row=%+v header=%+v scroll=%+v", key, row, header, p.tableY)
		}
	}
	check("b")
	p.applyInspectorFilter("")
	tt.Frame()
	check("a")
	p.followInspectorLatest()
	tt.Frame()
	check("b")
	p.revealInspectorRow("a:40", rows)
	tt.Frame()
	check("a")
	p.revealInspectorRow("b:40", rows)
	tt.Frame()
	check("b")
}

func TestInspectorEndRevealFollowsSubsequentAppend(t *testing.T) {
	rows := inspectorLongRows(0)
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	p.applyInspectorFilter("")
	tt.Frame()
	p.revealInspectorRow("next", rows)
	tt.Frame()
	if p.scroll.mode != "tail" || !nearInspector(p.tableY.Y, p.tableY.MaxY) {
		t.Fatalf("end reveal not classified at final extent: mode=%s scroll=%+v", p.scroll.mode, p.tableY)
	}
	rows = append(rows, session.InspectorRecord{Key: "after", Type: "after"})
	tt.Frame()
	if !nearInspector(p.tableY.Y, p.tableY.MaxY) {
		t.Fatalf("end reveal did not follow append: %+v", p.tableY)
	}
	p.applyInspectorFilter("block-start")
	tt.Frame()
	if p.tableY.Y != 0 || p.tableY.MaxY != 0 || p.scroll.mode != "tail" {
		t.Fatalf("filter scroll did not classify short content: mode=%s scroll=%+v", p.scroll.mode, p.tableY)
	}
}

func TestInspectorFollowUsesInclusiveFortyDIPThreshold(t *testing.T) {
	for _, distance := range []float32{40, 41} {
		t.Run(fmt.Sprint(distance), func(t *testing.T) {
			rows := inspectorLongRows(0)
			p := SessionInspectorPaneState{}
			tt := ui.NewTester(func(c *ui.Context) {
				inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
			}, 780, 300)
			header, _ := tt.Find("Inspector table header")
			tt.Scroll(700, header.Y+header.H+160, 0, -distance)
			tt.Frame()
			before := p.tableY.Y
			rows = append(rows, session.InspectorRecord{Key: "after", Type: "after"})
			tt.Frame()
			if distance == 40 {
				if !nearInspector(p.tableY.Y, p.tableY.MaxY) {
					t.Fatalf("40 DIP did not follow: Y=%+v mode=%s", p.tableY, p.scroll.mode)
				}
			} else if !nearInspector(p.tableY.Y, before) {
				t.Fatalf("41 DIP followed: before%v Y=%+v mode=%s", before, p.tableY, p.scroll.mode)
			}
		})
	}
}
