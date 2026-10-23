package frame

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

const inspectorPulseDuration = 700 * time.Millisecond

type inspectorPulseState struct {
	arrivals         map[string]time.Time
	initialized      bool
	revealKey        string
	revealGeneration uint64
	paintGeneration  uint64
	mounted          map[string]*inspectorRowPulse
}

func (s *inspectorPulseState) updateRows(rows []session.InspectorRecord, now time.Time) {
	s.paintGeneration++
	for key, pulse := range s.mounted {
		if pulse.lastPaint+1 < s.paintGeneration {
			delete(s.mounted, key)
		}
	}
	next := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		arrival, exists := s.arrivals[row.Key]
		if !exists && s.initialized {
			arrival = now
		}
		next[row.Key] = arrival
	}
	if len(rows) > 0 {
		s.initialized = true
	}
	s.arrivals = next
}

type inspectorPulseRecord struct {
	identity, location, kind, raw, summary string
	time                                   float64
}

func inspectorPulseValue(row session.InspectorRecord) inspectorPulseRecord {
	return inspectorPulseRecord{row.Identity, row.Location, row.Type, string(row.Raw), row.CollapsedSummary, row.Time}
}

type inspectorRowPulse struct {
	record     inspectorPulseRecord
	added      bool
	flash      bool
	generation uint64
	started    time.Time
	lastPaint  uint64
}

func (s *inspectorRowPulse) update(row session.InspectorRecord, added, flash, reduced bool, generation uint64, now time.Time) {
	value := inspectorPulseValue(row)
	changed, navigated := s.record != value, generation > 0 && s.generation != generation
	effect := changed || s.added != added || s.flash != flash || s.generation != generation
	s.record, s.added, s.flash, s.generation = value, added, flash, generation
	if !effect {
		return
	}
	s.started = time.Time{}
	if !reduced && (flash || navigated) && (changed || added || navigated) {
		s.started = now
	}
}

func (s *inspectorPulseState) paint(c *ui.Context, rowElement ui.Element, row session.InspectorRecord, flash bool, colors theme.Colors) {
	now := c.Now()
	arrival := s.arrivals[row.Key]
	added := !arrival.IsZero() && now.Sub(arrival) < inspectorPulseDuration
	generation := uint64(0)
	if s.revealKey == row.Key {
		generation = s.revealGeneration
	}
	if s.mounted == nil {
		s.mounted = map[string]*inspectorRowPulse{}
	}
	local := s.mounted[row.Key]
	if local == nil {
		local = &inspectorRowPulse{record: inspectorPulseValue(row)}
		s.mounted[row.Key] = local
	}
	local.lastPaint = s.paintGeneration
	local.update(row, added, flash, c.Preferences().ReduceMotion, generation, now)
	if local.started.IsZero() || now.Sub(local.started) >= inspectorPulseDuration || c.Preferences().ReduceMotion {
		return
	}
	// Draw, rather than rebuilding the hierarchy each animation frame. At expiry
	// the next build restores the row's ordinary hover/selection background.
	start := local.started
	color := chrome.Color(colors.BubbleHighlight)
	rowElement.Background(ui.Transparent).Draw(func(p *ui.Painter, r ui.Rect) {
		elapsed := p.Now().Sub(start)
		if elapsed >= inspectorPulseDuration {
			return
		}
		t := max(float32(0), float32(elapsed)/float32(inspectorPulseDuration))
		p.Fill(r, color.Alpha(1-t), 0)
		p.AnimationFrame()
	})
	c.After(inspectorPulseDuration - now.Sub(start))
}
