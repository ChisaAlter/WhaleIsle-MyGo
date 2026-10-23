package frame

import (
	"math"
	"slices"

	"whaleisle/internal/store/session"
)

// Geometry includes the reference's 30 DIP table header. The native List owns
// only the area below that header; both scroll offsets are otherwise identical.
type inspectorTableLayout struct {
	rows                         []session.InspectorRecord
	branches                     map[string]bool
	indexes                      map[string]int
	minimumExtent, bottomPadding float32
	scrollOffset                 *float32
	preservedCollapse            bool
}

type inspectorRowAnchor struct {
	key string
	top float32
}

type inspectorScrollState struct {
	layout         *inspectorTableLayout
	anchor         *inspectorRowAnchor
	mode           string
	reset          bool
	started        bool
	viewportHeight float32
	lastOffset     float32
	viewportTop    float32
	requestedFrom  *float32
}

func newInspectorTableLayout(rows []session.InspectorRecord, branches map[string]bool, previous *inspectorTableLayout, offset, height float32, anchor *inspectorRowAnchor) *inspectorTableLayout {
	l := &inspectorTableLayout{rows: rows, branches: branches, indexes: make(map[string]int, len(rows))}
	for i, row := range rows {
		l.indexes[row.Key] = i
	}
	if previous != nil && len(rows) < len(previous.rows) {
		for _, row := range rows {
			if i, ok := previous.indexes[row.Key]; ok && row.Disclosure == "closed" && previous.rows[i].Disclosure == "open" {
				l.preservedCollapse = true
				break
			}
		}
	}
	if anchor != nil {
		if i, ok := l.indexes[anchor.key]; ok {
			y := max(float32(0), float32(i+1)*inspectorRowHeight-anchor.top)
			l.scrollOffset = &y
		}
	} else if l.preservedCollapse {
		y := l.anchorOffset(previous, offset)
		l.scrollOffset = &y
	}
	natural := float32(len(rows)+1) * inspectorRowHeight
	minimum := float32(0)
	if previous != nil {
		minimum = previous.minimumExtent
	}
	if l.scrollOffset != nil {
		minimum = max(*l.scrollOffset+height, minimum+*l.scrollOffset-offset)
	}
	if minimum > natural {
		l.minimumExtent = minimum
	}
	l.bottomPadding = max(float32(0), l.minimumExtent-natural)
	return l
}

func (l *inspectorTableLayout) anchorOffset(previous *inspectorTableLayout, offset float32) float32 {
	first := max(0, min(len(previous.rows)-1, int(math.Floor(float64(offset/inspectorRowHeight)))))
	at := min(len(previous.rows)-1, first+len(inspectorStickyIndexes(previous.rows, previous.branches, offset)))
	row := previous.rows[at]
	if next, ok := l.indexes[row.Key]; ok {
		return max(float32(0), offset+float32(next-at)*inspectorRowHeight)
	}
	var parents []string
	for parent := row.Parent; parent != ""; {
		parents = append(parents, parent)
		i, ok := previous.indexes[parent]
		if !ok {
			break
		}
		parent = previous.rows[i].Parent
	}
	// A branch is itself included in the reference sticky ancestry.
	if previous.branches[row.Key] {
		parents = append([]string{row.Key}, parents...)
	}
	slices.Reverse(parents)
	for level := len(parents) - 1; level >= 0; level-- {
		if next, ok := l.indexes[parents[level]]; ok {
			return max(float32(0), float32(next-level)*inspectorRowHeight)
		}
	}
	return 0
}

func inspectorSameLayout(l *inspectorTableLayout, rows []session.InspectorRecord, branches map[string]bool) bool {
	if l == nil || len(l.rows) != len(rows) {
		return false
	}
	for i, row := range rows {
		old := l.rows[i]
		if old.Key != row.Key || old.Parent != row.Parent || old.Disclosure != row.Disclosure || l.branches[row.Key] != branches[row.Key] {
			return false
		}
	}
	return true
}

func (p *SessionInspectorPaneState) prepareInspectorLayout(rows []session.InspectorRecord, branches map[string]bool, height float32) *inspectorTableLayout {
	s := &p.scroll
	if !s.started {
		s.started = true
		if s.mode == "" {
			// A confirmed filter or a queued reveal explicitly starts at scroll.
			s.mode = "tail"
		}
	} else if p.tableY.Y != s.lastOffset && s.mode != "manual" {
		if p.tableY.MaxY-p.tableY.Y <= 40 {
			s.mode = "tail"
		} else {
			s.mode = "scroll"
		}
	}
	changed := s.reset || s.anchor != nil || !inspectorSameLayout(s.layout, rows, branches)
	if changed {
		previous := s.layout
		if s.reset {
			previous = nil
		}
		s.layout = newInspectorTableLayout(rows, branches, previous, p.tableY.Y, height+inspectorRowHeight, s.anchor)
		s.anchor, s.reset = nil, false
		if s.layout.scrollOffset != nil {
			y := *s.layout.scrollOffset
			if y != p.tableY.Y && s.mode != "manual" {
				// A programmatic scroll fires against the new content extent.
				maximum := max(float32(0), float32(len(rows))*inspectorRowHeight+s.layout.bottomPadding-height)
				if maximum-y <= 40 {
					s.mode = "tail"
				} else {
					s.mode = "scroll"
				}
			}
			p.list.ScrollToOffset(y)
			p.tableY.Y, s.lastOffset = y, y
		} else if s.mode == "tail" && s.layout.bottomPadding == 0 {
			p.list.ScrollToEnd()
		}
	} else if height != s.viewportHeight && s.mode == "tail" && s.layout.bottomPadding == 0 {
		p.list.ScrollToEnd()
	}
	// Following is owned here: the toolkit uses an exact-end threshold and would
	// override the reference's manual mode and retained trailing extent.
	p.list.FollowEnd = false
	s.viewportHeight = height
	return s.layout
}

func (p *SessionInspectorPaneState) toggleInspectorDisclosure(row session.InspectorRecord, top float32) {
	p.scroll.mode = "manual"
	p.scroll.anchor = &inspectorRowAnchor{key: row.Key, top: top + inspectorRowHeight}
	p.scroll.reset = true
	p.setInspectorCollapsed(row, !inspectorRowCollapsed(row, p.collapsed))
	p.list.FollowEnd = false
}

func (p *SessionInspectorPaneState) commitInspectorViewport(top float32) {
	s := &p.scroll
	changed := p.tableY.Y != s.lastOffset
	if s.requestedFrom != nil {
		changed = changed || p.tableY.Y != *s.requestedFrom
		s.requestedFrom = nil
	}
	if changed && s.mode != "manual" {
		if p.tableY.MaxY-p.tableY.Y <= 40 {
			s.mode = "tail"
		} else {
			s.mode = "scroll"
		}
	}
	s.lastOffset, s.viewportTop = p.tableY.Y, top
}

func (p *SessionInspectorPaneState) followInspectorLatest() {
	p.scroll.mode, p.scroll.reset = "tail", true
	p.list.ScrollToEnd()
}
