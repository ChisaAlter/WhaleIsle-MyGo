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

const crawlSpeed = float32(30) // installed Rows: 0.03 DIP per millisecond
const minTitleReveal = float32(8)

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
	if !t.active || t.cell <= 0 || overflow <= minTitleReveal {
		return 0, false
	}
	elapsed := max(time.Duration(0), now.Sub(t.since))
	at := min(overflow, crawlSpeed*float32(elapsed.Seconds()))
	return -at, at < overflow
}

// titleFadeEdges mirrors Rows' data-scrolled/data-clipped mask states. Resting
// ellipses and barely clipped titles stay at full opacity.
func titleFadeEdges(hovered bool, offset, content, cell float32) (left, right float32) {
	if !hovered || cell <= 0 || content-cell <= minTitleReveal {
		return
	}
	if offset < 0 {
		left = 8
	}
	if -offset < content-cell {
		right = 8
	}
	return
}

// sessionTitleCell paints a Session title, crawling it under the pointer when it
// does not fit. The lane keeps its original pointer and layout bounds while
// only its text moves, clipped away from the leading status and trailing actions.
func sessionTitleCell(c *ui.Context, row ui.Element, id, label string, ink ui.Color) ui.Element {
	crawl := ui.Local(row, "title-crawl:"+id, func() titleCrawl { return titleCrawl{} })
	hovered := row.Hovered()
	if !hovered {
		crawl.stop()
	}
	// The padding stays on the text so a resting row paints where the plain
	// single-line title did; the crawl compares the padded text width with the
	// title's allocated cell, measured from the previous painted frame.
	content := float32(0)
	offset := float32(0)
	moving := false
	if hovered {
		measured, _ := c.MeasureText(0, ui.Span{Text: label, Size: theme.InterfaceSize(14)})
		content = measured + 10
		crawl.start(c.Now())
		if c.Preferences().ReduceMotion {
			if overflow := content - crawl.cell; crawl.cell > 0 && overflow > minTitleReveal {
				offset = -overflow
			}
		} else {
			offset, moving = crawl.offset(c.Now(), content)
		}
	}
	lane := ui.Box(c).MinWidth(0).Grow(1).Clip()
	left, right := titleFadeEdges(hovered, offset, content, crawl.cell)
	lane.FadeHorizontal(left, right)
	lane.ObserveLayout(func(r ui.Rect) { crawl.cell = r.W })
	lane.Children(func() {
		text := ui.Text(c, label).FontSize(theme.InterfaceSize(14)).TextColor(ink).MinWidth(0).FillWidth().
			Padding(0, 6, 0, 4)
		if moving || offset != 0 {
			// Keep the far edge exposed after settling, including reduced motion.
			text.NoWrap().Width(content)
		} else {
			text.SingleLine()
		}
		text.OffsetAfterLayout(func(r ui.Rect) ui.Point { return ui.Point{X: offset} })
	})
	if moving {
		c.After(16 * time.Millisecond)
	}
	return lane
}
