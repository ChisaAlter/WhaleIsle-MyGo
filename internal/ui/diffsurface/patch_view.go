package diffsurface

import (
	"strconv"
	"strings"
	"unicode/utf8"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/diff"
)

func paintDiffControls(c *ui.Context, m Model) {
	if m.Workbench == nil {
		return
	}
	view := &m.Workbench.View
	source := ""
	if m.Text != nil {
		source = *m.Text
	}
	view.update(source)
	ui.Row(c).Gap(6).Padding(4, 12, 6, 12).Wrap().Children(func() {
		mode := word("切换为分栏", "Use split view")
		if view.Split {
			mode = word("切换为统一视图", "Use unified view")
		}
		if ui.Button(c, mode).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			view.Split = !view.Split
		}
		wrap := word("启用折行", "Wrap lines")
		if view.Wrap {
			wrap = word("取消折行", "Do not wrap")
		}
		if ui.Button(c, wrap).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			view.Wrap = !view.Wrap
		}
		if ui.Button(c, word("全部折叠", "Collapse all")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			view.setAll(false)
		}
		if ui.Button(c, word("全部展开", "Expand all")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			view.setAll(true)
		}
	})
}

func paintPatchFiles(c *ui.Context, m Model, source string, st diff.Style) {
	if m.Workbench == nil {
		diff.View(c, source, st)
		return
	}
	view := &m.Workbench.View
	for _, file := range view.update(source) {
		f := file
		ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				toggle := word("展开", "Expand")
				if view.Open[f.Path] {
					toggle = word("折叠", "Collapse")
				}
				if ui.Button(c, toggle).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
					view.Open[f.Path] = !view.Open[f.Path]
				}
				title := f.Path
				if title == "" {
					title = word("差异内容", "Diff contents")
				}
				if ui.Button(c, title).FontSize(fonttheme.InterfaceSize(11)).Grow(1).MinWidth(0).Disabled(m.OnOpenFile == nil || !safePatchPath(f.Path)).Clicked() {
					if err := m.OnOpenFile(f.Path); err != nil {
						m.Workbench.Status = err.Error()
					}
				}
			})
			if !view.Open[f.Path] {
				return
			}
			render := func() {
				if !view.Split {
					text := ui.RichText(c, diff.Spans(f.Source, st)...).FontSize(st.FontSize).TextColor(st.FG)
					if view.Wrap {
						text.FillWidth().MinWidth(0)
					} else {
						text.NoWrap()
					}
					return
				}
				minWidth := float32(0)
				if !view.Wrap {
					minWidth = splitColumnWidth(f.Rows)
				}
				ui.Column(c).FillWidth().MinWidth(0).Children(func() {
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, word("原始", "Before")).Grow(1).MinWidth(minWidth).TextColor(st.Muted)
						ui.Text(c, word("修改后", "After")).Grow(1).MinWidth(minWidth).TextColor(st.Muted)
					})
					for _, row := range f.Rows {
						r := row
						if r.Meta != "" {
							text := ui.Text(c, r.Meta).Font("monospace").FontSize(st.FontSize).TextColor(st.Hunk)
							if view.Wrap {
								text.FillWidth().MinWidth(0)
							} else {
								text.NoWrap()
							}
							continue
						}
						ui.Row(c).FillWidth().Gap(8).Children(func() {
							paintPatchCell(c, r.Old, r.OldNumber, view.Wrap, minWidth, st)
							paintPatchCell(c, r.New, r.NewNumber, view.Wrap, minWidth, st)
						})
					}
				})
			}
			if view.Wrap {
				render()
			} else {
				ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(render)
			}
		})
	}
}
func splitColumnWidth(rows []patchRow) float32 {
	longest := 20
	for _, r := range rows {
		for _, line := range []string{r.Old, r.New} {
			if n := utf8.RuneCountInString(strings.ReplaceAll(line, "\t", "    ")); n > longest {
				longest = n
			}
		}
	}
	return float32(longest+7) * 7.5
}
func paintPatchCell(c *ui.Context, line string, number int, wrap bool, minWidth float32, st diff.Style) {
	color := st.FG
	if strings.HasPrefix(line, "+") {
		color = st.Add
	} else if strings.HasPrefix(line, "-") {
		color = st.Del
	}
	prefix := ""
	if number > 0 {
		prefix = strconv.Itoa(number) + "  "
	}
	if line == "" {
		line = " "
	}
	text := ui.RichText(c, ui.Span{Text: prefix, Color: st.Muted, Font: "monospace"}, ui.Span{Text: line, Color: color, Font: "monospace"}).FontSize(st.FontSize).Grow(1).MinWidth(minWidth)
	if !wrap {
		text.NoWrap()
	}
}
