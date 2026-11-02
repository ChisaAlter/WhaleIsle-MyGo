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

// Production always delegates to NewWindow. The seam lets the public open path
// be tested without starting a desktop or making network requests.
var openQuickChatWindow = mygo.NewWindow
var focusQuickChatWindow = func(w *mygo.Window) { w.Focus() }

func quickChatSendKey(ev ui.InputEvent, composing bool) bool {
	return ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter && ev.Mods&ui.Shift == 0 && !composing
}

// quickChatPosition mirrors the native chatAnchor(): centred on her head, 8px
// above the pet before flipping below it, clamped 4px inside the work area so
// the card never leaves the screen, and keeping her body uncovered.
func quickChatPosition(pet, card mygo.Rectangle, area motionRect) motionPoint {
	return quickChatPositionInSpace(pet, card, area, 1)
}
func quickChatPositionInSpace(pet, card mygo.Rectangle, area motionRect, scale float64) motionPoint {
	inset, gap := float64(quickChatInset)*scale, float64(quickChatAnchorGap)*scale
	left := float64(area.x) + inset
	right := float64(area.x+area.width) - float64(card.Width) - inset
	top := float64(area.y) + inset
	bottom := float64(area.y+area.height) - float64(card.Height) - inset

	x := float64(pet.X) + float64(pet.Width)/2 - float64(card.Width)/2
	x = math.Min(math.Max(x, left), math.Max(left, right))

	petTop, petBottom := float64(pet.Y), float64(pet.Y+pet.Height)
	y := petTop - float64(card.Height) - gap
	if y < top {
		y = petBottom + gap
		if y < petBottom && float64(area.y+area.height)-petBottom < petTop-float64(area.y) {
			y = top
		}
	}
	y = math.Min(math.Max(y, top), math.Max(top, bottom))
	return motionPoint{x: math.Round(x), y: math.Round(y)}
}

func (q *QuickChat) positionNearPet() {
	if q.positioning || q.closed || q.win == nil || q.win.IsDestroyed() || q.parent == nil || q.parent.IsDestroyed() {
		return
	}
	q.positioning = true
	defer func() { q.positioning = false }()
	bounds := q.owner.petBounds()
	if area, ok := motionWorkArea(bounds); ok {
		scale := petCoordinateScale(activePetDisplay(petDisplays(), bounds))
		card := petWindowBounds(q.win)
		card.Width, card.Height = int(math.Round(quickChatWidth*scale)), int(math.Round(float64(q.win.Bounds().Height)*scale))
		point := quickChatPositionInSpace(bounds, card, area, scale)
		card.X, card.Y = int(point.x), int(point.y)
		petPlaceNative(q.win, card)
	}
}

func (q *QuickChat) followPet() {
	q.positionNearPet()
	q.offAnchor = append(q.offAnchor, q.parent.OnMove(q.positionNearPet), q.win.OnResize(q.positionNearPet), mygo.Screen.OnDisplaysChanged(q.positionNearPet))
}
