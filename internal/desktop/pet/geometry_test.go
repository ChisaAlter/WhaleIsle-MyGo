package pet

import (
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// TestNativePetGeometry pins the window footprint to the native live2d pet:
// PET_WIDTH = 240 and PET_HEIGHT = 260 (reference/electron/src/main/
// desktop-live2d.js:35-36) with EDGE_GAP = 24 (:37). MyGo also carries the
// page-only status panel, so its window is the native stage plus that panel.
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
	if PetWindowHeight != PetStageHeight+PetStatusPanelHeight {
		t.Fatalf("window height = %d, want stage %d plus the page status panel %d", PetWindowHeight, PetStageHeight, PetStatusPanelHeight)
	}
	w, h := petWindowSize()
	if w != PetWindowWidth || h != PetWindowHeight {
		t.Fatalf("petWindowSize = %dx%d", w, h)
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
