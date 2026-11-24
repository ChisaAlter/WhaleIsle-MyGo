package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// ExitCodeBadge is a neutral Tag with a 4px error dot. Its text stays neutral
// while the surrounding row transitions between tertiary and secondary ink.
func terminalExitBadge(c *ui.Context, text string, _ ToolCardStyle) {
	t := design.For(c)
	ui.Row(c).Shrink(0).Gap(design.Sp(1.25)).AlignItems(ui.Center).
		Padding(design.Sp(0.25), design.Sp(2), design.Sp(0.25), design.Sp(2)).
		Margin(0, 0, 0, design.Sp(2)).
		Background(t.Muted).Radius(design.RadiusFull).Label(text).Children(func() {
		ui.Box(c).Size(4, 4).Radius(design.RadiusFull).Background(t.Destructive).Shrink(0)
		ui.Text(c, text).Font(theme.RenderPreferences().Chat.Family).FontSize(11).FontWeight(design.WeightMedium).FixedLineHeight(17).NoWrap().TextColor(t.Foreground)
	})
}
