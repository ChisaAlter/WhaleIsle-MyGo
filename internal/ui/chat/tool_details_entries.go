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
//	heading (status, title, badge, status text) · subtitle · description · fields · lines · markdown · code · groups
//
// with a .5px hairline between list items. This file owns the generic entry
// parts the per-tool families had not reached yet: the `lines` bullet list, the
// nested `groups` disclosures, the `unchanged` container, the `markdown` prose,
// the clickable file heading, the todo status markers and the shared CodeBlock
// chrome. Measurements follow the installed module CSS, whose values are the
// ones in reference/electron/.../tool/components/ToolDetails.module.css.
// The inspected app ships the same rules minified in
// <install>/resources/vendor/deepseek-harness/packages/client/ui-tool/lib/client.js:1507.

// toolDetailsChevron is IconChevronRightOutlineRegular from the installed
// primitives: group and unchanged summaries turn it 90° while open.
var toolDetailsChevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M6 12L9.29289 8.70711C9.68342 8.31658 9.68342 7.68342 9.29289 7.29289L6 4" stroke="currentColor"/></svg>`))

// toolDetailsCheckIcon and toolDetailsPlayIcon are the installed
// IconCheckOutlineRegular and IconPlayOutlineRegular status markers.
var (
	toolDetailsCheckIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M2.25 8.5L5.49732 11.7473C5.90519 12.1552 6.57263 12.1344 6.95426 11.7018L13.75 4" stroke="currentColor"/></svg>`))
	toolDetailsPlayIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M8 14.5C11.5899 14.5 14.5 11.5899 14.5 8C14.5 4.41015 11.5899 1.5 8 1.5C4.41015 1.5 1.5 4.41015 1.5 8C1.5 11.5899 4.41015 14.5 8 14.5Z" stroke="currentColor"/><path d="M10.3329 7.91346C10.3996 7.95195 10.3996 8.04818 10.3329 8.08667L6.78304 10.1362C6.71638 10.1747 6.63304 10.1266 6.63304 10.0496L6.63304 5.95055C6.63304 5.87357 6.71638 5.82546 6.78304 5.86395L10.3329 7.91346Z" stroke="currentColor"/></svg>`))
)

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

// toolDetailsCode paints one installed code entry with the shared CodeBlock
// chrome. Its banner carries the recorded language, or the localized fallback,
// with the wrap toggle and the copy action; the owner bounds the viewport.
func toolDetailsCode(c *ui.Context, code toolDetailCode, st ToolCardStyle, ink toolDetailsPalette, codeOnly bool) {
	region := ui.Scroll(c.Key("code")).FillWidth().MinWidth(0).MaxHeight(240)
	if !codeOnly {
		region.Margin(8, 0, 0, 0)
	}
	language := code.Language
	if language == "" {
		language = i18n.T(i18n.KeyToolDetailCodeLabel)
	}
	// The code card keeps its banner at the top of its own viewport.
	position := ui.Local(region, "position", func() ui.ScrollState { return ui.ScrollState{} })
	region.TrackScroll(position).Children(func() {
		markdown.ToolCodeBlock(c, language, code.Text, toolDetailsCodeStyle(st, ink), position.Y)
	})
}

// toolDetailsMarkdown paints the installed `markdown` prose entry with the
// transcript's Markdown preset at the entry typography.
func toolDetailsMarkdown(c *ui.Context, text string, st ToolCardStyle, ink toolDetailsPalette) {
	column := ui.Column(c.Key("prose")).FillWidth().MinWidth(0).Margin(8, 0, 0, 0)
	column.Children(func() {
		markdown.View(c, text, transcriptMarkdownStyle(ink.secondary, ink.tertiary, ink.background, st.Accent, 13, darkSurface(st.FG)))
	})
}

// toolDetailsHeadingTitle paints the recorded title, as a file link when the
// entry carries a location and the owner supplied an opener. The installed
// heading gives the title a 12em basis whenever a status text shares the line.
func toolDetailsHeadingTitle(c *ui.Context, item toolDetailItem, st ToolCardStyle, ink toolDetailsPalette, hasStatusText bool) {
	removed := item.Change != nil && item.Change.Value == "removed"
	colour := ink.secondary
	if removed {
		colour = ink.tertiary
	}
	if item.Location != nil && ink.openFile != nil {
		label := item.Location.Path
		if item.Location.Line != nil {
			label += ":" + strconv.Itoa(*item.Location.Line)
		}
		button := ui.ButtonBase(c.Key("path")).Padding(0).Radius(0).Background(ui.Transparent).
			AlignItems(ui.Start).Justify(ui.Start).Grow(1).MinWidth(0).Label(item.Title)
		if hasStatusText {
			button.MinWidth(12 * 13)
		}
		if button.Clicked() {
			ink.openFile(label)
		}
		button.Children(func() {
			text := ui.Text(c, item.Title).Font(ink.font).FontSize(13).FixedLineHeight(20).TextColor(colour).MinWidth(0).PassThrough()
			if removed {
				text.Strikethrough()
			}
			if button.Hovered() {
				text.Underline()
			}
		})
		return
	}
	text := ui.Text(c, item.Title).Font(ink.font).FontSize(13).FixedLineHeight(20).TextColor(colour).MinWidth(0).Selectable().Grow(1)
	if removed {
		text.Strikethrough()
	}
	if hasStatusText {
		text.MinWidth(12 * 13)
	}
}

// toolDetailsStatusGlyph paints the installed todo marker: the change sign, the
// completed check, the in-progress play triangle or the pending square.
func toolDetailsStatusGlyph(c *ui.Context, item toolDetailItem, st ToolCardStyle, ink toolDetailsPalette) {
	colour := ink.secondary
	if item.Change != nil {
		switch item.Change.Value {
		case "added":
			colour = st.OK
		case "removed":
			colour = st.Err
		}
	}
	ui.Row(c.Key("status")).Width(14).Height(18).Shrink(0).AlignItems(ui.Center).Justify(ui.Start).Children(func() {
		switch {
		case item.Change != nil && item.Change.Value == "added":
			ui.Text(c, "+").Font(ink.font).FontSize(16).FixedLineHeight(18).TextColor(colour)
		case item.Change != nil && item.Change.Value == "removed":
			ui.Text(c, "−").Font(ink.font).FontSize(16).FixedLineHeight(18).TextColor(colour)
		case item.Status != nil && item.Status.Value == "completed":
			ui.Icon(c, toolDetailsCheckIcon).Size(14, 14).TextColor(colour).Role(ui.RoleNone)
		case item.Status != nil && item.Status.Value == "in_progress":
			ui.Icon(c, toolDetailsPlayIcon).Size(14, 14).TextColor(colour).Role(ui.RoleNone)
		default:
			ui.Box(c).Size(10, 10).Shrink(0).Border(1, ink.tertiary).Radius(2)
		}
	})
}

// toolDetailsStatusText paints the installed trailing status text: the previous
// status, an arrow, the localized status and the change suffix.
func toolDetailsStatusText(c *ui.Context, item toolDetailItem, ink toolDetailsPalette) {
	spans := make([]ui.Span, 0, 4)
	if item.PreviousStatus != nil {
		spans = append(spans, ui.Span{Text: *item.PreviousStatus, Color: ink.tertiary}, ui.Span{Text: " → "})
	}
	spans = append(spans, ui.Span{Text: item.Status.Label})
	if item.Change != nil && item.Change.Value == "updated" && item.PreviousStatus == nil {
		spans = append(spans, ui.Span{Text: " · " + item.Change.Label})
	}
	ui.RichText(c, spans...).Font(ink.font).FontSize(12).FixedLineHeight(20).TextColor(ink.caption).NoWrap().Shrink(0)
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
