package composer

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func effectNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %.9f, want %.9f", got, want)
	}
}

func TestEffectTimingCSSBezier(t *testing.T) {
	// Exact coordinates at bezier parameter .25 or .5, independent of
	// the inversion implementation. All three source curves are distinct.
	for _, tc := range []struct {
		easing string
		x, y   float64
	}{
		{"linear", .25, .25}, {"ease", .3125, .5375},
		{"ease-in-out", .274375, .15625}, {"cubic-bezier", .2125, .15625},
	} {
		effectNear(t, beamEase(tc.x, tc.easing), tc.y)
		effectNear(t, beamEase(0, tc.easing), 0)
		effectNear(t, beamEase(1, tc.easing), 1)
	}
	effectNear(t, typingEase(.35), .5)
}

func TestEffectTimingTypingKeyframes(t *testing.T) {
	for _, tc := range []struct {
		effect                  string
		p, alpha, offset, scale float64
	}{
		{"drop", 0, 0, -.55, 1.18}, {"drop", .55, -1, 0, 1}, {"drop", 1, 0, 0, 1},
		{"drop", .35, 1, 1, -1}, {"drop", .1225, .5, 1, -1},
		{"drop", .1925, -1, -.275, 1.09},
		{"rise", 0, .9, 0, 1}, {"rise", .35, .45, -.35, 1}, {"rise", 1, 0, -.7, 1},
		{"flash", 0, 0, 0, 1.35}, {"flash", .2, 1, 0, -1},
		{"flash", .07, .5, 0, -1}, {"flash", .35, -1, 0, 1.175}, {"flash", 1, 0, 0, 1},
	} {
		t.Run(tc.effect+"/"+time.Duration(tc.p*1000).String(), func(t *testing.T) {
			a, y, s := typingKeyframe(tc.effect, tc.p)
			if tc.alpha >= 0 {
				effectNear(t, float64(a), tc.alpha)
			}
			if tc.offset <= 0 {
				effectNear(t, float64(y), tc.offset)
			}
			if tc.scale >= 0 {
				effectNear(t, float64(s), tc.scale)
			}
		})
	}
	if typingDuration(100) != 400*time.Millisecond || typingDuration(200) != 200*time.Millisecond {
		t.Fatal("typing duration no longer derives from 400ms source token")
	}
}

func TestEffectTimingCaretScheduling(t *testing.T) {
	for _, tc := range []struct {
		ms      int
		visible bool
		next    int
	}{
		{0, true, 500}, {450, true, 50}, {499, true, 1}, {500, false, 500},
		{950, false, 50}, {999, false, 1}, {1000, true, 500}, {1370, true, 130},
	} {
		visible, next := typingCaretBlink(time.Duration(tc.ms) * time.Millisecond)
		if visible != tc.visible || next != time.Duration(tc.next)*time.Millisecond {
			t.Fatalf("at %dms got visible=%v next=%v", tc.ms, visible, next)
		}
	}
	// A late repaint still schedules the next phase boundary, without drift.
	visible, next := typingCaretBlink(1703 * time.Millisecond)
	if visible || next != 297*time.Millisecond {
		t.Fatal(visible, next)
	}
}

func TestEffectTimingBeamActivationAndReducedMotion(t *testing.T) {
	origin := time.Date(2026, 10, 19, 12, 34, 56, 370_000_000, time.UTC)
	var timeline beamTimeline
	a, elapsed, animate := timeline.update(origin, false, false)
	if a != 0 || elapsed != 0 || animate {
		t.Fatal(a, elapsed, animate)
	}
	a, elapsed, animate = timeline.update(origin, true, false)
	if a != 0 || elapsed != 0 || !animate {
		t.Fatal(a, elapsed, animate)
	}
	a, elapsed, _ = timeline.update(origin.Add(210*time.Millisecond), true, false)
	effectNear(t, a, .802403387584857)
	effectNear(t, elapsed, .21)
	a, _, _ = timeline.update(origin.Add(420*time.Millisecond), true, false)
	effectNear(t, a, 1)
	a, elapsed, animate = timeline.update(origin.Add(time.Second), false, false)
	if a != 1 || elapsed != 0 || !animate {
		t.Fatal(a, elapsed, animate)
	}
	a, _, animate = timeline.update(origin.Add(1420*time.Millisecond), false, false)
	if a != 0 || animate {
		t.Fatal(a, animate)
	}
	_, elapsed, _ = timeline.update(origin.Add(2*time.Second), true, false)
	if elapsed != 0 {
		t.Fatal("reactivation retained old animation phase")
	}
	a, elapsed, animate = timeline.update(origin.Add(2100*time.Millisecond), true, true)
	if a != 0 || elapsed != 0 || animate || timeline.active {
		t.Fatal("reduced motion retained beam")
	}
	_, elapsed, _ = timeline.update(origin.Add(3*time.Second), true, false)
	if elapsed != 0 {
		t.Fatal("reduced-motion release retained old animation phase")
	}
}

func TestEffectTimingBeamTransitionReversal(t *testing.T) {
	origin := time.Unix(100, 0)
	var timeline beamTimeline
	timeline.update(origin, true, false)
	before := timeline.opacity(origin.Add(210 * time.Millisecond))
	a, _, _ := timeline.update(origin.Add(210*time.Millisecond), false, false)
	effectNear(t, a, before)
	if timeline.duration >= beamOpacityDuration || timeline.duration <= 0 {
		t.Fatal("CSS reverse did not shorten transition", timeline.duration)
	}
	a, _, animate := timeline.update(origin.Add(210*time.Millisecond+timeline.duration), false, false)
	if a != 0 || animate {
		t.Fatal("fade did not retire", a, animate)
	}
}

func TestEffectTimingBeamChildActivation(t *testing.T) {
	origin := time.Unix(100, 0)
	var timeline beamTimeline
	style := readBeamStyle(nil)
	style.HueCycle.Enabled = false
	timeline.update(origin, true, false)
	timeline.effectTimes(origin, style)
	style.HueCycle.Enabled = true
	b, h := timeline.effectTimes(origin.Add(2*time.Second), style)
	if b != 0 || h != 0 {
		t.Fatal("new child inherited parent phase", b, h)
	}
	style.Breathing.Enabled = true
	b, h = timeline.effectTimes(origin.Add(3*time.Second), style)
	if b != 0 || h != 1 {
		t.Fatal("enabling breathing restarted existing hue", b, h)
	}
	style.HueCycle.Enabled = false
	timeline.effectTimes(origin.Add(4*time.Second), style)
	style.HueCycle.Enabled = true
	b, h = timeline.effectTimes(origin.Add(5*time.Second), style)
	if b != 2 || h != 0 {
		t.Fatal("hue reactivation retained phase or restarted breathing", b, h)
	}
	timeline.update(origin.Add(6*time.Second), false, false)
	b, h = timeline.effectTimes(origin.Add(6*time.Second), style)
	if b != 0 || h != 0 || !timeline.hueStarted.IsZero() || !timeline.breatheStarted.IsZero() {
		t.Fatal("inactive class retained child animations")
	}
}

func TestEffectTimingBeamTrajectories(t *testing.T) {
	style := readBeamStyle(nil)
	style.Period, style.Easing = 2, "ease"
	style.Direction = "clockwise"
	effectNear(t, beamPhase(.625, style), .5375)
	style.Direction = "counterclockwise"
	effectNear(t, beamPhase(1.375, style), .5375)
	style.Direction = "pingPong"
	effectNear(t, beamPhase(3.375, style), .5375)
	style.Breathing.Enabled, style.Breathing.Period, style.Breathing.Amplitude = true, 6, .3
	for _, tc := range []struct{ seconds, want float64 }{{0, 1}, {3, 1.3}, {6, 1}, {9, 1.3}, {12, 1}} {
		effectNear(t, beamBreathing(tc.seconds, style), tc.want)
	}
	style.Hue, style.HueCycle.Range, style.HueCycle.Period = 10, 30, 8
	effectNear(t, beamHue(0, style), -20)
	effectNear(t, beamHue(4, style), 40)
	effectNear(t, beamHue(8, style), -20)
	style.Breathing.Enabled = false
	effectNear(t, beamHue(6, style), 40) // source standalone hue duration is 12s
	style.HueCycle.Enabled = false
	effectNear(t, beamHue(6, style), 0)
}

func TestEffectTimingTypingLifecycle(t *testing.T) {
	draft := ""
	client := NewDraftClient(&draft)
	m := Model{TypingFx: true, TypingFxStyle: json.RawMessage(`{"effect":"drop","cursor":"block"}`)}
	client.configureTyping(m, false)
	born := time.Unix(100, 123)
	client.typing.caretBorn = born
	client.captureTyping("", "a")
	client.configureTyping(m, false)
	if len(client.typing.echoes) != 1 || client.typing.caretBorn != born {
		t.Fatal("ordinary frame restarted typing effects")
	}
	m.TypingFxStyle = json.RawMessage(`{"effect":"rise","cursor":"block"}`)
	client.configureTyping(m, false)
	if len(client.typing.echoes) != 0 {
		t.Fatal("effect switch retained canceled echoes")
	}
	client.captureTyping("", "a")
	client.configureTyping(m, true)
	if len(client.typing.echoes) != 0 || !client.typing.caretBorn.IsZero() {
		t.Fatal("reduced motion retained typing animation")
	}
}

func BenchmarkEffectTimingKeyframes(b *testing.B) {
	style := readBeamStyle(nil)
	style.Easing = "cubic-bezier"
	style.Breathing.Enabled = true
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := float64(i%1000) / 1000
		typingKeyframe("drop", p)
		beamPhase(p, style)
		beamBreathing(p, style)
		beamHue(p, style)
	}
}
