package ui

import (
	"os"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// memoOff turns memo regions off, so that every region builds in every
// pass, for comparing frames with and without them: MYGO_MEMO=off.
var memoOff = os.Getenv("MYGO_MEMO") == "off"

// Memo makes the element a memo region: the children its next Children
// call builds are kept from pass to pass and frame to frame, and Children
// skips its function while version, the element's identity (its place or
// Key) and what the region was built against stay the same. Bump version
// whenever what the function shows changes, as with a store's revision:
//
//	ui.Column(c).Width(240).Memo(app.sessions.Rev).Children(func() {
//		app.sidebar(c)
//	})
//
// A kept region is neither built nor laid out again; its hits, focus
// order, selectable texts and accessibility nodes are those it had. It is
// built again, all the same, in the frame after input reached it (the
// pointer over it, a key while the focus is in it, an action of assistive
// technology on one of its elements), while the focus is in it or one of
// its elements is pressed or dragged, after the window's size, theme,
// fonts or scale changed, and in every frame while something in it asks
// for frames (animations, transitions, timers) or keeps what a region
// cannot (overlays, routers, lists, focus bindings, Handles, input
// observers). The element itself builds in every pass as any other: only
// its children are kept.
//
// The function must depend on nothing but what version covers; a region
// inside another builds only when the one around it does, whose version
// must then change with the inner one's. Draw
// callbacks in a kept region run again in later frames: they must not use
// Elements they captured, which expire as usual. Call Memo before
// Children, after Key. MYGO_MEMO=off builds every region in every pass.
func (e *node) Memo(version uint64) *node {
	if e.kind != kindText {
		e.memoOn, e.memoVersion = true, version
	}
	return e
}

// memoRegion is what the engine keeps of a memo region between passes:
// the children last built, in an arena of their own, and what the frame
// derived from them.
type memoRegion struct {
	id      uint64
	version uint64
	print   memoPrint
	// valid is set once a build completed whose children may be kept;
	// live when that build asked for frames or kept what a region cannot,
	// and dirty once input reached the region since.
	valid, live, dirty bool
	// keys is set when the region's elements take keys through shortcuts or
	// actions; actions tracks non-shortcut actions that may react to any key.
	keys    bool
	actions bool
	// focus is set when the keyboard focus was in the region as it built.
	focus bool
	// seen and seenPass are the frame and pass that last built or kept the
	// region; parent is the region it is in, 0 for none.
	seen     uint64
	seenPass int
	parent   uint64
	reused   bool // in the pass that last saw it
	// The arena of the region's elements, as the context's.
	chunks      [][]node
	used, spent int
	first, last *node
	nchild      int
	depth       int
	// nodes are the elements inside the region, in build order, inner
	// regions' included, and states theirs.
	nodes  []*node
	states []*state
	// What the region's build registered with the pass, which keeping it
	// registers again.
	regs        []shortcutReg
	dividers    []dividers
	binds       []memoBind
	transitions []transitionUse
	// flags of what the region holds.
	texts, observers, selectables bool
	// The layout the region's children had in the last frame: the size of
	// the element holding them, its layout properties and text style, and
	// the size of its content when it scrolls.
	laid               bool
	lw, lh             float32
	lprops             memoLayout
	contentW, contentH float64
	// The commit of the region's children: where the region was, against
	// which clip, scope and page, and what they added to the hit list, the
	// focus order and the labels; bounds is the union of the hits, which
	// pointer input reaching dirties the region.
	committed   bool
	cx, cy      float32
	clip        Rect
	scope       focusScope
	page        uint64
	disabled    bool
	plain       bool // nothing in it changed the scope or the groups
	hits        []hit
	focusOrder  []uint64
	focusScopes []focusScope
	labels      []labelNode
	selTexts    []*state
	bounds      Rect
	// scrolls are the scroll containers in the region, with their offsets
	// as it committed, which its hits and nodes follow.
	scrolls []scrollMark
	// The accessibility nodes of the region's children, with parents
	// relative to the first (-1 for the node of the region's element),
	// where the region was as they were made, and whether in a scroll
	// container.
	// The measures of the region's children: their max-content and
	// min-content widths, and their heights at widths, for the properties
	// of the element holding them mprops; and the selection scope their
	// texts were prepared in.
	mprops      memoLayout
	measured    [2]bool
	widths      [2]float32
	heights     []measure
	selPrepared bool
	selScope    uint64
	selControl  bool
	described   bool
	ax, ay      float32
	scrolled    bool
	access      []platform.AccessNode
}

// memoPrint is what a region is built against, besides its version: a
// region built against another is built again.
type memoPrint struct {
	theme    Theme
	gen      uint64
	scale    float32
	w, h     float32
	inert    bool
	buttons  buttonStyle
	router   *Router
	vibrancy bool
	titleBar TitleBar
}

// memoLayout is what of the element holding a region its children's layout
// depends on.
type memoLayout struct {
	row, wrap, reverse, wrapReverse, grid, scrollOverflowX bool
	justify, align, alignContent, justifyItems             Align
	gapX, gapY                                             float32
	pad, border                                            [4]float32
	flags                                                  uint32
	ts                                                     textStyle
}

func (c *context) memoPrint() memoPrint {
	return memoPrint{theme: *c.theme, gen: c.rt.gen, scale: c.rt.painted[2], w: c.w, h: c.h, inert: c.inert,
		buttons: c.buttons, router: c.router, vibrancy: c.vibrancy, titleBar: c.titleBar}
}

// samePrint compares prints; themes whose materials do not compare differ.
func samePrint(a, b *memoPrint) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()
	return *a == *b
}

// memoMarks are the lengths of what a pass registers, before a region
// builds, to tell what its build registered.
type memoMarks struct {
	parts, observers, captures, reveal, clicks, toasts, regs, phys int
	transitions, dividers, binds, members, routers, kept, actions  int
	inputs, physTerminals                                          int
	overlay                                                        *node
	overlayN                                                       int
	animating, consumed, late                                      bool
	wake                                                           time.Time
}

func (c *context) memoMarks() memoMarks {
	rt := c.rt
	m := memoMarks{parts: len(rt.parts), observers: len(rt.inputObservers), captures: len(rt.keyCaptures),
		reveal: len(c.reveal), clicks: len(rt.clickLater), toasts: len(rt.toasts), regs: len(rt.nextRegs),
		phys: len(rt.physical.next), transitions: len(c.transitions), dividers: len(c.dividers), binds: len(rt.bindLog),
		routers: c.routers, kept: len(rt.kept), actions: len(rt.actions), inputs: len(rt.inputs),
		physTerminals: len(rt.physical.nextTerminals),
		overlay:       c.overlay, animating: rt.animating, consumed: rt.consumed, late: rt.late}
	for _, f := range rt.focusFields {
		m.members += len(f.members)
	}
	if c.overlay != nil {
		m.overlayN = c.overlay.nchild
	}
	rt.wakeMu.Lock()
	m.wake = rt.wakeAt
	rt.wakeMu.Unlock()
	return m
}

// memoChildren builds the children of e, a memo region, or keeps those it
// built before.
func (c *context) memoChildren(e *node, fn func()) {
	rt := c.rt
	if rt.memos == nil {
		rt.memos = map[uint64]*memoRegion{}
	}
	r := rt.memos[e.id]
	if r == nil {
		r = &memoRegion{id: e.id}
		rt.memos[e.id] = r
	}
	if r.seen == rt.frame && r.seenPass == rt.pass {
		// Built twice in a pass, as two elements with one key: build
		// this one as any element.
		saved := c.parent
		c.parent = e
		fn()
		c.parent = saved
		return
	}
	r.parent = 0
	if c.region != nil {
		r.parent = c.region.id
	}
	print := c.memoPrint()
	if rt.memoKeep(r, e, &print) {
		rt.keepRegion(r, e)
		return
	}
	rt.buildRegion(r, e, fn, &print)
}

// memoKeep reports whether the region's children may be kept in this pass.
func (rt *engine) memoKeep(r *memoRegion, e *node, print *memoPrint) bool {
	c := &rt.c
	switch {
	case memoOff, rt.insp.open, !r.valid, r.live, r.dirty, r.focus, r.version != e.memoVersion:
		return false
	case c.tree != nil, c.form != nil, c.accordion != nil, c.checks != nil, c.row != nil, c.sidebar != nil:
		return false
	case !samePrint(&r.print, print):
		return false
	case rt.drag != nil, rt.incoming != nil, rt.scrollDrag.st != nil && (rt.scrollDrag.st == e.st || rt.inRegion(rt.scrollDrag.st, r.id)):
		return false
	case rt.pressed != nil && rt.inRegion(rt.pressed, r.id):
		return false
	case rt.focused != 0 && rt.inRegion(rt.states[rt.focused], r.id):
		return false
	case rt.selection.scope != 0 && rt.inRegion(rt.states[rt.selection.scope], r.id):
		return false
	case rt.wheelID != 0 && (rt.wheelRemainX != 0 || rt.wheelRemainY != 0) && (rt.wheelID == e.id || rt.inRegion(rt.states[rt.wheelID], r.id)):
		return false
	case rt.memoTransitionBusy(r):
		return false
	}
	// The region was in the frame before, or in an earlier pass of this
	// one: its states are still there.
	return r.seen == rt.frame || r.seen+1 == rt.frame // pruneMemos lets go of one missing from a frame
}

// memoTransitionBusy reports whether any transition inside r is currently
// animating layout geometry or an exit ghost (requiring a rebuild). Resting
// transitions and paint-only color/alpha transitions do not bust memo regions.
func (rt *engine) memoTransitionBusy(r *memoRegion) bool {
	if len(r.transitions) == 0 || len(rt.trans) == 0 {
		return false
	}
	for _, u := range r.transitions {
		if tr := rt.trans[u.e.id]; tr != nil && (tr.moving&^moveAlpha != 0 || tr.ghost != nil) {
			return true
		}
	}
	return false
}

// inRegion reports whether the state s was in the region id as the last
// frame committed it, or in a region inside it.
func (rt *engine) inRegion(s *state, id uint64) bool {
	if s == nil {
		return false
	}
	for m, n := s.memo, 0; m != 0 && n < 64; n++ {
		if m == id {
			return true
		}
		r := rt.memos[m]
		if r == nil {
			return false
		}
		m = r.parent
	}
	return false
}

// keepRegion gives e the children the region kept, as this pass's: their
// states count as built, and what they registered with the pass is
// registered again.
func (rt *engine) keepRegion(r *memoRegion, e *node) {
	c := &rt.c
	e.first, e.last, e.nchild = r.first, r.last, r.nchild
	for ch := e.first; ch != nil; ch = ch.next {
		ch.parent = e
	}
	if d := e.depth + 1; d != r.depth {
		for ch := e.first; ch != nil; ch = ch.next {
			setDepth(ch, d)
		}
		r.depth = d
	}
	for _, n := range r.nodes {
		// A handle made now for a kept element is this pass's; those made
		// as it was built expire, as handles do.
		c.slots = append(c.slots, n)
		n.serial = int32(len(c.slots))
		n.epoch = rt.epoch
	}
	for _, s := range r.states {
		if !(s.seen == rt.frame && s.pass == rt.pass) {
			rt.touched = append(rt.touched, s)
		}
		s.seen, s.pass = rt.frame, rt.pass
	}
	for _, n := range r.nodes {
		if n.memoOn {
			// A region inside, kept with this one.
			if in := rt.memos[n.id]; in != nil {
				in.seen, in.seenPass, in.reused = rt.frame, rt.pass, true
			}
		}
	}
	rt.nextRegs = append(rt.nextRegs, r.regs...)
	c.dividers = append(c.dividers, r.dividers...)
	c.transitions = append(c.transitions, r.transitions...)
	for _, b := range r.binds {
		b.ref.element = wrapElement(b.n)
		if c.region != nil {
			rt.bindLog = append(rt.bindLog, b)
		}
	}
	r.seen, r.seenPass, r.reused = rt.frame, rt.pass, true
	rt.memoKept++
}

func setDepth(e *node, d int) {
	e.depth = d
	for ch := e.first; ch != nil; ch = ch.next {
		setDepth(ch, d+1)
	}
}

// buildRegion builds the children of e into the region's arena, and notes
// what the region keeps of them.
func (rt *engine) buildRegion(r *memoRegion, e *node, fn func(), print *memoPrint) {
	c := &rt.c
	before := c.memoMarks()
	// What the region's build asks for, whatever the pass asked for before.
	rt.consumed, rt.animating, rt.late = false, false, false
	// The arena of the build before gives way: those elements are gone.
	r.spent = max(r.spent, r.used)
	r.used = 0
	saved, savedRegion := c.parent, c.region
	c.parent, c.region = e, r
	fn()
	c.parent, c.region = saved, savedRegion
	r.trim()
	after := c.memoMarks()
	rt.consumed = before.consumed || after.consumed
	rt.animating = before.animating || after.animating
	rt.late = before.late || after.late
	r.first, r.last, r.nchild, r.depth = e.first, e.last, e.nchild, e.depth+1
	r.version, r.print = e.memoVersion, *print
	r.seen, r.seenPass, r.reused = rt.frame, rt.pass, false
	r.dirty, r.laid, r.committed, r.described = false, false, false, false
	r.measured, r.heights, r.selPrepared = [2]bool{}, r.heights[:0], false
	r.focus = false
	r.live = after.consumed || after.animating || !after.wake.Equal(before.wake) ||
		after.parts != before.parts || after.observers != before.observers || after.captures != before.captures ||
		after.reveal != before.reveal || after.clicks != before.clicks || after.toasts != before.toasts ||
		after.phys != before.phys || after.physTerminals != before.physTerminals ||
		after.members != before.members || after.routers != before.routers ||
		after.kept != before.kept || after.overlay != before.overlay || after.overlayN != before.overlayN ||
		after.late
	r.actions = after.actions != before.actions
	r.keys = after.regs != before.regs || r.actions
	r.regs = append(r.regs[:0], rt.nextRegs[before.regs:]...)
	r.dividers = append(r.dividers[:0], c.dividers[before.dividers:]...)
	r.transitions = append(r.transitions[:0], c.transitions[before.transitions:]...)
	r.binds = append(r.binds[:0], rt.bindLog[before.binds:]...)
	if savedRegion == nil {
		clear(rt.bindLog)
		rt.bindLog = rt.bindLog[:0]
	}
	clear(r.nodes)
	clear(r.states)
	r.nodes, r.states = r.nodes[:0], r.states[:0]
	r.texts, r.observers, r.selectables = false, false, false
	for ch := e.first; ch != nil; ch = ch.next {
		r.collect(ch)
	}
	if rt.focused != 0 {
		for _, s := range r.states {
			if s.id == rt.focused {
				r.focus = true
			}
		}
	}
	r.valid = !memoOff
	rt.memoBuilt++
}

// collect notes the elements inside the region, and what of them the
// frame walks for.
func (r *memoRegion) collect(e *node) {
	r.nodes = append(r.nodes, e)
	r.states = append(r.states, e.st)
	switch {
	case e.kind == kindText:
		r.texts = true
		if e.flags&flagSelectable != 0 || e.st.editor != nil {
			r.selectables = true
		}
	case e.flags&flagSelectable != 0:
		r.selectables = true
	}
	if e.observeLayout != nil || e.offsetAfterLayout != nil {
		r.observers = true
	}
	// What refers to elements outside the region, or holds what lists,
	// tables, toolbars and forms make as they lay out, is not kept.
	if e.list != nil || e.rowsOf != nil || e.overflow != nil || e.form != nil || e.field != nil || e.colFit != nil ||
		e.gridFit != nil || e.baselines || e.flags&(flagModal|flagPage) != 0 {
		r.live = true
	}
	for _, p := range [...]*node{e.popover, e.nameFrom, e.activeDescendant, e.followX} {
		if p != nil && !r.holds(p) {
			r.live = true
		}
	}
	for ch := e.first; ch != nil; ch = ch.next {
		r.collect(ch)
	}
}

// holds reports whether e is inside the region, as it is being collected:
// one of its ancestors is the region's element.
func (r *memoRegion) holds(e *node) bool {
	for p := e.parent; p != nil; p = p.parent {
		if p.id == r.id && p.memoOn {
			return true
		}
	}
	return false
}

// alloc returns a zeroed element from the region's arena.
func (r *memoRegion) alloc() *node {
	ci, ei := r.used/chunkSize, r.used%chunkSize
	if ci == len(r.chunks) {
		r.chunks = append(r.chunks, make([]node, chunkSize))
	}
	r.used++
	return &r.chunks[ci][ei]
}

// trim frees what elements of the region's builds before referred to.
func (r *memoRegion) trim() {
	r.chunks = trimArena(r.chunks, r.used, max(r.spent, r.used))
	r.spent = r.used
}

// release lets go of everything the region keeps.
func (r *memoRegion) release() {
	for _, ch := range r.chunks {
		clear(ch)
	}
	*r = memoRegion{id: r.id}
}

// memoLayoutOf returns what of e the layout of its children depends on.
func memoLayoutOf(e *node) memoLayout {
	return memoLayout{row: e.row, wrap: e.wrap, reverse: e.reverse, wrapReverse: e.wrapReverse, grid: e.grid,
		scrollOverflowX: e.scrollOverflowX, justify: e.justify, align: e.align, alignContent: e.alignContent,
		justifyItems: e.justifyItems, gapX: e.gapX, gapY: e.gapY, pad: e.pad, border: e.border,
		flags: e.flags & (flagScrollX | flagScrollY | flagInvisible), ts: e.resolvedText()}
}

// keptLayout lays out the children of e, a region kept in the last pass,
// as the last frame did, when e has the size and the properties it had:
// they go back where the layout put them, relative to their parents, for
// place. It reports false when they need laying out.
func keptLayout(e *node, w, h float32) bool {
	rt := e.c.rt
	r := rt.memos[e.id]
	if r == nil || !r.reused || !r.laid || r.seen != rt.frame || r.first != e.first || e.grid && len(e.cols)+len(e.rows) > 0 {
		return false
	}
	if r.lw != w || r.lh != h || r.lprops != memoLayoutOf(e) {
		// Laid out anew: what its elements measured at the old size goes.
		for _, n := range r.nodes {
			n.nmeasure = 0
		}
		return false
	}
	for _, n := range r.nodes {
		n.x, n.y = n.rx, n.ry
	}
	if e.scrolls() {
		e.contentW, e.contentH = r.contentW, r.contentH
	}
	rt.memoLaid++
	return true
}

// noteLayout notes the layout of the children of e, a memo region, which
// the next frame may keep.
func noteLayout(e *node, w, h float32) {
	rt := e.c.rt
	r := rt.memos[e.id]
	if r == nil || r.seen != rt.frame || r.first != e.first {
		return
	}
	r.laid, r.lw, r.lh, r.lprops = true, w, h, memoLayoutOf(e)
	r.contentW, r.contentH = e.contentW, e.contentH
}

// memoInput marks the regions an event reaches, which build again in the
// next frame: those under the pointer, before and after it moved, those
// holding the focus for keys, the element assistive technology acts on,
// and all of them for what may reach any.
func (rt *engine) memoInput(ev platform.SurfaceEvent) {
	if len(rt.memos) == 0 {
		return
	}
	switch ev.Kind {
	case platform.SurfaceFrame, platform.SurfaceResize, platform.SurfaceShown, platform.AccessibilityOn:
		return
	case platform.PointerMove:
		// Handled selectively by pointerMove / setHover so moving within the
		// same element without hover/track/drag changes never busts Memo.
		return
	case platform.PointerLeave:
		for _, id := range rt.hover {
			if s := rt.states[id]; s != nil && s.memo != 0 {
				rt.memoDirty(s.memo)
			}
		}
		if rt.pressed != nil && rt.pressed.memo != 0 {
			rt.memoDirty(rt.pressed.memo)
		}
		return
	case platform.PointerScroll:
		// Scroll dirties only the scroll container's region (handled by scroll)
		// unless a custom input handler intercepts it; skip busting unrelated regions.
		x, y := float32(ev.X), float32(ev.Y)
		for _, r := range rt.memos {
			if r.bounds.Contains(x, y) && (ev.Precise || rt.isHeadless() && !rt.smoothWheelInTest || rt.preferences().ReduceMotion) {
				rt.memoDirty(r.id)
			}
		}
		return
	case platform.PointerDown, platform.PointerUp:
		x, y := float32(ev.X), float32(ev.Y)
		for _, r := range rt.memos {
			if r.bounds.Contains(x, y) || rt.pointerIn && r.bounds.Contains(rt.pointerX, rt.pointerY) {
				rt.memoDirty(r.id)
			}
		}
	case platform.TextInput, platform.TextComposition, platform.SurfaceCommand:
		// Text editing only targets the focused element's region; never bust
		// unrelated sibling regions that happen to register shortcuts.
		if s := rt.states[rt.focused]; s != nil {
			rt.memoDirty(s.memo)
		}
	case platform.KeyPressed, platform.KeyReleased, platform.ModifiersChanged:
		if s := rt.states[rt.focused]; s != nil {
			rt.memoDirty(s.memo)
		}
		k, m := Key(ev.Key), Modifiers(ev.Mods)
		for _, r := range rt.memos {
			if !r.keys {
				continue
			}
			if r.actions || ev.Kind == platform.ModifiersChanged {
				rt.memoDirty(r.id)
				continue
			}
			for _, reg := range r.regs {
				if reg.key == k && reg.mods == m {
					rt.memoDirty(r.id)
					break
				}
			}
		}
	case platform.AccessAction:
		if s := rt.states[ev.ID]; s != nil {
			rt.memoDirty(s.memo)
		}
	default:
		for _, r := range rt.memos {
			r.dirty = true
		}
	}
}

// memoDirty marks the region id, and those it is in, to build again.
func (rt *engine) memoDirty(id uint64) {
	for n := 0; id != 0 && n < 64; n++ {
		r := rt.memos[id]
		if r == nil {
			return
		}
		r.dirty = true
		id = r.parent
	}
}

// memoHover marks the regions of the elements the pointer actually enters or
// leaves (the symmetric difference of from and to), leaving unchanged ancestor
// or sibling regions clean.
func (rt *engine) memoHover(from, to []uint64) {
	if len(rt.memos) == 0 {
		return
	}
	for _, id := range from {
		if !containsID(to, id) {
			if s := rt.states[id]; s != nil && s.memo != 0 {
				rt.memoDirty(s.memo)
			}
		}
	}
	for _, id := range to {
		if !containsID(from, id) {
			if s := rt.states[id]; s != nil && s.memo != 0 {
				rt.memoDirty(s.memo)
			}
		}
	}
}

func containsID(list []uint64, id uint64) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

// pruneMemos lets go of the regions the frame did not build or keep.
func (rt *engine) pruneMemos() {
	for id, r := range rt.memos {
		if r.seen != rt.frame || r.seenPass != rt.pass {
			r.release()
			delete(rt.memos, id)
		}
	}
}

// memoWalks reports whether a walk of the frame goes into the children of
// e, a memo region: those of a region kept in this pass only when they hold
// what it looks for, selectable texts or layout observers.
func (rt *engine) memoWalks(e *node, selectable bool) bool {
	r := rt.memos[e.id]
	if r == nil || !r.reused || r.seen != rt.frame || r.seenPass != rt.pass || r.first != e.first {
		return true
	}
	if selectable {
		return r.selectables
	}
	return r.observers
}

// commitMarks are the lengths of what commit makes, before the children
// of a memo region commit, and what they may change.
type commitMarks struct {
	hits, focus, labels, groups int
	modal, modalLayer           uint64
	scope                       focusScope
}

func (rt *engine) commitMarks() commitMarks {
	return commitMarks{hits: len(rt.hits), focus: len(rt.focusOrder), labels: len(rt.labels), groups: len(rt.groups),
		modal: rt.modal, modalLayer: rt.modalLayer, scope: rt.commitScope}
}

// keptHere returns the region of e when this pass kept it, with the
// children it kept.
func (rt *engine) keptHere(e *node) *memoRegion {
	r := rt.memos[e.id]
	if r == nil || !r.reused || r.seen != rt.frame || r.seenPass != rt.pass || r.first != e.first {
		return nil
	}
	return r
}

// commitKept commits the children of e, a region this pass kept, as the
// last frame did, when nothing they depend on moved: their states are as
// that frame left them, and what they added to the hit list, the focus
// order and the labels goes there again.
func (rt *engine) commitKept(e *node, clip Rect) bool {
	r := rt.keptHere(e)
	if r == nil || !r.committed || !r.plain || r.cx != e.x || r.cy != e.y || r.clip != clip || !r.scrollsAsCommitted() ||
		r.scope != rt.commitScope || r.page != rt.commitPage || r.disabled != (e.st.flags&flagDisabled != 0) {
		return false
	}
	rt.hits = append(rt.hits, r.hits...)
	rt.focusOrder = append(rt.focusOrder, r.focusOrder...)
	rt.focusScopes = append(rt.focusScopes, r.focusScopes...)
	rt.labels = append(rt.labels, r.labels...)
	rt.memoCommitted++
	return true
}

// noteCommit notes what the children of e, a memo region, made as they
// committed, for the next frame to keep.
func (rt *engine) noteCommit(e *node, clip Rect, m *commitMarks) {
	r := rt.memos[e.id]
	if r == nil || r.seen != rt.frame || r.seenPass != rt.pass || r.first != e.first {
		return
	}
	r.committed = true
	r.cx, r.cy, r.clip = e.x, e.y, clip
	r.scope, r.page, r.disabled = rt.commitScope, rt.commitPage, e.st.flags&flagDisabled != 0
	r.plain = m.scope.group == 0 && rt.modal == m.modal && rt.modalLayer == m.modalLayer && len(rt.groups) == m.groups
	r.hits = append(r.hits[:0], rt.hits[m.hits:]...)
	r.focusOrder = append(r.focusOrder[:0], rt.focusOrder[m.focus:]...)
	r.focusScopes = append(r.focusScopes[:0], rt.focusScopes[m.focus:]...)
	r.labels = append(r.labels[:0], rt.labels[m.labels:]...)
	if !r.scrollsAsCommitted() {
		r.described = false
	}
	r.scrolls = r.scrolls[:0]
	if e.scrolls() {
		r.scrolls = append(r.scrolls, scrollMark{e.st, e.st.scrollX, e.st.scrollY})
	}
	for _, n := range r.nodes {
		if n.scrolls() {
			r.scrolls = append(r.scrolls, scrollMark{n.st, n.st.scrollX, n.st.scrollY})
		}
	}
	r.bounds = Rect{}
	for _, h := range r.hits {
		r.bounds = unionNonEmpty(r.bounds, h.r)
	}
}

// unionNonEmpty returns the smallest rectangle holding a and b, either
// empty.
func unionNonEmpty(a, b Rect) Rect {
	switch {
	case a.W <= 0 || a.H <= 0:
		return b
	case b.W <= 0 || b.H <= 0:
		return a
	}
	return union(a, b)
}

// describeKept describes the children of e, a region this pass kept, for
// assistive technology as the last frame did, when the region did not move
// and the focus is not in it: below the node parent, with the parents of
// those nodes made relative to where they go now.
func (rt *engine) describeKept(t *platform.AccessTree, e *node, parent int, scrolled bool) bool {
	r := rt.keptHere(e)
	if r == nil || !r.described || r.ax != e.x || r.ay != e.y || r.scrolled != scrolled || !r.committed || !r.scrollsAsCommitted() ||
		rt.focused != 0 && rt.inRegion(rt.states[rt.focused], r.id) {
		return false
	}
	start := len(t.Nodes)
	for _, n := range r.access {
		if n.Parent == -2 {
			n.Parent = parent
		} else {
			n.Parent += start
		}
		t.Nodes = append(t.Nodes, n)
	}
	rt.memoDescribed++
	return true
}

// noteDescribe notes the nodes the children of e, a memo region, gave
// assistive technology from start on, for the next frame to keep.
func (rt *engine) noteDescribe(t *platform.AccessTree, e *node, parent int, scrolled bool, start int) {
	r := rt.memos[e.id]
	if r == nil || r.seen != rt.frame || r.seenPass != rt.pass || r.first != e.first {
		return
	}
	r.described, r.ax, r.ay, r.scrolled = true, e.x, e.y, scrolled
	r.access = append(r.access[:0], t.Nodes[start:]...)
	for i := range r.access {
		if p := r.access[i].Parent; p < start {
			r.access[i].Parent = -2 // the node of the region's element, or above it
		} else {
			r.access[i].Parent = p - start
		}
	}
}

// hereNow returns the region of e when this pass built or kept it, with
// its children.
func (rt *engine) hereNow(e *node) *memoRegion {
	r := rt.memos[e.id]
	if r == nil || r.seen != rt.frame || r.seenPass != rt.pass || r.first != e.first {
		return nil
	}
	return r
}

// measuresOf returns the region of e whose measures of its children hold
// for e as it is, letting go of those that do not.
func measuresOf(e *node) *memoRegion {
	r := e.c.rt.hereNow(e)
	if r == nil {
		return nil
	}
	if p := memoLayoutOf(e); p != r.mprops {
		r.mprops, r.measured, r.heights = p, [2]bool{}, r.heights[:0]
	}
	return r
}

// memoMeasured sets *w to the max-content or min-content width of the
// children of e, a memo region, as measured before, and reports whether
// it was.
func memoMeasured(e *node, maxContent bool, w *float32) bool {
	r := measuresOf(e)
	if r == nil || !r.measured[b2i(maxContent)] {
		return false
	}
	*w = r.widths[b2i(maxContent)]
	if !maxContent && e.scrolls() {
		*w = 0
	}
	return true
}

func noteMeasured(e *node, maxContent bool, w float32) {
	if r := measuresOf(e); r != nil {
		r.widths[b2i(maxContent)], r.measured[b2i(maxContent)] = w, true
	}
}

// memoHeight returns the height of the children of e, a memo region, in a
// content box cw wide, as measured before.
func memoHeight(e *node, cw float32) (float32, bool) {
	r := measuresOf(e)
	if r == nil {
		return 0, false
	}
	for _, m := range r.heights {
		if m.availW == cw {
			return m.h, true
		}
	}
	return 0, false
}

func noteHeight(e *node, cw, h float32) {
	if r := measuresOf(e); r != nil && len(r.heights) < 8 {
		r.heights = append(r.heights, measure{availW: cw, h: h})
	}
}

// selectionScope returns what decides the selection scope of the texts of
// e's children that reach out of e: the nearest selectable box from e up,
// or the control owning their text.
func selectionScope(e *node) (scope uint64, control bool) {
	for p := e; p != nil; p = p.parent {
		if p.kind == kindBox && p.flags&flagSelectable != 0 {
			return p.id, false
		}
		if p.flags&(flagClickable|flagFocusable|flagEditable|flagUnselectable) != 0 || p.inputFn != nil {
			return p.id, true
		}
	}
	return 0, false
}

// selectionKept reports whether the texts of the children of e, a region
// this pass kept, are prepared for selection as the last frame left them.
func (rt *engine) selectionKept(e *node) bool {
	r := rt.keptHere(e)
	if r == nil || !r.selPrepared {
		return false
	}
	scope, control := selectionScope(e)
	return scope == r.selScope && control == r.selControl
}

func (rt *engine) noteSelection(e *node) {
	if r := rt.hereNow(e); r != nil {
		r.selPrepared = true
		r.selScope, r.selControl = selectionScope(e)
	}
}

// scrollMark is the offset of a scroll container as its region committed.
type scrollMark struct {
	st     *state
	sx, sy float64
}

// scrollsAsCommitted reports whether the scroll containers of the region
// have the offsets they had as it committed.
func (r *memoRegion) scrollsAsCommitted() bool {
	for _, m := range r.scrolls {
		if m.st.scrollX != m.sx || m.st.scrollY != m.sy {
			return false
		}
	}
	return true
}

// memoBind is a Handle bound to an element of a memo region.
type memoBind struct {
	ref *refState
	n   *node
}
