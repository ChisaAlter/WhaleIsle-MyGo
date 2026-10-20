package diffsurface

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

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
	Text   string
	Number int
	Kind   diffLineKind
}

// diffRowHeight is the painted height of one line; the original workbench gives
// every add, remove and context line the same 20px band so changes read as
// blocks rather than as differently sized text.
const diffRowHeight float32 = 20

// diffGutterWidth is the line-number column, and diffSignWidth the column that
// carries the +/- marker.
const (
	diffNumberWidth float32 = 32
	diffSignWidth   float32 = 12
)

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
			}
			lines = append(lines, diffLine{Kind: kind, Text: row.Meta})
			continue
		}
		removed := strings.HasPrefix(row.Old, "-")
		added := strings.HasPrefix(row.New, "+")
		switch {
		case removed && added:
			lines = append(lines,
				diffLine{Kind: diffLineDel, Number: row.OldNumber, Text: strings.TrimPrefix(row.Old, "-")},
				diffLine{Kind: diffLineAdd, Number: row.NewNumber, Text: strings.TrimPrefix(row.New, "+")})
		case removed:
			lines = append(lines, diffLine{Kind: diffLineDel, Number: row.OldNumber, Text: strings.TrimPrefix(row.Old, "-")})
		case added:
			lines = append(lines, diffLine{Kind: diffLineAdd, Number: row.NewNumber, Text: strings.TrimPrefix(row.New, "+")})
		default:
			text := row.Old
			if text == "" {
				text = row.New
			}
			lines = append(lines, diffLine{Kind: diffLineContext, Number: row.NewNumber, Text: strings.TrimPrefix(text, " ")})
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

// paintUnifiedRows paints one file change as a single column of line bands. A
// folded hunk keeps its header and drops its lines.
func paintUnifiedRows(c *ui.Context, rows []patchRow, st diff.Style, wrap bool, fold hunkFold) {
	minWidth := float32(0)
	if !wrap {
		minWidth = unifiedMinWidth(rows)
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
		hunk, hidden := -1, false
		for _, line := range unifiedLines(rows) {
			if line.Kind == diffLineHunk {
				hunk++
				hidden = fold.collapsed[hunk]
				paintHunkHeader(c, line.Text, hunk, st, fold)
				continue
			}
			if hidden {
				continue
			}
			paintDiffLine(c, line, st, minWidth, wrap)
		}
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
		ui.Row(c).FillWidth().MinWidth(0).Padding(4, 8).Radius(6).Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .16)).Children(func() {
			ui.Text(c, text).Font("monospace").FontSize(st.FontSize).TextColor(st.Hunk).NoWrap()
		})
		return
	}
	row := ui.Row(c).FillWidth().MinWidth(0).Gap(4).Height(24).Padding(0, 6).Radius(6).AlignItems(ui.Center).Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .16))
	row.Children(func() {
		caption := text
		if folded {
			caption = text + "  …"
		}
		toggle := ui.Button(c, "").Label(text).FillWidth().Height(22).Padding(0).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Start).Tooltip(text)
		toggle.Children(func() {
			if folded {
				ui.Icon(c, diffChevronRightIcon).Size(9, 9).TextColor(st.Muted).PassThrough()
			} else {
				ui.Icon(c, diffChevronDownIcon).Size(9, 9).TextColor(st.Muted).PassThrough()
			}
			ui.Text(c, " "+caption).Font("monospace").FontSize(st.FontSize).TextColor(st.Hunk).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		})
		if toggle.Hovered() {
			toggle.Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .12))
		}
		if toggle.Clicked() {
			fold.onToggle(index)
		}
	})
}

// paintDiffLine paints one line band: number gutter, +/- marker, then the text.
func paintDiffLine(c *ui.Context, line diffLine, st diff.Style, minWidth float32, wrap bool) {
	if line.Kind == diffLineHunk {
		ui.Row(c).FillWidth().MinWidth(0).Padding(4, 8).Radius(6).Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .16)).Children(func() {
			ui.Text(c, line.Text).Font("monospace").FontSize(st.FontSize).TextColor(st.Hunk).NoWrap()
		})
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
	row := ui.Row(c).FillWidth().MinWidth(minWidth).Height(diffRowHeight).Gap(6).Padding(0, 8).AlignItems(ui.Center)
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		number := ""
		if line.Number > 0 {
			number = strconv.Itoa(line.Number)
		}
		ui.Text(c, number).Width(diffNumberWidth).Shrink(0).Font("monospace").FontSize(st.FontSize - 1).TextColor(st.Muted).TextAlign(ui.End)
		ui.Text(c, sign).Width(diffSignWidth).Shrink(0).Font("monospace").FontSize(st.FontSize).TextColor(color).TextAlign(ui.Center)
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
	text := line
	if sign != " " {
		text = line[1:]
	} else if strings.HasPrefix(text, " ") {
		text = text[1:]
	}
	row := ui.Row(c).FillWidth().MinWidth(minWidth).Height(diffRowHeight).Gap(6).Padding(0, 4).AlignItems(ui.Center)
	if tint.A > 0 {
		row.Background(tint)
	}
	row.Children(func() {
		label := ""
		if number > 0 {
			label = strconv.Itoa(number)
		}
		ui.Text(c, label).Width(diffNumberWidth - 6).Shrink(0).Font("monospace").FontSize(st.FontSize - 1).TextColor(st.Muted).TextAlign(ui.End)
		ui.Text(c, sign).Width(diffSignWidth).Shrink(0).Font("monospace").FontSize(st.FontSize).TextColor(color).TextAlign(ui.Center)
		body := ui.Text(c, text).Font("monospace").FontSize(st.FontSize).TextColor(color).Grow(1).MinWidth(0)
		if !wrap {
			body.NoWrap()
		}
	})
}
