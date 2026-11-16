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
	lastHeight    int
	themes        chan surfaceTheme
	armExpire     chan struct{}
	onExpire      func()
	onHeight      func()
	failure       error
	failureSignal chan struct{}
}

func newPublishedHTMLSurface(handle *previewprocess.Session, win *mygo.Window) *publishedHTMLSurface {
	s := &publishedHTMLSurface{
		Session:       handle,
		parent:        win,
		themes:        make(chan surfaceTheme, 1),
		armExpire:     make(chan struct{}, 1),
		failureSignal: make(chan struct{}, 1),
	}
	go func() {
		timer := time.NewTimer(300 * time.Millisecond)
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		var timerC <-chan time.Time
		resetTimer := func(d time.Duration) {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(d)
			timerC = timer.C
		}
		defer timer.Stop()
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
			case <-s.armExpire:
				resetTimer(300 * time.Millisecond)
			case <-timerC:
				timerC = nil
				s.mu.Lock()
				elapsed := time.Since(s.lastPaint)
				onExpire := s.onExpire
				s.mu.Unlock()
				if remaining := 300*time.Millisecond - elapsed; remaining > 0 {
					resetTimer(remaining)
					continue
				}
				_ = handle.PresentSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
				if onExpire != nil {
					onExpire()
				}
			}
		}
	}()
	return s
}

// SetHooks registers expiry and content-height callbacks on the published surface.
func (s *publishedHTMLSurface) SetHooks(onExpire, onHeight func()) {
	s.mu.Lock()
	s.onExpire = onExpire
	s.onHeight = onHeight
	s.mu.Unlock()
}

func (s *publishedHTMLSurface) Present(x, y, w, h, cx, cy, cw, ch int, appearance string, variables map[string]string) error {
	s.mu.Lock()
	s.lastPaint = time.Now()
	measured := s.Session.ContentHeight()
	heightChanged := measured >= 80 && measured != s.lastHeight
	if heightChanged {
		s.lastHeight = measured
	}
	onHeight := s.onHeight
	s.mu.Unlock()
	select {
	case s.armExpire <- struct{}{}:
	default:
	}
	if heightChanged && onHeight != nil {
		onHeight()
	}
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
