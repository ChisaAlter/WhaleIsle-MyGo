package ui

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// A frame trace is the frame statistics of every frame, not only the slow
// ones, written where a probe can read them back: no screenshot shows the
// pacing of frames, and the log's own timestamps are too coarse and too
// perturbing to take one line per frame at 120 Hz.
//
//	MYGO_FRAME_TRACE=<path>  append one CSV row per frame to <path>
//
// Unset, frames trace nothing and the engine does not measure them. The trace
// adds the frame's start on the wall clock and on the monotonic clock, the gap
// since the previous frame started (frame pace), how long the frame took and
// how its parts divided, and whether it built the view or only painted it.
//
// FrameTraceMark writes a row naming what the application was doing, so a
// reader of the trace can tell the frames of one interaction from the next.

var frameTrace = newFrameTrace(os.Getenv("MYGO_FRAME_TRACE"))

// frameTraceFlush is how stale the trace on disk may be: rows arrive per
// frame, so a run that is killed still leaves nearly all of them.
const frameTraceFlush = 50 * time.Millisecond

type frameTracer struct {
	w         *bufio.Writer
	file      *os.File
	lastStart time.Time
	lastFlush time.Time
}

func newFrameTrace(path string) *frameTracer {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		log.Printf("mygo: MYGO_FRAME_TRACE=%q: %v", path, err)
		return nil
	}
	t := &frameTracer{w: bufio.NewWriterSize(f, 1<<16), file: f}
	fmt.Fprintln(t.w, "kind,frame,start_unix_ns,gap_ns,dur_ns,build_ns,layout_ns,paint_ns,present_ns,passes,path,name,cycles,cycles_build,cycles_layout,cycles_paint,cycles_present,cycles_total,alloc_bytes,gc_cycles,present_upload_ns,present_draw_ns,present_swap_ns,cause")
	fmt.Fprintf(t.w, "cal,0,0,-1,0,0,0,0,0,0,none,clock,%.0f,0,0,0,0,0,0,0,0,0,0,-\n", measureCycleRate())
	return t
}

// measureCycleRate returns the cycles this thread uses in a second, by working
// it for a stretch long enough that the monotonic clock measures it: the trace
// rows carry cycles, which resolve a frame finer than that clock can, and the
// rate turns them into time.
func measureCycleRate() float64 {
	const stretch = 60 * time.Millisecond
	before, tid := readThreadCycles()
	if before == 0 {
		return 0
	}
	start := time.Now()
	for time.Since(start) < stretch {
	}
	after, afterTid := readThreadCycles()
	elapsed := time.Since(start)
	if after <= before || afterTid != tid {
		return 0
	}
	return float64(after-before) / elapsed.Seconds()
}

// frame appends the row of one frame, which f measured.
func (t *frameTracer) frame(rt *engine, f *frameStats, total time.Duration) {
	if t == nil {
		return
	}
	gap := int64(-1)
	if !t.lastStart.IsZero() {
		gap = int64(f.start.Sub(t.lastStart))
	}
	t.lastStart = f.start
	path := "none"
	if p, ok := rt.host.(interface{ framePath() string }); ok {
		if s := p.framePath(); s != "" {
			path = strings.ReplaceAll(s, " ", "_")
		}
	}
	fmt.Fprintf(t.w, "frame,%d,%d,%d,%d,%d,%d,%d,%d,%d,%s,,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%s\n",
		rt.frame, f.start.UnixNano(), gap, int64(total),
		int64(f.laps[phaseBuild]), int64(f.laps[phaseLayout]), int64(f.laps[phasePaint]), int64(f.laps[phasePresent]),
		f.passes, path,
		f.cycles[phaseBuild]+f.cycles[phaseLayout]+f.cycles[phasePaint]+f.cycles[phasePresent],
		f.cycles[phaseBuild], f.cycles[phaseLayout], f.cycles[phasePaint], f.cycles[phasePresent],
		f.cycles[phaseBuild]+f.cycles[phaseLayout]+f.cycles[phasePaint]+f.cycles[phasePresent],
		uint64(f.delta[statBytes]), uint64(f.delta[statCycles]),
		int64(f.presentUpload), int64(f.presentDraw), int64(f.presentSwap), f.cause)
	t.flush()
}

// mark names what the application is doing in the trace.
func (t *frameTracer) mark(name string) {
	if t == nil {
		return
	}
	fmt.Fprintf(t.w, "mark,0,%d,-1,0,0,0,0,0,0,none,%s,0,0,0,0,0,0,0,0,0,0,0,-\n", time.Now().UnixNano(), csvToken(name))
	t.flush()
}

// flush writes out the rows of the last frames, at most every
// frameTraceFlush, so a frame does not wait on the disk.
func (t *frameTracer) flush() {
	now := time.Now()
	if !t.lastFlush.IsZero() && now.Sub(t.lastFlush) < frameTraceFlush {
		return
	}
	t.lastFlush = now
	_ = t.w.Flush()
}

// close flushes and closes the trace file. Tests use it to release the file;
// the process keeps the trace open for its whole run instead.
func (t *frameTracer) close() {
	if t == nil {
		return
	}
	_ = t.w.Flush()
	_ = t.file.Close()
}

// csvToken fits a name into one CSV field.
func csvToken(s string) string {
	s = strings.NewReplacer(",", "_", " ", "_", "\n", "_", "\r", "_").Replace(s)
	if s == "" {
		return "-"
	}
	return s
}

// FrameTraceMark names an interaction in the frame trace, from the view that
// starts it: the frames after the mark belong to it until the next mark. It
// does nothing unless MYGO_FRAME_TRACE is set.
func FrameTraceMark(name string) { frameTrace.mark(name) }

// Frame causes: why a frame ran, appended to each trace row as the cause
// column. A request records its cause while frames are traced; the frame
// that starts next takes every cause recorded since the previous one began,
// so a cause names what asked, not which frame finally served it. Untraced,
// recording costs one nil check.
const (
	causeChanged    uint32 = 1 << iota // the app changed state: Window.Invalidate/Update
	causeRequest                       // the engine asked for a frame built anew (input handling, focus, resize, …)
	causeAnimation                     // Context.AnimationFrame: the view animates
	causeRepaint                       // Painter.AnimationFrame: a drawing moves
	causeAfter                         // Context.After's timer came due
	causePaintAfter                    // Painter.After's timer came due
	causeTheme                         // the system appearance or preferences changed
	causeInput                         // an input event arrived since the last frame
	causeKinds
)

var causeNames = [...]string{"changed", "request", "animation", "repaint", "after", "paint-after", "theme", "input"}

// frameCauses holds each window engine's pending causes; frameCauseTags the
// tags the app passed with FrameTraceCause, which name no window.
var (
	frameCauses    sync.Map // *engine → *atomic.Uint32
	frameCauseMu   sync.Mutex
	frameCauseTags map[string]struct{}
)

// traceCause records why rt is about to ask for a frame.
func traceCause(rt *engine, cause uint32) {
	if frameTrace == nil || rt == nil {
		return
	}
	v, ok := frameCauses.Load(rt)
	if !ok {
		v, _ = frameCauses.LoadOrStore(rt, new(atomic.Uint32))
	}
	v.(*atomic.Uint32).Or(cause)
}

// FrameTraceCause names the source of an Invalidate or Update the app is
// about to make, for the cause column of the frame trace ("tag:<source>").
// It does nothing unless MYGO_FRAME_TRACE is set, so call sites need no
// guard. Tags are process-wide: with several windows a tag may land on the
// next frame of another window.
func FrameTraceCause(source string) {
	if frameTrace == nil || source == "" {
		return
	}
	frameCauseMu.Lock()
	if frameCauseTags == nil {
		frameCauseTags = map[string]struct{}{}
	}
	frameCauseTags[source] = struct{}{}
	frameCauseMu.Unlock()
}

// takeFrameCause consumes rt's pending causes for the frame starting now.
// lastInput is the engine's last input time before the frame; an input
// since the previous frame counts as a cause. A frame nothing recorded a
// cause for was asked for by the platform (first show, expose, a host
// invalidate outside the engine): "surface".
func takeFrameCause(rt *engine, lastInput, prevInput time.Time) string {
	if frameTrace == nil {
		return ""
	}
	var bits uint32
	if v, ok := frameCauses.Load(rt); ok {
		bits = v.(*atomic.Uint32).Swap(0)
	}
	if !lastInput.IsZero() && lastInput.After(prevInput) {
		bits |= causeInput
	}
	var parts []string
	for i := 0; i < len(causeNames); i++ {
		if bits&(1<<i) != 0 {
			parts = append(parts, causeNames[i])
		}
	}
	frameCauseMu.Lock()
	if len(frameCauseTags) > 0 {
		tags := make([]string, 0, len(frameCauseTags))
		for t := range frameCauseTags {
			tags = append(tags, "tag:"+t)
		}
		clear(frameCauseTags)
		sort.Strings(tags)
		parts = append(parts, tags...)
	}
	frameCauseMu.Unlock()
	if len(parts) == 0 {
		return "surface"
	}
	return csvToken(strings.Join(parts, "+"))
}

// forgetFrameCauses drops a closed engine's pending causes.
func forgetFrameCauses(rt *engine) {
	if frameTrace != nil {
		frameCauses.Delete(rt)
	}
}
