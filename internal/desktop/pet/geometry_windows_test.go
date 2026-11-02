//go:build windows

package pet

import (
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

func TestPhysicalDisplayHopMigrationAndCSSOrigin(t *testing.T) {
	displays := []mygo.Display{
		{ID: 1, Primary: true, Bounds: mygo.Rectangle{Width: 1920, Height: 1080}, WorkArea: mygo.Rectangle{Width: 1920, Height: 1040}, ScaleFactor: 1},
		{ID: 2, Bounds: mygo.Rectangle{X: -2560, Y: -300, Width: 2560, Height: 1440}, WorkArea: mygo.Rectangle{X: -2560, Y: -300, Width: 2560, Height: 1400}, ScaleFactor: 1.5},
	}
	primary := mygo.Rectangle{X: 400, Y: 500, Width: 240, Height: 260}
	left := mygo.Rectangle{X: -2400, Y: 200, Width: 360, Height: 390}
	if displayOverlayBounds(displays, primary) != displays[0].Bounds || displayOverlayBounds(displays, left) != displays[1].Bounds {
		t.Fatal("overlay did not hop to a single active display")
	}
	// A point in the virtual gap is never a union-window origin.
	if got := displayOverlayBounds(displays, mygo.Rectangle{X: -3000, Y: -500, Width: 360, Height: 390}); got != displays[0].Bounds {
		t.Fatalf("gap origin=%+v", got)
	}
	legacy := petRestorePosition(&config.PetPosition{X: -1600, Y: 200}, displays)
	if legacy == nil || legacy.X != -2400 || legacy.Y != 300 || legacy.Space != config.PetPositionWindowsPhysical {
		t.Fatalf("legacy migration=%+v", legacy)
	}
	physical := petRestorePosition(&config.PetPosition{X: -2400, Y: 300, Space: config.PetPositionWindowsPhysical}, displays)
	if physical == nil || *physical != *legacy {
		t.Fatalf("physical position scaled twice: %+v", physical)
	}
	if petRestorePosition(&config.PetPosition{X: -5000, Y: -5000}, displays) != nil {
		t.Fatal("unmatched legacy position was not retired")
	}
	// Native HWND may report a relocated origin; CSS must use that actual
	// content origin, not the requested display origin or its old DPI.
	actual := mygo.Rectangle{X: -2550, Y: -285, Width: 2560, Height: 1440}
	got := localPetLayout(left, actual, displays[1].WorkArea, 1.5)
	if got.X != 100 || got.Y != 323 || got.Width != 240 || got.Height != 260 || got.Area.X != -7 || got.Area.Y != -10 {
		t.Fatalf("physical to CSS=%+v", got)
	}
	previous := petDisplays
	petDisplays = func() []mygo.Display { return displays }
	defer func() { petDisplays = previous }()
	p := &Companion{Behavior: config.DefaultPetBehavior(), stage: left}
	p.sizeStage()
	if p.stage.Width != 360 || p.stage.Height != 390 {
		t.Fatalf("150%% stage=%+v", p.stage)
	}
	p.stage.X, p.stage.Y = 300, 400
	p.sizeStage()
	if p.stage.Width != 240 || p.stage.Height != 260 {
		t.Fatalf("100%% hop scaled incorrectly=%+v", p.stage)
	}
	// Scaling must not move the display-selection point across the boundary.
	p.stage = mygo.Rectangle{X: -150, Y: 300, Width: 360, Height: 390}
	for i := 0; i < 4; i++ {
		p.sizeStage()
		if p.stage.Width != 240 || activePetDisplay(displays, p.petBounds()).ID != 1 {
			t.Fatalf("boundary hop oscillated at %d: %+v", i, p.stage)
		}
	}
	area := motionRect{x: -2560, y: -300, width: 2560, height: 1400}
	card := mygo.Rectangle{Width: 372, Height: 540}
	point := quickChatPositionInSpace(left, card, area, 1.5)
	if point.y != 554 {
		t.Fatalf("150%% card gap=%+v want work-area bottom minus 6px inset", point)
	}
}
