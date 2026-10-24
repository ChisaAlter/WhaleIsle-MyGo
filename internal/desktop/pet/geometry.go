package pet

// Window geometry mirrors the native desktop pet. The modern pet is the live2d
// overlay: reference/electron/src/main/desktop-live2d.js:35-37 pins the sprite
// footprint PET_WIDTH = 240, PET_HEIGHT = 260 and EDGE_GAP = 24 relative to the
// display work area (defaultPosition, :82-88). Its OS window is the union of
// every display (:27-33, overlayBounds), transparent, frameless, resizable:false
// and never resized for growth or behavior: 240x260 is the pet's footprint, not
// a window rectangle. The older sprite pet (desktop-pet.js:8-9, PET_SIZE = 88,
// EDGE_GAP = 16) is behind DESKTOP_PET_FEATURE = false and is not modelled.
//
// MyGo draws the pet as a web page instead of compositing a live2d sprite, so
// the window is a real page surface: the sprite stage keeps the native 240x260
// footprint, and the status panel below it is where the page puts the actions
// the native overlay asks its separate dialogue/status surfaces for. The height
// of that panel is therefore a page cost, not a native window size.
const (
	// PetStageWidth/PetStageHeight are the native sprite footprint.
	PetStageWidth  = 240
	PetStageHeight = 260
	// PetEdgeGap is the native gap kept from the work-area edges when the pet is
	// placed at its default bottom-right position (desktop-live2d.js:37,82-88).
	PetEdgeGap = 24
	// PetStatusPanelHeight is the page-only status/actions panel under the stage.
	PetStatusPanelHeight = 178

	PetWindowWidth  = PetStageWidth
	PetWindowHeight = PetStageHeight + PetStatusPanelHeight
)

// petWindowSize is the window rectangle in DIPs. It does not depend on the
// behavior: the native pet never resizes when its growth scale changes, the
// renderer only draws the same footprint larger (desktop-live2d.js:437 scales
// the hit bounds, not the window).
func petWindowSize() (int, int) {
	return PetWindowWidth, PetWindowHeight
}
