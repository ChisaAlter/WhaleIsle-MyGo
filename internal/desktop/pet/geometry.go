package pet

import (
	"github.com/egoist/mygo"

	"whaleisle/internal/config"
)

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

// openCompanionWindow creates the pet's OS window. The shell can only create a
// window once the application loop is running, and a main-window recreation
// (launcher -> desktop) closes the companion, so the manager must be able to
// create it again on the next SetReady. Tests replace this seam to observe the
// lifecycle without a GUI.
var openCompanionWindow = func(opts mygo.WindowOptions) *mygo.Window {
	return mygo.NewWindow(opts)
}


// defaultPetPosition places the pet at the native default: the bottom-right of
// the work area, EDGE_GAP from both edges (desktop-live2d.js:82-88). Coordinates
// are DIPs in the same space as the window API.
func defaultPetPosition(workArea mygo.Rectangle) (int, int) {
	x := workArea.X + max(0, workArea.Width-PetWindowWidth-PetEdgeGap)
	y := workArea.Y + max(0, workArea.Height-PetWindowHeight-PetEdgeGap)
	return x, y
}

// clampPetPosition keeps a remembered position fully on the work area, like the
// native clamp into the home display (desktop-live2d.js:950-956). A remembered
// position off every display falls back to the default, so the pet cannot be
// lost after a monitor change.
func clampPetPosition(workArea mygo.Rectangle, x, y int) (int, int) {
	if workArea.Width <= 0 || workArea.Height <= 0 {
		return x, y
	}
	maxX := workArea.X + max(0, workArea.Width-PetWindowWidth)
	maxY := workArea.Y + max(0, workArea.Height-PetWindowHeight)
	return min(max(x, workArea.X), maxX), min(max(y, workArea.Y), maxY)
}

// petWorkArea is the primary display's usable area, falling back to the display
// bounds and then to no clamping.
func petWorkArea() mygo.Rectangle {
	displays := mygo.Screen.Displays()
	for _, d := range displays {
		if d.Primary && d.WorkArea.Width > 0 && d.WorkArea.Height > 0 {
			return d.WorkArea
		}
	}
	for _, d := range displays {
		if d.WorkArea.Width > 0 && d.WorkArea.Height > 0 {
			return d.WorkArea
		}
	}
	return mygo.Rectangle{}
}

// placePetWindow positions a pet window: the remembered anchor when the user
// dragged it, otherwise the native bottom-right default.
func placePetWindow(w *mygo.Window, remembered *config.PetPosition) {
	if w == nil || w.IsDestroyed() {
		return
	}
	area := petWorkArea()
	x, y := defaultPetPosition(area)
	if remembered != nil {
		x, y = clampPetPosition(area, remembered.X, remembered.Y)
	}
	w.SetPosition(x, y)
}

