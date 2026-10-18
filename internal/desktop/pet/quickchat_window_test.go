package pet

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"testing"
)

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
	card := mygo.Rectangle{Width: 460, Height: 620}
	for _, tc := range []struct {
		pet  mygo.Rectangle
		want motionPoint
	}{
		{mygo.Rectangle{X: -1800, Y: 500, Width: 220, Height: 438}, motionPoint{x: -1572, y: 318}},
		{mygo.Rectangle{X: -220, Y: 800, Width: 220, Height: 438}, motionPoint{x: -688, y: 460}},
		{mygo.Rectangle{X: -1900, Y: 0, Width: 220, Height: 438}, motionPoint{x: -1672, y: 0}},
	} {
		if got := quickChatPosition(tc.pet, card, area); got != tc.want {
			t.Fatalf("got %+v want %+v", got, tc.want)
		}
	}
}
