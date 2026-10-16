package pet

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func quickChatSendKey(ev ui.InputEvent, composing bool) bool {
	return ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter && ev.Mods&ui.Shift == 0 && !composing
}

func quickChatPosition(pet, card mygo.Rectangle, area motionRect) motionPoint {
	const gap = 8
	x := float64(pet.X + pet.Width + gap)
	if x+float64(card.Width) > area.x+area.width {
		x = float64(pet.X - card.Width - gap)
	}
	return clampMotionPoint(motionPoint{x: x, y: float64(pet.Y + pet.Height - card.Height)}, float64(card.Width), float64(card.Height), area)
}

func (q *QuickChat) positionNearPet() {
	if q.closed || q.win == nil || q.win.IsDestroyed() || q.parent == nil || q.parent.IsDestroyed() {
		return
	}
	bounds := q.parent.Bounds()
	if area, ok := motionWorkArea(bounds); ok {
		point := quickChatPosition(bounds, q.win.Bounds(), area)
		q.win.SetPosition(int(point.x), int(point.y))
	}
}

func (q *QuickChat) followPet() {
	q.positionNearPet()
	q.offAnchor = append(q.offAnchor, q.parent.OnMove(q.positionNearPet), q.win.OnResize(q.positionNearPet), mygo.Screen.OnDisplaysChanged(q.positionNearPet))
}
