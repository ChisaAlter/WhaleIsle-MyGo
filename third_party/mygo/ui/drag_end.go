package ui

// DragEndResult is the completion of one window-local value drag. Dropped means
// a compatible target received the value through Drop/DropAt; a source can use
// an unhandled, uncanceled completion to apply its last insertion marker.
// Coordinates are the release position in window DIPs, fixed at completion.
// Escape, window focus loss and releases outside this window are Canceled.
// Native operating-system data drags keep their transfer.DragOptions.Done API.
type DragEndResult[T any] struct {
	Value    T
	X, Y     float32
	Canceled bool
	Dropped  bool
}

type dragEndKey struct{}

type dragEndReceipt struct {
	value    any
	x, y     float32
	canceled bool
	dropped  bool
	ready    bool
	frame    uint64
}

// DragEnd reports one completion on the source passed to Element.Drag, including
// a release over no target. It does not turn the source into a drop target.
// Read it while building that same source on each frame. Each receipt is read
// once; a source removed from the frame cannot deliver it after remounting.
func DragEnd[T any](e Element) (result DragEndResult[T], completed bool) {
	n := e.node()
	if n == nil {
		return
	}
	n = e.nodeFor(n.c.rt)
	r := coreLocal(n, dragEndKey{}, func() dragEndReceipt { return dragEndReceipt{} })
	if !r.ready {
		return
	}
	if n.c.rt.frame > r.frame+1 {
		r.ready, r.value = false, nil
		return
	}
	value, ok := r.value.(T)
	if !ok {
		return
	}
	result = DragEndResult[T]{Value: value, X: r.x, Y: r.y, Canceled: r.canceled, Dropped: r.dropped}
	r.ready, r.value = false, nil
	n.c.rt.consumed = true
	return result, true
}

func (rt *engine) cancelOrphanedValueDrag() {
	d := rt.drag
	if d == nil || d.native {
		return
	}
	if s := rt.states[d.src]; s == nil || s.seen != rt.frame || s.pass != rt.pass {
		rt.drag = nil
	}
}

func (rt *engine) cancelDragOnBlur() {
	rt.dragCancel()
	if d := rt.drag; d != nil && !d.native {
		rt.drag = nil
	}
}

func (rt *engine) recordDragEnd(d *valueDrag, dropped bool) {
	s := rt.states[d.src]
	if s == nil || s.seen != rt.frame || s.pass != rt.pass {
		return
	}
	r, _ := s.locals[dragEndKey{}].(*dragEndReceipt)
	if r != nil {
		*r = dragEndReceipt{value: d.value, x: rt.pointerX, y: rt.pointerY, canceled: d.canceled,
			dropped: dropped, ready: true, frame: rt.frame}
	}
}
