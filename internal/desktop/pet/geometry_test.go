package pet

import (
	"math"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// TestNativePetGeometry pins the window footprint to the native live2d pet:
// PET_WIDTH = 240 and PET_HEIGHT = 260 (reference/electron/src/main/
// desktop-live2d.js:35-36) with EDGE_GAP = 24 (:37). Status cards float on the
// display overlay; they never reserve a second panel below the character.
func TestNativePetGeometry(t *testing.T) {
	if PetStageWidth != 240 || PetStageHeight != 260 {
		t.Fatalf("native pet stage = %dx%d, want 240x260", PetStageWidth, PetStageHeight)
	}
	if PetEdgeGap != 24 {
		t.Fatalf("native edge gap = %d, want 24", PetEdgeGap)
	}
	if PetWindowWidth != PetStageWidth {
		t.Fatalf("window width = %d, want the native stage width %d", PetWindowWidth, PetStageWidth)
	}
	if PetWindowHeight != PetStageHeight {
		t.Fatalf("nominal height = %d, want character %d without reserved status space", PetWindowHeight, PetStageHeight)
	}
	w, h := petWindowSize()
	if w != PetWindowWidth || h != PetWindowHeight {
		t.Fatalf("petWindowSize = %dx%d", w, h)
	}
}

func TestOverlayUnionAndLogicalCharacterCoordinates(t *testing.T) {
	displays := []mygo.Display{
		{Bounds: mygo.Rectangle{X: -1920, Y: -200, Width: 1920, Height: 1080}, ScaleFactor: 1.5},
		{Bounds: mygo.Rectangle{X: 0, Y: 0, Width: 2560, Height: 1440}, ScaleFactor: 2},
		{Bounds: mygo.Rectangle{}},
	}
	want := mygo.Rectangle{X: -1920, Y: -200, Width: 1920, Height: 1080}
	stage := mygo.Rectangle{X: -1800, Y: 400, Width: 384, Height: 416}
	if got := displayOverlayBounds(displays, stage); got != want {
		t.Fatalf("union = %+v want %+v", got, want)
	}
	opts := companionWindowOptions(displays, stage)
	scale := petCoordinateScale(displays[0])
	if opts.X != int(float64(want.X)/scale) || opts.Y != int(math.Round(float64(want.Y)/scale)) || opts.Width != int(float64(want.Width)/scale) || opts.Height != int(float64(want.Height)/scale) || !opts.Transparent || !opts.Frameless || !opts.DisableMove || !opts.DisableResize || !opts.SkipTaskbar || !opts.AlwaysOnTop || !opts.DisableShadow || opts.StateKey != "" {
		t.Fatalf("overlay construction options = %+v", opts)
	}
	area := mygo.Rectangle{X: -1920, Y: -200, Width: 1920, Height: 1040}
	got := localPetLayout(stage, want, area, 1)
	if got.X != 120 || got.Y != 600 || got.Width != 384 || got.Height != 416 || got.Area != (mygo.Rectangle{Width: 1920, Height: 1040}) {
		t.Fatalf("DIP viewport layout = %+v", got)
	}
	previousDisplays := petDisplays
	petDisplays = func() []mygo.Display {
		return []mygo.Display{{Bounds: mygo.Rectangle{X: -2000, Y: -2000, Width: 4000, Height: 4000}, ScaleFactor: 1}}
	}
	t.Cleanup(func() { petDisplays = previousDisplays })
	p := &Companion{Behavior: config.DefaultPetBehavior(), stage: stage}
	p.Behavior.Scale = 1.6
	p.sizeStage()
	if p.stage.Width != 384 || p.stage.Height != 416 || p.stage.X != stage.X || p.stage.Y != stage.Y {
		t.Fatalf("scaled logical stage = %+v", p.stage)
	}
}

// TestPetWindowDoesNotFollowBehaviorScale mirrors the native renderer: growth
// scales the drawn pet (desktop-live2d.js:437 scales hit bounds), never the
// window, so every behavior reports the same footprint.
func TestPetWindowDoesNotFollowBehaviorScale(t *testing.T) {
	for _, scale := range []float64{1, 1.1, 1.5, 2, 3} {
		behavior := config.DefaultPetBehavior()
		behavior.Scale = scale
		if got := petWindowHeight(behavior); got != PetWindowHeight {
			t.Fatalf("scale %v resized the pet window to %d, want %d", scale, got, PetWindowHeight)
		}
	}
}

// TestDefaultPetPositionMatchesNative pins the native placement rule: the pet
// sits at the bottom-right of the work area, EDGE_GAP from both edges
// (desktop-live2d.js:82-88), and a remembered position is clamped onto it.
func TestDefaultPetPositionMatchesNative(t *testing.T) {
	area := mygo.Rectangle{X: 0, Y: 0, Width: 1440, Height: 920}
	x, y := defaultPetPosition(area)
	if x != 1440-PetWindowWidth-PetEdgeGap || y != 920-PetWindowHeight-PetEdgeGap {
		t.Fatalf("default position = (%d,%d), want (%d,%d)", x, y, 1440-PetWindowWidth-PetEdgeGap, 920-PetWindowHeight-PetEdgeGap)
	}
	// Negative-origin work areas (a left/secondary display) keep the same rule.
	left := mygo.Rectangle{X: -1920, Y: 0, Width: 1920, Height: 1080}
	if x, y := defaultPetPosition(left); x != -1920+1920-PetWindowWidth-PetEdgeGap || y != 1080-PetWindowHeight-PetEdgeGap {
		t.Fatalf("left display position = (%d,%d)", x, y)
	}
	// A remembered position off the work area is clamped back inside it.
	if x, y := clampPetPosition(area, -500, 5000); x != 0 || y != 920-PetWindowHeight {
		t.Fatalf("clamped position = (%d,%d), want (0,%d)", x, y, 920-PetWindowHeight)
	}
	if x, y := clampPetPosition(area, 100, 200); x != 100 || y != 200 {
		t.Fatalf("on-area position moved to (%d,%d)", x, y)
	}
}
