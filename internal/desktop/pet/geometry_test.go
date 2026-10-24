package pet

import (
	"testing"

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
