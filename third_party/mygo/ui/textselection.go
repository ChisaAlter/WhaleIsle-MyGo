package ui

import (
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"
)

// textPoint identifies a rune of a paragraph by its stable element ID.
type textPoint struct {
	id uint64
	at int
}

// textPointMeta records the enclosing virtual List coordinates for a paragraph
// when rendered inside a List, decoupling selection lifetime from mounted DOM nodes.
type textPointMeta struct {
	listID   uint64
	rowID    uint64
	rowIndex int
	inRow    int
}

type textCacheRowKey struct {
	listID uint64
	rowID  uint64
}

type cachedParagraph struct {
	id      uint64
	ordinal int
	text    string
}

const maxCachedSelectionRows = 2048

// textSelectionLogic holds reference-typed logical state outside the comparable
// textSelection struct so structural equality checks in read-only queries remain intact.
type textSelectionLogic struct {
	anchorKey      any
	caretKey       any
	startKeys      [2]any
	orderedRev     bool
	snapshots      map[uint64]string
	selTexts       map[uint64]string
	rowParagraphs  map[textCacheRowKey][]cachedParagraph
	rowOrder       []textCacheRowKey
	sourceRevision uint64
}

// textSelection is one selection across the paragraphs of a container.
// Each paragraph keeps its own layout and editor; syncTextSelection projects
// the shared endpoints onto those editors for painting.
type textSelection struct {
	scope         uint64
	anchor, caret textPoint
	start         [2]textPoint
	anchorMeta    textPointMeta
	caretMeta     textPointMeta
	startMeta     [2]textPointMeta
	unit          int
	dragging      bool
	desiredX      float32
	hasDesired    bool
	logic         *textSelectionLogic
}

// prepareSelectable waits until inline children have finished their text,
// so rebuilding a paragraph does not reset its selection to its own text.
func (rt *engine) prepareSelectable(e *node) {
	e.prepareSelectable()
	if e.memoOn {
		if rt.selectionKept(e) {
			return // a region kept, within what it was
		}
		defer rt.noteSelection(e)
	}
	for ch := e.first; ch != nil; ch = ch.next {
		rt.prepareSelectable(ch)
	}
}

func (e *node) prepareSelectable() {
	if e.kind != kindText {
		return
	}
	if e.isInline() {
		e.st.textScope = 0
		return
	}
	scope := uint64(0)
	if e.flags&(flagClickable|flagFocusable|flagUnselectable) == 0 && e.inputFn == nil {
		for p := e.parent; p != nil; p = p.parent {
			if p.kind == kindBox && p.flags&flagSelectable != 0 {
				scope = p.id
				break
			}
			if p.flags&(flagClickable|flagFocusable|flagEditable|flagUnselectable) != 0 || p.inputFn != nil {
				break // a control owns its text
			}
		}
	}
	if old := e.st.textScope; old != 0 && old != scope {
		if e.c.rt.selection.scope == old && e.c.rt.textSelectionIncludes(e.id) {
			e.c.rt.selection = textSelection{}
		}
		if ed := e.st.editor; ed != nil {
			ed.anchor, ed.caret = 0, 0
		}
	}
	e.st.textScope = scope
	if scope != 0 {
		e.flags |= flagSelectable
	}
	if e.flags&flagSelectable == 0 {
		return
	}
	ed := e.st.editor
	if ed == nil {
		ed = newEditor()
		ed.readOnly, ed.multiline = true, true
		e.st.editor = ed
	}
	if ed.source != e.text || len(ed.buf.paras) == 0 {
		rt := e.c.rt
		preserved, newAnchor, newCaret := preserveStreamingParagraphSelection(rt, ed, scope, e.id, e.text)
		if !preserved {
			if scope != 0 && rt.selection.scope == scope && rt.textSelectionIncludes(e.id) {
				rt.selection = textSelection{}
			}
		} else if scope != 0 && rt.selection.scope == scope {
			rt.adjustEndpointOffsets(e.id, ed.anchor, ed.caret, newAnchor, newCaret)
		}
		ed.source = e.text
		ed.setText(e.text)
		if preserved {
			ed.anchor, ed.caret = newAnchor, newCaret
		} else {
			ed.caret, ed.anchor = 0, 0
		}
	}
	if e.c.rt.focused == e.id || len(ed.queue) > 0 {
		ed.process(e.c, e)
	}
}

// collectSelectable follows build order, independently of painting order
// (absolute children paint last). Inline children are already in their
// paragraph's text, and must not be copied a second time.
func (rt *engine) collectSelectable(e *node, hidden bool) {
	hidden = hidden || e.flags&(flagInvisible|flagInert) != 0
	if hidden {
		return // nothing inside takes part in selection
	}
	if !hidden && e.kind == kindText && !e.isInline() && e.st.flags&flagDisabled == 0 && e.flags&flagSelectable != 0 {
		rt.texts = append(rt.texts, e.st)
	}
	if e.memoOn && !rt.memoWalks(e, true) {
		return // a region kept, with nothing selectable
	}
	for ch := e.first; ch != nil; ch = ch.next {
		rt.collectSelectable(ch, hidden)
	}
}

func (rt *engine) textPointIndex(p textPoint) int {
	for i, s := range rt.texts {
		if s.id == p.id && s.textScope == rt.selection.scope {
			return i
		}
	}
	return -1
}

func (rt *engine) compareTextPoints(a, b textPoint) int {
	i, j := rt.textPointIndex(a), rt.textPointIndex(b)
	if i != j {
		return i - j
	}
	return a.at - b.at
}

func (rt *engine) textSelectionBounds() (a, b textPoint, selected bool) {
	if rt.selection.scope == 0 {
		return textPoint{}, textPoint{}, false
	}
	a, b = rt.selection.anchor, rt.selection.caret
	ia, ib := rt.textPointIndex(a), rt.textPointIndex(b)
	if ia >= 0 && ib >= 0 {
		d := rt.compareTextPoints(a, b)
		if d > 0 {
			a, b = b, a
		}
		return a, b, d != 0
	}
	if !rt.selectionVirtualized() {
		return textPoint{}, textPoint{}, false
	}
	start, startMeta, end, endMeta, ok := rt.orderedLogicalSelection()
	if !ok {
		return textPoint{}, textPoint{}, false
	}
	var firstP, lastP *state
	for _, s := range rt.texts {
		if s.textScope != rt.selection.scope || !rt.selectionStateActive(s.id) {
			continue
		}
		if rt.paragraphInLogicalRange(s, start, startMeta, end, endMeta) {
			if firstP == nil {
				firstP = s
			}
			lastP = s
		}
	}
	if firstP == nil || lastP == nil {
		return textPoint{}, textPoint{}, false
	}
	outA := textPoint{firstP.id, 0}
	if firstP.id == start.id {
		outA.at = min(start.at, firstP.editor.buf.n)
	}
	outB := textPoint{lastP.id, lastP.editor.buf.n}
	if lastP.id == end.id {
		outB.at = min(end.at, lastP.editor.buf.n)
	}
	nonCollapsed := rt.selectionLogicalNonCollapsed(start, startMeta, end, endMeta) && (outA != outB || outB.at > outA.at)
	return outA, outB, nonCollapsed
}

func (rt *engine) textSelectionIncludes(id uint64) bool {
	if rt.selection.scope == 0 {
		return false
	}
	a, b, _ := rt.textSelectionBounds()
	if a.id != 0 {
		i := rt.textPointIndex(textPoint{id: id})
		if i >= 0 && i >= rt.textPointIndex(a) && i <= rt.textPointIndex(b) {
			return true
		}
	}
	if !rt.selectionVirtualized() {
		return false
	}
	s := rt.states[id]
	if s == nil || s.textScope != rt.selection.scope || !rt.selectionStateActive(id) {
		return false
	}
	start, startMeta, end, endMeta, ok := rt.orderedLogicalSelection()
	return ok && rt.paragraphInLogicalRange(s, start, startMeta, end, endMeta)
}

func (rt *engine) textSelectionVisible(s *state) bool {
	if !rt.windowFocused || s.flags&flagDisabled != 0 {
		return false
	}
	if s.textScope == 0 {
		return rt.focused == s.id
	}
	if s.textScope != rt.selection.scope {
		return false
	}
	if f := rt.states[rt.focused]; f != nil {
		return (f.textScope == s.textScope || f.id == s.textScope) && f.flags&flagDisabled == 0
	}
	return rt.focused == 0 && rt.selectionVirtualized()
}

func (rt *engine) syncTextSelection() {
	for _, s := range rt.texts {
		if s.textScope != 0 {
			s.editor.anchor, s.editor.caret = 0, 0
		}
	}
	if rt.selection.scope == 0 {
		return
	}
	if !rt.reconcileSelectionEndpoints() {
		// An endpoint was removed, hidden, disabled, or moved to another
		// container. Never let a recycled state inherit the selection.
		rt.selection = textSelection{}
		return
	}
	if rt.selection.dragging && rt.pressed != nil {
		rt.moveTextSelection(rt.pointerX, rt.pointerY)
		rt.captureSelectionEndpoints()
	}
	start, startMeta, end, endMeta, ok := rt.orderedLogicalSelection()
	if !ok {
		return
	}
	rt.cacheMountedSelectedRows(start, startMeta, end, endMeta)
	for _, s := range rt.texts {
		if s.textScope != rt.selection.scope || !rt.selectionStateActive(s.id) {
			continue
		}
		from, to, hit := rt.paragraphSliceInLogicalRange(s, start, startMeta, end, endMeta)
		if hit {
			s.editor.anchor, s.editor.caret = from, to
		}
	}
}

// textSelectionAt finds the nearest visible paragraph in scope. Gaps,
// padding, and a pointer dragged outside the container still have an
// endpoint; the editor's hit test clamps it to that paragraph's text.
func (rt *engine) textSelectionAt(scope uint64, x, y float32) textPoint {
	// Respect the topmost hit, including a link inside a rich paragraph.
	for _, id := range rt.hitChain(x, y) {
		if s := rt.states[id]; s != nil && s.textScope == scope && s.editor != nil && s.flags&flagDisabled == 0 {
			return textHit(s, x, y)
		}
	}
	var nearest *state
	distance := inf
	for _, s := range rt.texts {
		if s.textScope != scope || s.vw <= 0 || s.vh <= 0 || !s.editor.laidOut() {
			continue
		}
		for _, line := range s.editor.layout.Lines {
			r := intersect(Rect{s.x + s.editor.originX + line.X, s.y + s.editor.originY + line.Y, line.Width, line.Height}, Rect{s.vx, s.vy, s.vw, s.vh})
			if r.W <= 0 || r.H <= 0 {
				continue
			}
			dx, dy := max(r.X-x, x-r.X-r.W, 0), max(r.Y-y, y-r.Y-r.H, 0)
			if d := dx*dx + dy*dy; nearest == nil || d < distance {
				nearest, distance = s, d
			}
		}
	}
	if nearest == nil {
		return textPoint{}
	}
	return textHit(nearest, x, y)
}

func textHit(s *state, x, y float32) textPoint {
	ed := s.editor
	at := ed.hit(x-s.x, y-s.y)
	if y < s.y+ed.originY {
		at = 0
	} else if y >= s.y+ed.originY+ed.layout.Height {
		at = ed.buf.n
	}
	return textPoint{s.id, at}
}

// scrollTextSelection follows a drag near a scroll container's edges. The
// captured paragraph identifies the scroll container even outside its clip;
// after layout, syncTextSelection hit-tests the newly visible text.
func (rt *engine) scrollTextSelection() {
	if !rt.selection.dragging || rt.pressed == nil {
		return
	}
	const edge = 32
	for s := rt.pressed; s != nil; s = rt.states[s.parent] {
		if s.flags&flagScrollY == 0 {
			continue
		}
		var dy float32
		switch top, bottom := rt.pointerY-s.vy, s.vy+s.vh-rt.pointerY; {
		case top < edge:
			dy = -min(edge-top, edge) / 2
		case bottom < edge:
			dy = min(edge-bottom, edge) / 2
		}
		if dy != 0 && scrollBy(s, 0, dy) {
			rt.animating = true
		}
		return
	}
}

func textUnit(s *state, at, unit int) (int, int) {
	switch unit {
	case 2:
		return s.editor.buf.wordAt(at)
	case 3:
		return s.editor.lineEdges(at)
	}
	return at, at
}

func (rt *engine) pressTextSelection(s *state, x, y float32, mods Modifiers, clicks int) bool {
	if s.flags&flagDisabled != 0 {
		return false
	}
	scope := s.textScope
	if s.editor == nil && s.flags&flagSelectable != 0 {
		scope = s.id
	}
	if scope == 0 {
		return false
	}
	for _, id := range rt.hitChain(x, y) {
		if id == scope {
			break
		}
		if p := rt.states[id]; p != nil && p.flags&flagUnselectable != 0 {
			return false
		}
	}
	p := rt.textSelectionAt(scope, x, y)
	if p.id == 0 {
		return false
	}
	t := &rt.selection
	if t.scope != scope || mods&Shift == 0 || !rt.selectionPointAlive(t.anchor, t.anchorMeta, rt.anchorKey()) {
		*t = textSelection{scope: scope, anchor: p, caret: p}
	}
	t.caret, t.unit, t.dragging, t.hasDesired = p, min(clicks, 3), true, false
	if t.unit > 1 {
		a, b := textUnit(rt.states[p.id], p.at, t.unit)
		t.anchor, t.caret = textPoint{p.id, a}, textPoint{p.id, b}
	}
	t.start = [2]textPoint{t.anchor, t.caret}
	rt.captureSelectionEndpoints()
	// A press in padding starts in the nearest paragraph, which keeps the
	// focus and pointer capture until the release.
	s.pressed = false
	s = rt.states[p.id]
	rt.focused, rt.focusVisible = s.id, false
	rt.pressed = s
	s.pressed, s.pressMods = true, mods
	s.editor.dragging = false
	return true
}

func (rt *engine) moveTextSelection(x, y float32) {
	t := &rt.selection
	p := rt.textSelectionAt(t.scope, x, y)
	if p.id == 0 {
		return
	}
	if t.unit > 1 {
		a, b := textUnit(rt.states[p.id], p.at, t.unit)
		if rt.compareMountedToSavedStart(p, 0) < 0 {
			t.anchor, t.caret = t.start[1], textPoint{p.id, a}
			t.anchorMeta = t.startMeta[1]
			if t.logic != nil {
				t.logic.anchorKey = t.logic.startKeys[1]
			}
		} else {
			t.anchor, t.caret = t.start[0], textPoint{p.id, b}
			t.anchorMeta = t.startMeta[0]
			if t.logic != nil {
				t.logic.anchorKey = t.logic.startKeys[0]
			}
			if rt.compareMountedToSavedStart(t.caret, 1) < 0 {
				t.caret = t.start[1]
				t.caretMeta = t.startMeta[1]
				if t.logic != nil {
					t.logic.caretKey = t.logic.startKeys[1]
				}
			}
		}
	} else {
		t.caret = p
	}
	rt.captureSelectionEndpoints()
}

func (rt *engine) textSelectionCommand(s *state, name string) bool {
	if s.textScope == 0 {
		return false
	}
	if s.flags&flagDisabled != 0 {
		return true
	}
	switch name {
	case "selectAll":
		if rt.selectAllListText(s.textScope) {
			break
		}
		var first, last *state
		for _, p := range rt.texts {
			if p.textScope == s.textScope {
				if first == nil {
					first = p
				}
				last = p
			}
		}
		if first != nil {
			rt.selection = textSelection{scope: s.textScope, anchor: textPoint{first.id, 0}, caret: textPoint{last.id, last.editor.buf.n}}
			rt.captureSelectionEndpoints()
		}
	case "copy":
		if rt.selection.scope != s.textScope {
			break
		}
		if !rt.reconcileSelectionEndpoints() {
			rt.selection = textSelection{}
			break
		}
		start, startMeta, end, endMeta, ok := rt.orderedLogicalSelection()
		if !ok || !rt.selectionLogicalNonCollapsed(start, startMeta, end, endMeta) {
			break
		}
		if ls, source := rt.selectionTextSource(s.textScope, startMeta.listID); source && startMeta.listID == endMeta.listID {
			if text, valid := rt.copyListText(ls, start, startMeta, end, endMeta); valid {
				rt.host.writeClipboard(text)
			}
			break
		}
		if first, last := rt.textPointIndex(start), rt.textPointIndex(end); first >= 0 && last >= 0 {
			var out strings.Builder
			wrote := false
			for i := first; i <= last; i++ {
				p := rt.texts[i]
				if p.textScope != s.textScope || !rt.selectionStateActive(p.id) {
					continue
				}
				from, to := 0, p.editor.buf.n
				if i == first {
					from = min(start.at, to)
				}
				if i == last {
					to = min(end.at, to)
				}
				if wrote {
					out.WriteByte('\n')
				}
				out.WriteString(p.editor.buf.slice(from, to))
				wrote = true
			}
			rt.host.writeClipboard(out.String())
			break
		}
		if text, complete := rt.copyVirtualizedSelection(s.textScope, start, startMeta, end, endMeta); complete {
			rt.host.writeClipboard(text)
		}
	}
	return true // other editing commands cannot change selectable text
}

func (rt *engine) neighboringText(p textPoint, direction int) *state {
	for i := rt.textPointIndex(p) + direction; i >= 0 && i < len(rt.texts); i += direction {
		if s := rt.texts[i]; s.textScope == rt.selection.scope {
			return s
		}
	}
	return nil
}

func (rt *engine) textSelectionKey(s *state, k keyEvent) bool {
	if s.textScope == 0 {
		return false
	}
	if s.flags&flagDisabled != 0 {
		return true
	}
	if k.mods == Cmd && (k.key == KeyA || k.key == KeyC) {
		name := "copy"
		if k.key == KeyA {
			name = "selectAll"
		}
		return rt.textSelectionCommand(s, name)
	}
	t := &rt.selection
	if t.scope != s.textScope || rt.textPointIndex(t.caret) < 0 {
		p := textPoint{s.id, s.editor.caret}
		*t = textSelection{scope: s.textScope, anchor: p, caret: p}
		rt.captureSelectionEndpoints()
	}
	s = rt.states[t.caret.id]
	ed, at := s.editor, t.caret.at
	mods := k.mods &^ Shift
	if (runtime.GOOS == "darwin" && mods == Cmd && (k.key == KeyUp || k.key == KeyDown)) ||
		(mods == Ctrl && (k.key == KeyHome || k.key == KeyEnd)) {
		direction := 1
		if k.key == KeyUp || k.key == KeyHome {
			direction = -1
		}
		for next := rt.neighboringText(textPoint{s.id, 0}, direction); next != nil; next = rt.neighboringText(textPoint{s.id, 0}, direction) {
			s = next
		}
		at = s.editor.buf.n
		if direction < 0 {
			at = 0
		}
		t.caret, t.hasDesired = textPoint{s.id, at}, false
		rt.captureSelectionEndpoints()
		return true
	}
	if k.key == KeyUp || k.key == KeyDown {
		x, y, h := ed.layout.Caret(at)
		if !t.hasDesired {
			t.desiredX, t.hasDesired = s.x+ed.originX+x, true
		}
		direction := 1
		if k.key == KeyUp {
			direction = -1
		}
		y += h/2 + float32(direction)*h
		if y < 0 || y >= ed.layout.Height {
			if next := rt.neighboringText(t.caret, direction); next != nil {
				s, ed = next, next.editor
				y = 0
				if direction < 0 {
					y = ed.layout.Height - 1
				}
			} else {
				at = ed.buf.n
				if direction < 0 {
					at = 0
				}
				t.caret = textPoint{s.id, at}
				rt.captureSelectionEndpoints()
				return true
			}
		}
		t.caret = textPoint{s.id, ed.layout.IndexAt(t.desiredX-s.x-ed.originX, y)}
		rt.captureSelectionEndpoints()
		return true
	}
	t.hasDesired = false
	if (k.key == KeyLeft && at == 0 || k.key == KeyRight && at == ed.buf.n) && !(runtime.GOOS == "darwin" && mods == Cmd) {
		direction := 1
		if k.key == KeyLeft {
			direction = -1
		}
		if next := rt.neighboringText(t.caret, direction); next != nil {
			at = 0
			if direction < 0 {
				at = next.editor.buf.n
			}
			t.caret = textPoint{next.id, at}
			rt.captureSelectionEndpoints()
			return true
		}
	}
	ed.caret = at
	ed.key(&rt.c, s, editEvent{kind: editKey, mods: k.mods, key: k.key})
	t.caret = textPoint{s.id, ed.caret}
	rt.captureSelectionEndpoints()
	return true
}

func (rt *engine) textSelectionMenu(s *state, x, y float32) bool {
	if s.textScope == 0 {
		return false
	}
	p := textHit(s, x, y)
	a, b, selected := rt.textSelectionBounds()
	if !rt.textSelectionVisible(s) || !selected || rt.compareTextPoints(p, a) < 0 || rt.compareTextPoints(p, b) > 0 {
		rt.selection = textSelection{scope: s.textScope, anchor: p, caret: p}
		rt.captureSelectionEndpoints()
	}
	rt.focused, rt.focusVisible = s.id, false
	return true
}

func (rt *engine) selectionVirtualized() bool {
	return rt.selection.anchorMeta.listID != 0 || rt.selection.caretMeta.listID != 0
}

func (rt *engine) ensureSelectionLogic() *textSelectionLogic {
	if rt.selection.logic == nil {
		rt.selection.logic = &textSelectionLogic{
			snapshots:     map[uint64]string{},
			selTexts:      map[uint64]string{},
			rowParagraphs: map[textCacheRowKey][]cachedParagraph{},
		}
	}
	return rt.selection.logic
}

func (rt *engine) anchorKey() any {
	if rt.selection.logic == nil {
		return nil
	}
	return rt.selection.logic.anchorKey
}

func (rt *engine) listAlive(listID uint64) (*ListState, bool) {
	if listID == 0 {
		return nil, false
	}
	st := rt.states[listID]
	if st == nil || st.list == nil || !rt.selectionStateActive(listID) {
		return nil, false
	}
	ls := st.list
	if st.seen != rt.frame || st.pass != rt.pass || ls.frame.frame != rt.frame || ls.frame.pass != rt.pass {
		return nil, false
	}
	return ls, true
}

func resolveListKey(ls *ListState, near int, key any) (int, bool) {
	n := ls.frame.n
	if n <= 0 {
		return -1, false
	}
	if ls.Key == nil {
		if near >= 0 && near < n {
			return near, true
		}
		return -1, false
	}
	if key == nil {
		return -1, false
	}
	if near >= 0 && near < n && ls.Key(near) == key {
		return near, true
	}
	if idx, ok := ls.find(key, max(0, min(near, n-1)), n); ok {
		return idx, true
	}
	for i := 0; i < n; i++ {
		if ls.Key(i) == key {
			return i, true
		}
	}
	return -1, false
}

func (rt *engine) paragraphRowMeta(p *state) (textPointMeta, any, bool) {
	if p == nil {
		return textPointMeta{}, nil, false
	}
	for cur := p; cur != nil && cur.parent != 0; {
		par := rt.states[cur.parent]
		if par == nil {
			break
		}
		if par.list != nil {
			meta := textPointMeta{listID: par.id, rowID: cur.id, rowIndex: -1}
			var key any
			if r, ok := par.list.rows[cur.id]; ok {
				meta.rowIndex = r.row
				key = r.key
			} else if par.list.Key == nil {
				meta.rowIndex = -1
			}
			ord := 0
			for _, t := range rt.texts {
				if t == p {
					meta.inRow = ord
					return meta, key, true
				}
				if t.textScope == p.textScope && rt.within(t.id, cur.id) {
					ord++
				}
			}
			return meta, key, true
		}
		cur = par
	}
	return textPointMeta{}, nil, false
}

func (rt *engine) stateInTexts(id uint64) *state {
	for _, s := range rt.texts {
		if s.id == id && (rt.selection.scope == 0 || s.textScope == rt.selection.scope) {
			return s
		}
	}
	return nil
}

func (rt *engine) selectionPointAlive(pt textPoint, meta textPointMeta, key any) bool {
	if idx := rt.textPointIndex(pt); idx >= 0 {
		return rt.selectionStateActive(pt.id)
	}
	if meta.listID == 0 {
		return false
	}
	ls, ok := rt.listAlive(meta.listID)
	if !ok {
		return false
	}
	if ls.TextSource.Paragraphs != nil && rt.selection.scope == meta.listID {
		row, found := resolveListKey(ls, meta.rowIndex, key)
		return found && meta.inRow >= 0 && meta.inRow < len(ls.TextSource.Paragraphs(row))
	}
	if _, rowMounted := ls.rows[meta.rowID]; rowMounted {
		return false
	}
	_, found := resolveListKey(ls, meta.rowIndex, key)
	return found
}

func (rt *engine) captureSelectionEndpoints() {
	t := &rt.selection
	if t.scope == 0 {
		return
	}
	logic := rt.ensureSelectionLogic()
	if s := rt.stateInTexts(t.anchor.id); s != nil {
		if meta, key, ok := rt.paragraphRowMeta(s); ok {
			t.anchorMeta, logic.anchorKey = meta, key
		} else {
			t.anchorMeta, logic.anchorKey = textPointMeta{}, nil
		}
		if s.editor != nil {
			logic.snapshots[s.id] = s.editor.source
		}
	}
	if s := rt.stateInTexts(t.caret.id); s != nil {
		if meta, key, ok := rt.paragraphRowMeta(s); ok {
			t.caretMeta, logic.caretKey = meta, key
		} else {
			t.caretMeta, logic.caretKey = textPointMeta{}, nil
		}
		if s.editor != nil {
			logic.snapshots[s.id] = s.editor.source
		}
	}
	for k := range 2 {
		if s := rt.stateInTexts(t.start[k].id); s != nil {
			if meta, key, ok := rt.paragraphRowMeta(s); ok {
				t.startMeta[k], logic.startKeys[k] = meta, key
			} else {
				t.startMeta[k], logic.startKeys[k] = textPointMeta{}, nil
			}
		}
	}
	ia, ib := rt.textPointIndex(t.anchor), rt.textPointIndex(t.caret)
	if ia >= 0 && ib >= 0 {
		logic.orderedRev = ia > ib || (ia == ib && t.anchor.at > t.caret.at)
	} else if t.anchorMeta.listID != 0 && t.anchorMeta.listID == t.caretMeta.listID {
		d := compareMetaPoints(t.anchor, t.anchorMeta, t.caret, t.caretMeta, logic.orderedRev)
		logic.orderedRev = d > 0
	}
	if ls, ok := rt.selectionTextSource(t.scope, t.anchorMeta.listID); ok {
		logic.sourceRevision = ls.TextSource.Revision
	}
}

func (rt *engine) compareMountedToSavedStart(p textPoint, idx int) int {
	t := &rt.selection
	if rt.textPointIndex(t.start[idx]) >= 0 {
		return rt.compareTextPoints(p, t.start[idx])
	}
	ps := rt.stateInTexts(p.id)
	if ps == nil {
		return 0
	}
	pMeta, _, ok := rt.paragraphRowMeta(ps)
	sMeta := t.startMeta[idx]
	if !ok || pMeta.listID == 0 || pMeta.listID != sMeta.listID {
		return 0
	}
	return compareMetaPoints(p, pMeta, t.start[idx], sMeta, false)
}

func compareMetaPoints(a textPoint, aMeta textPointMeta, b textPoint, bMeta textPointMeta, samePosRev bool) int {
	if aMeta.rowIndex != bMeta.rowIndex {
		if aMeta.rowIndex < bMeta.rowIndex {
			return -1
		}
		return 1
	}
	if aMeta.inRow != bMeta.inRow {
		if aMeta.inRow < bMeta.inRow {
			return -1
		}
		return 1
	}
	if a.id != b.id {
		if samePosRev {
			return 1
		}
		return -1
	}
	return a.at - b.at
}

func (rt *engine) reconcileSelectionEndpoints() bool {
	t := &rt.selection
	if sc := rt.states[t.scope]; sc != nil && sc.seen == rt.frame && sc.pass == rt.pass && sc.flags&(flagInvisible|flagInert|flagDisabled) != 0 {
		return false
	}
	logic := rt.ensureSelectionLogic()
	origAnchorID, origCaretID := t.anchor.id, t.caret.id
	rebound := map[uint64]*state{}
	reconcileOne := func(pt *textPoint, origOtherID uint64, meta *textPointMeta, keyPtr *any) bool {
		if ls, ok := rt.selectionTextSource(t.scope, meta.listID); ok {
			return rt.reconcileListTextPoint(ls, pt, meta, *keyPtr)
		}
		if idx := rt.textPointIndex(*pt); idx >= 0 {
			s := rt.texts[idx]
			if !rt.selectionStateActive(s.id) {
				return false
			}
			if m, key, ok := rt.paragraphRowMeta(s); ok {
				*meta, *keyPtr = m, key
			} else {
				*meta, *keyPtr = textPointMeta{}, nil
			}
			if s.editor != nil {
				logic.snapshots[s.id] = s.editor.source
			}
			return true
		}
		if s, ok := rebound[pt.id]; ok && s != nil && s.editor != nil {
			oldText := logic.snapshots[pt.id]
			oldSel := logic.selTexts[pt.id]
			rebindPointOffsets(pt, s, oldText, oldSel)
			if m, key, ok := rt.paragraphRowMeta(s); ok {
				*meta, *keyPtr = m, key
			} else {
				*meta, *keyPtr = textPointMeta{}, nil
			}
			return true
		}
		if meta.listID != 0 {
			ls, ok := rt.listAlive(meta.listID)
			if !ok {
				return false
			}
			if _, rowMounted := ls.rows[meta.rowID]; rowMounted {
				oldID := pt.id
				if rt.rebindReflowedPoint(pt, origOtherID, meta, keyPtr) {
					rebound[oldID] = rt.stateInTexts(pt.id)
					return true
				}
				return false
			}
			rowIdx, found := resolveListKey(ls, meta.rowIndex, *keyPtr)
			if !found {
				return false
			}
			meta.rowIndex = rowIdx
			return true
		}
		oldID := pt.id
		if rt.rebindReflowedPoint(pt, origOtherID, meta, keyPtr) {
			rebound[oldID] = rt.stateInTexts(pt.id)
			return true
		}
		return false
	}
	if !reconcileOne(&t.anchor, origCaretID, &t.anchorMeta, &logic.anchorKey) {
		return false
	}
	if !reconcileOne(&t.caret, origAnchorID, &t.caretMeta, &logic.caretKey) {
		return false
	}
	for k := range 2 {
		if t.start[k].id == t.anchor.id {
			t.startMeta[k] = t.anchorMeta
			logic.startKeys[k] = logic.anchorKey
		} else if t.start[k].id == t.caret.id {
			t.startMeta[k] = t.caretMeta
			logic.startKeys[k] = logic.caretKey
		} else if t.startMeta[k].listID != 0 {
			if ls, ok := rt.listAlive(t.startMeta[k].listID); ok {
				if idx, found := resolveListKey(ls, t.startMeta[k].rowIndex, logic.startKeys[k]); found {
					t.startMeta[k].rowIndex = idx
				}
			}
		}
	}
	if ls, ok := rt.selectionTextSource(t.scope, t.anchorMeta.listID); ok {
		logic.sourceRevision = ls.TextSource.Revision
	}
	return true
}

func (rt *engine) rebindReflowedPoint(pt *textPoint, otherID uint64, meta *textPointMeta, keyPtr *any) bool {
	if rt.selection.logic == nil {
		return false
	}
	logic := rt.selection.logic
	oldText := logic.snapshots[pt.id]
	if oldText == "" {
		return false
	}
	oldSel := logic.selTexts[pt.id]
	var candidate *state
	for _, s := range rt.texts {
		if s.textScope != rt.selection.scope || !rt.selectionStateActive(s.id) || s.editor == nil {
			continue
		}
		if pt.id != otherID && s.id == otherID {
			continue
		}
		if meta.listID != 0 && !rt.within(s.id, meta.rowID) {
			continue
		}
		cur := s.editor.source
		if cur == "" {
			continue
		}
		if cur == oldText || strings.HasPrefix(cur, oldText) {
			candidate = s
			break
		}
		if oldSel != "" && (strings.HasPrefix(oldText, cur) || (utf8.RuneCountInString(oldSel) >= 2 && strings.Contains(cur, oldSel) && (strings.HasPrefix(cur, oldSel) || commonPrefixRunes([]rune(oldText), []rune(cur)) > 0))) {
			candidate = s
			break
		}
	}
	if candidate == nil {
		return false
	}
	oldID := pt.id
	rebindPointOffsets(pt, candidate, oldText, oldSel)
	if rt.focused == oldID {
		rt.focused = candidate.id
	}
	if m, key, ok := rt.paragraphRowMeta(candidate); ok {
		*meta, *keyPtr = m, key
	}
	logic.snapshots[candidate.id] = candidate.editor.source
	if oldSel != "" && candidate.id != oldID {
		logic.selTexts[candidate.id] = oldSel
	}
	return true
}

func rebindPointOffsets(pt *textPoint, candidate *state, oldText, oldSel string) {
	if candidate == nil || candidate.editor == nil {
		return
	}
	cur := candidate.editor.source
	pt.id = candidate.id
	if oldText == "" || strings.HasPrefix(cur, oldText) || strings.HasPrefix(oldText, cur) {
		pt.at = min(max(0, pt.at), candidate.editor.buf.n)
		return
	}
	if oldSel != "" {
		if oldOff, _, okOld := findPreservedSubstringRunes(oldText, oldSel); okOld {
			if newOff, _, okNew := findPreservedSubstringRunes(cur, oldSel); okNew {
				pt.at = min(max(0, pt.at+(newOff-oldOff)), candidate.editor.buf.n)
				return
			}
		}
	}
	pt.at = min(max(0, pt.at), candidate.editor.buf.n)
}

func (rt *engine) orderedLogicalSelection() (start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta, ok bool) {
	t := &rt.selection
	if t.scope == 0 {
		return textPoint{}, textPointMeta{}, textPoint{}, textPointMeta{}, false
	}
	ia, ib := rt.textPointIndex(t.anchor), rt.textPointIndex(t.caret)
	if ia >= 0 && ib >= 0 {
		d := rt.compareTextPoints(t.anchor, t.caret)
		if d > 0 {
			return t.caret, t.caretMeta, t.anchor, t.anchorMeta, true
		}
		return t.anchor, t.anchorMeta, t.caret, t.caretMeta, true
	}
	if !rt.selectionVirtualized() {
		return textPoint{}, textPointMeta{}, textPoint{}, textPointMeta{}, false
	}
	if t.anchorMeta.listID == 0 || t.anchorMeta.listID != t.caretMeta.listID {
		return textPoint{}, textPointMeta{}, textPoint{}, textPointMeta{}, false
	}
	if _, ok := rt.listAlive(t.anchorMeta.listID); !ok {
		return textPoint{}, textPointMeta{}, textPoint{}, textPointMeta{}, false
	}
	rev := t.logic != nil && t.logic.orderedRev
	d := compareMetaPoints(t.anchor, t.anchorMeta, t.caret, t.caretMeta, rev)
	if d > 0 {
		return t.caret, t.caretMeta, t.anchor, t.anchorMeta, true
	}
	return t.anchor, t.anchorMeta, t.caret, t.caretMeta, true
}

func (rt *engine) selectionLogicalNonCollapsed(start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) bool {
	if start.id != end.id {
		return start.id != 0 && end.id != 0
	}
	if startMeta.listID != endMeta.listID || startMeta.rowID != endMeta.rowID || startMeta.inRow != endMeta.inRow {
		return true
	}
	return start.at != end.at
}

func (rt *engine) paragraphSliceInLogicalRange(s *state, start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) (from, to int, hit bool) {
	ia, ib := rt.textPointIndex(start), rt.textPointIndex(end)
	if ia >= 0 && ib >= 0 {
		i := rt.textPointIndex(textPoint{id: s.id})
		if i < ia || i > ib {
			return 0, 0, false
		}
		from, to = 0, s.editor.buf.n
		if i == ia {
			from = min(start.at, to)
		}
		if i == ib {
			to = min(end.at, to)
		}
		return from, to, true
	}
	if startMeta.listID == 0 {
		return 0, 0, false
	}
	meta, _, ok := rt.paragraphRowMeta(s)
	if !ok || meta.listID != startMeta.listID {
		return 0, 0, false
	}
	cmpStart := compareRowOrder(meta, startMeta, s.id, start.id)
	cmpEnd := compareRowOrder(meta, endMeta, s.id, end.id)
	if cmpStart < 0 || cmpEnd > 0 {
		return 0, 0, false
	}
	from, to = 0, s.editor.buf.n
	if cmpStart == 0 {
		from = min(start.at, to)
	}
	if cmpEnd == 0 {
		to = min(end.at, to)
	}
	return from, to, true
}

func (rt *engine) paragraphInLogicalRange(s *state, start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) bool {
	_, _, hit := rt.paragraphSliceInLogicalRange(s, start, startMeta, end, endMeta)
	return hit
}

func compareRowOrder(pMeta, targetMeta textPointMeta, pID, targetID uint64) int {
	if pMeta.rowIndex != targetMeta.rowIndex {
		if pMeta.rowIndex < targetMeta.rowIndex {
			return -1
		}
		return 1
	}
	if pMeta.inRow != targetMeta.inRow {
		if pMeta.inRow < targetMeta.inRow {
			return -1
		}
		return 1
	}
	if pID == targetID {
		return 0
	}
	return 0
}

func (rt *engine) cacheMountedSelectedRows(start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) {
	logic := rt.ensureSelectionLogic()
	clear(logic.selTexts)
	_, hasSource := rt.selectionTextSource(rt.selection.scope, startMeta.listID)
	if hasSource {
		for id := range logic.snapshots {
			if id != rt.selection.anchor.id && id != rt.selection.caret.id && id != rt.selection.start[0].id && id != rt.selection.start[1].id {
				delete(logic.snapshots, id)
			}
		}
	}
	for _, s := range rt.texts {
		if s.textScope != rt.selection.scope || !rt.selectionStateActive(s.id) || s.editor == nil {
			continue
		}
		from, to, hit := rt.paragraphSliceInLogicalRange(s, start, startMeta, end, endMeta)
		if !hit {
			continue
		}
		endpoint := s.id == rt.selection.anchor.id || s.id == rt.selection.caret.id || s.id == rt.selection.start[0].id || s.id == rt.selection.start[1].id
		if !hasSource || endpoint {
			logic.snapshots[s.id] = s.editor.source
		}
		if to > from {
			if !hasSource || endpoint {
				logic.selTexts[s.id] = s.editor.buf.slice(from, to)
			}
		}
		if hasSource {
			continue
		}
		meta, _, inList := rt.paragraphRowMeta(s)
		if !inList {
			continue
		}
		rk := textCacheRowKey{listID: meta.listID, rowID: meta.rowID}
		if _, exists := logic.rowParagraphs[rk]; !exists {
			if len(logic.rowOrder) >= maxCachedSelectionRows {
				evict := logic.rowOrder[0]
				logic.rowOrder = logic.rowOrder[1:]
				delete(logic.rowParagraphs, evict)
			}
			logic.rowOrder = append(logic.rowOrder, rk)
		}
		if meta.inRow == 0 {
			logic.rowParagraphs[rk] = logic.rowParagraphs[rk][:0]
		}
		rowList := logic.rowParagraphs[rk]
		if meta.inRow == len(rowList) {
			logic.rowParagraphs[rk] = append(rowList, cachedParagraph{id: s.id, ordinal: meta.inRow, text: s.editor.source})
		} else {
			updated := false
			for i := range rowList {
				if rowList[i].id == s.id {
					rowList[i].text = s.editor.source
					rowList[i].ordinal = meta.inRow
					updated = true
					break
				}
			}
			if !updated {
				logic.rowParagraphs[rk] = append(rowList, cachedParagraph{id: s.id, ordinal: meta.inRow, text: s.editor.source})
			}
		}
	}
}

func (rt *engine) copyVirtualizedSelection(scope uint64, start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) (string, bool) {
	ls, ok := rt.listAlive(startMeta.listID)
	if !ok || startMeta.listID != endMeta.listID {
		return "", false
	}
	logic := rt.selection.logic
	var out strings.Builder
	wrote := false
	for r := startMeta.rowIndex; r <= endMeta.rowIndex; r++ {
		if r < 0 || r >= ls.frame.n {
			return "", false
		}
		var key any = r
		if ls.Key != nil {
			key = ls.Key(r)
		}
		rowID := keyedID(startMeta.listID, key)
		var mounted []*state
		for _, s := range rt.texts {
			if s.textScope == scope && rt.selectionStateActive(s.id) && rt.within(s.id, rowID) {
				mounted = append(mounted, s)
			}
		}
		if len(mounted) > 0 {
			for idx, p := range mounted {
				if r == startMeta.rowIndex && idx < startMeta.inRow || r == endMeta.rowIndex && idx > endMeta.inRow {
					continue
				}
				from, to := 0, p.editor.buf.n
				if r == startMeta.rowIndex && (p.id == start.id || idx == startMeta.inRow) {
					from = min(start.at, to)
				}
				if r == endMeta.rowIndex && (p.id == end.id || idx == endMeta.inRow) {
					to = min(end.at, to)
				}
				if wrote {
					out.WriteByte('\n')
				}
				if to > from {
					out.WriteString(p.editor.buf.slice(from, to))
				}
				wrote = true
			}
			continue
		}
		if logic == nil {
			return "", false
		}
		cached := logic.rowParagraphs[textCacheRowKey{listID: startMeta.listID, rowID: rowID}]
		if len(cached) == 0 {
			// Legacy lists without a logical text source can only copy ranges
			// whose full text was observed. Never publish a partial clipboard.
			return "", false
		}
		for _, cp := range cached {
			idx := cp.ordinal
			if r == startMeta.rowIndex && idx < startMeta.inRow || r == endMeta.rowIndex && idx > endMeta.inRow {
				continue
			}
			runes := []rune(cp.text)
			from, to := 0, len(runes)
			if r == startMeta.rowIndex && (cp.id == start.id || idx == startMeta.inRow) {
				from = min(max(0, start.at), to)
			}
			if r == endMeta.rowIndex && (cp.id == end.id || idx == endMeta.inRow) {
				to = min(max(0, end.at), to)
			}
			if wrote {
				out.WriteByte('\n')
			}
			if to > from {
				out.WriteString(string(runes[from:to]))
			}
			wrote = true
		}
	}
	return out.String(), true
}

func (rt *engine) adjustEndpointOffsets(id uint64, oldA, oldB, newA, newB int) {
	t := &rt.selection
	if newA == oldA && newB == oldB {
		return
	}
	oldStart, oldEnd := min(oldA, oldB), max(oldA, oldB)
	newStart, newEnd := min(newA, newB), max(newA, newB)
	if t.anchor.id == id {
		if t.anchor.at == oldStart {
			t.anchor.at = newStart
		} else if t.anchor.at == oldEnd {
			t.anchor.at = newEnd
		}
	}
	if t.caret.id == id {
		if t.caret.at == oldStart {
			t.caret.at = newStart
		} else if t.caret.at == oldEnd {
			t.caret.at = newEnd
		}
	}
	for k := range 2 {
		if t.start[k].id == id {
			if t.start[k].at == oldStart {
				t.start[k].at = newStart
			} else if t.start[k].at == oldEnd {
				t.start[k].at = newEnd
			}
		}
	}
}

func preserveStreamingParagraphSelection(rt *engine, ed *editor, scope, id uint64, newText string) (preserved bool, anchor, caret int) {
	if len(ed.buf.paras) == 0 || !utf8.ValidString(ed.source) {
		return false, 0, 0
	}
	if strings.HasPrefix(newText, ed.source) {
		return true, ed.anchor, ed.caret
	}
	selA, selB := ed.selection()
	if selA == selB && scope != 0 && rt.selection.scope == scope {
		// Scoped multi-paragraph selection may leave edge paragraphs with collapsed or
		// partial offsets matching rt.selection directly.
		if rt.selection.anchor.id == id {
			selA = min(rt.selection.anchor.at, ed.buf.n)
		}
		if rt.selection.caret.id == id {
			selB = min(rt.selection.caret.at, ed.buf.n)
		}
		if selA > selB {
			selA, selB = selB, selA
		}
	}
	if selA == selB {
		return false, 0, 0
	}
	oldRunes := []rune(ed.source)
	newRunes := []rune(newText)
	if selB > len(oldRunes) {
		return false, 0, 0
	}
	oldSel := oldRunes[selA:selB]
	// Case 1: Trailing Markdown syntax/fence reflow preserved text up through selB identically.
	if selB <= len(newRunes) && slices.Equal(oldSel, newRunes[selA:selB]) {
		if selA == 0 || commonPrefixRunes(oldRunes, newRunes) > 0 {
			return true, ed.anchor, ed.caret
		}
	}
	// Case 2: Block split truncated trailing lines from this paragraph cleanly.
	if strings.HasPrefix(ed.source, newText) && newText != "" && len(newRunes) > selA {
		na, nb := min(ed.anchor, len(newRunes)), min(ed.caret, len(newRunes))
		if na != nb {
			return true, na, nb
		}
	}
	// Case 3: Leading inline/block Markdown markers (e.g. "#", "*", "`", ">") collapsed while reflowing.
	if len(oldSel) >= 2 && newText != "" {
		if off, end, ok := findPreservedSubstringRunes(newText, string(oldSel)); ok {
			prefix := strings.TrimLeft(ed.source, "#>*`- \t")
			if strings.HasPrefix(newText, prefix) || commonPrefixRunes(oldRunes, newRunes) > 0 || off <= selA {
				if ed.anchor <= ed.caret {
					return true, off, end
				}
				return true, end, off
			}
		}
	}
	return false, 0, 0
}

func commonPrefixRunes(a, b []rune) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func findPreservedSubstringRunes(haystack, needle string) (start, end int, ok bool) {
	idx := strings.Index(haystack, needle)
	if idx < 0 {
		return 0, 0, false
	}
	start = utf8.RuneCountInString(haystack[:idx])
	end = start + utf8.RuneCountInString(needle)
	return start, end, true
}
