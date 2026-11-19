//go:build windows && inspectorfixture

package frame

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestFrameProbe runs the frame probe's interactions in memory at the pace of a
// display, and leaves the engine's frame trace for scripts/frame-time-report.mjs
// to summarise. Nothing here touches a window, a compositor or the operator's
// screen, so the same numbers come out on any machine:
//
//	WHALEISLE_FRAME_PROBE=1 MYGO_FRAME_TRACE=out.csv \
//	  go test -tags inspectorfixture -run TestFrameProbe ./internal/ui/frame
//
// WHALEISLE_FRAME_PROBE_REPEAT runs the whole script that many times, each on a
// fresh view, which gives the report several samples of every interaction and a
// CPU profile enough frames to attribute.
//
// The frame cost is the engine's, in memory: build and layout are the same code
// the window runs, while paint and present are the software renderer, which is
// the slowest path a frame can take.
func TestFrameProbe(t *testing.T) {
	if os.Getenv("WHALEISLE_FRAME_PROBE") == "" {
		t.Skip("frame probe not requested: set WHALEISLE_FRAME_PROBE=1")
	}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	repeat := 1
	if raw := os.Getenv("WHALEISLE_FRAME_PROBE_REPEAT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			t.Fatalf("WHALEISLE_FRAME_PROBE_REPEAT=%q is not a count", raw)
		}
		repeat = n
	}
	for i := 0; i < repeat; i++ {
		prefix := ""
		if repeat > 1 {
			prefix = fmt.Sprintf("r%d_", i)
		}
		runFrameProbeOnce(t, prefix)
	}
}

// runFrameProbeOnce drives every interaction once, at the interval a display at
// the probe budget refreshes at, and marks the trace before each.
func runFrameProbeOnce(t *testing.T, prefix string) {
	budget := time.Second / 165
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	model := newFrameProbeModel()
	tester := ui.NewTester(model.viewFunc(false), model.width, model.height)
	mark := func(name string) { ui.FrameTraceMark(prefix + name) }
	frame := func() {
		now = now.Add(budget)
		tester.SetTime(now)
		tester.Frame()
	}
	if !tester.HasText(fmt.Sprintf("Probe answer %d", messages)) {
		t.Fatal("the store's messages did not reach the view")
	}
	t.Logf("messages in store: %d", len(model.view().Messages))
	frames := func(n int) {
		for i := 0; i < n; i++ {
			frame()
		}
	}

	mark("warm")
	frames(5)

	mark("scroll")
	tester.Move(400, 300)
	start := model.list.ScrollOffset()
	for i := 0; i < 40; i++ {
		tester.Scroll(400, 300, 0, -99)
		frame()
	}
	frames(20)
	up := model.list.ScrollOffset()
	if up >= start {
		t.Fatalf("wheel up did not scroll: offset stayed %.1f, then %.1f", start, up)
	}
	for i := 0; i < 40; i++ {
		tester.Scroll(400, 300, 0, 99)
		frame()
	}
	frames(20)
	down := model.list.ScrollOffset()
	if down <= up {
		t.Fatalf("wheel down did not scroll: offset stayed %.1f, then %.1f", up, down)
	}
	t.Logf("scroll %.1f up to %.1f, down to %.1f", start, up, down)

	mark("stream")
	for i := 0; i < 120; i++ {
		model.appendStreamToken()
		frame()
	}
	if after := model.streamedText(); len(after) == 0 {
		t.Fatal("streaming applied no text")
	}
	last := model.view().Messages
	if !strings.Contains(last[len(last)-1].Body, "token") {
		t.Fatal("the streamed text did not reach the store's last message")
	}

	mark("expand_open")
	if !model.toggleExpanded() {
		t.Fatal("the tool details did not open")
	}
	frames(30)
	mark("expand_close")
	if model.toggleExpanded() {
		t.Fatal("the tool details did not close")
	}
	frames(30)

	// The picker's popup is the proof: its options paint over the view only
	// while it is open, which the frame's pixels show and the text tree does
	// not. The buffers are reused so the check itself allocates nothing.
	var closed, opened, after []byte
	pixels := func(dst *[]byte) {
		pix := tester.Image().Pix
		if cap(*dst) < len(pix) {
			*dst = make([]byte, len(pix))
		} else {
			*dst = (*dst)[:len(pix)]
		}
		copy(*dst, pix)
	}
	changed := func(a, b []byte) int {
		n := 0
		for i := 0; i+3 < len(a) && i+3 < len(b); i += 4 {
			if a[i] != b[i] || a[i+1] != b[i+1] || a[i+2] != b[i+2] {
				n++
			}
		}
		return n
	}
	// A click away from the picker drops its focus ring, and the pointer goes
	// back to the chat: the frames before and after then differ only by what
	// opened, not by where the pointer or the focus sat.
	neutral := func() {
		tester.ClickAt(400, 690)
		tester.Move(400, 300)
		frames(5)
	}
	pickerClick := func() { tester.ClickAt(float32(model.width-pickerX), 14) }
	neutral()
	pixels(&closed)
	mark("picker_open")
	pickerClick()
	frames(30)
	pixels(&opened)
	if changed(closed, opened) < 1000 {
		t.Fatalf("the picker opened nothing: %d pixels changed", changed(closed, opened))
	}
	mark("picker_close")
	pickerClick()
	frames(30)
	neutral()
	pixels(&after)
	if n, back := changed(opened, after), changed(closed, after); back > 0 && n < 1000 {
		t.Fatalf("the picker did not close: %d of its %d pixels went away", n, back)
	}

	mark("theme_dark")
	model.setDark(true)
	frames(20)
	mark("theme_light")
	model.setDark(false)
	frames(20)

	mark("resize_small")
	tester.SetSize(980, 640)
	frames(20)
	mark("resize_wide")
	tester.SetSize(1280, 720)
	frames(20)
}
