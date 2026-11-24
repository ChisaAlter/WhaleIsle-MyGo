package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// titleCrawl drives Rows.module.css's hover crawl: a Session title that does not
// fit its 16px-leading cell scrolls while the pointer rests on the row. The cell
// width arrives from the previous painted frame, so a title that fits never
// moves and the crawl can start on the same frame the pointer lands.
type titleCrawl struct {
	cell       float32
	since      time.Time
	active     bool
	label      string
	fontFamily string
	fontSize   float32
	measured   float32
	layout     *ui.TextLayout
	layoutH    float32
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
// Crawling runs strictly inside Draw with paint-only p.After timers so hovering
// a long title never triggers full-tree rebuilds.
func sessionTitleCell(c *ui.Context, row ui.Element, id, label string, ink ui.Color) ui.Element {
	crawl := ui.Local(row, "title-crawl:"+id, func() titleCrawl { return titleCrawl{} })
	hovered := row.Hovered()
	if !hovered {
		crawl.stop()
	}
	fontSize := fs(14)
	fontFamily := ""
	if th := c.Theme(); th != nil {
		fontFamily = th.Font
	}
	content := float32(0)
	crawling := false
	reduceMotion := c.Preferences().ReduceMotion
	if hovered {
		if crawl.label != label || crawl.fontSize != fontSize || crawl.fontFamily != fontFamily || crawl.layout == nil {
			layout := ui.ShapeTextLine(label, ui.Font{Family: fontFamily, Size: fontSize}, 0)
			w, h := layout.Size()
			crawl.label = label
			crawl.fontSize = fontSize
			crawl.fontFamily = fontFamily
			crawl.layout = layout
			crawl.measured = w
			crawl.layoutH = h
		}
		content = crawl.measured + 10
		crawl.start(c.Now())
		if crawl.cell > 0 && content-crawl.cell > minTitleReveal {
			crawling = true
		}
	}
	lane := ui.Box(c).MinWidth(0).Grow(1).Clip()
	lane.ObserveLayout(func(r ui.Rect) { crawl.cell = r.W })
	lane.Children(func() {
		text := ui.Text(c, label).FontSize(fontSize).TextColor(ink).MinWidth(0).FillWidth().
			Padding(0, design.Sp(1.5), 0, design.Sp(1)).SingleLine()
		if crawling {
			// Keep identical height, pointer lane geometry, and accessible text
			// while painting the horizontally crawling text in Draw.
			text.Opacity(0)
		}
	})
	if crawling {
		layout := crawl.layout
		layoutH := crawl.layoutH
		padLeft := design.Sp(1)
		lane.Draw(func(p *ui.Painter, r ui.Rect) {
			if layout == nil {
				return
			}
			var offset float32
			var moving bool
			if reduceMotion {
				if overflow := content - crawl.cell; crawl.cell > 0 && overflow > minTitleReveal {
					offset = -overflow
				}
			} else {
				offset, moving = crawl.offset(p.Now(), content)
			}
			y := r.Y + (r.H-layoutH)/2
			p.Clip(r, 0, func() {
				p.TextLayout(layout, r.X+padLeft+offset, y, ink)
			})
			if moving {
				p.After(16 * time.Millisecond)
			}
		})
	}
	return lane
}
