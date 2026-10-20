package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// titleCrawl drives Rows.module.css's hover crawl: a Session title that does not
// fit its 16px-leading cell scrolls while the pointer rests on the row. The cell
// width arrives from the previous painted frame, so a title that fits never
// moves and the crawl can start on the same frame the pointer lands.
type titleCrawl struct {
	cell   float32
	since  time.Time
	active bool
}

const (
	crawlSpeed = float32(40)      // DIP per second
	crawlHold  = 1200 * time.Millisecond // pause at each end
)

func (t *titleCrawl) start(now time.Time) {
	if !t.active {
		t.active, t.since = true, now
	}
}

func (t *titleCrawl) stop() {
	t.active, t.since = false, time.Time{}
}

// offset is the displacement for a content width at now, and whether the title
// is still moving (the caller keeps the frame loop alive while it is).
func (t *titleCrawl) offset(now time.Time, content float32) (float32, bool) {
	overflow := content - t.cell
	if !t.active || t.cell <= 0 || overflow <= 1 {
		return 0, false
	}
	run := time.Duration(float64(overflow) / float64(crawlSpeed) * float64(time.Second))
	if run <= 0 {
		return 0, false
	}
	total := 2*run + 2*crawlHold
	elapsed := now.Sub(t.since)
	if elapsed < 0 {
		elapsed = 0
	}
	phase := elapsed % total
	at := float32(0)
	switch {
	case phase < run:
		at = crawlSpeed * float32(phase.Seconds())
	case phase < run+crawlHold:
		at = overflow
	case phase < 2*run+crawlHold:
		at = overflow - crawlSpeed*float32((phase-run-crawlHold).Seconds())
	}
	return -at, true
}

// sessionTitleCell paints a Session title, crawling it under the pointer when it
// does not fit. At rest the title ellipsizes in place, as Rows.module.css does;
// the row clips the crawling text so it cannot paint over the trailing cell.
func sessionTitleCell(c *ui.Context, row ui.Element, id, label string, ink ui.Color) {
	crawl := ui.Local(row, "title-crawl:"+id, func() titleCrawl { return titleCrawl{} })
	hovered := row.Hovered()
	if !hovered {
		crawl.stop()
	}
	// The padding stays on the text so a resting row paints where the plain
	// single-line title did; the crawl compares the padded text width with the
	// title's allocated cell, measured from the previous painted frame.
	measured, _ := c.MeasureText(0, ui.Span{Text: label, Size: theme.InterfaceSize(14)})
	content := measured + 10
	offset := float32(0)
	moving := false
	if hovered {
		crawl.start(c.Now())
		if !c.Preferences().ReduceMotion {
			offset, moving = crawl.offset(c.Now(), content)
		}
	}
	text := ui.Text(c, label).FontSize(theme.InterfaceSize(14)).TextColor(ink).MinWidth(0).Grow(1).
		Padding(0, 6, 0, 4)
	if moving {
		// Content width, clipped by the row, so the displacement reveals the rest
		// of the title instead of an ellipsis.
		text.NoWrap()
	} else {
		text.SingleLine()
	}
	text.ObserveLayout(func(r ui.Rect) { crawl.cell = r.W })
	text.OffsetAfterLayout(func(r ui.Rect) ui.Point { return ui.Point{X: offset} })
	if moving {
		c.After(16 * time.Millisecond)
	}
}
