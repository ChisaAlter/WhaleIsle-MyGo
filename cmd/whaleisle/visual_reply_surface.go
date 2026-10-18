package main

import (
	"errors"
	"github.com/egoist/mygo"
	"sync"
	"time"
	"whaleisle/internal/desktop/previewprocess"
)

// The live helper survives inline/expanded moves. A paint lease hides native
// children when virtualization, another pane or a modal removes their seat.
type surfaceTheme struct {
	appearance string
	variables  map[string]string
}
type publishedHTMLSurface struct {
	*previewprocess.Session
	parent        *mygo.Window
	mu            sync.Mutex
	lastPaint     time.Time
	themes        chan surfaceTheme
	failure       error
	failureSignal chan struct{}
}

func newPublishedHTMLSurface(handle *previewprocess.Session, win *mygo.Window) *publishedHTMLSurface {
	s := &publishedHTMLSurface{Session: handle, parent: win, themes: make(chan surfaceTheme, 1), failureSignal: make(chan struct{}, 1)}
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-handle.Done():
				return
			case <-s.failureSignal:
				_ = handle.Close()
				return
			case theme := <-s.themes:
				if err := handle.SetTheme(theme.appearance, theme.variables); err != nil {
					s.fail(err)
				}
			case <-tick.C:
				s.mu.Lock()
				expired := time.Since(s.lastPaint) > 300*time.Millisecond
				s.mu.Unlock()
				if expired {
					_ = handle.PresentSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
				}
			}
		}
	}()
	return s
}
func (s *publishedHTMLSurface) Present(x, y, w, h, cx, cy, cw, ch int, appearance string, variables map[string]string) error {
	s.mu.Lock()
	s.lastPaint = time.Now()
	s.mu.Unlock()
	select {
	case s.themes <- surfaceTheme{appearance, variables}:
	default:
	}
	err := s.Session.PresentSurface(s.parent.NativeHandle(), x, y, w, h, cx, cy, cw, ch)
	if err != nil {
		s.fail(err)
	}
	return err
}

func (s *publishedHTMLSurface) fail(err error) {
	s.mu.Lock()
	if s.failure == nil {
		s.failure = err
	}
	s.mu.Unlock()
	select {
	case s.failureSignal <- struct{}{}:
	default:
	}
}
func (s *publishedHTMLSurface) Close() error {
	err := s.Session.Close()
	s.mu.Lock()
	failure := s.failure
	s.mu.Unlock()
	return errors.Join(err, failure)
}
