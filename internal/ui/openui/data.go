package openui

import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

func init() {
	Register("Table", tableView)
	Register("Progress", progressView)
	Register("Icon", iconView)
}

// numeric reports a number, or text that reads as a number or amount
// ("12.5%", "$1,200"), so the column aligns right.
func numeric(v lang.Value) bool {
	switch t := v.(type) {
	case float64:
		return true
	case string:
		s := strings.TrimSpace(t)
		s = strings.TrimLeft(s, "+-−$€£¥")
		s = strings.TrimRight(s, "%")
		s = strings.ReplaceAll(s, ",", "")
		if s == "" {
			return false
		}
		return !math.IsNaN(lang.StringToNumber(s))
	}
	return false
}

// tableGrid draws a coss-styled table (40 DIP muted header, bordered rows,
// 10 DIP cell padding) whose cells may hold components; numeric columns
// align right. It scrolls sideways when the columns do not fit.
func (r *Render) tableGrid(columns []string, rows [][]lang.Value) ui.Element {
	t := r.T
	right := make([]bool, len(columns))
	for i := range columns {
		n, all := 0, true
		for _, row := range rows {
			if i < len(row) && row[i] != nil && row[i] != "" {
				n++
				all = all && numeric(row[i])
			}
		}
		right[i] = n > 0 && all
	}
	return coss.ScrollArea(r.C, coss.ScrollAreaOptions{Horizontal: true}, func() {
		g := ui.Grid(r.C).MinWidth(0).FillWidth().Role(ui.RoleTable)
		tracks := make([]ui.Track, len(columns))
		for i := range tracks {
			tracks[i] = ui.FitContent()
		}
		if len(tracks) > 0 {
			tracks[0] = ui.Fr(1)
		}
		g.ColumnTracks(tracks...)
		g.Children(func() {
			cell := func(i int, last bool) ui.Element {
				e := ui.Row(r.C).Padding(design.Sp(2.5)).AlignItems(ui.Center).MinWidth(0)
				if !last {
					e.BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border)
				}
				if right[i] {
					e.Justify(ui.End)
				}
				return e
			}
			for i, col := range columns {
				r.C.Key(i)
				cell(i, len(rows) == 0).Role(ui.RoleColumnHeader).Children(func() {
					r.text(col, design.TextSM, design.WeightMedium, t.MutedForeground)
				})
			}
			for ri, row := range rows {
				for i := range columns {
					r.C.Key(ri*len(columns) + i + len(columns))
					var v lang.Value
					if i < len(row) {
						v = row[i]
					}
					cell(i, ri == len(rows)-1).Role(ui.RoleCell).Children(func() {
						switch x := v.(type) {
						case *lang.Element, []lang.Value:
							r.Content(x)
						case nil:
						default:
							e := r.inlineText(lang.ToString(x), design.TextSM, design.WeightNormal, t.Foreground, nil)
							if right[i] {
								e.FontFeatures("tnum")
							}
						}
					})
				}
			}
		})
	})
}

func tableView(r *Render, el *lang.Element) ui.Element {
	var cols []string
	for _, c := range list(el, "columns") {
		cols = append(cols, lang.ToString(c))
	}
	var rows [][]lang.Value
	for _, row := range list(el, "rows") {
		if l, ok := row.([]lang.Value); ok {
			rows = append(rows, l)
		}
	}
	return r.tableGrid(cols, rows)
}

func progressView(r *Render, el *lang.Element) ui.Element {
	v := numOr(el, "value", 0)
	hi := numOr(el, "max", 100)
	if hi <= 0 {
		hi = 100
	}
	frac := max(0, min(v/hi, 1))
	o := coss.ProgressOptions{Label: str(el, "label")}
	if o.Label != "" {
		o.Value = lang.NumberToString(math.Round(frac*100)) + "%"
	}
	return coss.Progress(r.C, frac, o)
}

func iconSize(s string) float32 {
	switch s {
	case "sm":
		return design.IconXS
	case "lg":
		return design.IconLG
	case "xl":
		return design.Sp(6)
	}
	return design.IconSM
}

func iconView(r *Render, el *lang.Element) ui.Element {
	col := r.T.Foreground
	if c, ok := r.color(str(el, "color")); ok {
		col = c
	}
	return r.Inline(r.icon(str(el, "name"), iconSize(str(el, "size")), col))
}
