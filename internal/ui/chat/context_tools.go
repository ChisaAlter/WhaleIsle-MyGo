package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var contextBrowseIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M4.9375 5.90295H11.0625M4.9375 9.02991H8.27841" stroke="currentColor"/><path d="M12.5 1.32617C13.3039 1.32617 14 1.95171 14 2.77637V13.2246C13.9996 14.0489 13.3036 14.6738 12.5 14.6738H3.5C2.69637 14.6738 2.00042 14.0489 2 13.2246V2.77637C2 1.95171 2.69613 1.32617 3.5 1.32617H12.5ZM3.5 2.32617C3.1993 2.32617 3 2.55186 3 2.77637V13.2246C3.00044 13.4489 3.19963 13.6738 3.5 13.6738H12.5C12.8004 13.6738 12.9996 13.4489 13 13.2246V2.77637C13 2.55186 12.8007 2.32617 12.5 2.32617H3.5Z" fill="currentColor"/></svg>`))
var contextInjectionIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M5 2.5H3.5C2.94772 2.5 2.5 2.94772 2.5 3.5V13.5C2.5 14.0523 2.94772 14.5 3.5 14.5H12.5C13.0523 14.5 13.5 14.0523 13.5 13.5V3.5C13.5 2.94772 13.0523 2.5 12.5 2.5H11M8 .5V7.5M5.5 5L8 7.5L10.5 5M5.5 11H10.5" stroke="currentColor"/></svg>`))
var contextChevronIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor"/></svg>`))

type contextPresence struct {
	open  bool
	until time.Time
}

// paintToolContext specializes only an all-tool-change context. A mixed block
// remains an opaque context disclosure, while either variant stays in Chat.
func paintToolContext(c *ui.Context, msg session.Message, key string, open *bool, muted ui.Color) bool {
	_, content := contextMessageParts(msg)
	if len(content) == 0 {
		return false
	}
	var added, removed []string
	for _, block := range content {
		switch block.Type {
		case "tool-addition":
			added = append(added, block.ToolName)
		case "tool-removal":
			removed = append(removed, block.ToolName)
		default:
			return false
		}
	}
	title := i18n.T(i18n.KeyChatToolsUpdated)
	summary := ""
	single := len(content) == 1
	if single {
		label := i18n.KeyChatToolAdded
		if content[0].Type == "tool-removal" {
			label = i18n.KeyChatToolRemoved
		}
		title = fmt.Sprintf(i18n.T(label), content[0].ToolName)
	} else if len(added) > 0 && len(removed) > 0 {
		summary = fmt.Sprintf(i18n.T(i18n.KeyChatToolsChanged), len(added), len(removed))
	} else if len(added) > 0 {
		summary = fmt.Sprintf(i18n.T(i18n.KeyChatToolsAddedCount), len(added))
	} else {
		summary = fmt.Sprintf(i18n.T(i18n.KeyChatToolsRemovedCount), len(removed))
	}
	size := secondaryChatSize()
	delta := theme.CurrentPreferences().Chat.Size - 14
	root := ui.Column(c.Key("tool-context:" + key)).FillWidth().MinWidth(0)
	presence := ui.Local(root, "presence", func() contextPresence { return contextPresence{} })
	root.Children(func() {
		name := strings.TrimSpace(title + " " + summary)
		var row ui.Element
		if single {
			row = ui.Row(c).Label(name)
		} else {
			row = ui.Button(c, "").Label(name).Role(ui.RoleDisclosure).Expanded(*open).
				Padding(0).BorderWidth(0).Background(ui.Transparent).Radius(4).FocusRing(false)
			if row.Clicked() {
				*open = !*open
				c.Invalidate()
			}
		}
		row.FillWidth().MinWidth(0).Height(24 + delta).AlignItems(ui.Center).Justify(ui.Start)
		hovered := !single && row.Hovered()
		if hovered {
			row.Background(c.Theme().SurfaceHover.Alpha(.5))
		}
		row.Children(func() {
			icon := contextBrowseIcon
			if !single && (hovered || *open) {
				icon = contextChevronIcon
			}
			ui.Box(c).Size(16+delta, 16+delta).Margin(0, 6, 0, 0).Shrink(0).Center().Children(func() {
				glyph := ui.Icon(c, icon).Size(14+delta, 14+delta).TextColor(muted).Role(ui.RoleNone)
				if !single && *open {
					glyph.Rotate(180)
				}
			})
			ui.Text(c, title).FontSize(size).FixedLineHeight(24 + delta).TextColor(muted).SingleLine().Shrink(1).MinWidth(0)
			if summary != "" {
				ui.Box(c).Size(2, 2).Radius(1).Background(muted.Alpha(.5)).Margin(0, 8, 0, 8).Shrink(0).Role(ui.RoleNone)
				ui.Text(c, summary).FontSize(size).FixedLineHeight(24 + delta).TextColor(muted).SingleLine().Grow(1).MinWidth(0)
			}
		})
		row.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if row.FocusVisible() {
				p.Stroke(ui.Rect{X: r.X + 1, Y: r.Y + 2, W: r.W - 2, H: r.H - 4}, c.Theme().Focus, 4, 2)
			}
		})
		shown := !single && *open
		if presence.open != shown {
			presence.open = shown
			if !shown {
				presence.until = c.Now().Add(200 * time.Millisecond)
			}
		}
		target := float32(0)
		if shown {
			target = 1
		}
		opacity := root.Animate("context-opacity", target, 100*time.Millisecond)
		if !shown && (c.Preferences().ReduceMotion || !c.Now().Before(presence.until)) {
			return
		}
		if !shown {
			c.Invalidate()
		}
		body := ui.Column(c).FillWidth().MinWidth(0).Margin(4, 0, 4, 22+delta).Opacity(opacity)
		if !shown {
			body.PassThrough()
		}
		body.Children(func() {
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(141).Padding(10, 16, 12, 12).
				Background(c.Theme().Surface).Radius(12).Children(func() {
				ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
					ui.Column(c).Gap(0).Children(func() {
						if len(added) > 0 {
							ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatToolsAdded), strings.Join(added, ", "))).
								Font(theme.RenderPreferences().Code.Family).FontSize(11).FixedLineHeight(16).TextColor(muted).NoWrap()
						}
						if len(removed) > 0 {
							ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatToolsRemoved), strings.Join(removed, ", "))).
								Font(theme.RenderPreferences().Code.Family).FontSize(11).FixedLineHeight(16).TextColor(muted).NoWrap()
						}
					})
				})
			})
		})
	})
	return true
}
