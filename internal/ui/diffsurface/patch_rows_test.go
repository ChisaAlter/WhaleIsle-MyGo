package diffsurface

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/diff"
)

// TestUnifiedLinesSplitsPairedRows covers the unified view's line model: the
// parser pairs a removed line with the added line that replaced it, and the
// single-column view must show them as two bands with their own numbers.
func TestUnifiedLinesSplitsPairedRows(t *testing.T) {
	rows := splitPatchRows("@@ -1,2 +1,2 @@\n-old line\n+new line\n context\n")
	lines := unifiedLines(rows)
	if len(lines) != 4 {
		t.Fatalf("expected hunk header plus three lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].Kind != diffLineHunk {
		t.Fatalf("first line is not the hunk header: %+v", lines[0])
	}
	if lines[1].Kind != diffLineDel || lines[1].Text != "old line" || lines[1].Number != 1 {
		t.Fatalf("removed line not split out: %+v", lines[1])
	}
	if lines[2].Kind != diffLineAdd || lines[2].Text != "new line" || lines[2].Number != 1 {
		t.Fatalf("added line not split out: %+v", lines[2])
	}
	if lines[3].Kind != diffLineContext || lines[3].Text != "context" {
		t.Fatalf("context line lost its leading marker: %+v", lines[3])
	}
}

// TestUnifiedMinWidthFollowsTheLongestLine keeps every band wide enough for the
// widest line, so a change tint spans the whole row inside the scroller.
func TestUnifiedMinWidthFollowsTheLongestLine(t *testing.T) {
	short := unifiedMinWidth(splitPatchRows("@@ -1 +1 @@\n-a\n+b\n"))
	long := unifiedMinWidth(splitPatchRows("@@ -1 +1 @@\n-" + "x0123456789abcdefghijklmnopqrst" + "\n+b\n"))
	if long <= short {
		t.Fatalf("longer line did not widen the band: short=%v long=%v", short, long)
	}
}

// TestDiffToolRowKeepsBothViewModes checks that the tool row exposes the two
// view modes as a choice instead of a single toggling label.
func TestDiffToolRowKeepsBothViewModes(t *testing.T) {
	workbench := &Workbench{}
	text := ""
	m := Model{Workbench: workbench, Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		paintDiffControls(c, m)
	}, 720, 200)
	if !tt.HasText(i18n.T(i18n.KeySurfacesDiffUnified)) || !tt.HasText(i18n.T(i18n.KeySurfacesDiffSplit)) {
		t.Fatalf("tool row does not offer both view modes: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffSplit)); err != nil {
		t.Fatalf("split chip is not reachable: %v", err)
	}
	if !workbench.View.Split {
		t.Fatal("split chip did not switch the view")
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffWrap)); err != nil {
		t.Fatalf("wrap chip is not reachable: %v", err)
	}
	if !workbench.View.Wrap {
		t.Fatal("wrap chip did not switch wrapping")
	}
}

const singleFilePatch = "diff --git a/a.txt b/a.txt\n--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old one\n+new one\n"

func patchBodyTester(t *testing.T, workbench *Workbench) *ui.Tester {
	t.Helper()
	text := singleFilePatch
	m := Model{Workbench: workbench, Text: &text}
	st := diff.Style{
		FontSize: 12,
		FG:       ui.RGB(24, 24, 24),
		Add:      ui.RGB(61, 153, 112),
		Del:      ui.RGB(224, 102, 102),
		Hunk:     ui.RGB(40, 90, 200),
		File:     ui.RGB(120, 120, 120),
		Muted:    ui.RGB(120, 120, 120),
	}
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { paintPatchFiles(c, m, text, st) })
	}, 900, 420)
}

// TestPatchBodyShowsLinesByDefault keeps a diff readable without a click: a
// fresh patch shows its hunk and its lines, and only an explicit collapse hides
// them.
func TestPatchBodyShowsLinesByDefault(t *testing.T) {
	workbench := &Workbench{}
	tt := patchBodyTester(t, workbench)
	if !tt.HasText("old one") || !tt.HasText("new one") {
		t.Fatalf("diff body did not render its lines: %v", tt.Texts())
	}
	if !tt.HasText("@@ -1 +1 @@") {
		t.Fatalf("hunk header missing: %v", tt.Texts())
	}
}

// TestFoldedHunkHidesItsLines covers per-hunk folding: clicking the header keeps
// the header and drops that hunk's lines.
func TestFoldedHunkHidesItsLines(t *testing.T) {
	workbench := &Workbench{}
	tt := patchBodyTester(t, workbench)
	if err := tt.Click("@@ -1 +1 @@"); err != nil {
		t.Fatalf("hunk header is not reachable: %v", err)
	}
	tt.Frame()
	if tt.HasText("old one") || tt.HasText("new one") {
		t.Fatalf("folded hunk still paints its lines: %v", tt.Texts())
	}
	if !tt.HasText("@@ -1 +1 @@") {
		t.Fatalf("folded hunk lost its header: %v", tt.Texts())
	}
	if err := tt.Click("@@ -1 +1 @@"); err != nil {
		t.Fatalf("folded header is not reachable: %v", err)
	}
	tt.Frame()
	if !tt.HasText("old one") || !tt.HasText("new one") {
		t.Fatalf("unfold did not restore the hunk: %v", tt.Texts())
	}
}
