package pluginui

import (
	"context"
	_ "embed"
	"encoding/json"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed mirror_guard.js
var mirrorGuardSource []byte

// MirrorRect is source-main-frame CSS geometry, not desktop screen pixels.
type MirrorRect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}
type MirrorOccurrence struct {
	Handle      string     `json:"handle"`
	Entry       string     `json:"entry"`
	Label       string     `json:"label"`
	Turn        int        `json:"turn"`
	Seq         uint64     `json:"seq"`
	Rect        MirrorRect `json:"rect"`
	VisibleRect MirrorRect `json:"visibleRect"`
	Connected   bool       `json:"connected"`
	Visible     bool       `json:"visible"`
	Text        string     `json:"text"`
}
type mirrorGeometry struct {
	DocumentKey string   `json:"documentKey"`
	Revision    uint64   `json:"revision"`
	Closed      bool     `json:"closed"`
	Subject     *Subject `json:"subject"`
	Viewport    struct {
		W       float64 `json:"w"`
		H       float64 `json:"h"`
		OffsetX float64 `json:"offsetX"`
		OffsetY float64 `json:"offsetY"`
		Scale   float64 `json:"scale"`
	} `json:"viewport"`
	Owners   []MirrorOccurrence `json:"owners"`
	Portals  []MirrorPortal     `json:"portals"`
	Dirty    uint64             `json:"dirty"`
	Animated bool               `json:"animated"`
}

// MirrorFrame is an immutable, bounded capture shared by every native tail row.
// Its bitmap is the single original WebView image; occurrences never own React.
type MirrorFrame struct {
	Bitmap                     *ui.Bitmap
	PNGWidth, PNGHeight        int
	Occurrences                []MirrorOccurrence
	subject                    Subject
	page, revision, generation uint64
	geometry                   mirrorGeometry
	source                     *mygo.Window
	textFocus                  *mirrorTextFocus
	textAt                     time.Time
}
type mirrorState struct {
	generation          uint64
	inflight, inputBusy bool
	captureDone         chan struct{}
	width, height       int
	last                time.Time
	frame               *MirrorFrame
	layout              *MirrorFrame
	error               string
	pointerStage        string
	textFocusError      string
	pixelDirty          bool
	gesture             *mirrorGesture
	anchor              mirrorNativeAnchor
	pendingAnchor       mirrorNativeAnchor
	text                *mirrorTextState
}

func (o *Owner) invalidateMirrorLocked() {
	if o.mirror != nil {
		if o.mirror.gesture != nil {
			o.mirror.gesture.cancel()
		}
		if o.mirror.frame != nil {
			o.mirror.layout = o.mirror.frame
		}
		o.mirror.generation++
		o.mirror.frame = nil
	}
}
func (o *Owner) Inline() bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return !o.closed && o.mirror != nil
}

// RequestMirror coalesces a source refresh. It may be called during frame build;
// native capture, Eval and input run only on its worker, never inside Draw.
func (o *Owner) RequestMirror(width, height int) {
	if o == nil {
		return
	}
	o.consumeMirrorNativeTail()
	width = max(320, min(2048, width))
	height = max(240, min(1536, height))
	o.mu.Lock()
	m := o.mirror
	if m == nil || o.closed || o.masked || o.incompatible || !o.state.Connected || o.state.Subject == nil || !o.state.Subject.equal(o.selected) {
		o.mu.Unlock()
		return
	}
	if m.width != width || m.height != height {
		m.width, m.height = width, height
		o.invalidateMirrorLocked()
	}
	if m.inflight || m.inputBusy || m.text.captureDeferred() || m.frame != nil && time.Since(m.last) < 350*time.Millisecond {
		o.mu.Unlock()
		return
	}
	m.inflight = true
	completion := make(chan struct{})
	m.captureDone = completion
	h, capability, subject, page, revision, generation := o.host, o.gateway, o.selected, o.state.Page, o.state.Revision, m.generation
	previous := m.frame
	o.mu.Unlock()
	go func() {
		frame := previous
		var err error
		if !o.probeMirror(previous) {
			frame, err = o.captureMirror(h, capability, width, height, subject, page, revision, generation)
		}
		o.mu.Lock()
		if m.captureDone == completion {
			close(completion)
			m.captureDone = nil
			m.inflight = false
		}
		m.last = time.Now()
		if !o.closed && o.mirror == m && m.generation == generation && !o.masked && o.state.Page == page && o.selected.equal(subject) {
			m.error = ""
			if err != nil {
				m.error = err.Error()
			} else if !m.inputBusy {
				m.frame = frame
				m.layout = frame
				m.pixelDirty = false
			}
		}
		o.mu.Unlock()
		o.notify()
	}()
}

// MirrorSnapshot returns the retained frame only while its exact native/Host
// generation, Page and subject remain current. A revision change invalidates it.
func (o *Owner) MirrorSnapshot() *MirrorFrame {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror == nil || !o.mirrorCurrentLocked(o.mirror.frame) {
		return nil
	}
	return o.mirror.frame
}
func (o *Owner) MirrorError() string {
	if o == nil {
		return ""
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror == nil {
		return ""
	}
	return o.mirror.error
}
func (o *Owner) mirrorCurrent(f *MirrorFrame) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mirrorCurrentLocked(f)
}
func (o *Owner) mirrorCurrentLocked(f *MirrorFrame) bool {
	return f != nil && !o.closed && !o.masked && !o.incompatible && o.mirror != nil && o.mirror.frame == f && o.mirror.generation == f.generation && o.state.Connected && o.state.Page == f.page && o.state.Revision == f.revision && o.selected.equal(f.subject)
}

func mirrorEval[T any](ctx context.Context, p *mygo.Page, script string) (T, error) {
	var v T
	result, err := p.EvalContext(ctx, script)
	if err == nil {
		raw, e := json.Marshal(result)
		err = e
		if err == nil {
			err = json.Unmarshal(raw, &v)
		}
	}
	return v, err
}
func validMirrorGeometry(g mirrorGeometry, subject Subject) bool {
	finite := func(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
	if g.Closed || g.DocumentKey == "" || g.Revision == 0 || g.Subject == nil || !g.Subject.equal(subject) || g.Viewport.W <= 0 || g.Viewport.H <= 0 || g.Viewport.W > 2048 || g.Viewport.H > 1536 || g.Viewport.Scale != 1 || g.Viewport.OffsetX != 0 || g.Viewport.OffsetY != 0 || len(g.Owners) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range g.Owners {
		r := row.Rect
		if row.Handle == "" || seen[row.Handle] || len(row.Handle) > 64 || len(row.Entry) > 8192 || len(row.Text) > 65536 || row.Turn < 0 || row.Seq > 1<<53-1 || !finite(r.X) || !finite(r.Y) || !finite(r.W) || !finite(r.H) || r.W < 0 || r.H < 0 {
			return false
		}
		seen[row.Handle] = true
		v := row.VisibleRect
		if row.Visible && (!row.Connected || r.W <= 0 || r.H <= 0 || !mirrorFragmentValid(v) || v.X < -.01 || v.Y < -.01 || v.X+v.W > g.Viewport.W+.01 || v.Y+v.H > g.Viewport.H+.01 || v.X < r.X-.01 || v.Y < r.Y-.01 || v.X+v.W > r.X+r.W+.01 || v.Y+v.H > r.Y+r.H+.01) {
			return false
		}
	}
	return true
}

// PaintMirror draws only a current retained bitmap. Source/crop/native DIP and
// PNG pixel ratios stay separate; target aspect ratio follows original bounds.
func (o *Owner) PaintMirror(p *ui.Painter, dst ui.Rect, f *MirrorFrame, row MirrorOccurrence) {
	if !o.mirrorCurrent(f) || !row.Visible || row.Rect.W <= 0 || row.Rect.H <= 0 {
		return
	}
	fragment, ok := NewMirrorFragment(row.Rect, MirrorRect{W: f.geometry.Viewport.W, H: f.geometry.Viewport.H}, row.VisibleRect)
	if !ok {
		return
	}
	mapped, ok := MapMirrorFragment(dst, p.NativeSurfaceClip(), fragment)
	if !ok {
		return
	}
	s := mapped.Scale
	whole := ui.Rect{X: dst.X - float32(s*row.Rect.X), Y: dst.Y - float32(s*row.Rect.Y), W: float32(f.geometry.Viewport.W * s), H: float32(f.geometry.Viewport.H * s)}
	p.Clip(mapped.Clip, 0, func() { p.Image(f.Bitmap, whole, ui.FillBox) })
}
