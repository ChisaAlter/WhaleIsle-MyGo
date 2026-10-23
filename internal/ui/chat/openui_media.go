package chat

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/desktop/webview"
)

// All fields other than b's own mutex-protected fields are UI-thread owned.
// A native browser is a sibling of the helper surface, never its descendant.
// The helper continues to receive only typed local rendering/state messages.
type openUIMedia struct {
	ctx        context.Context
	block      *openUIBlock
	helper     *previewprocess.Session
	parent     *mygo.Window
	valid      func() bool
	invalidate func()
	revision   uint64
	seats      map[string]*openUIMediaSeat
	removed    map[string]bool
	rect       ui.Rect
	clip       ui.Rect
	holes      []mygo.Rectangle
	painted    time.Time
	masked     bool
	off        []func()
	closed     bool
}

type openUIMediaHost interface {
	Alive() bool
	State() webview.State
	SetVisible(bool)
	PresentMedia(mygo.Rectangle, mygo.Rectangle, []mygo.Rectangle) (bool, error)
	Close() error
}

type openUIMediaSeat struct {
	request  openui.MediaRequest
	sequence uint64
	host     openUIMediaHost
	failed   string
}

func newOpenUIMedia(ctx context.Context, b *openUIBlock, helper *previewprocess.Session, parent *mygo.Window, valid func() bool, revision uint64, invalidate func()) *openUIMedia {
	m := &openUIMedia{ctx: ctx, block: b, helper: helper, parent: parent, valid: valid, revision: revision, invalidate: invalidate, seats: map[string]*openUIMediaSeat{}, removed: map[string]bool{}}
	mygo.RunOnMain(func() {
		if parent == nil || parent.IsDestroyed() {
			m.closed = true
			return
		}
		m.off = []func(){parent.OnHide(m.suspend), parent.OnMinimize(m.suspend), parent.OnClosed(m.closeMain), parent.OnResize(func() { m.suspend(); invalidate() }), parent.OnMove(func() { m.suspend(); invalidate() })}
	})
	return m
}

func (m *openUIMedia) current() bool {
	if m == nil || m.closed || m.ctx.Err() != nil || m.parent == nil || m.parent.IsDestroyed() || m.valid == nil || !m.valid() {
		return false
	}
	select {
	case <-m.helper.Done():
		return false
	default:
	}
	m.block.mu.Lock()
	defer m.block.mu.Unlock()
	return m.block.handle == m.helper && m.block.media == m && m.block.revision == m.revision && m.block.parent == m.parent
}

func (m *openUIMedia) closeSeat(s *openUIMediaSeat) {
	if s.host != nil {
		h := s.host
		s.host = nil
		_ = h.Close()
	}
}

// Hidden/unpainted seats destroy the private browser, stopping hidden playback
// and releasing its private profile. Their next visible seat starts paused.
func (m *openUIMedia) suspend() {
	m.painted = time.Time{}
	for _, seat := range m.seats {
		m.closeSeat(seat)
	}
}

func (m *openUIMedia) closeMain() {
	if m.closed {
		return
	}
	m.closed = true
	m.suspend()
	for _, off := range m.off {
		off()
	}
	m.off = nil
	m.seats = nil
	m.removed = nil
}

func (m *openUIMedia) Close() {
	if m != nil {
		mygo.RunOnMain(m.closeMain)
	}
}

// Retire runs as soon as the native UI observes a source/streaming replacement,
// before its coalesced render reaches the helper. Late old-revision events can
// neither revive old components nor affect this revision's new components.
func (m *openUIMedia) Retire(revision uint64) {
	if m == nil {
		return
	}
	mygo.RunOnMain(func() {
		if m.closed || revision <= m.revision {
			return
		}
		m.suspend()
		m.revision = revision
		m.seats = map[string]*openUIMediaSeat{}
		m.removed = map[string]bool{}
	})
}

func (m *openUIMedia) ExpirePaint() {
	if m == nil {
		return
	}
	mygo.RunOnMain(func() {
		if !m.current() || time.Since(m.painted) >= 300*time.Millisecond || !m.parent.IsVisible() || m.parent.IsMinimized() {
			m.suspend()
		}
	})
}

// Paint receives coordinates only from MyGo's trusted native painter. Modal,
// ancestor-scroll and workspace-float clipping stay independent of DOM claims.
func (m *openUIMedia) Paint(rect, clip ui.Rect, holes []mygo.Rectangle, masked bool) {
	if m == nil {
		return
	}
	if !m.current() || !childImageVisible(rect, clip) {
		m.suspend()
		return
	}
	m.rect, m.clip, m.painted, m.masked = rect, clip, time.Now(), masked
	m.holes = append(m.holes[:0], holes...)
	for _, seat := range m.seats {
		if _, err := m.present(seat); err != nil {
			m.report(err)
		}
	}
}

func (m *openUIMedia) report(err error) {
	if err == nil {
		return
	}
	m.block.mu.Lock()
	changed := m.block.err != err.Error()
	m.block.err = err.Error()
	m.block.mu.Unlock()
	if changed {
		m.invalidate()
	}
}

func (m *openUIMedia) Handle(event openui.Event) openui.APIResponse {
	status, state := 409, ""
	err := errors.New("OpenUI media owner is no longer active")
	mygo.RunOnMain(func() {
		// Consume even a stale request so the bounded session queue cannot retain
		// an authorization ticket after its original source has been replaced.
		if claimErr := m.helper.ClaimOpenUIMedia(event); claimErr != nil {
			status, err = 403, claimErr
			return
		}
		if !m.current() || event.Media.Revision != m.revision {
			return
		}
		r := *event.Media
		seat := m.seats[r.ComponentID]
		if seat != nil && event.Sequence <= seat.sequence {
			err = errors.New("OpenUI media update was superseded")
			return
		}
		if r.Operation == "remove" {
			if seat != nil {
				m.closeSeat(seat)
				delete(m.seats, r.ComponentID)
			}
			m.removed[r.ComponentID] = true
			status, state, err = 200, "closed", nil
			return
		}
		if m.removed[r.ComponentID] {
			err = errors.New("OpenUI media component was removed")
			return
		}
		if seat == nil || seat.request.VideoID != r.VideoID || seat.failed != "" {
			if !event.UserGesture {
				status, err = 403, errors.New("Click the YouTube component to activate this private player")
				return
			}
			if seat != nil {
				m.closeSeat(seat)
			}
			seat = &openUIMediaSeat{}
			m.seats[r.ComponentID] = seat
		}
		seat.request, seat.sequence = r, event.Sequence
		state, err = m.present(seat)
		status = 200
		if err != nil {
			status = 503
		}
	})
	body := map[string]any{"ok": err == nil}
	if err != nil {
		body["error"] = err.Error()
	} else {
		body["state"] = state
	}
	raw, _ := json.Marshal(body)
	return openui.APIResponse{InstanceID: event.InstanceID, RequestID: event.RequestID, Status: status, Body: raw}
}

func (m *openUIMedia) present(seat *openUIMediaSeat) (string, error) {
	if !m.current() || !seat.request.Visible || time.Since(m.painted) >= 300*time.Millisecond || !m.parent.IsVisible() || m.parent.IsMinimized() {
		m.closeSeat(seat)
		return "hidden", nil
	}
	bounds, clip, ok := openUIMediaGeometry(seat.request, m.rect, m.clip, runtime.GOOS)
	if !ok {
		m.closeSeat(seat)
		return "hidden", nil
	}
	return m.presentHost(seat, bounds, clip)
}

func (m *openUIMedia) presentHost(seat *openUIMediaSeat, bounds, clip mygo.Rectangle) (string, error) {
	if m.masked {
		// Inspector picking is a temporary occlusion, not an offscreen
		// lifecycle change. Preserve the private player and its playback state.
		if seat.host != nil {
			seat.host.SetVisible(false)
		}
		return "hidden", nil
	}
	if seat.failed != "" {
		return "", errors.New(seat.failed)
	}
	if seat.host != nil {
		if state := seat.host.State(); state.Error != "" || !seat.host.Alive() {
			seat.failed = "YouTube private browser could not load this video"
			m.closeSeat(seat)
			return "", errors.New(seat.failed)
		}
	}
	if seat.host == nil {
		// The fixed player wrapper supplies the installed application's identity
		// as its HTTPS base/referrer, in its own fresh, binding-free profile.
		title := strings.TrimSpace(seat.request.Title)
		if title == "" {
			title = "YouTube"
		}
		host, err := webview.NewYouTubeHost(m.parent, seat.request.VideoID, title, m.invalidate, m.invalidate)
		if err != nil {
			seat.failed = err.Error()
			return "", err
		}
		seat.host = host
		if !m.current() {
			m.closeSeat(seat)
			return "hidden", nil
		}
	}
	shown, err := seat.host.PresentMedia(bounds, clip, m.holes)
	if err != nil {
		seat.failed = err.Error()
		m.closeSeat(seat)
		return "", err
	}
	if !shown {
		m.closeSeat(seat)
		return "hidden", nil
	}
	// This acknowledges only the native view. YouTube owns playback errors;
	// neither document load nor a successful seat is proof of video playback.
	return "native-view-created", nil
}

func openUIMediaGeometry(r openui.MediaRequest, seat, paintClip ui.Rect, platform string) (mygo.Rectangle, mygo.Rectangle, bool) {
	if seat.W <= 0 || seat.H <= 0 || paintClip.W <= 0 || paintClip.H <= 0 || r.Viewport.Width <= 0 || r.Viewport.Height <= 0 {
		return mygo.Rectangle{}, mygo.Rectangle{}, false
	}
	sx, sy := float64(seat.W)/r.Viewport.Width, float64(seat.H)/r.Viewport.Height
	x, y := float64(seat.X), float64(seat.Y)
	if platform == "darwin" {
		// The existing macOS helper image is always 900x700, contain-fitted by
		// Session.RenderPixels. Match its exact letterbox and input transform.
		if r.Viewport.Width != 900 || r.Viewport.Height != 700 {
			return mygo.Rectangle{}, mygo.Rectangle{}, false
		}
		sx = math.Min(sx, sy)
		sy = sx
		x += (float64(seat.W) - 900*sx) / 2
		y += (float64(seat.H) - 700*sy) / 2
	} else if platform == "windows" {
		// HWND resizing reflows the DOM. Never stretch stale pre-resize DOM
		// rectangles into a newer paint seat; wait for its viewport update.
		if math.Abs(r.Viewport.Width-float64(int(seat.W))) > 1 || math.Abs(r.Viewport.Height-float64(int(seat.H))) > 1 {
			return mygo.Rectangle{}, mygo.Rectangle{}, false
		}
	} else {
		return mygo.Rectangle{}, mygo.Rectangle{}, false
	}
	mapRect := func(v openui.MediaRect) openui.MediaRect {
		return openui.MediaRect{X: x + v.X*sx, Y: y + v.Y*sy, Width: v.Width * sx, Height: v.Height * sy}
	}
	bounds, clip := mapRect(r.Rect), mapRect(r.Clip)
	left := math.Max(clip.X, math.Max(bounds.X, float64(paintClip.X)))
	top := math.Max(clip.Y, math.Max(bounds.Y, float64(paintClip.Y)))
	right := math.Min(clip.X+clip.Width, math.Min(bounds.X+bounds.Width, float64(paintClip.X+paintClip.W)))
	bottom := math.Min(clip.Y+clip.Height, math.Min(bounds.Y+bounds.Height, float64(paintClip.Y+paintClip.H)))
	// Clip inward at integer DIP boundaries; never expose a fractional strip
	// outside the parent's accepted paint region.
	native := mygo.Rectangle{X: int(math.Floor(bounds.X)), Y: int(math.Floor(bounds.Y)), Width: int(math.Ceil(bounds.X+bounds.Width)) - int(math.Floor(bounds.X)), Height: int(math.Ceil(bounds.Y+bounds.Height)) - int(math.Floor(bounds.Y))}
	visible := mygo.Rectangle{X: int(math.Ceil(left)), Y: int(math.Ceil(top)), Width: int(math.Floor(right)) - int(math.Ceil(left)), Height: int(math.Floor(bottom)) - int(math.Ceil(top))}
	return native, visible, native.Width > 0 && native.Height > 0 && visible.Width > 0 && visible.Height > 0
}
