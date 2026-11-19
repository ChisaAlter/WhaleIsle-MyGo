package theme

import (
	"reflect"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// PaletteRevealDuration is how long a window takes to reach a switched
// palette: the 200 ms the interface's other transitions use, so the window does
// not move at a speed nothing else in it does.
const PaletteRevealDuration = 200 * time.Millisecond

// paletteSwitch is the switch a window is showing: where its palette came from,
// where it goes, and what it shows now.
type paletteSwitch struct {
	enabled bool
	primed  bool
	on      bool
	from    Colors
	shown   Colors
	to      Colors
	now     time.Time
	start   time.Time
}

var paletteReveal paletteSwitch

// EnablePaletteReveal makes a switched theme reach the window over
// PaletteRevealDuration instead of in one frame. The app calls this once at
// startup; a process that does not keeps switching palettes in one frame, as
// the tests that read exact palette values expect.
func EnablePaletteReveal() { paletteReveal.enabled = true }

// RevealFrame moves a running palette switch on and keeps the frames coming
// until it lands. A window calls this once per frame, before it reads a
// palette; a window that does not never starts a reveal.
func RevealFrame(c *ui.Context) {
	paletteReveal.now = c.Now()
	if !paletteReveal.on {
		return
	}
	k, done := paletteReveal.progress()
	if done {
		paletteReveal.on, paletteReveal.shown = false, paletteReveal.to
		return
	}
	paletteReveal.shown = mixPalettes(paletteReveal.from, paletteReveal.to, k)
	c.AnimationFrame()
}

// progress returns how far the switch has gone, and whether it has landed.
func (s *paletteSwitch) progress() (float32, bool) {
	k := float32(s.now.Sub(s.start)) / float32(PaletteRevealDuration)
	if k >= 1 {
		return 1, true
	}
	// Eased, so the window spends its time at the two palettes rather than at
	// the color between them, which is the one nobody chose.
	return ui.EaseInOut(max(k, 0)), false
}

// shownPalette returns what a window switching to palette to draws now. It
// starts the switch the first time a window asks for a palette other than the
// one it showed, and returns that one until the next frame moves it on.
func shownPalette(to Colors) Colors {
	if !paletteReveal.enabled {
		return to
	}
	if paletteReveal.on {
		if paletteReveal.to == to {
			return paletteReveal.shown
		}
		// Switched again before the last switch landed: start from what the
		// window shows.
		paletteReveal.from = paletteReveal.shown
	} else if paletteReveal.primed {
		paletteReveal.from = paletteReveal.to
	}
	if !paletteReveal.primed {
		// The first palette of the process arrives as it is: no window is
		// showing anything else yet.
		paletteReveal.primed, paletteReveal.shown, paletteReveal.to = true, to, to
		return to
	}
	paletteReveal.to, paletteReveal.on, paletteReveal.start = to, true, paletteReveal.now
	paletteReveal.shown = paletteReveal.from
	return paletteReveal.from
}

// mixPalettes returns the palette k of the way from a to b, every color mixed
// with premultiplied alpha so a color fading from transparent keeps its hue.
func mixPalettes(a, b Colors, k float32) Colors {
	var out Colors
	av, bv, ov := reflect.ValueOf(a), reflect.ValueOf(b), reflect.ValueOf(&out).Elem()
	for i := 0; i < av.NumField(); i++ {
		if ov.Field(i).Kind() != reflect.String {
			continue
		}
		ov.Field(i).SetString(mixToken(av.Field(i).String(), bv.Field(i).String(), k))
	}
	return out
}

// mixToken returns the CSS color k of the way from a to b.
func mixToken(a, b string, k float32) string {
	if a == b {
		return a
	}
	from, to := Color(a), Color(b)
	aa, ba := float32(from.A)/255, float32(to.A)/255
	alpha := aa + (ba-aa)*k
	if alpha <= 0 {
		return "rgba(0, 0, 0, 0)"
	}
	channel := func(x, y uint8) int {
		v := (float32(x)*aa + (float32(y)*ba-float32(x)*aa)*k) / alpha
		return int(max(0, min(v+0.5, 255)))
	}
	return "rgba(" + strconv.Itoa(channel(from.R, to.R)) + ", " +
		strconv.Itoa(channel(from.G, to.G)) + ", " +
		strconv.Itoa(channel(from.B, to.B)) + ", " +
		strconv.FormatFloat(float64(alpha), 'f', 3, 32) + ")"
}
