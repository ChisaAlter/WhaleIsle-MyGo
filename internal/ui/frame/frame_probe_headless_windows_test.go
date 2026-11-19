//go:build windows && inspectorfixture

package frame

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestFrameProbe runs the frame probe's interactions in memory at the pace of a
// display, and leaves the engine's frame trace for scripts/frame-time-report.mjs
// to summarise. Nothing here touches a window, a compositor or the operator's
// screen, so the same numbers come out on any machine:
//
//	WHALEISLE_FRAME_PROBE=1 MYGO_FRAME_TRACE=out.csv \
//	  go test -count=1 -tags inspectorfixture -run TestFrameProbe ./internal/ui/frame
//
// WHALEISLE_FRAME_PROBE_REPEAT runs the whole script that many times, each on a
// fresh view; WHALEISLE_FRAME_PROBE_ONLY=resize runs one interaction, which is
// what a profile needs; WHALEISLE_FRAME_PROBE_MESSAGES sizes the transcript;
// WHALEISLE_FRAME_PROBE_DUMP=<dir> writes the settled frame of each interaction
// so two builds of the code can be compared pixel for pixel.
//
// -count=1 is not optional: the go command's test cache does not key on the
// environment, so a second run with other variables replays the first.
//
// The frame cost is the engine's, in memory: build and layout are the same code
// the window runs, while paint and present are the software renderer, which is
// the slowest path a frame can take.
func TestFrameProbe(t *testing.T) {
	if os.Getenv("WHALEISLE_FRAME_PROBE") == "" {
		t.Skip("frame probe not requested: set WHALEISLE_FRAME_PROBE=1")
	}
	if openProfile != "" {
		runtime.MemProfileRate = 4096
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
	if report := session.ViewTraceReport(); report != "" {
		t.Logf("who advanced the snapshot revision:\n%s", report)
	}
}

// openProfile names a pair of heap profiles to write either side of the open
// frame, so that one run says what that frame allocated and where: go tool
// pprof -sample_index=alloc_space -diff_base=open.before open.after.
var openProfile = os.Getenv("WHALEISLE_FRAME_PROBE_OPEN_PROFILE")

// openCPU names a prefix to write a CPU profile of each run's open frame to.
// The frames are 50 ms apart and 50 ms long, so one profile holds a handful of
// samples; pprof merges the files a run leaves behind:
// go tool pprof open-cpu.*.prof
var openCPU = os.Getenv("WHALEISLE_FRAME_PROBE_OPEN_CPU")
var openRun int

// writeAllocProfile writes the allocator's profile to path.
func writeAllocProfile(path string) {
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if err := pprof.Lookup("allocs").WriteTo(file, 0); err != nil {
		panic(err)
	}
}

// resizeRounds is how many times the probe changes the window size and back,
// resizeFrames how many frames it draws at each size, dragSteps how many frames
// a drag of the window edge takes and dragStep how far each one moves it. A
// window remeasures what it shows when its width changes, so the frames a
// resize or a drag produce are what a run is measuring.
var resizeRounds = probeInt("WHALEISLE_FRAME_PROBE_RESIZE_ROUNDS", 6)
var resizeFrames = probeInt("WHALEISLE_FRAME_PROBE_RESIZE_FRAMES", 4)
var dragSteps = probeInt("WHALEISLE_FRAME_PROBE_DRAG_STEPS", 24)
var dragStep = probeInt("WHALEISLE_FRAME_PROBE_DRAG_STEP", 12)

func probeInt(name string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return fallback
}

// settledSpare and settledBefore are the buffers settled compares, kept across
// calls so that comparing two frames allocates nothing after the first round.
var settledSpare, settledBefore = make([]byte, 0, 1280*720*4), make([]byte, 0, 1280*720*4)

// runFrameProbeOnce drives the interactions, each at the interval a display at
// the probe budget refreshes at, and marks the trace before each.
func runFrameProbeOnce(t *testing.T, prefix string) {
	budget := time.Second / 165
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	only := os.Getenv("WHALEISLE_FRAME_PROBE_ONLY")
	want := func(name string) bool { return only == "" || only == name }
	current := ""
	mark := func(name string) { current = prefix + name; ui.FrameTraceMark(current) }
	// The first frame a session draws is where the store projects its view and
	// the chat projects its messages. The trace names a frame after the last
	// mark set before it, so the mark goes here, before the view exists: set
	// later, the open of each run lands under the previous run's last mark.
	mark("open")
	model := newFrameProbeModel()
	var tester *ui.Tester
	open := func() {
		tester = ui.NewTester(model.viewFunc(false), model.width, model.height)
	}
	if openCPU != "" {
		file, err := os.Create(fmt.Sprintf("%s.%d.prof", openCPU, openRun))
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(file); err != nil {
			panic(err)
		}
		open()
		pprof.StopCPUProfile()
		file.Close()
	} else if openProfile != "" {
		writeAllocProfile(openProfile + ".before")
		open()
		writeAllocProfile(openProfile + ".after")
	} else {
		open()
	}
	openRun++
	frame := func() {
		now = now.Add(budget)
		tester.SetTime(now)
		tester.Frame()
	}
	frames := func(n int) {
		for i := 0; i < n; i++ {
			frame()
		}
	}

	if !tester.HasText(fmt.Sprintf("Probe answer %d", messages)) {
		t.Fatal("the store's messages did not reach the view")
	}
	t.Logf("messages in store: %d", len(model.view().Messages))

	if want("open") {
		// The open frame is the one ui.NewTester drew above: it is where the
		// store projects the view and the chat projects its messages, and it
		// is the only frame a long session pays for once.
		builds, hits := model.store.SnapshotStats()
		t.Logf("open: store projections=%d cache hits=%d", builds, hits)
		dump(tester, "open")
	}

	if want("scroll") {
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
		if down := model.list.ScrollOffset(); down <= up {
			t.Fatalf("wheel down did not scroll: offset stayed %.1f, then %.1f", up, down)
		}
		dump(tester, "scroll")
	}

	if want("stream") {
		mark("stream")
		for i := 0; i < 120; i++ {
			model.appendStreamToken()
			frame()
		}
		if after := model.streamedText(); len(after) == 0 {
			t.Fatal("streaming applied no text")
		}
		last := model.view().Messages
		if !containsToken(last[len(last)-1].Body) {
			t.Fatal("the streamed text did not reach the store's last message")
		}
		dump(tester, "stream")
	}

	if want("expand") {
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
		dump(tester, "expand")
	}

	if want("picker") {
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
	}

	if want("theme") {
		mark("theme_dark")
		model.setDark(true)
		frames(20)
		mark("theme_light")
		model.setDark(false)
		frames(20)
		dump(tester, "theme")
	}

	if want("resize") {
		// A resize remeasures the transcript for a new width. The frames after
		// it must settle: the last resize returns to the size the probe started
		// at, so the settled image can be compared with the one before.
		before := settled(tester, settledBefore[:0])
		for i := 0; i < resizeRounds; i++ {
			mark(fmt.Sprintf("resize_small_%d", i))
			tester.SetSize(980, 640)
			frames(resizeFrames)
			mark(fmt.Sprintf("resize_wide_%d", i))
			tester.SetSize(1280, 720)
			frames(resizeFrames)
		}
		if now := settled(tester, settledSpare[:0]); !sameFrame(before, now) {
			t.Fatalf("a resize and back changed the settled image: %d pixels differ", differentPixels(before, now))
		}
		dump(tester, "resize")
	}

	if want("drag") {
		// A drag of the window edge changes the width on every frame the
		// pointer moves, so each step here is one frame at a new width.
		before := settled(tester, settledBefore[:0])
		for i := 1; i <= dragSteps; i++ {
			mark(fmt.Sprintf("drag_%d", i))
			tester.SetSize(1280-dragStep*i, 720)
			tester.Frame()
			frame()
		}
		mark("drag_back")
		tester.SetSize(1280, 720)
		tester.Frame()
		frames(2)
		if now := settled(tester, settledSpare[:0]); !sameFrame(before, now) {
			t.Fatalf("a drag and back changed the settled image: %d pixels differ", differentPixels(before, now))
		}
	}
}

// settled draws a frame that changes nothing and returns its pixels in buf,
// which it reuses: a copy the size of the window allocated here is charged to
// the frame beside it, and the frames it stands between are what it measures.
func settled(tester *ui.Tester, buf []byte) []byte {
	pix := append(buf, tester.Pixels()...)
	tester.Frame()
	if next := tester.Pixels(); differentPixels(pix, next) != 0 {
		panic("a still view kept changing")
	}
	return pix
}

func sameFrame(a, b []byte) bool { return len(a) == len(b) && differentPixels(a, b) == 0 }

func differentPixels(a, b []byte) int {
	n := 0
	for i := 0; i+3 < len(a) && i+3 < len(b); i += 4 {
		if a[i] != b[i] || a[i+1] != b[i+1] || a[i+2] != b[i+2] {
			n++
		}
	}
	return n
}

func containsToken(body string) bool {
	return len(body) >= 5 && body[len(body)-5:] == "token"
}

// dump writes the frame's pixels where WHALEISLE_FRAME_PROBE_DUMP points, and
// logs a digest of them, so two builds of this code can be compared.
func dump(tester *ui.Tester, name string) {
	dir := os.Getenv("WHALEISLE_FRAME_PROBE_DUMP")
	if dir == "" {
		return
	}
	pix := tester.Pixels()
	sum := sha256.Sum256(pix)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(dir+"/"+name+".rgba", pix, 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("dump %s %d %s\n", name, len(pix), hex.EncodeToString(sum[:8]))
}
