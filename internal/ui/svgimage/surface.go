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

type imageTheme struct{ appearance, background string }

type Surface struct {
	handle    *previewprocess.Session
	parent    *mygo.Window
	cancel    context.CancelFunc
	once      sync.Once
	mu        sync.Mutex
	lastPaint time.Time
	err       error
	themes    chan imageTheme
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
	handle, err := previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: helperPath}, previewprocess.Document{HTML: markup, Width: max(240, min(1600, width)), InitialHeight: max(80, min(2000, height)), Appearance: appearance, Inline: true})
	if err != nil {
		cancel()
		return nil, err
	}
	if ctx.Err() != nil || parent.IsDestroyed() || !valid() {
		cancel()
		_ = handle.Close()
		return nil, errors.New("SVG image owner changed while loading")
	}
	s := &Surface{handle: handle, parent: parent, cancel: cancel, themes: make(chan imageTheme, 1)}
	go s.watch(ctx, valid)
	return s, nil
}

func (s *Surface) watch(ctx context.Context, valid func() bool) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	defer s.Close()
	var last imageTheme
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.handle.Done():
			s.fail(errors.New("isolated SVG image closed"))
			return
		case theme := <-s.themes:
			if theme != last {
				if err := s.handle.SetTheme(theme.appearance, map[string]string{"--background": theme.background}); err != nil {
					s.fail(err)
					return
				}
				last = theme
			}
		case <-tick.C:
			if s.parent.IsDestroyed() || !valid() {
				return
			}
			s.mu.Lock()
			hidden := time.Since(s.lastPaint) > 300*time.Millisecond
			s.mu.Unlock()
			if hidden {
				_ = s.handle.PresentImageSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
			}
		}
	}
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
func (s *Surface) Close() error {
	s.once.Do(func() {
		s.cancel()
		if err := s.handle.Close(); err != nil {
			s.fail(err)
		}
	})
	return s.Error()
}

func (s *Surface) Hide() { _ = s.handle.PresentImageSurface(0, 0, 0, 0, 0, 0, 0, 0, 0) }

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
	s.mu.Lock()
	s.lastPaint = time.Now()
	s.mu.Unlock()
	select {
	case s.themes <- imageTheme{appearance, background}:
	default:
	}
	if s.handle.PaintImagePixels(p, r) {
		return nil
	}
	clip := p.NativeSurfaceClip()
	if err := s.handle.PresentImageSurfaceClipped(s.parent.NativeHandle(), int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H), shape); err != nil {
		s.fail(err)
		go s.Close()
		return err
	}
	p.After(100 * time.Millisecond)
	return nil
}
