//go:build !windows

package pet

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

func platformPetDisplays() []mygo.Display            { return mygo.Screen.Displays() }
func petCursorPoint() mygo.Point                     { return mygo.Screen.CursorScreenPoint() }
func petWindowBounds(w *mygo.Window) mygo.Rectangle  { return w.Bounds() }
func petContentBounds(w *mygo.Window) mygo.Rectangle { return w.ContentBounds() }
func petWindowScale(w *mygo.Window) float64          { return 1 }
func petPlaceNative(w *mygo.Window, b mygo.Rectangle) {
	if w.Bounds() != b {
		w.SetBounds(b)
	}
}
func petCoordinateScale(d mygo.Display) float64 { return 1 }
func petStoredPosition(position motionPoint) *config.PetPosition {
	return &config.PetPosition{X: int(position.x), Y: int(position.y)}
}
func petRestorePosition(saved *config.PetPosition, displays []mygo.Display) *config.PetPosition {
	if saved == nil || saved.Space == config.PetPositionWindowsPhysical {
		return nil
	}
	copy := *saved
	return &copy
}
