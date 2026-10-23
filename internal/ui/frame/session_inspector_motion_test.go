package frame

import (
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestInspectorPulseHistoricalBaselineAndArrivalsFollowSource(t *testing.T) {
	var state inspectorPulseState
	now := time.Unix(6000, 0)
	row := session.InspectorRecord{Key: "old"}
	state.updateRows(nil, now)
	state.updateRows([]session.InspectorRecord{row}, now)
	if !state.arrivals["old"].IsZero() {
		t.Fatal("first nonempty history flashed")
	}
	rows := []session.InspectorRecord{row, {Key: "new"}}
	state.updateRows(rows, now.Add(time.Second))
	state.updateRows(rows, now.Add(2*time.Second))
	if state.arrivals["new"] != now.Add(time.Second) || !state.arrivals["old"].IsZero() {
		t.Fatal("arrival restarted on unchanged source")
	}
	state.updateRows([]session.InspectorRecord{row}, now.Add(3*time.Second))
	if _, ok := state.arrivals["new"]; ok {
		t.Fatal("removed key retained arrival")
	}
}

func TestInspectorRowPulseModeCancellationAndRepeatReveal(t *testing.T) {
	now := time.Unix(6000, 0)
	row := session.InspectorRecord{Key: "r", Type: "text"}
	state := inspectorRowPulse{record: inspectorPulseValue(row)}
	state.update(row, true, false, false, 0, now)
	if !state.started.IsZero() {
		t.Fatal("Raw Log arrival flashed")
	}
	state.update(row, true, false, false, 1, now)
	if state.started != now {
		t.Fatal("Raw Log reveal did not flash")
	}
	state.update(row, true, false, false, 1, now.Add(300*time.Millisecond))
	if state.started != now {
		t.Fatal("unchanged render restarted pulse")
	}
	state.update(row, true, false, false, 2, now.Add(350*time.Millisecond))
	if state.started != now.Add(350*time.Millisecond) {
		t.Fatal("repeated reveal failed to restart pulse")
	}
	row.Raw = []byte(`{"text":"updated"}`)
	state.update(row, true, false, false, 2, now.Add(400*time.Millisecond))
	if !state.started.IsZero() {
		t.Fatal("Raw Log record update must cancel previous pulse without flashing")
	}
	state.update(row, false, true, false, 0, now)
	row.Raw = []byte(`{"text":"next"}`)
	state.update(row, false, true, false, 0, now.Add(time.Millisecond))
	if state.started != now.Add(time.Millisecond) {
		t.Fatal("Chat Group update failed to flash")
	}
	state.update(row, false, true, true, 1, now.Add(2*time.Millisecond))
	if !state.started.IsZero() {
		t.Fatal("reduced motion scheduled reveal pulse")
	}
}

func TestInspectorPulseKeepsClockAcrossPinnedPresentationChanges(t *testing.T) {
	p := SessionInspectorPaneState{}
	row := session.InspectorRecord{Key: "r", Type: "text", Raw: []byte(`{"text":"same"}`)}
	sticky := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			p.pulse.updateRows([]session.InspectorRecord{row}, c.Now())
			key := "ordinary"
			if sticky {
				key = "pinned"
			}
			inspectorTableRow(c.Key(key), &p, row, 0, false, sticky, 780, theme.Default().Light)
		})
	}, 780, 100)
	now := time.Unix(6000, 0)
	tt.SetTime(now)
	p.pulse.revealKey, p.pulse.revealGeneration = "r", 1
	tt.Frame()
	clock := p.pulse.mounted["r"].started
	row.Disclosure = "closed" // Lifecycle metadata alone is not a changed record.
	sticky = true
	tt.SetTime(now.Add(350 * time.Millisecond))
	if p.pulse.mounted["r"].started != clock {
		t.Fatal("pinning restarted the row pulse")
	}
	sticky = false
	tt.SetTime(now.Add(800 * time.Millisecond))
	if p.pulse.mounted["r"].started != clock || tt.Image().RGBAAt(700, 5) != (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("unpinning replayed expired reveal")
	}
}

func TestInspectorRenderedPulseIsLinearSevenHundredMillisecondsAndReducedMotion(t *testing.T) {
	for _, mode := range []string{"session-log", "chat-node"} {
		t.Run(mode, func(t *testing.T) {
			p := SessionInspectorPaneState{Mode: mode}
			row := session.InspectorRecord{Key: "r", Type: "text", Raw: []byte(`{"text":"same"}`)}
			colors := theme.Default().Light
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() {
					p.pulse.updateRows([]session.InspectorRecord{row}, c.Now())
					inspectorTableRow(c, &p, row, 0, false, false, 780, colors)
				})
			}, 780, 100)
			now := time.Unix(6000, 0)
			tt.SetTime(now)
			sample := func() color.RGBA { return tt.Image().RGBAAt(700, 5) }
			save := func(phase string) {
				if prefix := os.Getenv("INSPECTOR_PULSE_EVIDENCE"); prefix != "" {
					file, err := os.Create(prefix + "-" + mode + "-" + phase + ".png")
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(file, tt.Image())
					closeErr := file.Close()
					if err != nil {
						t.Fatal(err)
					}
					if closeErr != nil {
						t.Fatal(closeErr)
					}
				}
			}
			if sample() != (color.RGBA{255, 255, 255, 255}) {
				t.Fatalf("history flashed: %v", sample())
			}
			p.pulse.revealKey, p.pulse.revealGeneration = "r", 1
			tt.Frame()
			save("0")
			if sample() != (color.RGBA{211, 226, 255, 255}) {
				t.Fatalf("initial pulse color=%v", sample())
			}
			tt.SetTime(now.Add(350 * time.Millisecond))
			save("350")
			half := sample()
			if half.R < 231 || half.R > 234 || half.G < 239 || half.G > 242 || half.B != 255 {
				t.Fatalf("pulse not linear at350ms: %v", half)
			}
			tt.SetTime(now.Add(700 * time.Millisecond))
			save("700")
			if sample() != (color.RGBA{255, 255, 255, 255}) {
				t.Fatalf("pulse survived700ms: %v", sample())
			}
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			p.pulse.revealGeneration++
			tt.Frame()
			save("reduced")
			if sample() != (color.RGBA{255, 255, 255, 255}) {
				t.Fatalf("reduced-motion pulse: %v", sample())
			}
		})
	}
}
