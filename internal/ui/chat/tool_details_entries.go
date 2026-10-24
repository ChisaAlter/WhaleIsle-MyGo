package chat

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// The installed ToolDetails module paints one recorded item as
//
//	heading (title, badge) · subtitle · description · fields · lines · code · groups
//
// with a .5px hairline between list items. This file owns the generic entry
// parts the per-tool families had not reached yet: the `lines` bullet list, the
// nested `groups` disclosures, the `unchanged` container and the shared
// CodeBlock chrome. Measurements follow the installed module CSS, whose values
// are the ones in reference/electron/.../tool/components/ToolDetails.module.css.
// The inspected app ships the same rules minified in
// <install>/resources/vendor/deepseek-harness/packages/client/ui-tool/lib/client.js:1507.

// toolDetailsChevron is IconChevronRightOutlineRegular from the installed
// primitives: group and unchanged summaries turn it 90° while open.
var toolDetailsChevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M6 12L9.29289 8.70711C9.68342 8.31658 9.68342 7.68342 9.29289 7.29289L6 4" stroke="currentColor"/></svg>`))

// toolDetailsLines paints the installed `ul.lines`: one disc per recorded string
// inset 18px, a 4px gap between lines, and the recorded whitespace kept.
func toolDetailsLines(c *ui.Context, lines []string, font string, ink ui.Color) {
	list := ui.Column(c.Key("lines")).FillWidth().MinWidth(0).Margin(6, 0, 0, 0).Gap(4)
	list.Children(func() {
		for index, line := range lines {
			ui.Row(c.Key(index)).FillWidth().MinWidth(0).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "•").Width(18).NoWrap().Font(font).FontSize(13).FixedLineHeight(20).TextColor(ink).Shrink(0)
				ui.Text(c, line).Font(font).FontSize(13).FixedLineHeight(20).TextColor(ink).Grow(1).MinWidth(0).Selectable()
			})
		}
	})
}

// toolDetailsCode paints one installed code entry. A recorded language selects
// the shared CodeBlock card, whose banner carries the language, the wrap toggle
// and the copy action. An entry without one keeps the plain selectable region
// round125 shipped for the background-job receipt, so those cards are unchanged.
func toolDetailsCode(c *ui.Context, code toolDetailCode, st ToolCardStyle, ink toolDetailsPalette, codeOnly bool) {
	region := ui.Scroll(c.Key("code")).FillWidth().MinWidth(0).MaxHeight(240)
	if !codeOnly {
		region.Margin(8, 0, 0, 0)
	}
	if code.Language == "" {
		region.Children(func() {
			ui.Text(c, code.Text).Font(ink.codeFont).FontSize(12).FixedLineHeight(18).TextColor(ink.secondary).
				FillWidth().MinWidth(0).Selectable()
		})
		return
	}
	// The code card keeps its banner at the top of its own viewport.
	position := ui.Local(region, "position", func() ui.ScrollState { return ui.ScrollState{} })
	region.TrackScroll(position).Children(func() {
		markdown.ToolCodeBlock(c, code.Language, code.Text, toolDetailsCodeStyle(st, ink), position.Y)
	})
}

// toolDetailsCodeStyle localizes the shared code card for this card's ink.
func toolDetailsCodeStyle(st ToolCardStyle, ink toolDetailsPalette) markdown.Style {
	return markdown.Style{FontSize: 13, FG: ink.secondary, Muted: ink.secondary, CodeBG: ink.background, Dark: darkSurface(st.FG),
		ToolCodeLanguage: st.Muted, ToolCodeHover: chrome.Color(st.Colors.Hover),
		CopyLabel: i18n.T(i18n.KeyToolDetailCopy), CopiedLabel: i18n.T(i18n.KeyToolDetailCopied),
		WrapLabel: i18n.T(i18n.KeyToolDetailWrap), NoWrapLabel: i18n.T(i18n.KeyToolDetailUnwrap),
		CodeLabel: i18n.T(i18n.KeyToolDetailCodeLabel)}
}

// toolDetailsGroupView paints one installed `groups` disclosure: an 8px top
// margin above its hairline, the 6px chevron label, and children at 9/0/2/20
// with the ordinary hairline between them. A code-only child keeps 4/0/0/20,
// as the installed `:has(> code:only-child)` rule does.
func toolDetailsGroupView(c *ui.Context, group toolDetailGroup, st ToolCardStyle, ink toolDetailsPalette, index int) {
	root := ui.Column(c.Key("group:"+strconv.Itoa(index))).FillWidth().MinWidth(0).
		Margin(8, 0, 0, 0).BorderWidth(.5, 0, 0, 0).BorderColor(st.Border)
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		toolDetailsSummary(c, "summary", group.Label, ink.caption, ink.font, [4]float32{8, 0, 0, 0}, open)
		if !*open {
			return
		}
		items := ui.Column(c.Key("items")).FillWidth().MinWidth(0)
		items.Children(func() {
			for childIndex, child := range group.Items {
				if childIndex > 0 {
					ui.Box(c).FillWidth().Height(toolCSSHalfBorder(c)).Shrink(0).Background(st.Border)
				}
				toolDetailsItem(c, child, st, ink, 0, true, childIndex)
			}
		})
	})
}

// toolDetailsUnchangedView paints the installed `unchanged` container after the
// list. Its summary is the muted count at 9/14, and its entries use the ordinary
// item padding once opened, so the container never reserves the Inspect inset.
func toolDetailsUnchangedView(c *ui.Context, unchanged toolDetailsUnchanged, st ToolCardStyle, ink toolDetailsPalette) {
	root := ui.Column(c.Key("unchanged")).FillWidth().MinWidth(0).
		BorderWidth(.5, 0, 0, 0).BorderColor(st.Border)
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		toolDetailsSummary(c, "summary", unchanged.Label, ink.tertiary, ink.font, [4]float32{9, 14, 9, 14}, open)
		if !*open {
			return
		}
		items := ui.Column(c.Key("items")).FillWidth().MinWidth(0)
		items.Children(func() {
			for index, item := range unchanged.Items {
				if index > 0 {
					ui.Box(c).FillWidth().Height(toolCSSHalfBorder(c)).Shrink(0).Background(st.Border)
				}
				toolDetailsItem(c, item, st, ink, 14, false, index)
			}
		})
	})
}

// toolDetailsSummary paints one installed `<summary>` row: the chevron, a 6px
// gap and the label. It is the whole hit target and a keyboard disclosure, as
// the installed native element is.
func toolDetailsSummary(c *ui.Context, key, label string, ink ui.Color, font string, padding [4]float32, open *bool) {
	row := ui.ButtonBase(c.Key(key)).FillWidth().MinWidth(0).Shrink(0).
		Padding(padding[0], padding[1], padding[2], padding[3]).Gap(6).
		AlignItems(ui.Center).Justify(ui.Start).Background(ui.Transparent).Radius(0).
		Role(ui.RoleDisclosure).Expanded(*open).Label(label)
	if row.Clicked() {
		*open = !*open
		c.Invalidate()
	}
	row.Children(func() {
		chevron := ui.Icon(c, toolDetailsChevron).Size(14, 14).Shrink(0).TextColor(ink).Role(ui.RoleNone).PassThrough()
		if *open {
			chevron.Rotate(90)
		}
		ui.Text(c, label).Font(font).FontSize(13).FixedLineHeight(20).TextColor(ink).MinWidth(0).PassThrough()
	})
}
