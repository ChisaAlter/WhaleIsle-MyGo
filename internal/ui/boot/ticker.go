package boot

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
func bootTicker(c *ui.Context, m Model, palette *scenePalette, openLog ...func()) {
	// The line multiplexes action feedback, the state's hint and the newest log
	// entry, and shows nothing when all three are empty (boot.js:44,57-63).
	line := ""
	lineColor := palette.log
	switch {
	case m.Status.State == harness.Failed && m.StatusText == "":
		line = bootText("可立即重启，或根据日志调整配置。", "Restart now, or adjust the configuration from the log.")
	case len(m.Logs) > 0:
		line = m.Logs[len(m.Logs)-1]
		// .ticker-line.important: a log line that explains a failure is the one
		// place the foot turns to the alert colour (boot.css:442-444).
		if importantBootLog.MatchString(line) {
			lineColor = palette.underAlert
		}
	}
	// The dot alone carries the failure colour; the hint line keeps the foot's
	// own ink (boot.css:432-434,442-444).
	dotColor := palette.log
	if m.Status.State == harness.Failed {
		dotColor = palette.underAlert
	}
	const size = 11.5
	ui.Row(c).Key("boot-ticker").Absolute().Left(0).Right(0).Bottom(0).Justify(ui.Center).
		Padding(0, tickerInsetX, tickerInsetBottom, tickerInsetX).Children(func() {
		button := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Key:     "boot-ticker-inner",
			Label:   bootText("全部日志", "All logs"),
			Content: func() {
				dot := ui.Box(c).Key("boot-ticker-dot").Size(5, 5).Radius(design.RadiusFull).Shrink(0)
				alpha := float32(1)
				if !c.Preferences().ReduceMotion {
					c.After(motionWait(c.Now()))
					alpha = 1 - .65*ui.Bounce(ui.EaseInOut)(loopPhase(c.Now(), tickerPulse))
				}
				dot.Background(dotColor.Alpha(alpha))
				ui.Text(c, line).Font(bootCodeFont).FontSize(size).FixedLineHeight(18).
					TextColor(lineColor).MinWidth(0).SingleLine().Ellipsis("…")
				// L 00: the count the drawer's title repeats (boot.js updateLogCount).
				ui.Text(c, fmt.Sprintf("L %02d", len(m.Logs))).Font(bootCodeFont).FontSize(size).
					LetterSpacing(size * .08).FixedLineHeight(18).TextColor(palette.log.Alpha(.6)).Shrink(0).NoWrap()
				ui.Text(c, bootText("全部日志", "All logs")).FontSize(size).TextColor(palette.log.Alpha(.55)).Shrink(0).NoWrap()
			},
		}).Background(ui.Transparent).Padding(design.Sp(1), design.Sp(3.5)).MaxWidth(tickerWidth).MaxWidthPercent(100).
			Gap(design.Sp(2.5)).AlignItems(ui.Center)
		if button.Clicked() && len(openLog) > 0 && openLog[0] != nil {
			openLog[0]()
		}
	})
}
