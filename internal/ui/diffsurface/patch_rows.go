package diffsurface

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/widgets/diff"
)

// diffLineKind is the visual role of one rendered patch line.
type diffLineKind int

const (
	diffLineContext diffLineKind = iota
	diffLineAdd
	diffLineDel
	diffLineHunk
	diffLineMeta
)

// diffLine is one single-column line of a unified view.
type diffLine struct {
	Text string
	// OldNumber and NewNumber fill the two number columns ReviewDiff lays
	// out; a zero column paints empty, like a del line's new side.
	OldNumber, NewNumber int
	Kind                 diffLineKind
	// Hunk is the line's hunk ordinal, set on headers so a folded header can
	// name the hunk it hides.
	Hunk int
}

// ReviewDiff.module.css `.line` gives every add, del and context row the same
// 22px band (min-height + line-height 22), so changes read as blocks rather
// than as differently sized text.
const diffRowHeight float32 = 22

// ReviewDiff.module.css `.hunkHeader` pads 4px around the 11px/19px code font.
const diffHunkHeaderHeight float32 = 27

// ReviewDiff.module.css `.hunks` indents the hunk block 20px under its file
// header, so every hunk-section row shifts right by that margin.
const diffHunksIndent float32 = 20

// The original's `.line` grid lays out two 3.5em number columns and a 1.2em
// sign column against the 11px code font.
func diffNumberWidth(st diff.Style) float32 { return 3.5 * st.FontSize }
func diffSignWidth(st diff.Style) float32   { return 1.2 * st.FontSize }

// diffMetricFontSize sizes the scroll-width estimates; the production styles
// paint at the original's 11px `--dsw-font-markdown-code-block` token.
const diffMetricFontSize float32 = 11

// unifiedLines flattens the paired rows the patch parser produces into the
// single column the unified view paints: a removed line and the added line that
// replaced it become two rows.
func unifiedLines(rows []patchRow) []diffLine {
	lines := make([]diffLine, 0, len(rows))
	for _, row := range rows {
		if row.Meta != "" {
			kind := diffLineMeta
			if strings.HasPrefix(row.Meta, "@@") {
				kind = diffLineHunk
			} else if diffNoiseMeta(row.Meta) {
				continue
			}
			lines = append(lines, diffLine{Kind: kind, Text: row.Meta})
			continue
		}
		removed := strings.HasPrefix(row.Old, "-")
		added := strings.HasPrefix(row.New, "+")
		switch {
		case removed && added:
			lines = append(lines,
				diffLine{Kind: diffLineDel, OldNumber: row.OldNumber, Text: strings.TrimPrefix(row.Old, "-")},
				diffLine{Kind: diffLineAdd, NewNumber: row.NewNumber, Text: strings.TrimPrefix(row.New, "+")})
		case removed:
			lines = append(lines, diffLine{Kind: diffLineDel, OldNumber: row.OldNumber, Text: strings.TrimPrefix(row.Old, "-")})
		case added:
			lines = append(lines, diffLine{Kind: diffLineAdd, NewNumber: row.NewNumber, Text: strings.TrimPrefix(row.New, "+")})
		default:
			text := row.Old
			if text == "" {
				text = row.New
			}
			lines = append(lines, diffLine{Kind: diffLineContext, OldNumber: row.OldNumber, NewNumber: row.NewNumber, Text: strings.TrimPrefix(text, " ")})
		}
	}
	return lines
}

// unifiedMinWidth is the widest rendered line in DIPs, so every band can span
// the same width inside a horizontal scroller.
func unifiedMinWidth(rows []patchRow) float32 {
	longest := 20
	for _, line := range unifiedLines(rows) {
		if n := utf8.RuneCountInString(strings.ReplaceAll(line.Text, "\t", "    ")); n > longest {
			longest = n
		}
	}
	return float32(longest+10) * 7.5
}

// visibleUnifiedLines is the unified view's line list after folding: a folded
// hunk keeps its header and loses its lines, and every header learns its index.
func visibleUnifiedLines(rows []patchRow, fold hunkFold) []diffLine {
	lines := unifiedLines(rows)
	visible := make([]diffLine, 0, len(lines))
	hunk, hidden := -1, false
	for _, line := range lines {
		if line.Kind == diffLineHunk {
			hunk++
			hidden = fold.collapsed[hunk]
			line.Hunk = hunk
			visible = append(visible, line)
			continue
		}
		if hidden {
			continue
		}
		visible = append(visible, line)
	}
	return visible
}

// diffLineBox is the height one line occupies in both the gutter and the text
// column; keeping the two columns on the same sequence of heights is what aligns
// them.
func diffLineBox(kind diffLineKind) float32 {
	if kind == diffLineHunk {
		return diffHunkHeaderHeight
	}
	return diffRowHeight
}

// paintUnifiedRows paints one file change as a single column of line bands.
//
// Without wrapping the text scrolls sideways inside its own scroller while the
// number gutter stays put, so the end of a long line is reachable and the line
// still shows its own number. With wrapping there is nothing to scroll and the
// rows paint as one column.
func paintUnifiedRows(c *ui.Context, rows []patchRow, st diff.Style, wrap bool, fold hunkFold) {
	lines := visibleUnifiedLines(rows, fold)
	if wrap {
		ui.Column(c).FillWidth().MinWidth(0).Gap(1).Selectable().Children(func() {
			for _, line := range lines {
				if line.Kind == diffLineHunk {
					paintHunkHeader(c, line.Text, line.Hunk, st, fold)
					continue
				}
				paintDiffLine(c, line, st, 0, true)
			}
		})
		return
	}
	minWidth := unifiedMinWidth(rows)
	ui.Row(c).FillWidth().MinWidth(0).Gap(0).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(2*diffNumberWidth(st) + diffSignWidth(st)).Shrink(0).Gap(1).Unselectable().Children(func() {
			for _, line := range lines {
				paintDiffGutter(c, line, st)
			}
		})
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
			ui.Column(c).MinWidth(minWidth).Gap(1).Selectable().Children(func() {
				for _, line := range lines {
					paintDiffText(c, line, st, fold)
				}
			})
		})
	})
}

// diffNoiseMeta names the git file-header lines the original's ReviewDiff
// drops: the file pair, index and ---/+++ markers, and the extended git
// headers. `\` markers (no newline at EOF) stay, as do real notes.
func diffNoiseMeta(text string) bool {
	for _, prefix := range []string{
		"diff --git ", "index ", "--- ", "+++ ",
		"old mode", "new mode", "new file mode", "deleted file mode",
		"similarity index", "dissimilarity index", "rename from", "rename to",
		"copy from", "copy to", "GIT binary patch", "Binary files ", "literal ",
	} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// diffLineKindLabel is the accessible name a rendered patch row reports,
// matching the original's data-diff-line values.
func diffLineKindLabel(kind diffLineKind) string {
	switch kind {
	case diffLineAdd:
		return "add"
	case diffLineDel:
		return "del"
	case diffLineHunk:
		return "hunk"
	case diffLineMeta:
		return "meta"
	}
	return "context"
}

// paintDiffGutter paints one line's number and change marker outside the
// horizontal scroller.
func paintDiffGutter(c *ui.Context, line diffLine, st diff.Style) {
	color := st.FG
	sign := " "
	var tint ui.Color
	switch line.Kind {
	case diffLineAdd:
		color, tint, sign = st.Add, st.Add.Alpha(.08), "+"
	case diffLineDel:
		color, tint, sign = st.Del, st.Del.Alpha(.08), "-"
	}
	row := ui.Row(c).Width(2*diffNumberWidth(st) + diffSignWidth(st)).Height(diffLineBox(line.Kind)).Gap(0).AlignItems(ui.Center).Label("Diff gutter " + diffLineKindLabel(line.Kind))
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		if line.Kind == diffLineHunk || line.Kind == diffLineMeta {
			return
		}
		paintDiffNumberCell(c, line.OldNumber, st, tint, true)
		paintDiffNumberCell(c, line.NewNumber, st, tint, false)
		ui.Text(c, sign).Width(diffSignWidth(st)).Shrink(0).Font("monospace").FontSize(st.FontSize).TextColor(color).TextAlign(ui.Center)
	})
}

// paintDiffNumberCell paints one `.number` grid cell: 3.5em wide, right aligned
// with the original's 8px right padding, tertiary ink, and the gutter fill the
// original gives numbers on changed rows.
func paintDiffNumberCell(c *ui.Context, number int, st diff.Style, tint ui.Color, landmark bool) {
	cell := ui.Row(c).Width(diffNumberWidth(st)).Shrink(0).FillHeight().AlignItems(ui.Center)
	if tint.A > 0 {
		cell.Background(ui.RGBA(tint.R, tint.G, tint.B, .16))
	}
	if landmark {
		cell.Label("Diff gutter")
	}
	cell.Children(func() {
		text := ""
		if number > 0 {
			text = strconv.Itoa(number)
		}
		ui.Text(c, text).Grow(1).Font("monospace").FontSize(st.FontSize-1).TextColor(st.Muted).TextAlign(ui.End).Padding(0, 8, 0, 0).Unselectable()
	})
}

// paintDiffText paints one line's text inside the scroller, at the same height
// its gutter cell used.
func paintDiffText(c *ui.Context, line diffLine, st diff.Style, fold hunkFold) {
	if line.Kind == diffLineHunk {
		paintHunkHeader(c, line.Text, line.Hunk, st, fold)
		return
	}
	color := st.FG
	var tint ui.Color
	switch line.Kind {
	case diffLineAdd:
		color, tint = st.Add, st.Add.Alpha(.08)
	case diffLineDel:
		color, tint = st.Del, st.Del.Alpha(.08)
	case diffLineMeta:
		color = st.File
	}
	row := ui.Row(c).FillWidth().Height(diffLineBox(line.Kind)).MinWidth(0).Padding(0, 8).AlignItems(ui.Center).Label("Diff line " + diffLineKindLabel(line.Kind))
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		ui.Text(c, line.Text).Font("monospace").FontSize(st.FontSize).TextColor(color).NoWrap()
	})
}

// hunkFold is one file's per-hunk collapse state. A zero fold paints every hunk
// expanded and leaves the headers inert.
type hunkFold struct {
	collapsed map[int]bool
	onToggle  func(index int)
}

// paintHunkHeader paints one hunk header. A bound fold turns it into a control
// that hides the hunk's lines; an unbound one stays a static label.
func paintHunkHeader(c *ui.Context, text string, index int, st diff.Style, fold hunkFold) {
	folded := fold.collapsed[index]
	if fold.onToggle == nil {
		// ReviewDiff.module.css `.hunkHeader`: plain tertiary code text with
		// 4px/16px padding — no band, no radius.
		ui.Row(c).FillWidth().MinWidth(0).Padding(4, 16).Children(func() {
			ui.Text(c, text).Font("monospace").FontSize(st.FontSize).FixedLineHeight(19).TextColor(st.Hunk).NoWrap()
		})
		return
	}
	row := ui.Row(c).FillWidth().MinWidth(0).Gap(4).Height(diffHunkHeaderHeight).Padding(0, 16).AlignItems(ui.Center)
	row.Children(func() {
		caption := text
		if folded {
			caption = text + "  …"
		}
		toggle := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Label:   text,
			Content: func() {
				if folded {
					ui.Icon(c, diffChevronRightIcon).Size(9, 9).TextColor(st.Muted).PassThrough()
				} else {
					ui.Icon(c, diffChevronDownIcon).Size(9, 9).TextColor(st.Muted).PassThrough()
				}
				ui.Text(c, " "+caption).Font("monospace").FontSize(st.FontSize).TextColor(st.Hunk).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
			},
		}).FillWidth().Height(22).Padding(0).AlignItems(ui.Start).Tooltip(text)
		if toggle.Clicked() {
			fold.onToggle(index)
		}
	})
}

// paintDiffLine paints one line band: number gutter, +/- marker, then the text.
func paintDiffLine(c *ui.Context, line diffLine, st diff.Style, minWidth float32, wrap bool) {
	if line.Kind == diffLineHunk {
		paintHunkHeader(c, line.Text, line.Hunk, st, hunkFold{})
		return
	}
	if line.Kind == diffLineMeta {
		ui.Text(c, line.Text).FillWidth().MinWidth(0).Font("monospace").FontSize(st.FontSize).TextColor(st.File).NoWrap().Padding(2, 8)
		return
	}
	color := st.FG
	sign := " "
	var tint ui.Color
	switch line.Kind {
	case diffLineAdd:
		color, tint, sign = st.Add, st.Add.Alpha(.08), "+"
	case diffLineDel:
		color, tint, sign = st.Del, st.Del.Alpha(.08), "-"
	}
	row := ui.Row(c).FillWidth().MinWidth(minWidth).Height(diffRowHeight).Gap(0).Margin(0, 0, 0, diffHunksIndent).Padding(0, 16, 0, 0).AlignItems(ui.Center).Label("Diff line " + diffLineKindLabel(line.Kind))
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		paintDiffNumberCell(c, line.OldNumber, st, tint, true)
		paintDiffNumberCell(c, line.NewNumber, st, tint, false)
		ui.Text(c, sign).Width(diffSignWidth(st)).Shrink(0).Font("monospace").FontSize(st.FontSize).TextColor(color).TextAlign(ui.Center).Unselectable()
		text := ui.Text(c, line.Text).Font("monospace").FontSize(st.FontSize).TextColor(color).Grow(1).MinWidth(0)
		if !wrap {
			text.NoWrap()
		}
	})
}

// paintSplitCell paints one side of the split view with the same band height and
// change tint as the unified view.
func paintSplitCell(c *ui.Context, line string, number int, wrap bool, minWidth float32, st diff.Style) {
	color := st.FG
	sign := " "
	var tint ui.Color
	switch {
	case strings.HasPrefix(line, "+"):
		color, tint, sign = st.Add, st.Add.Alpha(.08), "+"
	case strings.HasPrefix(line, "-"):
		color, tint, sign = st.Del, st.Del.Alpha(.08), "-"
	}
	text := splitCellText(line)
	row := ui.Row(c).FillWidth().MinWidth(minWidth).Height(diffRowHeight).Gap(6).Padding(0, 4).AlignItems(ui.Center)
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		label := ""
		if number > 0 {
			label = strconv.Itoa(number)
		}
		ui.Text(c, label).Width(diffNumberWidth(st) - 6).Shrink(0).Font("monospace").FontSize(st.FontSize - 1).TextColor(st.Muted).TextAlign(ui.End).Unselectable()
		ui.Text(c, sign).Width(diffSignWidth(st)).Shrink(0).Font("monospace").FontSize(st.FontSize).TextColor(color).TextAlign(ui.Center).Unselectable()
		body := ui.Text(c, text).Font("monospace").FontSize(st.FontSize).TextColor(color).Grow(1).MinWidth(0)
		if !wrap {
			body.NoWrap()
		}
	})
}
