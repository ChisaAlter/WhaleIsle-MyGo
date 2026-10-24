package pet

import (
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Native quick-chat card geometry (reference/electron/src/renderer/pet-live2d.html):
// #pet-chat is a 248px fixed-width card anchored 8px above her head, flipped
// below when there is no room above, and kept 4px inside the host work area.
// The card is a DOM overlay there; this shell's pet is a native window, so the
// same numbers drive the separate chat window and its placement.
const (
	quickChatWidth     = 248
	quickChatHeight    = 360
	quickChatMinWidth  = 248
	quickChatMinHeight = 220
	quickChatAnchorGap = 8
	quickChatInset     = 4
)

func quickChatSendKey(ev ui.InputEvent, composing bool) bool {
	return ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter && ev.Mods&ui.Shift == 0 && !composing
}

// quickChatPosition mirrors the native chatAnchor(): centred on her head, 8px
// above the pet before flipping below it, clamped 4px inside the work area so
// the card never leaves the screen, and keeping her body uncovered.
func quickChatPosition(pet, card mygo.Rectangle, area motionRect) motionPoint {
	left := float64(area.x) + quickChatInset
	right := float64(area.x+area.width) - float64(card.Width) - quickChatInset
	top := float64(area.y) + quickChatInset
	bottom := float64(area.y+area.height) - float64(card.Height) - quickChatInset

	x := float64(pet.X) + float64(pet.Width)/2 - float64(card.Width)/2
	x = math.Min(math.Max(x, left), math.Max(left, right))

	petTop, petBottom := float64(pet.Y), float64(pet.Y+pet.Height)
	y := petTop - float64(card.Height) - quickChatAnchorGap
	if y < top {
		y = petBottom + quickChatAnchorGap
		if y < petBottom && float64(area.y+area.height)-petBottom < petTop-float64(area.y) {
			y = top
		}
	}
	y = math.Min(math.Max(y, top), math.Max(top, bottom))
	return motionPoint{x: math.Round(x), y: math.Round(y)}
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
