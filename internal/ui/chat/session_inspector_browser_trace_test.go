package chat

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

// Replays the observed Chromium frame times through the production native List
// and revealer. It does not drive the browser or modify the source recording.
func TestInspectorSmoothRecordedBrowserTrajectory(t *testing.T) {
	path := os.Getenv("INSPECTOR_BROWSER_SMOOTH_TRACE")
	if path == "" {
		t.Skip("opt-in recorded Chromium comparison")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		Cases []struct {
			Spec struct {
				Name          string
				Start, Target float64
				NodeKey       string
			}
			Frames  []struct{ Raf, ScrollTop float64 }
			Summary struct {
				SourceCurveComparison *struct{ InferredOriginRaf, NominalDurationMs, PhysicalDeltaScale float64 }
			}
		}
	}
	if err = json.Unmarshal(bytes, &reference); err != nil {
		t.Fatal(err)
	}
	var results []map[string]any
	for _, record := range reference.Cases {
		curve := record.Summary.SourceCurveComparison
		if curve == nil || len(record.Spec.Name) < 8 || record.Spec.Name[:8] != "distance" && record.Spec.Name[:7] != "reverse" {
			continue
		}
		tt, n, scroll, list, _ := inspectorRevealFixture(t)
		tt.SetScale(float32(curve.PhysicalDeltaScale))
		list.ScrollToOffset(float32(record.Spec.Start))
		tt.Frame()
		row := int((record.Spec.Target + 100) / 60)
		n.Target = &InspectorTarget{ID: fmt.Sprint(row)}
		tt.Frame()
		var samples []map[string]any
		var square, maxError float64
		for _, frame := range record.Frames {
			elapsed := frame.Raf - curve.InferredOriginRaf
			if elapsed < 0 || elapsed > curve.NominalDurationMs+20 {
				continue
			}
			tt.SetTime(time.Unix(7000, 0).Add(time.Duration(elapsed * float64(time.Millisecond))))
			error := math.Abs(float64(scroll.Y) - frame.ScrollTop)
			square += error * error
			maxError = max(maxError, error)
			samples = append(samples, map[string]any{"ms": elapsed, "nativeY": scroll.Y, "chromiumY": frame.ScrollTop, "errorDIP": error})
		}
		rmse := math.Sqrt(square / float64(len(samples)))
		if len(samples) < 10 || rmse > .5 || maxError > 1.2 {
			t.Fatalf("%s: samples%d RMSE%.4f max%.4f", record.Spec.Name, len(samples), rmse, maxError)
		}
		results = append(results, map[string]any{"case": record.Spec.Name, "scale": curve.PhysicalDeltaScale, "durationMs": curve.NominalDurationMs, "rmseDIP": rmse, "maxErrorDIP": maxError, "samples": samples})
	}
	if len(results) != 4 {
		t.Fatalf("expected4 Chromium trajectories, got%d", len(results))
	}
	output := os.Getenv("INSPECTOR_BROWSER_SMOOTH_COMPARISON")
	if output != "" {
		b, err := json.MarshalIndent(map[string]any{"source": "Actual installed Chromium frame times/offsets replayed by production native List and Inspector revealer with fixed clock. Source-recorded bounded phase alignment retained; durations not fitted. Different pixels/frame scheduling outside this comparison.", "cases": results}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(output, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
