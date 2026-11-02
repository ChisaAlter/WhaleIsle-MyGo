//go:build windows

package pet

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// Real native DPI is observed, never changed. Synthetic 100/150 monitor
// topology is measured separately by TestMeasurePetDisplayPolicies.
func TestRealPetCoordinateContract(t *testing.T) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" {
		t.Skip("isolated own GUI only")
	}
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	behavior := config.DefaultPetBehavior()
	behavior.Wander = false
	behavior.SelfTalk = false
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: true, Behavior: behavior}); err != nil {
		t.Fatal(err)
	}
	var p *Companion
	var w *mygo.Window
	var displays []mygo.Display
	mygo.RunOnMain(func() {
		p = New(store)
		p.SetReady(true)
		w = p.win
		displays = petDisplays()
		p.stopInteraction()
		if p.motion.timer != nil {
			p.motion.timer.Stop()
			p.motion.timer = nil
		}
	})
	if w == nil || w.IsDestroyed() {
		t.Fatal("own pet creation failed")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	realPetEvidence["coordinateDisplays"] = displays
	type sample struct {
		name           string
		display        mygo.Display
		position       motionPoint
		repeats        int
		localX, localY int
	}
	var samples []sample
	for _, d := range displays {
		dpi := petCoordinateScale(d)
		samples = append(samples, sample{name: "display-interior", display: d, position: motionPoint{x: float64(d.Bounds.X) + 120*dpi, y: float64(d.Bounds.Y) + 80*dpi}, repeats: 1, localX: 120, localY: 80})
	}
	// Exercise the original failure topology on the actual adjacent screens:
	// the nominal centre crosses while the anchor is still 150px on the left.
	for _, left := range displays {
		for _, right := range displays {
			if left.Bounds.X+left.Bounds.Width == right.Bounds.X && left.ID != right.ID {
				y := max(left.Bounds.Y, right.Bounds.Y) + 300
				if y+200 < min(left.Bounds.Y+left.Bounds.Height, right.Bounds.Y+right.Bounds.Height) {
					x := right.Bounds.X - 150
					selected := activePetDisplay(displays, mygo.Rectangle{X: x, Y: y})
					if selected.ID == right.ID {
						samples = append(samples, sample{name: "centre-crosses-before-anchor", display: right, position: motionPoint{x: float64(x), y: float64(y)}, repeats: 8, localX: int(math.Round(float64(x-right.Bounds.X) / petCoordinateScale(right))), localY: int(math.Round(float64(y-right.Bounds.Y) / petCoordinateScale(right)))})
					}
				}
			}
		}
	}
	if len(displays) > 1 {
		d := displays[0]
		dpi := petCoordinateScale(d)
		samples = append(samples, sample{name: "return-to-first-display", display: d, position: motionPoint{x: float64(d.Bounds.X) + 120*dpi, y: float64(d.Bounds.Y) + 80*dpi}, repeats: 4, localX: 120, localY: 80})
	}
	var records []any
	for _, sample := range samples {
		display := sample.display
		if display.Bounds.Width <= 0 || display.Bounds.Height <= 0 {
			continue
		}
		var expected petLayout
		var before mygo.Rectangle
		var scale float64
		var stage mygo.Rectangle
		var cursor mygo.Point
		var view interactionView
		mygo.RunOnMain(func() {
			dpi := petCoordinateScale(display)
			for i := 0; i < sample.repeats; i++ {
				p.setMotionPosition(sample.position)
				if got := petContentBounds(w); got != display.Bounds || petWindowScale(w) != dpi {
					t.Errorf("%s repeat%d overlay=%+v dpi=%g", sample.name, i, got, petWindowScale(w))
				}
			}
			before = petContentBounds(w)
			scale = petWindowScale(w)
			stage = p.petBounds()
			if before != display.Bounds || math.Abs(scale-dpi) > .001 {
				t.Errorf("HWND display placement mismatch: want=%+v at%g actual=%+v at%g", display.Bounds, dpi, before, scale)
			}
			if w.IsMovable() {
				t.Error("programmatic placement left user movement enabled")
			}
			expected = localPetLayout(stage, before, display.WorkArea, scale)
			if expected.X != sample.localX || expected.Y != sample.localY || expected.Width != 240 || expected.Height != 260 {
				t.Errorf("CSS unit mismatch: %+v", expected)
			}
			// These velocity fields simulate only the application's state, not
			// desktop input; the real cursor is sampled read-only.
			p.interaction.vx, p.interaction.vy = 300*scale, -200*scale
			p.tickInteraction(time.Now())
			cursor = petCursorPoint()
			view = p.interaction.view
		})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var observed map[string]any
		matches := func(v map[string]any) bool {
			if v == nil {
				return false
			}
			vw, wok := v["viewportWidth"].(float64)
			vh, hok := v["viewportHeight"].(float64)
			return wok && hok && v["left"] == float64(expected.X) && v["top"] == float64(expected.Y) && v["width"] == float64(expected.Width) && v["height"] == float64(expected.Height) && v["dpr"] == scale && math.Abs(vw-float64(before.Width)/scale) <= 1 && math.Abs(vh-float64(before.Height)/scale) <= 1
		}
		for ctx.Err() == nil {
			value, evalErr := w.Page().EvalContext(ctx, `(()=>{const r=document.getElementById("sprite").getBoundingClientRect();return {left:r.left,top:r.top,width:r.width,height:r.height,viewportWidth:innerWidth,viewportHeight:innerHeight,dpr:devicePixelRatio,renderer:document.body.dataset.renderer||""};})()`)
			if evalErr == nil {
				observed, _ = value.(map[string]any)
				if matches(observed) {
					break
				}
			}
			time.Sleep(25 * time.Millisecond)
		}
		cancel()
		if !matches(observed) {
			t.Errorf("display%d actual native=%+v scale=%g layout=%+v DOM=%v", display.ID, before, scale, expected, observed)
		}
		if math.Abs(view.VX-300) > 1e-6 || math.Abs(view.VY+200) > 1e-6 {
			t.Errorf("native-to-CSS velocity=%v,%v", view.VX, view.VY)
		}
		records = append(records, map[string]any{"sample": sample.name, "repeatedPlacements": sample.repeats, "hardwareDisplay": display, "nativeContent": before, "hardwareScale": scale, "stage": stage, "cssLayout": expected, "dom": observed, "cursorPhysicalReadOnly": cursor, "cursorCSSLastSample": mygo.Point{X: int(view.CursorX), Y: int(view.CursorY)}, "rendererVX": view.VX, "rendererVY": view.VY})
		if realPetEvidence["dom"] == nil {
			realPetEvidence["dom"] = observed
		}
		if image, err := w.CapturePage(); err == nil {
			savePetGUIImage(t, "coordinates-"+sample.name+"-"+display.Label[len(display.Label)-1:], image)
		} else {
			t.Error(err)
		}
		t.Logf("hardwareScale=%g content=%+v nativeStage=%+v CSS=%+v DOM=%v rendererV=(%g,%g)DIP/s", scale, before, stage, expected, observed, view.VX, view.VY)
	}
	realPetEvidence["coordinateHardwareSamples"] = records
	if len(records) == 0 {
		t.Fatal("no native display measured")
	}
	if image, err := w.CapturePage(); err == nil {
		savePetGUIImage(t, "coordinates", image)
	} else {
		t.Error(err)
	}
}
