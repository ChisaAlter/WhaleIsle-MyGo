package pet

import (
	"encoding/json"
	"html"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// Current native overlayBounds covers ONE display (desktop-live2d.js:663-676),
// hopping with the character. Its old file-header union comment is stale.
const (
	PetStageWidth   = 240
	PetStageHeight  = 260
	PetEdgeGap      = 24
	PetWindowWidth  = PetStageWidth
	PetWindowHeight = PetStageHeight
)

var (
	openCompanionWindow = mygo.NewWindow
	petDisplays         = platformPetDisplays
)

type petLayout struct {
	X, Y, Width, Height int
	Area                mygo.Rectangle
}

func primaryPetDisplay(displays []mygo.Display) mygo.Display {
	for _, d := range displays {
		if d.Primary && d.Bounds.Width > 0 && d.Bounds.Height > 0 {
			return d
		}
	}
	for _, d := range displays {
		if d.Bounds.Width > 0 && d.Bounds.Height > 0 {
			return d
		}
	}
	return mygo.Display{Bounds: mygo.Rectangle{Width: 1280, Height: 720}, WorkArea: mygo.Rectangle{Width: 1280, Height: 720}, ScaleFactor: 1}
}
func petDisplayAt(displays []mygo.Display, x, y float64) (mygo.Display, bool) {
	for _, d := range displays {
		b := d.Bounds
		if b.Width > 0 && b.Height > 0 && x >= float64(b.X) && x < float64(b.X+b.Width) && y >= float64(b.Y) && y < float64(b.Y+b.Height) {
			return d, true
		}
	}
	return mygo.Display{}, false
}

func activePetDisplay(displays []mygo.Display, stage mygo.Rectangle) mygo.Display {
	// Reference overlayBounds uses the FIXED nominal 120x130 DIP centre, then
	// the anchor, never the scaled/drawn bounds. Map this offset using the
	// anchor monitor's DPI, which cannot change when resizing on a target
	// monitor. Feeding the new stage.Width back into selection oscillates at
	// x=-150 between 150% (360px wide) and 100% (240px wide) screens.
	anchor, anchored := petDisplayAt(displays, float64(stage.X), float64(stage.Y))
	scale := 1.0
	if anchored {
		scale = petCoordinateScale(anchor)
	}
	if centre, ok := petDisplayAt(displays, float64(stage.X)+PetStageWidth/2*scale, float64(stage.Y)+PetStageHeight/2*scale); ok {
		return centre
	}
	if anchored {
		return anchor
	}
	return primaryPetDisplay(displays)
}
func displayOverlayBounds(displays []mygo.Display, stage mygo.Rectangle) mygo.Rectangle {
	return activePetDisplay(displays, stage).Bounds
}
func companionWindowOptions(displays []mygo.Display, stage mygo.Rectangle) mygo.WindowOptions {
	d := activePetDisplay(displays, stage)
	b := d.Bounds
	s := petCoordinateScale(d)
	return mygo.WindowOptions{Title: "Whale Isle companion", URL: "whale-pet://localhost/index.html",
		X: int(math.Round(float64(b.X) / s)), Y: int(math.Round(float64(b.Y) / s)), Width: int(math.Round(float64(b.Width) / s)), Height: int(math.Round(float64(b.Height) / s)),
		Hidden: true, Frameless: true, Transparent: true, AlwaysOnTop: true, SkipTaskbar: true, DisableMove: true, DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true, DisableShadow: true,
		Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}}
}
func petWindowSize() (int, int) { return PetStageWidth, PetStageHeight }
func defaultPetPosition(area mygo.Rectangle) (int, int) {
	return area.X + max(0, area.Width-PetStageWidth-PetEdgeGap), area.Y + max(0, area.Height-PetStageHeight-PetEdgeGap)
}
func clampPetPosition(area mygo.Rectangle, x, y int) (int, int) {
	if area.Width <= 0 || area.Height <= 0 {
		return x, y
	}
	return min(max(x, area.X), area.X+max(0, area.Width-PetStageWidth)), min(max(y, area.Y), area.Y+max(0, area.Height-PetStageHeight))
}
func petWorkArea() mygo.Rectangle { return primaryPetDisplay(petDisplays()).WorkArea }
func (p *Companion) petBounds() mygo.Rectangle {
	b := p.stage
	if b.Width <= 0 || b.Height <= 0 {
		b.Width, b.Height = PetStageWidth, PetStageHeight
	}
	return b
}
func (p *Companion) petPosition() motionPoint {
	b := p.petBounds()
	return motionPoint{x: float64(b.X), y: float64(b.Y)}
}

// Native coordinates are physical pixels on Windows, DIPs elsewhere. The
// actual content rectangle and its current scale are the only CSS origin.
func localPetLayout(stage, overlay, area mygo.Rectangle, scale float64) petLayout {
	if scale <= 0 {
		scale = 1
	}
	round := func(v int) int { return int(math.Round(float64(v) / scale)) }
	return petLayout{X: round(stage.X - overlay.X), Y: round(stage.Y - overlay.Y), Width: round(stage.Width), Height: round(stage.Height), Area: mygo.Rectangle{X: round(area.X - overlay.X), Y: round(area.Y - overlay.Y), Width: round(area.Width), Height: round(area.Height)}}
}
func renderLayoutPage(page []byte, layout petLayout) []byte {
	data, _ := json.Marshal(layout)
	return []byte(strings.ReplaceAll(string(page), "{{PET_LAYOUT}}", html.EscapeString(string(data))))
}
func (p *Companion) publishLayout() {
	if p.win == nil || p.win.IsDestroyed() || p.placingOverlay {
		return
	}
	p.placingOverlay = true
	defer func() { p.placingOverlay = false }()
	displays := petDisplays()
	petPlaceNative(p.win, displayOverlayBounds(displays, p.petBounds()))
	content := petContentBounds(p.win)
	area := activePetDisplay(displays, p.petBounds()).WorkArea
	if area.Width <= 0 || area.Height <= 0 {
		area = content
	}
	view := localPetLayout(p.petBounds(), content, area, petWindowScale(p.win))
	p.assetMu.Lock()
	p.layout = view
	p.assetMu.Unlock()
	if p.QuickChat != nil {
		p.QuickChat.positionNearPet()
	}
}
func (p *Companion) sizeStage() {
	scale := p.savedBehavior().Scale * petCoordinateScale(activePetDisplay(petDisplays(), p.petBounds()))
	p.stage.Width, p.stage.Height = int(math.Round(PetStageWidth*scale)), int(math.Round(PetStageHeight*scale))
}
func (p *Companion) preparePetStage(remembered *config.PetPosition) {
	displays := petDisplays()
	d := primaryPetDisplay(displays)
	s := petCoordinateScale(d)
	area := d.WorkArea
	if area.Width <= 0 || area.Height <= 0 {
		area = d.Bounds
	}
	p.stage = mygo.Rectangle{X: area.X, Y: area.Y, Width: int(PetStageWidth * s), Height: int(PetStageHeight * s)}
	p.stage.X += max(0, area.Width-p.stage.Width-int(PetEdgeGap*s))
	p.stage.Y += max(0, area.Height-p.stage.Height-int(PetEdgeGap*s))
	if restored := petRestorePosition(remembered, displays); restored != nil {
		p.stage.X, p.stage.Y = restored.X, restored.Y
	}
	p.sizeStage()
	if work, ok := motionWorkArea(p.petBounds()); ok {
		position := clampMotionPoint(p.petPosition(), float64(p.stage.Width), float64(p.stage.Height), work)
		p.stage.X, p.stage.Y = int(position.x), int(position.y)
	}
}
func (p *Companion) placePetWindow(w *mygo.Window, remembered *config.PetPosition) {
	if w == nil || w.IsDestroyed() {
		return
	}
	p.preparePetStage(remembered)
	p.publishLayout()
}
