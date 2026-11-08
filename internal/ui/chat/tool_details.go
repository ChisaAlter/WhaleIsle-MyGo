package chat

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// toolDetailsPalette is the ink the installed card resolves per palette: the
// interface and code families, the item text, the caption line, the muted
// subtitle and the code surface.
type toolDetailsPalette struct {
	font, codeFont               string
	secondary, tertiary, caption ui.Color
	background                   ui.Color
	// openFile opens a recorded file location; nil keeps the heading inert.
	openFile func(string)
}

// Goal details are a subset of the installed ToolDetails entity renderer. Values
// come from the immutable logged result, independently of the live Goals store.
func toolDetailsCard(c *ui.Context, model *toolDetailsModel, st ToolCardStyle, key string, hasInspect bool, openFile func(string)) {
	background, borderInk, secondary, caption := st.Field, st.Border.Alpha(.4), st.Muted, st.Muted.Alpha(.6)
	tertiary := st.Muted
	if st.Colors.CodeBlock != "" {
		background, borderInk = chrome.Color(st.Colors.CodeBlock), chrome.Color(st.Colors.BorderL1)
		secondary, caption = chrome.Color(st.Colors.Secondary), chrome.Color(st.Colors.Caption)
		tertiary = chrome.Color(st.Colors.Muted)
	}
	ink := toolDetailsPalette{font: theme.RenderPreferences().Interface.Family, codeFont: theme.RenderPreferences().Code.Family,
		secondary: secondary, tertiary: tertiary, caption: caption, background: background, openFile: openFile}
	border := toolCSSHalfBorder(c)
	// The installed 88px reserve clears the row's Inspect button. It applies to
	// the caption when a caption exists, and to the items only when the card has
	// no caption to carry it (`:not([data-caption])`).
	itemEnd := float32(14)
	if hasInspect && model.Caption == "" {
		itemEnd = 88
	}
	font := theme.RenderPreferences().Interface.Family
	card := ui.Column(c.Key("tool-details:"+key)).FillWidth().MinWidth(0).Margin(4, 0, 4, 4).
		Border(border, borderInk).Radius(16).Background(background).Clip().Selectable()
	card.Children(func() {
		if model.Caption != "" {
			end := float32(14)
			if hasInspect {
				end = 88
			}
			ui.Column(c.Key("caption")).FillWidth().MinWidth(0).Padding(10, end, 6, 14).Children(func() {
				ui.Text(c, model.Caption).Font(font).FontSize(12).FixedLineHeight(20).TextColor(caption).
					FillWidth().MinWidth(0).Selectable()
			})
		}
		if len(model.Items) == 0 {
			ui.Column(c.Key("empty")).FillWidth().MinWidth(0).Padding(10, itemEnd, 10, 14).Children(func() {
				toolDetailsText(c, model.Empty, font, secondary)
			})
		} else {
			ui.Scroll(c.Key("list")).FillWidth().MinWidth(0).MaxHeight(320).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Children(func() {
					for index, item := range model.Items {
						if index > 0 {
							ui.Box(c).FillWidth().Height(border).Shrink(0).Background(st.Border)
						}
						toolDetailsItem(c, item, st, ink, itemEnd, false, index)
					}
				})
			})
		}
		// The installed unchanged container follows the list and paints even
		// when the list itself is empty.
		if model.Unchanged != nil {
			toolDetailsUnchangedView(c, *model.Unchanged, st, ink)
		}
	})
}

// toolDetailsItem paints one installed DetailItem: the heading line with its
// status badge, the recorded subtitle and description, the named fields, the
// line list, the code entry and the nested groups, in the installed order. A
// code-only item drops the item padding, as the installed
// `:has(> code:only-child)` rule does.
func toolDetailsItem(c *ui.Context, item toolDetailItem, st ToolCardStyle, ink toolDetailsPalette, end float32, nested bool, index int) {
	// The badge keeps the heading row present even for an empty recorded title,
	// as the installed `title !== undefined` check does.
	hasHeading := item.Title != "" || item.Badge != nil
	codeOnly := item.Code != nil && !hasHeading && item.Subtitle == nil && item.Description == nil &&
		item.Markdown == nil && len(item.Fields) == 0 && len(item.Lines) == 0 && len(item.Groups) == 0
	body := ui.Column(c.Key(index)).FillWidth().MinWidth(0)
	switch {
	case nested && codeOnly:
		// The installed `.group .item:has(> .code:only-child)` inset.
		body.Padding(4, 0, 0, 20)
	case nested:
		body.Padding(9, 0, 2, 20)
	case codeOnly:
		body.Padding(0)
	default:
		body.Padding(10, end, 10, 14)
	}
	body.Children(func() {
		if hasHeading {
			// The installed heading keeps the status glyph, the title (or its file
			// link), the badge and the trailing status text on one wrapping line.
			hasStatusText := item.Status != nil
			ui.Row(c.Key("heading")).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
				if item.Status != nil {
					toolDetailsStatusGlyph(c, item, st, ink)
				}
				toolDetailsHeadingTitle(c, item, st, ink, hasStatusText)
				if item.Badge != nil {
					toolDetailsBadgeView(c, *item.Badge, st, ink.caption, ink.font, toolDetailsBadgeKey(index))
				}
				if item.Status != nil {
					toolDetailsStatusText(c, item, ink)
				}
			})
		}
		if item.Subtitle != nil {
			ui.Text(c, *item.Subtitle).Font(ink.font).FontSize(12).FixedLineHeight(20).TextColor(ink.tertiary).
				Margin(3, 0, 0, 0).FillWidth().MinWidth(0).Selectable()
		}
		if item.Description != nil {
			ui.Text(c, *item.Description).Font(ink.font).FontSize(13).FixedLineHeight(20).TextColor(ink.caption).
				Margin(6, 0, 6, 0).FillWidth().MinWidth(0).Selectable()
		}
		if len(item.Fields) > 0 {
			fields := ui.Column(c).FillWidth().MinWidth(0).Gap(4)
			// The installed `.heading + .fields` rule only applies when the fields
			// directly follow the heading line.
			if hasHeading && item.Subtitle == nil && item.Description == nil {
				fields.Margin(6, 0, 0, 0)
			}
			fields.Children(func() {
				for _, field := range item.Fields {
					ui.Grid(c).ColumnTracks(ui.Fixed(5.5*13), ui.Fr(1)).Gap(12).FillWidth().MinWidth(0).AlignItems(ui.Start).Children(func() {
						toolDetailsText(c, field.Label, ink.font, ink.caption)
						toolDetailsText(c, field.Value, ink.font, ink.secondary)
					})
				}
			})
		}
		if len(item.Lines) > 0 {
			toolDetailsLines(c, item.Lines, ink.font, ink.secondary)
		}
		if item.Markdown != nil {
			toolDetailsMarkdown(c, *item.Markdown, st, ink)
		}
		if item.Code != nil {
			toolDetailsCode(c, *item.Code, st, ink, codeOnly)
		}
		for groupIndex, group := range item.Groups {
			toolDetailsGroupView(c, group, st, ink, groupIndex)
		}
	})
}

func toolDetailsText(c *ui.Context, text, font string, ink ui.Color) ui.Element {
	return ui.Text(c, text).Font(font).FontSize(13).FixedLineHeight(20).TextColor(ink).MinWidth(0).Selectable()
}

func toolCSSHalfBorder(c *ui.Context) float32 {
	return float32(max(1, math.Floor(.5*float64(c.DeviceScale()))) / float64(c.DeviceScale()))
}

// Inspect stays in the expanded body. It is keyboard reachable at rest; hovering
// anywhere on the owner row or giving it visible focus fades its paint in.
func toolInspectButton(c *ui.Context, st ToolCardStyle, key string, inDetails, ownerHovered bool, inspect func()) {
	base, secondary, stroke, hover := ui.Transparent, st.Muted, st.Border, st.Field
	if st.Colors.BG != "" {
		base, secondary, stroke = chrome.Color(st.Colors.BG), chrome.Color(st.Colors.Secondary), chrome.Color(st.Colors.BorderL3)
		hover = ui.Hex("#f1f3f5")
		if darkSurface(st.FG) {
			hover = ui.Hex("#353638")
		}
	}
	button := ui.ButtonBase(c.Key("tool-inspect:"+key)).AlignSelf(ui.Start).Padding(2, 8, 2, 8).Gap(4).
		Border(toolCSSHalfBorder(c), stroke).Radius(999).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		Label(i18n.T(i18n.KeyChatToolInspect)).FocusRing(false)
	if inDetails {
		button.Absolute().Top(12).Right(12)
	} else {
		button.Margin(4, 0, 2, 4)
	}
	focused := button.FocusVisible()
	ink := secondary
	if button.Hovered() {
		base, ink = hover, st.FG
	}
	button.Background(base).TextColor(ink)
	visible := boolFloat(ownerHovered || focused)
	if !c.Preferences().ReduceMotion {
		visible = button.AnimateWith("inspect-visible", visible, 100*time.Millisecond, func(t float32) float32 { return contextBezier(t, .4, .2) })
	}
	button.Opacity(visible)
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if focused {
			p.Stroke(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}, st.Accent, 999, 2)
		}
	})
	if button.Clicked() {
		inspect()
	}
	button.Children(func() {
		ui.Icon(c, toolIconInspect).Size(12, 12).PassThrough()
		ui.Text(c, i18n.T(i18n.KeyChatToolInspect)).Font(theme.RenderPreferences().Chat.Family).FontSize(11).FixedLineHeight(16).TextColor(ink).PassThrough()
	})
}

var toolIconInspect = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M4.67398 4.25061L1.36094 7.86484C1.29085 7.9413 1.29085 8.05866 1.36094 8.13513L4.67398 11.7494" stroke="currentColor"/><path d="M11.3262 4.25061L14.6392 7.86484C14.7093 7.9413 14.7093 8.05866 14.6392 8.13513L11.3262 11.7494" stroke="currentColor"/><path d="M9.56222 3.62573L6.43774 12.3743" stroke="currentColor"/></svg>`))
