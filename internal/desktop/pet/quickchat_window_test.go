package pet

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestQuickChatEnterAndPlacement keeps Enter/Shift+Enter and the native card
// anchor: 248 wide, 8px above her head, flipped below when the head has no room,
// and clamped 4px inside the work area.
func TestQuickChatEnterAndPlacement(t *testing.T) {
	enter := ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}
	if !quickChatSendKey(enter, false) || quickChatSendKey(enter, true) {
		t.Fatal("Enter/IME routing")
	}
	enter.Mods = ui.Shift
	if quickChatSendKey(enter, false) {
		t.Fatal("Shift+Enter must insert newline")
	}
	area := motionRect{x: -1920, y: 0, width: 1920, height: 1080}
	card := mygo.Rectangle{Width: quickChatWidth, Height: quickChatHeight}
	for _, tc := range []struct {
		name string
		pet  mygo.Rectangle
		want motionPoint
	}{
		// Roomy: centred on her head, 8px above the sprite.
		{"above head", mygo.Rectangle{X: -1800, Y: 500, Width: 220, Height: 438}, motionPoint{x: -1814, y: 132}},
		// No room above: flipped below the sprite.
		{"flipped below", mygo.Rectangle{X: -1800, Y: 80, Width: 220, Height: 438}, motionPoint{x: -1814, y: 526}},
		// Right edge clamp keeps the card 4px inside the work area.
		{"right clamp", mygo.Rectangle{X: -100, Y: 500, Width: 220, Height: 438}, motionPoint{x: -252, y: 132}},
		// Left edge clamp.
		{"left clamp", mygo.Rectangle{X: -1920, Y: 500, Width: 220, Height: 438}, motionPoint{x: -1916, y: 132}},
	} {
		got := quickChatPosition(tc.pet, card, area)
		if got != tc.want {
			t.Fatalf("%s: got %+v want %+v", tc.name, got, tc.want)
		}
	}
}

// TestQuickChatNativeCardGeometry pins the numbers measured from the installed
// renderer (pet-live2d.html #pet-chat): 248px wide, 8px anchor gap, 4px inset,
// and a card that stays off her body in both placements.
func TestQuickChatNativeCardGeometry(t *testing.T) {
	if quickChatWidth != 248 || quickChatAnchorGap != 8 || quickChatInset != 4 {
		t.Fatalf("geometry drift: width=%d gap=%d inset=%d", quickChatWidth, quickChatAnchorGap, quickChatInset)
	}
	if quickChatWidth < quickChatMinWidth || quickChatHeight < quickChatMinHeight {
		t.Fatalf("minimum larger than default: %d×%d vs %d×%d", quickChatWidth, quickChatHeight, quickChatMinWidth, quickChatMinHeight)
	}
	area := motionRect{x: -1920, y: 0, width: 1920, height: 1080}
	card := mygo.Rectangle{Width: quickChatWidth, Height: quickChatHeight}
	head := mygo.Rectangle{X: -1800, Y: 500, Width: 220, Height: 438}

	above := quickChatPosition(head, card, area)
	if above.y+float64(card.Height)+float64(quickChatAnchorGap) != float64(head.Y) {
		t.Fatalf("above placement does not leave the native 8px gap: %+v", above)
	}
	if above.x+float64(card.Width)/2 != float64(head.X+head.Width/2) {
		t.Fatalf("card is not centred on her head: %+v", above)
	}

	low := mygo.Rectangle{X: -1800, Y: 80, Width: 220, Height: 438}
	below := quickChatPosition(low, card, area)
	if below.y != float64(low.Y+low.Height+quickChatAnchorGap) {
		t.Fatalf("flipped placement does not clear her feet: %+v", below)
	}
}
