//go:build !release

package ui

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestingT is the minimal testing interface accepted by audit helpers such as
// AssertHoverFocusZeroShift so non-test builds do not import package testing.
type TestingT interface {
	Helper()
	Errorf(format string, args ...any)
}

// NodeRect describes one non-overlay element's stable identity and laid-out box
// in DIPs, captured by Tester.FlowSnapshot.
type NodeRect struct {
	ID    uint64
	Label string
	Text  string
	Role  Role
	Rect  Rect
}

// String names the node rect for failure diagnostics.
func (n NodeRect) String() string {
	name := n.Text
	if name == "" {
		name = n.Label
	}
	return fmt.Sprintf("id=%d role=%v name=%q rect=%v", n.ID, n.Role, name, n.Rect)
}

// InteractiveBox describes an element of the last frame that takes the
// pointer or the focus, for audits of a view's interaction.
type InteractiveBox struct {
	// Rect is the visible box of the element, in DIPs.
	Rect Rect
	// Label is the text or label of the element, or of the nearest ancestor
	// that has one.
	Label string
	// Role is the element's role for assistive technology.
	Role Role
	// Kinds names what the element takes: click, focus, hover, drag, text.
	Kinds []string
	// Cursor is the pointer shape the window shows over the middle of the
	// element's box, which is what the user sees there.
	Cursor Cursor
}

// Interactives lists the enabled elements of the last frame that take
// clicks, the focus, drags or text, with the pointer shape shown over
// each. It moves the pointer over them, so call it on a view that is
// ready to be looked at, not in the middle of a gesture.
func (t *Tester) Interactives() []InteractiveBox {
	texts := map[uint64]string{}
	for _, n := range t.rt.labels {
		texts[n.id] = n.text
	}
	type entry struct {
		st *state
		r  Rect
	}
	var list []entry
	seen := map[uint64]bool{}
	for _, h := range t.rt.hits {
		s := h.st
		if s == nil || seen[s.id] || h.r.W <= 0 || h.r.H <= 0 {
			continue
		}
		if s.flags&(flagClickable|flagFocusable|flagDraggable|flagEditable) == 0 || s.flags&flagDisabled != 0 {
			continue
		}
		seen[s.id] = true
		list = append(list, entry{s, h.r})
	}
	var out []InteractiveBox
	for _, e := range list {
		s := e.st
		label := texts[s.id]
		for p := s; label == "" && p != nil; p = t.rt.states[p.parent] {
			label = texts[p.id]
			if p.parent == 0 {
				break
			}
		}
		if label == "" {
			// The first text inside the box.
			for _, n := range t.rt.labels {
				if n.r.X >= e.r.X && n.r.Y >= e.r.Y && n.r.X+n.r.W <= e.r.X+e.r.W+1 && n.r.Y+n.r.H <= e.r.Y+e.r.H+1 {
					label = n.text
					break
				}
			}
		}
		var kinds []string
		if s.flags&flagClickable != 0 {
			kinds = append(kinds, "click")
		}
		if s.flags&flagFocusable != 0 {
			kinds = append(kinds, "focus")
		}
		if s.flags&flagDraggable != 0 {
			kinds = append(kinds, "drag")
		}
		if s.flags&flagEditable != 0 {
			kinds = append(kinds, "text")
		}
		t.Move(e.r.X+e.r.W/2, e.r.Y+e.r.H/2)
		out = append(out, InteractiveBox{Rect: e.r, Label: label, Role: s.role, Kinds: kinds, Cursor: t.Cursor()})
	}
	return out
}

// String names the box for logs.
func (b InteractiveBox) String() string {
	return fmt.Sprintf("%v role=%v kinds=%v cursor=%v label=%q", b.Rect, b.Role, b.Kinds, b.Cursor, b.Label)
}

// FlowSnapshot returns the stable ID, label/text, role, and final laid-out
// Rect of every visible in-flow (non-overlay) element in the last frame,
// including button interiors, text nodes, and siblings. Transient overlays
// (tooltips, popovers, hover cards, menus) are excluded; already-open modal
// dialog content is included so dialogs can be audited too.
func (t *Tester) FlowSnapshot() []NodeRect {
	if t == nil || t.rt == nil || t.rt.c.root == nil {
		return nil
	}
	var out []NodeRect
	var walk func(e *node)
	walk = func(e *node) {
		if e == nil || e.flags&flagInvisible != 0 || e.leaving != 0 || e.ghost {
			return
		}
		if e.id == overlayID {
			for ch := e.first; ch != nil; ch = ch.next {
				if ch.flags&flagModal != 0 {
					walk(ch)
				}
			}
			return
		}
		if e.parent != nil {
			out = append(out, NodeRect{
				ID:    e.id,
				Label: e.label,
				Text:  e.text,
				Role:  e.role,
				Rect:  Rect{X: e.x, Y: e.y, W: e.w, H: e.h},
			})
		}
		for ch := e.first; ch != nil; ch = ch.next {
			walk(ch)
		}
	}
	walk(t.rt.c.root)
	return out
}

type flowAuditState struct {
	nodes       []NodeRect
	texts       []string
	scrolls     map[uint64][2]float64
	nodeScrolls map[uint64][]uint64
}

func (t *Tester) flowAuditState() flowAuditState {
	st := flowAuditState{
		scrolls:     map[uint64][2]float64{},
		nodeScrolls: map[uint64][]uint64{},
	}
	if t == nil || t.rt == nil || t.rt.c.root == nil {
		return st
	}
	var stack []uint64
	var walk func(e *node)
	walk = func(e *node) {
		if e == nil || e.flags&flagInvisible != 0 || e.leaving != 0 || e.ghost {
			return
		}
		if e.id == overlayID {
			for ch := e.first; ch != nil; ch = ch.next {
				if ch.flags&flagModal != 0 {
					walk(ch)
				}
			}
			return
		}
		if e.parent != nil {
			nr := NodeRect{
				ID:    e.id,
				Label: e.label,
				Text:  e.text,
				Role:  e.role,
				Rect:  Rect{X: e.x, Y: e.y, W: e.w, H: e.h},
			}
			st.nodes = append(st.nodes, nr)
			if e.kind == kindText && e.text != "" {
				st.texts = append(st.texts, e.text)
			}
			if len(stack) > 0 {
				anc := make([]uint64, len(stack))
				copy(anc, stack)
				st.nodeScrolls[e.id] = anc
			}
		}
		pushed := false
		if e.scrolls() && e.st != nil {
			st.scrolls[e.id] = [2]float64{e.st.scrollX, e.st.scrollY}
			stack = append(stack, e.id)
			pushed = true
		}
		for ch := e.first; ch != nil; ch = ch.next {
			if ch.flags&flagAbsolute != 0 && pushed {
				// Absolute children do not translate with their container's scroll.
				saved := stack
				stack = stack[:len(stack)-1]
				walk(ch)
				stack = saved
				continue
			}
			walk(ch)
		}
		if pushed {
			stack = stack[:len(stack)-1]
		}
	}
	walk(t.rt.c.root)
	return st
}

// interactiveTargets returns interactive elements in the active scope without
// moving the pointer.
func (t *Tester) interactiveTargets() []InteractiveBox {
	texts := map[uint64]string{}
	for _, n := range t.rt.labels {
		texts[n.id] = n.text
	}
	activeModal := t.rt.modal
	seen := map[uint64]bool{}
	var out []InteractiveBox
	for _, h := range t.rt.hits {
		s := h.st
		if s == nil || seen[s.id] || h.r.W <= 0 || h.r.H <= 0 {
			continue
		}
		if s.flags&(flagClickable|flagFocusable|flagDraggable|flagEditable) == 0 || s.flags&flagDisabled != 0 {
			continue
		}
		if activeModal != 0 && s.scope != activeModal {
			continue
		}
		seen[s.id] = true
		label := texts[s.id]
		for p := s; label == "" && p != nil; p = t.rt.states[p.parent] {
			label = texts[p.id]
			if p.parent == 0 {
				break
			}
		}
		if label == "" {
			for _, n := range t.rt.labels {
				if n.r.X >= h.r.X && n.r.Y >= h.r.Y && n.r.X+n.r.W <= h.r.X+h.r.W+1 && n.r.Y+n.r.H <= h.r.Y+h.r.H+1 {
					label = n.text
					break
				}
			}
		}
		out = append(out, InteractiveBox{Rect: h.r, Label: label, Role: s.role})
	}
	return out
}

const zeroShiftTolerance float32 = 0.01

func assertFlowUnchanged(t TestingT, phase string, base, got flowAuditState, compensateScroll bool) bool {
	t.Helper()
	ok := true
	if !slices.Equal(base.texts, got.texts) {
		t.Errorf("zero-shift[%s]: in-flow texts changed: base=%q got=%q", phase, base.texts, got.texts)
		ok = false
	}
	if len(base.nodes) != len(got.nodes) {
		t.Errorf("zero-shift[%s]: in-flow node count changed: base=%d got=%d", phase, len(base.nodes), len(got.nodes))
		ok = false
	}
	gotByID := make(map[uint64]NodeRect, len(got.nodes))
	for _, n := range got.nodes {
		gotByID[n.ID] = n
	}
	for _, b := range base.nodes {
		g, exists := gotByID[b.ID]
		if !exists {
			t.Errorf("zero-shift[%s]: in-flow node disappeared: %s", phase, b)
			ok = false
			continue
		}
		dx, dy := g.Rect.X-b.Rect.X, g.Rect.Y-b.Rect.Y
		if compensateScroll {
			for _, sid := range base.nodeScrolls[b.ID] {
				bs, bok := base.scrolls[sid]
				gs, gok := got.scrolls[sid]
				if bok && gok {
					dx += float32(gs[0] - bs[0])
					dy += float32(gs[1] - bs[1])
				}
			}
		}
		dw, dh := g.Rect.W-b.Rect.W, g.Rect.H-b.Rect.H
		if abs32(dx) > zeroShiftTolerance || abs32(dy) > zeroShiftTolerance || abs32(dw) > zeroShiftTolerance || abs32(dh) > zeroShiftTolerance {
			t.Errorf("zero-shift[%s]: node %s shifted by d=(%.3f, %.3f, %.3f, %.3f) (base=%v got=%v)",
				phase, b, dx, dy, dw, dh, b.Rect, g.Rect)
			ok = false
		}
	}
	return ok
}

// AssertHoverFocusZeroShift sweeps pointer hover over every interactive control
// (advancing virtual time by 250ms to catch animated width/layout growth, then
// leaving to check restoration) and sweeps Tab across all focusable controls,
// asserting that no in-flow non-overlay element changes geometry (> 0.01 DIP)
// and no in-flow text or child node is added or removed.
func AssertHoverFocusZeroShift(t TestingT, tt *Tester) {
	t.Helper()
	if tt == nil || tt.rt == nil || tt.rt.c.root == nil {
		return
	}
	origClock := tt.rt.clock
	defer func() { tt.rt.clock = origClock }()

	now := tt.rt.now()
	if now.IsZero() {
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	advance := func(d time.Duration) {
		now = now.Add(d)
		tt.SetTime(now)
	}
	// Establish a clean resting baseline with pointer outside the window.
	tt.SetTime(now)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerLeave})
	advance(250 * time.Millisecond)

	base := tt.flowAuditState()
	targets := tt.interactiveTargets()

	for i, target := range targets {
		cx, cy := target.Rect.X+target.Rect.W/2, target.Rect.Y+target.Rect.H/2
		name := target.Label
		if name == "" {
			name = fmt.Sprintf("#%d@%.0f,%.0f", i, cx, cy)
		}
		tt.Move(cx, cy)
		assertFlowUnchanged(t, fmt.Sprintf("hover(%s)+0ms", name), base, tt.flowAuditState(), false)
		advance(100 * time.Millisecond)
		assertFlowUnchanged(t, fmt.Sprintf("hover(%s)+100ms", name), base, tt.flowAuditState(), false)
		advance(150 * time.Millisecond)
		assertFlowUnchanged(t, fmt.Sprintf("hover(%s)+250ms", name), base, tt.flowAuditState(), false)

		tt.send(platform.SurfaceEvent{Kind: platform.PointerLeave})
		advance(250 * time.Millisecond)
		assertFlowUnchanged(t, fmt.Sprintf("leave(%s)", name), base, tt.flowAuditState(), false)
	}

	// Keyboard focus sweep across focusable elements in the active scope.
	focusCount := 0
	if m := tt.rt.modal; m != 0 {
		for _, sc := range tt.rt.focusScopes {
			if sc.modal == m {
				focusCount++
			}
		}
	} else {
		focusCount = len(tt.rt.focusOrder)
	}
	if focusCount > 64 {
		focusCount = 64
	}
	for i := range focusCount {
		tt.Key(0, KeyTab)
		assertFlowUnchanged(t, fmt.Sprintf("tab(%d)+0ms", i+1), base, tt.flowAuditState(), true)
		advance(100 * time.Millisecond)
		assertFlowUnchanged(t, fmt.Sprintf("tab(%d)+100ms", i+1), base, tt.flowAuditState(), true)
		advance(150 * time.Millisecond)
		assertFlowUnchanged(t, fmt.Sprintf("tab(%d)+250ms", i+1), base, tt.flowAuditState(), true)
	}
}
