package markdown

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// DiffLine is a flattened patch row: path, gap, context, del or add.
type DiffLine struct{ Kind, Text string }

// DiffLabels localizes the patch fold control.
type DiffLabels struct {
	Expand, ExpandAria     func(int) string
	Collapse, CollapseAria string
}

// DiffPalette contains the installed semantic diff colors and translucent fills.
type DiffPalette struct{ Added, Deleted, AddedBG, DeletedBG ui.Color }

// ToolDiffBlock draws the installed inline patch preset. Its owner derives exact
// local patch rows and full clipboard text. The card starts folded and unwrapped.
func ToolDiffBlock(c *ui.Context, language string, rows []DiffLine, clipboard string, maxLines int, st Style, palette DiffPalette, labels DiffLabels) ui.Element {
	root := ui.Column(c.Key("diff-card")).MinWidth(0).Shrink(0).Radius(16).Background(st.CodeBG).Clip()
	wrap := ui.Local(root, "wrap", func() bool { return false })
	expanded := ui.Local(root, "expanded", func() bool { return false })
	copiedUntil := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	copied := c.Now().Before(*copiedUntil)
	if copied {
		c.After(copiedUntil.Sub(c.Now()))
	}
	prefs := theme.RenderPreferences()
	font := prefs.Code.Family
	size := max(float32(1), prefs.Code.Size-2)
	height := max(float32(1), prefs.Code.Size+6)
	languageInk := st.ToolCodeLanguage
	if languageInk.A == 0 {
		languageInk = st.Muted
	}
	languageLabel := language
	if toolGrammarForHint(language) == "" {
		languageLabel = codeLabel(st)
	}
	hidden := len(rows) - maxLines
	head := (maxLines + 1) / 2
	tail := maxLines - head
	capped := hidden > 0 && !*expanded
	end := len(rows)
	if capped {
		end = head
	}
	caption, aria := "", ""
	if hidden > 0 {
		caption, aria = labels.Expand(hidden), labels.ExpandAria(hidden)
		if *expanded {
			caption, aria = labels.Collapse, labels.CollapseAria
		}
	}
	textFor := func(row DiffLine) string {
		switch row.Kind {
		case "add":
			return "+ " + row.Text
		case "del":
			return "- " + row.Text
		case "context":
			return "  " + row.Text
		}
		return row.Text
	}
	width := float32(0)
	measure := func(row DiffLine) {
		weight := 400
		if row.Kind == "path" {
			weight = 500
		}
		w, _ := c.MeasureText(0, ui.Span{Text: textFor(row), Font: font, Size: size, Weight: weight})
		width = max(width, w+44)
	}
	for i := 0; i < end; i++ {
		measure(rows[i])
	}
	if capped {
		for i := len(rows) - tail; i < len(rows); i++ {
			measure(rows[i])
		}
	}
	if hidden > 0 {
		measure(DiffLine{Text: caption})
	}
	root.Children(func() {
		ui.Row(c.Key("banner")).Height(42).Shrink(0).Unselectable().Padding(10, 18, 8, 22).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, languageLabel).Font(font).FontSize(11).FixedLineHeight(18).TextColor(languageInk).NoWrap().Grow(1).MinWidth(0)
			ui.Row(c).Gap(4).Shrink(0).Children(func() {
				wrapLabel, unwrapLabel := codeWrapLabels(st)
				icon, tooltip := codeIconWrap, wrapLabel
				if *wrap {
					icon, tooltip = codeIconNoWrap, unwrapLabel
				}
				if toolCodeButton(c.Key("wrap"), icon, wrapLabel, tooltip, st).Role(ui.RoleToggleButton).Checked(*wrap).Clicked() {
					*wrap = !*wrap
					c.Invalidate()
				}
				icon, label := codeIconCopy, codeCopyLabel(st)
				if copied {
					icon, label = codeIconCheck, codeCopiedLabel(st)
				}
				if toolCodeButton(c.Key("copy"), icon, label, label, st).Clicked() && !copied {
					c.WriteClipboard(clipboard)
					*copiedUntil = c.Now().Add(time.Second)
					c.Invalidate()
				}
			})
		})
		paint := func() {
			body := ui.Column(c.Key("lines")).MinWidth(0).Shrink(0).Selectable().Padding(6, 0, 20, 0)
			if !*wrap {
				body.MinWidth(width)
			}
			body.Children(func() {
				line := func(i int) {
					row := rows[i]
					ink, bg, stripe, weight := st.Muted, ui.Transparent, ui.Transparent, 400
					switch row.Kind {
					case "path":
						weight = 500
					case "gap":
						ink = languageInk
					case "add":
						ink, bg, stripe = palette.Added, palette.AddedBG, palette.Added
					case "del":
						ink, bg, stripe = palette.Deleted, palette.DeletedBG, palette.Deleted
					}
					text := ui.Text(c, textFor(row)).Font(font).FontSize(size).FixedLineHeight(height).FontWeight(weight).TextColor(ink).Background(bg).Padding(0, 22, 0, 22).MinHeight(height).MinWidth(0).Shrink(0).Selectable()
					if !*wrap {
						text.NoWrap()
					}
					if stripe.A > 0 {
						text.DrawOver(func(p *ui.Painter, r ui.Rect) { p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, stripe, 0) })
					}
				}
				for i := 0; i < end; i++ {
					line(i)
				}
				if hidden > 0 {
					button := ui.ButtonBase(c.Key("fold")).FillWidth().MinWidth(0).Height(height).Shrink(0).Padding(0, 22, 0, 22).Background(ui.Transparent).Justify(ui.Start).Role(ui.RoleDisclosure).Expanded(*expanded).Label(aria)
					if button.Clicked() {
						*expanded = !*expanded
						c.Invalidate()
					}
					ink := languageInk
					if button.Hovered() {
						ink = st.Muted
					}
					button.Children(func() {
						ui.Text(c, caption).Font(font).FontSize(size).FixedLineHeight(height).TextColor(ink).NoWrap().PassThrough()
					})
				}
				if capped {
					for i := len(rows) - tail; i < len(rows); i++ {
						line(i)
					}
				}
			})
		}
		if *wrap {
			paint()
		} else {
			ui.Column(c.Key("columns")).ScrollOverflowX().MinWidth(0).Shrink(0).Children(paint)
		}
	})
	return root
}
