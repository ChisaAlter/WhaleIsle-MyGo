package pet

import (
	"encoding/json"
	"math"
	"net/url"

	"github.com/egoist/mygo"
)

type petHitRect struct {
	X, Y, Width, Height float64
}

type petHitReport struct {
	Width, Height       float64
	Regions             []petHitRect
	Epoch               uint64
	MenuOpen, StageBusy bool
}

type petHitState struct {
	report              petHitReport
	ignoreSet, ignoring bool
}

// This accepts geometry of the packaged page only. It cannot move a native
// window, name a display, invoke input, or grant a general page/native bridge.
func decodePetHitReport(raw string) (petHitReport, bool) {
	var report petHitReport
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 8192 || u.Scheme != "whale-pet" || u.Host != "localhost" || u.User != nil || u.Fragment != "" || u.Path != "/interactive-regions" {
		return report, false
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) != 1 || len(query["data"]) != 1 || json.Unmarshal([]byte(query.Get("data")), &report) != nil {
		return petHitReport{}, false
	}
	finite := func(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
	if report.Epoch == 0 || len(report.Regions) > 16 || !finite(report.Width) || !finite(report.Height) || report.Width <= 0 || report.Height <= 0 || report.Width > 65536 || report.Height > 65536 {
		return petHitReport{}, false
	}
	clipped := make([]petHitRect, 0, min(len(report.Regions), 16))
	for _, rect := range report.Regions[:min(len(report.Regions), 16)] {
		if !finite(rect.X) || !finite(rect.Y) || !finite(rect.Width) || !finite(rect.Height) || math.Abs(rect.X) > 65536 || math.Abs(rect.Y) > 65536 || rect.Width <= 0 || rect.Height <= 0 || rect.Width > 65536 || rect.Height > 65536 {
			continue
		}
		x, y := math.Max(0, rect.X), math.Max(0, rect.Y)
		right, bottom := math.Min(report.Width, rect.X+rect.Width), math.Min(report.Height, rect.Y+rect.Height)
		if right > x && bottom > y {
			clipped = append(clipped, petHitRect{X: x, Y: y, Width: right - x, Height: bottom - y})
		}
	}
	report.Regions = clipped
	return report, true
}

// Content and cursor use the same native space (physical pixels on Windows,
// DIPs elsewhere). Page-to-native ratios handle scale and zoom once.
func petHitContains(report petHitReport, content mygo.Rectangle, cursor mygo.Point, dragging bool) bool {
	return petHitContainsScaled(report, content, cursor, dragging, 1)
}
func petHitContainsScaled(report petHitReport, content mygo.Rectangle, cursor mygo.Point, dragging bool, scale float64) bool {
	if content.Width <= 0 || content.Height <= 0 {
		return false
	}
	if dragging {
		return true // Keep pointer capture through release outside the character.
	}
	if report.Width <= 0 || report.Height <= 0 {
		return false
	}
	sx, sy := float64(content.Width)/report.Width, float64(content.Height)/report.Height
	pad := 8 * scale
	for _, rect := range report.Regions {
		left, top := float64(content.X)+rect.X*sx, float64(content.Y)+rect.Y*sy
		right, bottom := left+rect.Width*sx, top+rect.Height*sy
		if float64(cursor.X) >= left-pad && float64(cursor.X) <= right+pad && float64(cursor.Y) >= top-pad && float64(cursor.Y) <= bottom+pad {
			return true
		}
	}
	return false
}

func (p *Companion) resetHitRegions() {
	p.interaction.hit = petHitState{}
	if p.win != nil && !p.win.IsDestroyed() {
		p.win.SetIgnoreMouseEvents(true)
		p.interaction.hit.ignoreSet, p.interaction.hit.ignoring = true, true
	}
}

func (p *Companion) updateHitRegions(cursor mygo.Point) {
	w := p.win
	if w == nil || w.IsDestroyed() {
		return
	}
	s := &p.interaction
	interactive := w.IsVisible() && petHitContainsScaled(s.hit.report, petContentBounds(w), cursor, s.pressed || s.mode == "pickup", petWindowScale(w))
	ignore := !interactive
	if !s.hit.ignoreSet || s.hit.ignoring != ignore {
		w.SetIgnoreMouseEvents(ignore)
		s.hit.ignoreSet, s.hit.ignoring = true, ignore
	}
}

func (p *Companion) hitRegionAction(raw string) bool {
	report, ok := decodePetHitReport(raw)
	if !ok {
		return false
	}
	if p.win == nil || p.win.IsDestroyed() || !p.win.IsVisible() || !p.ready || !p.interaction.publishHitReport(report) {
		return true
	}
	p.updateHitRegions(petCursorPoint())
	return true
}

func (s *interactionState) publishHitReport(report petHitReport) bool {
	if report.Epoch != s.generation {
		return false
	}
	s.hit.report = report
	s.menuOpen, s.stageBusy = report.MenuOpen, report.StageBusy
	return true
}
