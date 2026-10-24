package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// ExitCodeBadge is a neutral Tag with a 4px error dot. Its text stays neutral
// while the surrounding row transitions between tertiary and secondary ink.
func terminalExitBadge(c *ui.Context, text string, st ToolCardStyle) {
	background, foreground := ui.RGB(245, 246, 247), st.Muted
	if darkSurface(st.Muted) {
		background = ui.RGB(53, 54, 56)
	}
	if st.Colors.Secondary != "" {
		foreground = chrome.Color(st.Colors.Secondary)
	}
	ui.Row(c).Shrink(0).Gap(5).AlignItems(ui.Center).Padding(1, 8, 1, 8).Margin(0, 0, 0, 8).
		Background(background).Radius(999).Label(text).Children(func() {
		ui.Box(c).Size(4, 4).Radius(2).Background(st.Err).Shrink(0)
		ui.Text(c, text).Font(theme.RenderPreferences().Chat.Family).FontSize(11).FontWeight(500).FixedLineHeight(17).NoWrap().TextColor(foreground)
	})
}
