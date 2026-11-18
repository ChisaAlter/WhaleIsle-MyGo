// Package svgimage owns a single authorized inert SVG image on the existing
// isolated helper surface. It starts no new browser shell or image interpreter.
package svgimage

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/previewprocess"
)

const defaultSVGOffscreenReclaimTTL = 5 * time.Second

type imageTheme struct{ appearance, background string }

type imageSession interface {
	Done() <-chan struct{}
	SetTheme(appearance string, vars map[string]string) error
	PresentImageSurface(parent uintptr, x, y, w, h, clipX, clipY, clipW, clipH int) error
	PresentImageSurfaceClipped(parent uintptr, x, y, w, h, clipX, clipY, clipW, clipH int, shape previewprocess.ImageSurfaceClip) error
	PaintImagePixels(p *ui.Painter, r ui.Rect) bool
	ImageSize() (float64, float64)
	Close() error
}

type Surface struct {
	handle      imageSession
	parent      *mygo.Window
	cancel      context.CancelFunc
	once        sync.Once
	mu          sync.Mutex
	lastPaint   time.Time
	visible     bool
	hiddenSince time.Time
	reclaimed   bool
	reclaimTTL  time.Duration
	err         error
	themes      chan imageTheme
	wake        chan struct{}
}

// Open consumes an already-authorized immutable snapshot. The caller owns ctx
// and valid: session, Host, tab, attachment/cache generation and window changes
// must cancel/revoke that exact owner. A slow old result is closed before return.
// Calls belong off the UI thread, like the existing published-page helper.
func Open(ctx context.Context, parent *mygo.Window, helperPath string, data []byte, width, height int, appearance string, valid func() bool) (*Surface, error) {
	if parent == nil || parent.IsDestroyed() || valid == nil || !valid() {
		return nil, errors.New("SVG image owner is no longer active")
	}
	markup, err := imagepreview.SVGSurfaceHTML(data)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	handle, err := previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: helperPath}, previewprocess.Document{HTML: markup, Width: max(240, min(1600, width)), InitialHeight: max(80, min(2000, height)), Appearance: appearance, Inline: true, InertImage: true})
	if err != nil {
		cancel()
		return nil, err
	}
	if ctx.Err() != nil || parent.IsDestroyed() || !valid() {
		cancel()
		_ = handle.Close()
		return nil, errors.New("SVG image owner changed while loading")
	}
	s := &Surface{
		handle:  handle,
		parent:  parent,
		cancel:  cancel,
		visible: true,
		themes:  make(chan imageTheme, 1),
		wake:    make(chan struct{}, 1),
	}
	go s.watch(ctx, valid)
	return s, nil
}

func newSurfaceForSession(ctx context.Context, parent *mygo.Window, handle imageSession, reclaimTTL time.Duration, valid func() bool) *Surface {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Surface{
		handle:     handle,
		parent:     parent,
		cancel:     cancel,
		visible:    true,
		reclaimTTL: reclaimTTL,
		themes:     make(chan imageTheme, 1),
		wake:       make(chan struct{}, 1),
	}
	go s.watch(ctx, valid)
	return s
}

func (s *Surface) effectiveReclaimTTL() time.Duration {
	if s.reclaimTTL > 0 {
		return s.reclaimTTL
	}
	return defaultSVGOffscreenReclaimTTL
}

func (s *Surface) signalWatch() {
	if s.wake == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Surface) markVisible() {
	s.mu.Lock()
	wasHidden := !s.visible
	s.visible = true
	s.hiddenSince = time.Time{}
	s.lastPaint = time.Now()
	s.mu.Unlock()
	if wasHidden {
		s.signalWatch()
	}
}

func (s *Surface) watch(ctx context.Context, valid func() bool) {
	defer s.Close()
	var last imageTheme
	var reclaimTimer *time.Timer
	stopTimer := func() {
		if reclaimTimer == nil {
			return
		}
		if !reclaimTimer.Stop() {
			select {
			case <-reclaimTimer.C:
			default:
			}
		}
		reclaimTimer = nil
	}
	defer stopTimer()

	syncHiddenTimer := func() {
		s.mu.Lock()
		visible := s.visible
		hiddenSince := s.hiddenSince
		s.mu.Unlock()
		if visible {
			stopTimer()
			return
		}
		if reclaimTimer != nil {
			return
		}
		wait := s.effectiveReclaimTTL()
		if !hiddenSince.IsZero() {
			if elapsed := time.Since(hiddenSince); elapsed >= wait {
				wait = time.Millisecond
			} else {
				wait -= elapsed
			}
		}
		reclaimTimer = time.NewTimer(wait)
	}

	for {
		var timerC <-chan time.Time
		if reclaimTimer != nil {
			timerC = reclaimTimer.C
		}
		select {
		case <-ctx.Done():
			return
		case <-s.handle.Done():
			s.mu.Lock()
			reclaimed := s.reclaimed
			s.mu.Unlock()
			if !reclaimed {
				s.fail(errors.New("isolated SVG image closed"))
			}
			return
		case theme := <-s.themes:
			if theme != last {
				if err := s.handle.SetTheme(theme.appearance, map[string]string{"--background": theme.background}); err != nil {
					s.fail(err)
					return
				}
				last = theme
			}
		case <-s.wake:
			if (s.parent != nil && s.parent.IsDestroyed()) || (valid != nil && !valid()) {
				return
			}
			syncHiddenTimer()
		case <-timerC:
			reclaimTimer = nil
			if (s.parent != nil && s.parent.IsDestroyed()) || (valid != nil && !valid()) {
				return
			}
			s.mu.Lock()
			stillHidden := !s.visible
			if stillHidden {
				s.reclaimed = true
			}
			s.mu.Unlock()
			if stillHidden {
				return
			}
		}
	}
}

func (s *Surface) Reclaimed() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	r := s.reclaimed
	s.mu.Unlock()
	return r
}

func (s *Surface) fail(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}
func (s *Surface) Error() error          { s.mu.Lock(); defer s.mu.Unlock(); return s.err }
func (s *Surface) Done() <-chan struct{} { return s.handle.Done() }
func (s *Surface) Size() (float32, float32) {
	w, h := s.handle.ImageSize()
	return float32(w), float32(h)
}
func (s *Surface) Close() error {
	s.once.Do(func() {
		s.cancel()
		if err := s.handle.Close(); err != nil {
			s.fail(err)
		}
	})
	return s.Error()
}

func (s *Surface) Hide() {
	if s == nil {
		return
	}
	s.mu.Lock()
	alreadyHidden := !s.visible
	if s.visible {
		s.visible = false
		s.hiddenSince = time.Now()
	}
	s.mu.Unlock()
	if alreadyHidden {
		return
	}
	s.signalWatch()
	_ = s.handle.PresentImageSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
}

// Paint receives an aspect-fitted image rectangle from the native owner, so
// existing zoom/pan/crop/thumbnail geometry and input remain authoritative.
// It returns the actual helper error; callers must display that failure instead
// of calling a static PNG an active animation. Paint never waits for IPC.
func (s *Surface) Paint(p *ui.Painter, r ui.Rect, appearance, background string) error {
	return s.PaintClipped(p, r, appearance, background, previewprocess.ImageSurfaceClip{})
}

func (s *Surface) PaintClipped(p *ui.Painter, r ui.Rect, appearance, background string, shape previewprocess.ImageSurfaceClip) error {
	if err := s.Error(); err != nil {
		return err
	}
	select {
	case <-s.Done():
		return errors.New("SVG image surface is closed")
	default:
	}
	s.markVisible()
	select {
	case s.themes <- imageTheme{appearance, background}:
	default:
	}
	if s.handle.PaintImagePixels(p, r) {
		return nil
	}
	clip := p.NativeSurfaceClip()
	var parentHandle uintptr
	if s.parent != nil && mygo.App.IsReady() {
		parentHandle = s.parent.NativeHandle()
	}
	if err := s.handle.PresentImageSurfaceClipped(parentHandle, int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H), shape); err != nil {
		s.fail(err)
		go s.Close()
		return err
	}
	return nil
}
