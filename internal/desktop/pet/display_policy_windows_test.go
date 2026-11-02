//go:build windows

package pet

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
	"whaleisle/internal/config"

	"github.com/egoist/mygo"
)

func coordinateTestDisplays() []mygo.Display {
	return []mygo.Display{
		{ID: 1, Bounds: mygo.Rectangle{X: -2560, Width: 2560, Height: 1440}, WorkArea: mygo.Rectangle{X: -2560, Width: 2560, Height: 1400}, ScaleFactor: 1.5},
		{ID: 2, Primary: true, Bounds: mygo.Rectangle{Width: 1920, Height: 1080}, WorkArea: mygo.Rectangle{Width: 1920, Height: 1040}, ScaleFactor: 1},
	}
}

func testDisplayAt(displays []mygo.Display, x, y float64) (mygo.Display, bool) {
	for _, d := range displays {
		b := d.Bounds
		if x >= float64(b.X) && x < float64(b.X+b.Width) && y >= float64(b.Y) && y < float64(b.Y+b.Height) {
			return d, true
		}
	}
	return mygo.Display{}, false
}

// The previous mutable-size centre is kept ONLY in this diagnostic, so its
// oscillation can be reproduced without reintroducing it into the application.
func mutableCentrePolicy(displays []mygo.Display, stage mygo.Rectangle) mygo.Display {
	if d, ok := testDisplayAt(displays, float64(stage.X+stage.Width/2), float64(stage.Y+stage.Height/2)); ok {
		return d
	}
	return primaryPetDisplay(displays)
}

func anchorFirstPolicy(displays []mygo.Display, stage mygo.Rectangle) mygo.Display {
	if d, ok := testDisplayAt(displays, float64(stage.X), float64(stage.Y)); ok {
		return d
	}
	return primaryPetDisplay(displays)
}

type coordinatePolicyTrace struct {
	Step      int     `json:"step"`
	CentreX   float64 `json:"centreX"`
	DisplayID int64   `json:"displayID"`
	Width     int     `json:"width"`
	CSSWidth  float64 `json:"cssWidth"`
}

func TestMeasurePetDisplayPolicies(t *testing.T) {
	displays := coordinateTestDisplays()
	type policy struct {
		name          string
		selectDisplay func([]mygo.Display, mygo.Rectangle) mygo.Display
		centre        func(mygo.Rectangle) float64
	}
	policies := []policy{
		{"mutable-centre", mutableCentrePolicy, func(s mygo.Rectangle) float64 { return float64(s.X + s.Width/2) }},
		{"anchor-first", anchorFirstPolicy, func(s mygo.Rectangle) float64 { return float64(s.X) }},
		{"fixed-nominal-centre", activePetDisplay, func(s mygo.Rectangle) float64 { return float64(s.X) + 180 }},
	}
	record := map[string]any{"kind": "synthetic-physical-monitor-model", "displays": displays, "anchorX": -150, "anchorY": 300}
	for _, policy := range policies {
		stage := mygo.Rectangle{X: -150, Y: 300, Width: 360, Height: 390}
		var trace []coordinatePolicyTrace
		for i := 0; i < 6; i++ {
			d := policy.selectDisplay(displays, stage)
			scale := petCoordinateScale(d)
			trace = append(trace, coordinatePolicyTrace{Step: i, CentreX: policy.centre(stage), DisplayID: d.ID, Width: stage.Width, CSSWidth: float64(stage.Width) / scale})
			stage.Width, stage.Height = int(math.Round(PetStageWidth*scale)), int(math.Round(PetStageHeight*scale))
		}
		record[policy.name] = trace
		t.Logf("%s: %+v", policy.name, trace)
		if policy.name == "mutable-centre" {
			for i := 1; i < len(trace); i++ {
				if trace[i].DisplayID == trace[i-1].DisplayID {
					t.Fatal("diagnostic no longer reproduces the mutable-size oscillation")
				}
			}
		} else {
			for i := 1; i < len(trace); i++ {
				if trace[i].DisplayID != trace[0].DisplayID {
					t.Fatalf("%s oscillated", policy.name)
				}
			}
		}
	}
	// Sweep 601 physical boundary anchors for both DPI directions and three
	// drawn-size preferences. Every repeated placement uses the same selector
	// as sizeStage, overlay, work area and quick-chat placement.
	checks := 0
	for _, pair := range [][2]float64{{1.5, 1}, {1, 1.5}} {
		displays[0].ScaleFactor, displays[1].ScaleFactor = pair[0], pair[1]
		for x := -300; x <= 300; x++ {
			for _, size := range []float64{.6, 1, 1.6} {
				stage := mygo.Rectangle{X: x, Y: 300, Width: 360, Height: 390}
				first := activePetDisplay(displays, stage).ID
				for i := 0; i < 6; i++ {
					d := activePetDisplay(displays, stage)
					if d.ID != first {
						t.Fatalf("sweep oscillated: pair=%v x=%d size=%g step=%d", pair, x, size, i)
					}
					stage.Width = int(math.Round(240 * size * d.ScaleFactor))
					stage.Height = int(math.Round(260 * size * d.ScaleFactor))
					checks++
				}
			}
		}
	}
	record["stableSweepChecks"] = checks
	record["stableSweepOscillations"] = 0
	// Persistence is tested through the real config codec, not a literal copy.
	var migration []any
	for _, dpi := range []float64{1, 1.5} {
		display := mygo.Display{Bounds: mygo.Rectangle{X: -3000, Y: -600, Width: 3000, Height: 1800}, ScaleFactor: dpi}
		legacy := &config.PetPosition{X: -1000, Y: 200}
		converted := petRestorePosition(legacy, []mygo.Display{display})
		if converted == nil || converted.X != int(-1000*dpi) || converted.Y != int(200*dpi) || converted.Space != config.PetPositionWindowsPhysical {
			t.Fatalf("dpi=%g migration=%+v", dpi, converted)
		}
		storeDir := t.TempDir()
		store, err := config.Open(storeDir)
		if err != nil {
			t.Fatal(err)
		}
		cfg := store.DesktopPet()
		cfg.Position = converted
		if err := store.SaveDesktopPet(cfg); err != nil {
			t.Fatal(err)
		}
		reopened, err := config.Open(storeDir)
		if err != nil {
			t.Fatal(err)
		}
		stored := reopened.DesktopPet().Position
		if !reflect.DeepEqual(stored, converted) {
			t.Fatalf("position marker codec=%+v want=%+v", stored, converted)
		}
		restored := petRestorePosition(stored, []mygo.Display{display})
		if !reflect.DeepEqual(restored, converted) {
			t.Fatal("physical marker scaled a second time")
		}
		migration = append(migration, map[string]any{"dpi": dpi, "legacy": legacy, "physical": converted, "reopened": stored})
	}
	record["markerRoundTrips"] = migration
	t.Logf("stable boundary checks=%d oscillations=0; marker=%v", checks, migration)
	if result := os.Getenv("WHALEISLE_PET_POLICY_RESULT"); result != "" {
		data, _ := json.MarshalIndent(record, "", "  ")
		if err := os.WriteFile(result, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
