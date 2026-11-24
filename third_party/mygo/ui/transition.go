package ui

import (
	"slices"
	"time"
)

// ElementTransition animates an element (Element.Transition) from what the
// last frame showed to what the layout makes of it in this one: where it
// is in its parent, its size, and its background and border colors. Enter
// and Exit animate it as it appears and as it goes:
//
//	ui.Row(c).Key(item.ID).Transition(ui.ElementTransition{
//		Enter: &ui.Motion{Collapse: true},
//		Exit:  &ui.Motion{Collapse: true},
//	})
type ElementTransition struct {
	// Duration is how long a change takes: 200 ms when zero.
	Duration time.Duration
	// ExitDuration is how long an exit takes: Duration when zero, so that an
	// element can go faster than it came, as the design system's overlay does
	// (200 ms in, 100 ms out).
	ExitDuration time.Duration
	// Ease is how the element goes, as for AnimateWith: EaseOut when nil.
	Ease Easing
	// Position, Size and Colors choose what animates: where the element is
	// in its parent, its width and height (its content laid out at each
	// size on the way), and its background and border colors. None set
	// animates them all.
	Position, Size, Colors bool
	// Enter, if set, is where the element comes from as it appears in a
	// parent that the last frame showed: it starts there, and goes to its
	// place.
	Enter *Motion
	// Exit, if set, is where the element goes once it is no longer built,
	// while its parent still is: a copy of it, as the last frame showed
	// it, goes there under its siblings, taking neither room, the pointer
	// nor the focus, and is then gone.
	Exit *Motion
}

// Motion is where an element appearing comes from, or where one going
// goes, from its place in the layout.
type Motion struct {
	// X and Y move it by as many DIPs.
	X, Y float32
	// Opacity is how opaque it is there: 0, as by default, is
	// transparent, and 1 opaque.
	Opacity float32
	// Collapse makes it nothing along its parent's direction: no height in
	// a column, no width in a row. Its content keeps its layout, clipped.
	Collapse bool
	// Scale shrinks (below 1) or grows (above 1) the element's box around
	// its centre there: 1, and 0, leave it at its size. The box clips what
	// it holds while it is not 1, so a panel shrinking in shows its middle
	// rather than spilling past its edges; the toolkit paints no scaled
	// subtree, so content keeps its own layout instead of shrinking with
	// the box.
	Scale float32
}

// defaultDuration is the Duration of a Transition without one.
const defaultDuration = 200 * time.Millisecond

// Transition animates the element as t asks. The element needs the same
// ID from frame to frame: a Key, or the same place among its siblings.
// Elements that the change moves as well need a Transition of their own to
// move smoothly, the same one to move in step with it, as the rows of a
// list:
//
//	for _, it := range app.items {
//		ui.Row(c).Key(it.ID).Transition(rowTransition).Children(func() {
//			ui.Text(c, it.Title)
//		})
//	}
//
// Changes made by resizing the window do not animate, and nothing does
// while the desktop asks for less motion (Preferences.ReduceMotion).
func (e *node) Transition(t ElementTransition) *node {
	e.c.transitions = append(e.c.transitions, transitionUse{e: e, t: t})
	return e
}

// transitionUse is a call of Transition in the frame being built.
type transitionUse struct {
	e *node
	t ElementTransition
	r *transition
}

// tbox is what a transition moves: an element's place in its parent's
// box, its size, its opacity and how much of its box it takes up.
type tbox struct {
	x, y, w, h, alpha, scale float32
}

// What a transition moves now.
const (
	movePos uint8 = 1 << iota
	moveSize
	moveAlpha
	moveScale
)

// The axes a Motion collapses an element along.
const (
	collapseWidth  = 1
	collapseHeight = 2
)

// transition is what an element's Transition keeps from frame to frame,
// by the element's ID. It outlives the element while a copy of it leaves.
type transition struct {
	t ElementTransition
	// built is the last frame that built the element, and laid the last
	// that laid it out: elem is its element then, and parent the ID of
	// its parent.
	built, laid uint64
	elem        *node
	parent      uint64
	// target is where the layout put the element in frame laid, next
	// where it puts it in the frame being laid out, and shown where it
	// shows: moving (movePos, …) from from since start.
	target, next, shown, from tbox
	start                     time.Time
	moving                    uint8
	fresh                     bool
	// collapse is the axis a Motion collapses the element along, while it
	// does.
	collapse uint8
	// The background and border colors, which move as the element
	// paints, after its styleFn set them.
	colorTarget, colorFrom, colorShown [2]Color
	colorStart                         time.Time
	colorMoving, colorSet              bool
	colorFrame                         uint64
	colorPaintSeq                      uint64
	// Pure-opacity transitions animate in paint without rebuilding the view.
	baseOpacity    float32
	baseOpacitySet bool
	alphaPaintSeq  uint64
	// ghost is the copy of the element going, which moves from gone,
	// where it showed as it went; opacity is its own.
	ghost   *node
	gone    tbox
	opacity float32
	// snapRoot is a copy of the element as the last frame that built it
	// showed it, made in snapNodes and snapStates, which becomes the ghost
	// when the element goes.
	snapRoot   *node
	snapNodes  []node
	snapStates []state
}

func (r *transition) duration() time.Duration {
	if r.t.Duration > 0 {
		return r.t.Duration
	}
	return defaultDuration
}

// exitDuration is how long the element takes to go: its own ExitDuration
// when it has one, and Duration otherwise.
func (r *transition) exitDuration() time.Duration {
	if r.t.ExitDuration > 0 {
		return r.t.ExitDuration
	}
	return r.duration()
}

// progress returns how far a move started at start has gone at now along
// the easing, and whether it is over.
func (r *transition) progress(start, now time.Time) (float32, bool) {
	return r.progressOver(r.duration(), start, now)
}

// progressExit is progress over the exit duration.
func (r *transition) progressExit(start, now time.Time) (float32, bool) {
	return r.progressOver(r.exitDuration(), start, now)
}

// progressOver returns how far a move of length d started at start has gone
// at now along the easing, and whether it is over.
func (r *transition) progressOver(d time.Duration, start, now time.Time) (float32, bool) {
	t := float32(now.Sub(start)) / float32(d)
	if t >= 1 {
		return 1, true
	}
	ease := r.t.Ease
	if ease == nil {
		ease = EaseOut
	}
	return ease(max(t, 0)), false
}

// What a transition animates.
const (
	propPosition uint8 = 1 << iota
	propSize
	propColors
)

func (r *transition) properties() uint8 {
	t := &r.t
	if !t.Position && !t.Size && !t.Colors {
		return propPosition | propSize | propColors
	}
	var p uint8
	if t.Position {
		p |= propPosition
	}
	if t.Size {
		p |= propSize
	}
	if t.Colors {
		p |= propColors
	}
	return p
}

func lerp(a, b, k float32) float32 { return a + (b-a)*k }

// at returns the box k of the way from b to o, in the properties of move.
func (b tbox) at(o tbox, k float32, move uint8) tbox {
	if move&movePos != 0 {
		b.x, b.y = lerp(b.x, o.x, k), lerp(b.y, o.y, k)
	} else {
		b.x, b.y = o.x, o.y
	}
	if move&moveSize != 0 {
		b.w, b.h = lerp(b.w, o.w, k), lerp(b.h, o.h, k)
	} else {
		b.w, b.h = o.w, o.h
	}
	if move&moveAlpha != 0 {
		b.alpha = lerp(b.alpha, o.alpha, k)
	} else {
		b.alpha = o.alpha
	}
	if move&moveScale != 0 {
		b.scale = lerp(b.scale, o.scale, k)
	} else {
		b.scale = o.scale
	}
	return b
}

// moved returns b moved as m says, collapsed along axis.
func (b tbox) moved(m *Motion, axis uint8) tbox {
	b.x += m.X
	b.y += m.Y
	b.alpha = max(0, min(m.Opacity, 1))
	b.scale = m.Scale
	if b.scale <= 0 {
		b.scale = 1
	}
	switch axis {
	case collapseWidth:
		b.w = 0
	case collapseHeight:
		b.h = 0
	}
	return b
}

// collapseAxis returns the axis a Motion collapses a child of parent along.
func collapseAxis(m *Motion, parent *node) uint8 {
	switch {
	case !m.Collapse:
		return 0
	case parent.row && parent.list == nil && !parent.grid:
		return collapseWidth
	}
	return collapseHeight
}

// animateLayout moves the elements given a Transition from where the last
// frame showed them to where the layout put them, and the copies of those
// gone with an Exit: after the layout of a frame in a window w×h, while
// boxes are relative to their parents.
func (rt *engine) animateLayout(root *node, w, h float32) {
	c := root.c
	// Changes that resizing the window makes do not animate.
	resized := w != rt.laidW || h != rt.laidH
	rt.laidW, rt.laidH = w, h
	clear(rt.exitTrans)
	rt.exitTrans = rt.exitTrans[:0]
	uses := c.transitions
	if len(uses) == 0 && len(rt.trans) == 0 {
		return
	}
	if rt.trans == nil {
		rt.trans = map[uint64]*transition{}
	}
	now := c.now
	still := rt.preferences().ReduceMotion
	// Parents first: one resizing lays out its children anew, which then
	// move themselves.
	uses = c.byDepth(uses)
	n := 0
	for _, u := range uses {
		e := u.e
		if e.parent == nil {
			continue // the root's
		}
		r := rt.trans[e.id]
		if r == nil {
			r = &transition{}
			rt.trans[e.id] = r
		} else if r.built == rt.frame {
			if r.elem == e {
				r.t = u.t // the element's last Transition counts
			}
			continue
		}
		r.fresh = r.laid+1 != rt.frame
		if g := r.ghost; g != nil {
			// Built again as a copy of it was going: it comes back from
			// where the copy shows.
			r.ghost, r.fresh = nil, false
			r.from, r.start, r.moving, r.collapse = r.shown, now, movePos|moveSize|moveAlpha|moveScale, 0
		}
		r.t, r.elem, r.parent, r.built = u.t, e, e.parent.id, rt.frame
		r.next = tbox{x: e.x, y: e.y, w: e.w, h: e.h, alpha: 1, scale: 1}
		e.st.trec = r
		if underInvisible(e) {
			// Not laid out (layoutBox): it shows at once when what holds it
			// does, as an element laid out anew.
			continue
		}
		u.r = r
		uses[n] = u
		n++
	}
	for _, u := range uses[:n] {
		e, r := u.e, u.r
		next := r.next
		switch {
		case r.fresh:
			r.moving, r.collapse = 0, 0
			// An element appearing in a parent that was there comes in;
			// whole new parts of the interface, as a page, appear at once.
			if m := r.t.Enter; m != nil && !still && !resized && e.st.born == rt.frame && e.parent.st.born != rt.frame {
				r.collapse = collapseAxis(m, e.parent)
				r.from = next.moved(m, r.collapse)
				var moving uint8
				if r.from.x != next.x || r.from.y != next.y {
					moving |= movePos
				}
				if r.from.w != next.w || r.from.h != next.h {
					moving |= moveSize
				}
				if r.from.alpha != next.alpha {
					moving |= moveAlpha
				}
				if r.from.scale != next.scale {
					moving |= moveScale
				}
				r.start, r.moving = now, moving
			}
		case still || resized:
			r.moving, r.collapse = 0, 0
		default:
			props := r.properties()
			var changed uint8
			if props&propPosition != 0 && (next.x != r.target.x || next.y != r.target.y) {
				changed |= movePos
			}
			if props&propSize != 0 && (next.w != r.target.w || next.h != r.target.h) {
				changed |= moveSize
			}
			if changed != 0 {
				r.from, r.start = r.shown, now
				r.moving |= changed
			}
		}
		r.target, r.laid = next, rt.frame
		r.baseOpacity, r.baseOpacitySet = e.opacity, e.opacitySet
		shown := next
		if r.moving != 0 {
			k, done := r.progress(r.start, now)
			if done {
				r.moving, r.collapse = 0, 0
			} else {
				shown = r.from.at(next, k, r.moving)
				if r.moving == moveAlpha {
					rt.repainting = true
				} else {
					rt.animating = true
				}
			}
		}
		r.shown = shown
		r.apply(e, shown)
		if r.t.Exit != nil && e.leaving == 0 {
			rt.exitTrans = append(rt.exitTrans, r)
		} else if r.snapRoot != nil {
			clear(r.snapNodes)
			clear(r.snapStates)
			r.snapRoot = nil
		}
	}
	rt.animateExits(root, now, still)
}

// byDepth returns the transitions asked for in the order of their
// elements' depth, those of one depth in the order asked: a counting sort,
// into a slice the context reuses.
func (c *context) byDepth(uses []transitionUse) []transitionUse {
	deepest := 0
	for i := range uses {
		deepest = max(deepest, uses[i].e.depth)
	}
	starts := c.depthStarts
	if cap(starts) < deepest+2 {
		starts = make([]int, deepest+2)
	} else {
		starts = starts[:deepest+2]
		clear(starts)
	}
	for i := range uses {
		starts[uses[i].e.depth+1]++
	}
	for d := 1; d < len(starts); d++ {
		starts[d] += starts[d-1]
	}
	sorted := slices.Grow(c.sortedUses[:0], len(uses))[:len(uses)]
	for _, u := range uses {
		sorted[starts[u.e.depth]] = u
		starts[u.e.depth]++
	}
	c.depthStarts, c.sortedUses = starts, sorted
	return sorted
}

// apply shows e at s.
func (r *transition) apply(e *node, s tbox) {
	w, h := max(s.w, 0), max(s.h, 0)
	switch {
	case s.scale != 1:
		// Scaled: the box keeps its centre, paints smaller and clips what it
		// holds, which keeps its own layout.
		scaledW, scaledH := w*max(s.scale, 0), h*max(s.scale, 0)
		s.x, s.y = s.x+(w-scaledW)/2, s.y+(h-scaledH)/2
		w, h = scaledW, scaledH
		e.w, e.h = w, h
		e.flags |= flagClip
	case w != e.w || h != e.h:
		if r.collapse != 0 {
			// Collapsing: the content keeps its layout, clipped.
			e.w, e.h = w, h
			e.flags |= flagClip
		} else {
			layoutBox(e, w, h)
		}
	}
	e.x, e.y = s.x, s.y
	if s.alpha < 1 {
		o := float32(1)
		if e.opacitySet {
			o = e.opacity
		}
		e.opacity, e.opacitySet = o*max(s.alpha, 0), true
	}
}

// animateExits moves the copies of the elements gone with an Exit, under
// their parents, and copies those that just went, from the frame before,
// which the engine kept for them.
func (rt *engine) animateExits(root *node, now time.Time, still bool) {
	for id, r := range rt.trans {
		if r.built == rt.frame {
			continue
		}
		if r.ghost == nil && (r.t.Exit == nil || still || r.laid+1 != rt.frame) {
			delete(rt.trans, id)
			continue
		}
		p := rt.builtNode(r.parent)
		if p == nil || p.leaving != 0 {
			// Its parent went too.
			delete(rt.trans, id)
			continue
		}
		if r.ghost == nil {
			g := r.snapRoot
			if g == nil {
				delete(rt.trans, id) // no copy of it was made as it showed
				continue
			}
			// The copy made as the frame before ended is the ghost's now.
			r.ghost, r.elem = g, nil
			r.snapRoot, r.snapNodes, r.snapStates = nil, nil, nil
			r.ghost.leaving = 1
			if g.flags&flagAbsolute != 0 {
				r.ghost.leaving = 2
			}
			r.ghost.flags |= flagAbsolute
			r.opacity = 1
			if g.opacitySet {
				r.opacity = g.opacity / max(r.shown.alpha, 0.001)
			}
			r.gone, r.start = r.shown, now
			r.collapse = collapseAxis(r.t.Exit, p)
		}
		k, done := r.progressExit(r.start, now)
		if done {
			delete(rt.trans, id)
			continue
		}
		s := r.gone.at(r.gone.moved(r.t.Exit, r.collapse), k, movePos|moveSize|moveAlpha|moveScale)
		g := r.ghost
		g.next = nil
		p.add(g)
		p.ghosts = true
		layoutBox(g, r.gone.w, r.gone.h)
		if s.scale != 1 {
			w, h := max(s.w, 0)*max(s.scale, 0), max(s.h, 0)*max(s.scale, 0)
			s.x, s.y = s.x+(max(s.w, 0)-w)/2, s.y+(max(s.h, 0)-h)/2
			g.w, g.h = w, h
			g.flags |= flagClip
		} else if s.w != r.gone.w || s.h != r.gone.h {
			g.w, g.h = max(s.w, 0), max(s.h, 0)
			g.flags |= flagClip
		}
		g.x, g.y = s.x, s.y
		g.opacity, g.opacitySet = r.opacity*max(s.alpha, 0), true
		r.shown = s
		rt.animating = true
	}
}

// builtNode returns the element with the ID id that the frame built, nil
// for none.
func (rt *engine) builtNode(id uint64) *node {
	s := rt.states[id]
	if s == nil || s.seen != rt.frame || s.pass != rt.pass || s.node == nil || s.node.st != s || s.node.id != id {
		return nil
	}
	return s.node
}

// snapshotExits copies, as the frame ends, the elements it built with an
// Exit and the elements inside them, as they showed: one not built in the
// next frame leaves as this copy (animateExits). When no element in the
// frame specifies an Exit motion, it skips iterating rt.trans completely.
func (rt *engine) snapshotExits() {
	if len(rt.exitTrans) == 0 {
		return
	}
	still := rt.preferences().ReduceMotion
	for _, r := range rt.exitTrans {
		if still || r.built != rt.frame || r.laid != rt.frame || r.t.Exit == nil || r.elem == nil || r.elem.leaving != 0 {
			if r.snapRoot != nil && r.built == rt.frame {
				clear(r.snapNodes)
				clear(r.snapStates)
				r.snapRoot = nil
			}
			continue
		}
		r.snapshot(r.elem)
	}
}

// snapshot copies e and the elements inside it into the transition's
// storage (ghostOf).
func (r *transition) snapshot(e *node) {
	n := ghostSize(e)
	if cap(r.snapNodes) < n {
		r.snapNodes, r.snapStates = make([]node, n), make([]state, n)
	} else {
		if n < len(r.snapNodes) {
			// Let go of what the copy before held beyond this one.
			clear(r.snapNodes[n:])
			clear(r.snapStates[n:])
		}
		r.snapNodes, r.snapStates = r.snapNodes[:n], r.snapStates[:n]
	}
	i := 0
	r.snapRoot = ghostInto(e, r.snapNodes, r.snapStates, &i)
}

// ghostSize counts the elements a copy of e has: e and those inside it,
// but copies leaving.
func ghostSize(e *node) int {
	n := 1
	for c := e.first; c != nil; c = c.next {
		if c.leaving == 0 {
			n += ghostSize(c)
		}
	}
	return n
}

// ghostInto copies e, an element of the frame, and the elements inside it
// into nodes and states from *i on, for its exit transition. The copy only
// shows: it refers to no other element nor to the app's state, has states
// of its own, and is inert.
func ghostInto(e *node, nodes []node, states []state, i *int) *node {
	g, st := &nodes[*i], &states[*i]
	*i++
	shadows, cols, rows, frags := g.shadows[:0], g.cols[:0], g.rows[:0], g.frags[:0]
	*g = *e
	*st = *e.st
	st.trec, st.node = nil, nil
	g.st = st
	g.parent, g.first, g.last, g.next, g.nchild = nil, nil, nil, nil, 0
	g.shadows, g.cols, g.rows, g.frags = append(shadows, e.shadows...), append(cols, e.cols...), append(rows, e.rows...), append(frags, e.frags...)
	g.track, g.popover, g.list, g.rowsOf, g.overflow = nil, nil, nil, nil, nil
	g.activeDescendant, g.nameFrom, g.field, g.form = nil, nil, nil, nil
	g.followX, g.colFit, g.gridFit = nil, nil, nil
	g.inputFn, g.ghosts = nil, false
	// The copy shows the frame it was made in: the callbacks of the app's
	// state — input, escape routing, value commits and layout observers —
	// do not run for it. Its paint callbacks still run so it looks as it
	// did.
	g.regionEscape, g.valueInput = nil, nil
	g.observeLayout, g.offsetAfterLayout = nil, nil
	g.tl, g.nmeasure, g.leafOK = nil, 0, false
	g.flags = g.flags&^(flagModal|flagPage|flagFocusTarget) | flagInert
	g.ghost = true
	for c := e.first; c != nil; c = c.next {
		if c.leaving == 0 {
			g.add(ghostInto(c, nodes, states, i))
		}
	}
	return g
}

// animateColors moves e's background and border colors, once its styleFn
// set them, as its Transition asks. Pure color transitions request paint-only
// frames (rt.repainting) instead of rebuilding the view tree.
func (rt *engine) animateColors(e *node, r *transition) {
	if r.properties()&propColors == 0 {
		return
	}
	if r.colorFrame != rt.frame {
		target := [2]Color{e.bg, e.borderC}
		switch {
		case !r.colorSet || r.colorFrame+1 != rt.frame:
			// Not painted in the frame before: its colors show at once.
			r.colorMoving = false
		case target != r.colorTarget:
			if rt.preferences().ReduceMotion {
				r.colorMoving = false
			} else {
				r.colorFrom, r.colorStart, r.colorMoving = r.colorShown, rt.c.now, true
			}
		}
		r.colorSet, r.colorTarget, r.colorFrame = true, target, rt.frame
		r.colorShown = target
	}
	if r.colorMoving && r.colorPaintSeq != rt.paintSeq {
		r.colorPaintSeq = rt.paintSeq
		if k, done := r.progress(r.colorStart, rt.c.now); done {
			r.colorMoving = false
			r.colorShown = r.colorTarget
		} else {
			r.colorShown = [2]Color{
				mixPremultiplied(r.colorFrom[0], r.colorTarget[0], k),
				mixPremultiplied(r.colorFrom[1], r.colorTarget[1], k),
			}
			rt.repainting = true
		}
	}
	e.bg, e.borderC = r.colorShown[0], r.colorShown[1]
}

// animateAlpha advances a pure-opacity Transition during paint-only frames
// (repaintFrame) without triggering a full view rebuild.
func (rt *engine) animateAlpha(e *node, r *transition) {
	if r.moving != moveAlpha {
		return
	}
	if r.alphaPaintSeq != rt.paintSeq {
		r.alphaPaintSeq = rt.paintSeq
		k, done := r.progress(r.start, rt.c.now)
		if done {
			r.moving = 0
			r.shown.alpha = r.target.alpha
			e.opacity, e.opacitySet = r.baseOpacity, r.baseOpacitySet
			return
		}
		alpha := lerp(r.from.alpha, r.target.alpha, k)
		r.shown.alpha = alpha
		base := float32(1)
		if r.baseOpacitySet {
			base = r.baseOpacity
		}
		e.opacity, e.opacitySet = base*max(alpha, 0), true
		rt.repainting = true
	}
}

// mixPremultiplied returns the color k of the way from a to b, mixed with
// premultiplied alpha, so that a color fading from transparent keeps its
// hue.
func mixPremultiplied(a, b Color, k float32) Color {
	aa, ba := float32(a.A)/255, float32(b.A)/255
	alpha := lerp(aa, ba, k)
	if alpha <= 0 {
		return Color{}
	}
	if a.wide.ok() || b.wide.ok() {
		// Premultiplied, b weighs this much of the straight colors.
		c := mixWide(a, b, k*ba/alpha)
		c.A = alphaByte(alpha)
		return c
	}
	ch := func(x, y uint8) uint8 {
		v := lerp(float32(x)*aa, float32(y)*ba, k) / alpha
		return uint8(max(0, min(v+0.5, 255)))
	}
	return Color{R: ch(a.R, b.R), G: ch(a.G, b.G), B: ch(a.B, b.B), A: alphaByte(alpha)}
}

// underInvisible reports whether an element is inside an invisible one,
// whose content is not laid out.
func underInvisible(e *node) bool {
	for p := e.parent; p != nil; p = p.parent {
		if p.flags&flagInvisible != 0 {
			return true
		}
	}
	return false
}
