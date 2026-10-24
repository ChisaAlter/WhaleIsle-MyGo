package markdown

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// ReadLine retains the source file's own numbering and exact returned text.
type ReadLine struct {
	Number int
	Text   string
}

// ReadLabels are localized by the conversation owner, including count labels.
type ReadLabels struct {
	Window                 func(int, int) string
	Expand, ExpandAria     func(int) string
	Collapse, CollapseAria string
}

// ToolReadBlock renders the installed read card preset. The owner supplies its
// margin. Unlike ToolCodeBlock, its banner does not stick and wrapping starts
// disabled. Clipboard content includes every returned line, including folds.
func ToolReadBlock(c *ui.Context, label, language string, lines []ReadLine, totalLines, maxLines int, st Style, labels ReadLabels) ui.Element {
	root := ui.Column(c.Key("read-card")).MinWidth(0).Shrink(0).Radius(16).Background(st.CodeBG)
	wrap := ui.Local(root, "wrap", func() bool { return false })
	expanded := ui.Local(root, "expanded", func() bool { return false })
	copiedUntil := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	copied := c.Now().Before(*copiedUntil)
	if copied {
		c.After(copiedUntil.Sub(c.Now()))
	}
	parts := make([]string, len(lines))
	digits := 3
	for i, line := range lines {
		parts[i] = line.Text
		digits = max(digits, len(strconv.Itoa(line.Number)))
	}
	raw := strings.Join(parts, "\n")
	prefs := theme.RenderPreferences()
	size := max(float32(1), prefs.Code.Size-2)
	height := max(float32(1), prefs.Code.Size+6)
	gutter, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("0", digits), Font: prefs.Code.Family, Size: size})
	grammar := toolGrammarForHint(language)
	rows := make([][]ui.Span, len(lines))
	if grammar != "" {
		rows = toolReadHighlightedLines(grammar, raw, len(lines), st, size)
	} else {
		for i, line := range lines {
			rows[i] = []ui.Span{{Text: line.Text, Color: st.FG, Size: size}}
		}
	}
	languageLabel := language
	if grammar == "" {
		languageLabel = codeLabel(st)
	}
	languageInk := st.ToolCodeLanguage
	if languageInk.A == 0 {
		languageInk = st.Muted
	}
	root.Children(func() {
		ui.Row(c.Key("banner")).MinWidth(0).Height(42).Shrink(0).Padding(10, 18, 8, 22).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Row(c).MinWidth(0).Grow(1).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, languageLabel).Font(prefs.Code.Family).FontSize(11).FixedLineHeight(18).TextColor(languageInk).NoWrap().Shrink(0)
				ui.Text(c, label).Font(prefs.Interface.Family).FontSize(11).FixedLineHeight(18).TextColor(languageInk).SingleLine().MinWidth(0).Shrink(1).Tooltip(label)
			})
			ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
				if len(lines) < totalLines {
					ui.Text(c, labels.Window(len(lines), totalLines)).Font(prefs.Interface.Family).FontSize(11).FixedLineHeight(18).TextColor(languageInk).NoWrap().Shrink(0).Margin(0, 8, 0, 0)
				}
				wrapLabel, unwrapLabel := codeWrapLabels(st)
				icon, tooltip := codeIconWrap, wrapLabel
				if *wrap {
					icon, tooltip = codeIconNoWrap, unwrapLabel
				}
				if toolCodeButton(c.Key("wrap"), icon, wrapLabel, tooltip, st).Role(ui.RoleToggleButton).Checked(*wrap).Clicked() {
					*wrap = !*wrap
					c.Invalidate()
				}
				if len(lines) > 0 {
					icon, label := codeIconCopy, codeCopyLabel(st)
					if copied {
						icon, label = codeIconCheck, codeCopiedLabel(st)
					}
					if toolCodeButton(c.Key("copy"), icon, label, label, st).Clicked() && !copied {
						c.WriteClipboard(raw)
						*copiedUntil = c.Now().Add(time.Second)
						c.Invalidate()
					}
				}
			})
		})
		hidden := len(lines) - maxLines
		head := (maxLines + 1) / 2
		tail := maxLines - head
		capped := hidden > 0 && !*expanded
		paint := func() {
			ui.Column(c.Key("lines")).MinWidth(0).Shrink(0).Padding(6, 22, 20, 22).Children(func() {
				line := func(i int) {
					ui.Row(c.Key(lines[i].Number)).MinWidth(0).Shrink(0).MinHeight(height).AlignItems(ui.Start).Children(func() {
						ui.Text(c, strconv.Itoa(lines[i].Number)).Role(ui.RoleNone).Font(prefs.Code.Family).FontSize(size).FixedLineHeight(height).TextColor(languageInk).Width(gutter).Margin(0, 12, 0, 0).TextAlign(ui.End).NoWrap().Shrink(0)
						text := ui.RichText(c, rows[i]...).Font(prefs.Code.Family).FontSize(size).FixedLineHeight(height).TextColor(st.FG).MinWidth(0).MinHeight(height).Shrink(0).Selectable()
						if !*wrap {
							text.NoWrap()
						} else {
							text.Shrink(1)
						}
					})
				}
				end := len(lines)
				if capped {
					end = head
				}
				for i := 0; i < end; i++ {
					line(i)
				}
				if hidden > 0 {
					caption, aria := labels.Expand(hidden), labels.ExpandAria(hidden)
					if *expanded {
						caption, aria = labels.Collapse, labels.CollapseAria
					}
					button := ui.ButtonBase(c.Key("fold")).FillWidth().Height(height).Shrink(0).MinWidth(0).Padding(0, 0, 0, gutter+12).Justify(ui.Start).Background(ui.Transparent).Label(aria).Role(ui.RoleDisclosure).Expanded(*expanded)
					if button.Clicked() {
						*expanded = !*expanded
						c.Invalidate()
					}
					ink := languageInk
					if button.Hovered() {
						ink = st.Muted
					}
					button.Children(func() {
						ui.Text(c, caption).Font(prefs.Code.Family).FontSize(size).FixedLineHeight(height).TextColor(ink).NoWrap().PassThrough()
					})
				}
				if capped {
					for i := len(lines) - tail; i < len(lines); i++ {
						line(i)
					}
				}
			})
		}
		if *wrap {
			paint()
		} else {
			ui.ScrollHorizontal(c.Key("columns")).MinWidth(0).Shrink(0).Children(paint)
		}
	})
	return root
}

func toolReadHighlightedLines(grammar, raw string, count int, st Style, size float32) [][]ui.Span {
	rows := make([][]ui.Span, count)
	if count == 0 {
		return rows
	}
	spans := []ui.Span{{Text: raw, Color: st.FG}}
	if grammar != "" {
		spans = toolCodeSpans(grammar, raw, st)
	}
	index := 0
	for _, span := range spans {
		for i, part := range strings.Split(span.Text, "\n") {
			if i > 0 {
				index++
			}
			if index >= count {
				break
			}
			if part != "" {
				copy := span
				copy.Text, copy.Size = part, size
				rows[index] = append(rows[index], copy)
			}
		}
	}
	return rows
}
