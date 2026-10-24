package chat

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func toolHeaderMetrics() (size, height, leading, glyph float32) {
	body := theme.RenderPreferences().Chat.Size
	delta := body - 14
	size = body - 1
	if body > 14 {
		size = body - 2
	}
	return size, 24 + delta, 16 + delta, 14 + delta
}

func toolHeaderColor(st ToolCardStyle, hover bool) ui.Color {
	if hover {
		if st.Colors.Secondary != "" {
			return chrome.Color(st.Colors.Secondary)
		}
		return st.Muted.Mix(st.FG, .35)
	}
	return st.Muted
}

// Tool rows keep their business icon at rest. Lifecycle cues affect the summary
// and accessible description; they never replace that icon with a status dot.
func toolHeader(c *ui.Context, info ToolCardInfo, st ToolCardStyle, key string, open *bool, expandable bool) ui.Element {
	size, height, leadingSize, glyph := toolHeaderMetrics()
	var row ui.Element
	if expandable {
		row = ui.ButtonBase(c.Key("tool-header:" + key)).Role(ui.RoleDisclosure).Expanded(*open).Cursor(ui.CursorPointer)
		if row.Clicked() {
			*open = !*open
			row.Expanded(*open)
			c.Invalidate()
		}
	} else {
		row = ui.Row(c.Key("tool-header:" + key))
	}
	row.FillWidth().MinWidth(0).Height(height).Shrink(0).Gap(0).Justify(ui.Start).AlignItems(ui.Center).Clip()
	title, summary := toolRowTitle(info.Name), toolRowSummary(info)
	row.Label(strings.TrimSpace(title + " " + summary)).Description(toolRowStatusLabel(info.Status))
	hovered := row.Hovered()
	ink := toolHeaderColor(st, false).Mix(toolHeaderColor(st, true), row.AnimateWith("tool-hover", boolFloat(hovered), 100*time.Millisecond, contextColorEase))
	caption := st.Muted.Alpha(.6)
	if st.Colors.Caption != "" {
		caption = chrome.Color(st.Colors.Caption)
	}
	summaryInk := ink
	switch toolRowStatus(info.Status) {
	case statusError:
		summaryInk = st.Err
	case statusWarning:
		summaryInk = ui.Hex("#dd8629")
	}
	running := toolRowStatus(info.Status) == statusRunning
	font := theme.RenderPreferences().Chat.Family
	row.Children(func() {
		ui.Box(c).Size(leadingSize, leadingSize).Shrink(0).Margin(0, 6, 0, 0).Children(func() {
			icon := toolKindIconForName(info.Name)
			if expandable && *open {
				icon = toolIconChevronUp
			}
			fade := float32(0)
			if expandable && !*open {
				fade = row.AnimateWith("tool-icon-hover", boolFloat(hovered), 100*time.Millisecond, func(t float32) float32 { return contextBezier(t, .4, .2) })
			}
			ui.Icon(c, icon).Absolute().Size(glyph, glyph).Left((leadingSize - glyph) / 2).Top((leadingSize - glyph) / 2).TextColor(ink).Opacity(1 - fade).PassThrough()
			if expandable && !*open {
				ui.Icon(c, toolIconChevronDown).Absolute().Size(glyph, glyph).Left((leadingSize - glyph) / 2).Top((leadingSize - glyph) / 2).TextColor(ink).Opacity(fade).PassThrough()
			}
		})
		weight := 400
		if strings.HasPrefix(info.Name, "cordis_") {
			weight = 500
		}
		if running {
			widgets.TextShimmer(c, widgets.TextShimmerOptions{Text: title, Active: true, Font: font, Size: size, Weight: weight, Color: ink}).Shrink(0)
		} else {
			ui.Text(c, title).Font(font).FontSize(size).FontWeight(weight).FixedLineHeight(height).TextColor(ink).NoWrap().Shrink(0)
		}
		if summary != "" {
			ui.Box(c).Size(2, 2).Radius(1).Background(caption).Shrink(0).Margin(0, 8, 0, 8)
			if running {
				widgets.TextShimmer(c, widgets.TextShimmerOptions{Text: summary, Active: true, Font: font, Size: size, Color: summaryInk}).Grow(1).MinWidth(0)
			} else {
				ui.Text(c, summary).Font(font).FontSize(size).FixedLineHeight(height).TextColor(summaryInk).SingleLine().Grow(1).MinWidth(0)
			}
		}
	})
	return row
}

func boolFloat(value bool) float32 {
	if value {
		return 1
	}
	return 0
}
