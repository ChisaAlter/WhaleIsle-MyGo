package diffsurface

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/diff"
)

func buildLargeSyntheticPatch(fileCount, linesPerFile int) string {
	var b strings.Builder
	for f := range fileCount {
		name := fmt.Sprintf("pkg/file_%02d.go", f)
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", name, name)
		fmt.Fprintf(&b, "--- a/%s\n", name)
		fmt.Fprintf(&b, "+++ b/%s\n", name)
		fmt.Fprintf(&b, "@@ -1,1 +1,%d @@\n", linesPerFile+1)
		b.WriteString(" package pkg\n")
		for i := range linesPerFile {
			fmt.Fprintf(&b, "+var row_%02d_%04d = %d\n", f, i, i)
		}
	}
	return b.String()
}

func TestPatchViewVirtualizesLargePatchAndSupportsScrollAndFold(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	patch := buildLargeSyntheticPatch(2, 600) // >1200 lines total
	root := t.TempDir()
	w := &Workbench{
		Root:        root,
		Loaded:      true,
		Repository:  true,
		AgainstHEAD: true,
		Changes:     []git.Change{{Path: "pkg/file_00.go", Worktree: 'M'}},
	}
	defer w.ClosePane()

	m := Model{
		WorkspaceRoot: root,
		Workbench:     w,
		Text:          &patch,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 900, 420)
	// Second frame picks up ObserveLayout viewport width for !Wrap mode.
	tt.Frame()

	first, last := w.View.list.Visible()
	if first != 0 || last <= first || last >= 100 {
		t.Fatalf("expected top viewport slice virtualized in [0..100), got visible=[%d..%d]", first, last)
	}
	if !tt.HasText("var row_00_0000 = 0") {
		t.Fatalf("expected top row visible at initial scroll: %v", tt.Texts())
	}
	if tt.HasText("var row_01_0599 = 599") {
		t.Fatal("tail row at line >1200 should not be built while scrolled at top")
	}

	// Scroll to the bottom of the virtualized list.
	w.View.list.ScrollToEnd()
	tt.Frame()
	firstEnd, lastEnd := w.View.list.Visible()
	if firstEnd <= 500 || lastEnd < 1200 {
		t.Fatalf("expected ScrollToEnd to reach bottom virtualized window, got visible=[%d..%d]", firstEnd, lastEnd)
	}
	if lastEnd-firstEnd >= 100 {
		t.Fatalf("built row span too large after ScrollToEnd: %d rows", lastEnd-firstEnd)
	}
	if !tt.HasText("var row_01_0599 = 599") {
		t.Fatalf("expected final line visible after ScrollToEnd: %v", tt.Texts())
	}
	if tt.HasText("var row_00_0000 = 0") {
		t.Fatal("top row should be unmounted after scrolling to bottom")
	}

	// Switch to Split + Wrap modes cleanly while keeping virtualization bounded.
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffSplit)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffWrap)); err != nil {
		t.Fatal(err)
	}
	sf, sl := w.View.list.Visible()
	if sl-sf >= 100 || !w.View.Split || !w.View.Wrap {
		t.Fatalf("unexpected split+wrap state: split=%v wrap=%v visible=[%d..%d]", w.View.Split, w.View.Wrap, sf, sl)
	}

	// Collapse all then expand all.
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffCollapseAll)); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("var row_01_0599 = 599") || tt.HasText("var row_00_0000 = 0") {
		t.Fatal("expected Collapse all to hide code rows")
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffExpandAll)); err != nil {
		t.Fatal(err)
	}
	w.View.list.ScrollToPosition(0, 0)
	tt.Frame()
	if !tt.HasText("var row_00_0000 = 0") {
		t.Fatal("expected Expand all to restore file rows")
	}
}

func TestPatchViewHorizontalScrollKeepsFileHeaderButtonsInViewport(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	veryLongLine := strings.Repeat("x", 240)
	patch := "diff --git a/long.go b/long.go\n--- a/long.go\n+++ b/long.go\n@@ -1,1 +1,1 @@\n-short\n+" + veryLongLine + "\n"
	root := t.TempDir()
	w := &Workbench{
		Root:        root,
		Loaded:      true,
		Repository:  true,
		AgainstHEAD: true,
		Changes:     []git.Change{{Path: "long.go", Worktree: 'M'}},
	}
	defer w.ClosePane()

	m := Model{WorkspaceRoot: root, Workbench: w, Text: &patch}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 640, 360)
	tt.Frame()

	rect, ok := tt.Find("Stage")
	if !ok {
		t.Fatal("Stage action button not found")
	}
	if rect.X+rect.W > 640 {
		t.Fatalf("file header Stage button pushed outside 640px viewport by long line: rect=%+v", rect)
	}
}

func TestPatchViewVirtualizedSelectionExcludesHeadersAndGutters(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	patch := "diff --git a/sample.go b/sample.go\n--- a/sample.go\n+++ b/sample.go\n@@ -10,2 +10,2 @@\n-oldLine\n+newLine\n sameLine\n"
	root := t.TempDir()
	w := &Workbench{
		Root:        root,
		Loaded:      true,
		Repository:  true,
		AgainstHEAD: true,
		Changes:     []git.Change{{Path: "sample.go", Worktree: 'M'}},
	}
	defer w.ClosePane()
	m := Model{WorkspaceRoot: root, Workbench: w, Text: &patch}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 700, 320)
	tt.Frame()

	r, ok := tt.Find("newLine")
	if !ok {
		t.Fatal("code line not found")
	}
	tt.ClickAt(r.X+4, r.Y+r.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Command("copy")
	copied := tt.Clipboard()
	if !strings.Contains(copied, "newLine") || !strings.Contains(copied, "oldLine") || !strings.Contains(copied, "sameLine") {
		t.Fatalf("Select All missed expected diff code lines: %q", copied)
	}
	if strings.Contains(copied, "@@") || strings.Contains(copied, "10") || strings.Contains(copied, "Stage") || strings.Contains(copied, "Expand all") || strings.Contains(copied, "Collapse all") {
		t.Fatalf("Select All included unselectable header or gutter tokens: %q", copied)
	}
	// Verify file header path button itself is unselectable (excluded from lines).
	for _, line := range strings.Split(copied, "\n") {
		if strings.TrimSpace(line) == "sample.go" {
			t.Fatalf("Select All included file header button row: %q", copied)
		}
	}

	// Switch to Split mode and verify "Before"/"After" headers are also excluded.
	if err := tt.Click(i18n.T(i18n.KeySurfacesDiffSplit)); err != nil {
		t.Fatal(err)
	}
	rs, ok := tt.Find("newLine")
	if !ok {
		t.Fatal("split code line not found")
	}
	tt.ClickAt(rs.X+4, rs.Y+rs.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Command("copy")
	splitCopied := tt.Clipboard()
	if !strings.Contains(splitCopied, "newLine") {
		t.Fatalf("split Select All missed code line: %q", splitCopied)
	}
	if strings.Contains(splitCopied, "Before") || strings.Contains(splitCopied, "After") || strings.Contains(splitCopied, "10") || strings.Contains(splitCopied, "@@") {
		t.Fatalf("split Select All included unselectable column headers or gutters: %q", splitCopied)
	}
}

func TestPatchViewCopyIncludesUnvisitedLogicalRows(t *testing.T) {
	patch := buildLargeSyntheticPatch(1, 1000)
	view := &patchViewState{Wrap: true}
	st := diff.Style{FontSize: 12, FG: ui.RGB(20, 20, 20), Add: ui.RGB(0, 140, 0), Muted: ui.RGB(100, 100, 100)}
	tt := ui.NewTester(func(c *ui.Context) {
		paintVirtualizedPatchList(c, Model{}, view, patch, st)
	}, 700, 320)
	r, ok := tt.Find("var row_00_0000 = 0")
	if !ok {
		t.Fatal("first patch line not visible")
	}
	if tt.HasText("var row_00_0999 = 999") {
		t.Fatal("tail was built before copying")
	}
	tt.ClickAt(r.X+4, r.Y+r.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	// File metadata is rendered as selectable patch text. Only the native file
	// header, hunk controls and number/sign gutters are unselectable.
	want := []string{"diff --git a/pkg/file_00.go b/pkg/file_00.go", "--- a/pkg/file_00.go", "+++ b/pkg/file_00.go", "package pkg"}
	for i := 0; i < 1000; i++ {
		want = append(want, fmt.Sprintf("var row_00_%04d = %d", i, i))
	}
	if got := tt.Clipboard(); got != strings.Join(want, "\n") {
		t.Fatalf("logical patch copy = %d bytes, want %d", len(got), len(strings.Join(want, "\n")))
	}
	if tt.HasText("var row_00_0999 = 999") {
		t.Fatal("copying patch rendered offscreen tail")
	}
}

func TestRecordedPatchViewWorksInsideOuterScrollAcrossModes(t *testing.T) {
	patch := "diff --git a/rec.go b/rec.go\n--- a/rec.go\n+++ b/rec.go\n@@ -1,1 +1,2 @@\n lineA\n+lineB\n"
	st := diff.Style{
		FontSize: 12,
		FG:       ui.RGB(20, 20, 20),
		Add:      ui.RGB(0, 180, 0),
		Del:      ui.RGB(200, 0, 0),
		Hunk:     ui.RGB(0, 100, 200),
		File:     ui.RGB(100, 100, 100),
		Muted:    ui.RGB(120, 120, 120),
	}
	for _, tc := range []struct {
		name        string
		split, wrap bool
	}{
		{name: "unified-nowrap", split: false, wrap: false},
		{name: "unified-wrap", split: false, wrap: true},
		{name: "split-nowrap", split: true, wrap: false},
		{name: "split-wrap", split: true, wrap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Scroll(c).Fill().Children(func() {
					RecordedPatchView(c, patch, tc.split, tc.wrap, st)
				})
			}, 700, 320)
			if !tt.HasText("rec.go") || !tt.HasText("lineA") || !tt.HasText("lineB") {
				t.Fatalf("RecordedPatchView (%s) missing expected texts: %v", tc.name, tt.Texts())
			}
		})
	}
}

func TestPatchViewStandaloneNilWorkbenchVirtualizes(t *testing.T) {
	patch := buildLargeSyntheticPatch(1, 500)
	st := diff.Style{
		FontSize: 12,
		FG:       ui.RGB(20, 20, 20),
		Add:      ui.RGB(0, 180, 0),
		Del:      ui.RGB(200, 0, 0),
		Hunk:     ui.RGB(0, 100, 200),
		File:     ui.RGB(100, 100, 100),
		Muted:    ui.RGB(120, 120, 120),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintPatchFiles(c, Model{}, patch, st)
		})
	}, 700, 300)
	tt.Frame()
	if !tt.HasText("var row_00_0000 = 0") {
		t.Fatalf("standalone virtualized view missing top line: %v", tt.Texts())
	}
	if tt.HasText("var row_00_0499 = 499") {
		t.Fatal("standalone virtualized view built bottom row >500 lines away while at top")
	}
}
