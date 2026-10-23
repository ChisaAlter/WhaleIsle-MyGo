package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var contextBrowseIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M4.9375 5.90295H11.0625M4.9375 9.02991H8.27841" stroke="currentColor"/><path d="M12.5 1.32617C13.3039 1.32617 14 1.95171 14 2.77637V13.2246C13.9996 14.0489 13.3036 14.6738 12.5 14.6738H3.5C2.69637 14.6738 2.00042 14.0489 2 13.2246V2.77637C2 1.95171 2.69613 1.32617 3.5 1.32617H12.5ZM3.5 2.32617C3.1993 2.32617 3 2.55186 3 2.77637V13.2246C3.00044 13.4489 3.19963 13.6738 3.5 13.6738H12.5C12.8004 13.6738 12.9996 13.4489 13 13.2246V2.77637C13 2.55186 12.8007 2.32617 12.5 2.32617H3.5Z" fill="currentColor"/></svg>`))
var contextInjectionIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M5 2.5H3.5C2.94772 2.5 2.5 2.94772 2.5 3.5V13.5C2.5 14.0523 2.94772 14.5 3.5 14.5H12.5C13.0523 14.5 13.5 14.0523 13.5 13.5V3.5C13.5 2.94772 13.0523 2.5 12.5 2.5H11M8 .5V7.5M5.5 5L8 7.5L10.5 5M5.5 11H10.5" stroke="currentColor"/></svg>`))
var contextChevronIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor"/></svg>`))

// paintToolContext specializes only an all-tool-change context. A mixed block
// remains an opaque context disclosure, while either variant stays in Chat.
func paintToolContext(c *ui.Context, msg session.Message, root ui.Element, open *bool, palette theme.Colors) bool {
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
	style := contextBodyPalette(palette)
	paintContextDisclosure(c, root, open, contextDisclosure{Icon: contextBrowseIcon, Title: title, Summary: summary, Single: single,
		Body: func() {
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				ui.Column(c).Gap(0).Selectable().Children(func() {
					if len(added) > 0 {
						ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatToolsAdded), strings.Join(added, ", "))).
							Font(style.Font).FontSize(style.Size).FixedLineHeight(style.LineHeight).TextColor(style.Tertiary).NoWrap()
					}
					if len(removed) > 0 {
						ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatToolsRemoved), strings.Join(removed, ", "))).
							Font(style.Font).FontSize(style.Size).FixedLineHeight(style.LineHeight).TextColor(style.Tertiary).NoWrap()
					}
				})
			})
		}}, palette)
	return true
}
