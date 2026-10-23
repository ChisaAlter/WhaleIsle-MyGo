package chat

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func startInspectorSmooth(t *testing.T) (*ui.Tester, *InspectorNavigation, *ui.ScrollState, *ui.ListState) {
	t.Helper()
	tt, n, scroll, list, _ := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	tt.SetTime(time.Unix(7000, 0).Add(150 * time.Millisecond))
	if scroll.Y <= 100 || scroll.Y >= 1400 || !list.ScrollAnimating() || !n.reveal.pending {
		t.Fatalf("missing running navigation: %+v %+v", scroll, n.reveal)
	}
	return tt, n, scroll, list
}

func TestInspectorSmoothInputCancelsGloballyWithoutSwallowingWheel(t *testing.T) {
	for _, kind := range []string{"wheel-chat", "wheel-composer", "wheel-horizontal", "wheel-zero", "pointer-composer", "key-page", "key-modified", "key-escape"} {
		t.Run(kind, func(t *testing.T) {
			tt, n, scroll, list := startInspectorSmooth(t)
			before := scroll.Y
			switch kind {
			case "wheel-chat":
				tt.Scroll(250, 200, 0, 30)
			case "wheel-composer":
				tt.Scroll(250, 340, 0, 30)
			case "wheel-horizontal":
				tt.Scroll(250, 340, 20, 0)
			case "wheel-zero":
				tt.Scroll(250, 340, 0, 0)
			case "pointer-composer":
				tt.Press(250, 340)
				tt.Release(250, 340)
			case "key-page":
				tt.Key(0, ui.KeyPageDown)
			case "key-modified":
				tt.Key(ui.Ctrl|ui.Shift, ui.KeyHome)
			case "key-escape":
				tt.Key(0, ui.KeyEscape)
			}
			if n.reveal.pending || n.Highlight != "" || list.ScrollAnimating() || !n.reveal.started.IsZero() {
				t.Fatalf("input did not cancel: %+v", n.reveal)
			}
			want := before
			if kind == "wheel-chat" {
				want += 30
			}
			if math.Abs(float64(scroll.Y-want)) > .01 {
				t.Fatalf("interrupt lost normal input: before%v after%v want%v", before, scroll.Y, want)
			}
			tt.SetTime(time.Unix(7000, 0).Add(3 * time.Second))
			if math.Abs(float64(scroll.Y-want)) > .01 || !n.reveal.started.IsZero() {
				t.Fatal("cancelled generation resumed")
			}
		})
	}
}

func TestInspectorSmoothOrdinaryKeysContinueAndFinishedFlashSurvivesInput(t *testing.T) {
	tt, n, scroll, list := startInspectorSmooth(t)
	for _, key := range []ui.Key{ui.KeyA, ui.KeyLeft, ui.KeyRight, ui.KeyEnter} {
		tt.Key(0, key)
		if !n.reveal.pending || !list.ScrollAnimating() {
			t.Fatalf("key%v cancelled", key)
		}
	}
	duration := inspectorChatScrollDuration(1300)
	tt.SetTime(time.Unix(7000, 0).Add(duration))
	tt.SetTime(time.Unix(7000, 0).Add(duration + time.Millisecond))
	tt.SetTime(time.Unix(7000, 0).Add(duration + 2*time.Millisecond))
	if scroll.Y != 1400 || n.reveal.started.IsZero() {
		t.Fatal("ordinary keys prevented landing")
	}
	fixed := n.reveal.rect
	start := n.reveal.started
	tt.Scroll(250, 200, 0, 20)
	tt.Key(0, ui.KeyEscape)
	tt.Press(250, 340)
	tt.Release(250, 340)
	if n.reveal.rect != fixed || n.reveal.started != start {
		t.Fatal("finished animation still captured interruption")
	}
}

func TestInspectorSmoothTraceMatchesChromiumCurveAcrossDisplayScales(t *testing.T) {
	var traces []map[string]any
	for _, scale := range []float32{1, 1.5, 2} {
		tt, n, scroll, list, _ := inspectorRevealFixture(t)
		tt.SetScale(scale)
		list.ScrollToOffset(100)
		tt.Frame()
		n.Target = &InspectorTarget{ID: "25"}
		tt.Frame()
		duration := inspectorChatScrollDuration(1300 * scale)
		var samples []map[string]any
		for _, fraction := range []float32{0, .1, .25, .5, .75, 1} {
			elapsed := time.Duration(float64(duration) * float64(fraction))
			tt.SetTime(time.Unix(7000, 0).Add(elapsed))
			want := 100 + 1300*inspectorChatScrollEase(fraction)
			if math.Abs(float64(scroll.Y-want)) > .02 {
				t.Fatalf("scale%v fraction%v offset%v want%v", scale, fraction, scroll.Y, want)
			}
			samples = append(samples, map[string]any{"ms": float64(elapsed) / float64(time.Millisecond), "y": scroll.Y})
		}
		tt.SetTime(time.Unix(7000, 0).Add(duration + time.Millisecond))
		tt.SetTime(time.Unix(7000, 0).Add(duration + 2*time.Millisecond))
		if n.reveal.started.IsZero() {
			t.Fatal("final frame did not flash")
		}
		traces = append(traces, map[string]any{"scale": scale, "distanceDIP": 1300, "durationMs": float64(duration) / float64(time.Millisecond), "samples": samples})
	}
	if path := os.Getenv("INSPECTOR_SMOOTH_EVIDENCE"); path != "" {
		b, err := json.MarshalIndent(traces, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
