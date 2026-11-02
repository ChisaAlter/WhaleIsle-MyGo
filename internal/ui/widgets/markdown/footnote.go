package markdown

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	extast "github.com/yuin/goldmark/extension/ast"
)

func superscriptNumber(index int) string {
	const digits = "⁰¹²³⁴⁵⁶⁷⁸⁹"
	runes := []rune(digits)
	var out strings.Builder
	for _, digit := range strconv.Itoa(index) {
		out.WriteRune(runes[digit-'0'])
	}
	return out.String()
}

func viewFootnotes(c *ui.Context, node *extast.FootnoteList, source string, st Style) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		border := st.RuleBorder
		if border.A == 0 {
			border = c.Theme().Border
		}
		ui.Box(c).FillWidth().Height(.5).Background(border)
		for footnote := node.FirstChild(); footnote != nil; footnote = footnote.NextSibling() {
			item := footnote.(*extast.Footnote)
			ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Text(c, strconv.Itoa(item.Index)+".").FontSize(st.FontSize).FixedLineHeight(BodyLineHeight(st)).TextColor(st.Muted)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() { viewDocumentBlocks(c, item, source, st, true) })
			})
		}
	})
}
