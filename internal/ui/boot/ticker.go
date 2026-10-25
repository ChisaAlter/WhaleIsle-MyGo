package boot

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The reference page's foot is .ticker (boot.css:432-501): one line of the
// newest log entry at the bottom of the page, with the count of lines collected,
// over the deep water where it keeps its contrast. It replaces the ticker the
// page used to cram into the centred brand block.

const (
	// tickerPulse is the dot's 1.6s pulse to 35% (boot.css:486-489,505-507).
	tickerPulse = 1600 * time.Millisecond
	// tickerInsetX and tickerInsetBottom are .ticker's 0 56px 24px padding.
	tickerInsetX      = float32(56)
	tickerInsetBottom = float32(24)
	// tickerWidth keeps the line readable on a wide window: the reference caps
	// .ticker-inner at 72ch.
	tickerWidth = float32(560)
)

// bootTicker paints the newest log line and the count of lines collected.
func bootTicker(c *ui.Context, logs []string, under ui.Color) {
	line := i18n.T(KeyWaiting)
	if len(logs) > 0 {
		line = logs[len(logs)-1]
	}
	font := theme.RenderPreferences().Code.Family
	size := theme.InterfaceSize(11.5)
	ui.Row(c).Key("boot-ticker").FillWidth().Shrink(0).Justify(ui.Center).
		Padding(0, tickerInsetX, tickerInsetBottom, tickerInsetX).Children(func() {
		ui.Row(c).Key("boot-ticker-inner").Width(tickerWidth).MaxWidthPercent(100).
			Gap(10).AlignItems(ui.Center).Children(func() {
			dot := ui.Box(c).Key("boot-ticker-dot").Size(5, 5).Radius(2.5).Shrink(0)
			alpha := float32(1)
			if !c.Preferences().ReduceMotion {
				alpha = 1 - .65*dot.Loop("boot-ticker-pulse", tickerPulse, ui.Bounce(ui.EaseInOut))
			}
			dot.Background(under.Alpha(alpha))
			ui.Text(c, line).Font(font).FontSize(size).FixedLineHeight(theme.InterfaceSize(18)).
				TextColor(under).MinWidth(0).Grow(1).SingleLine().Ellipsis("…")
			// L 00: the count the drawer's title repeats (boot.js updateLogCount).
			ui.Text(c, fmt.Sprintf("L %02d", len(logs))).Font(font).FontSize(size).
				FixedLineHeight(theme.InterfaceSize(18)).TextColor(under.Alpha(.6)).Shrink(0).NoWrap()
		})
	})
}
