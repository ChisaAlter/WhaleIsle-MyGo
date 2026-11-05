package main

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/update"
)

const (
	nativeUpdateFirstCheck   = time.Minute
	nativeUpdateInterval     = 6 * time.Hour
	nativeUpdateNetworkRetry = 15 * time.Minute
)

// The app binds this once to its real main window. The launcher is only one
// consumer of the same center, so hiding/closing it does not stop discovery.
func (s *mainState) startNativeUpdateCenter(win *mygo.Window) {
	s.ensureNativeUpdateCenter()
	s.mu.Lock()
	r := &s.nativeReleases
	if r.backgroundCancel != nil || s.quitting {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.backgroundSequence++
	sequence := r.backgroundSequence
	r.backgroundCancel = cancel
	s.mu.Unlock()
	go func() {
		defer cancel()
		defer func() {
			s.mu.Lock()
			if s.nativeReleases.backgroundSequence == sequence {
				s.nativeReleases.backgroundCancel = nil
			}
			s.mu.Unlock()
		}()
		delay := nativeUpdateFirstCheck
		for {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			delay = nativeUpdateInterval
			settled := make(chan error, 1)
			if !s.refreshNativeReleasesOwned(win, ctx, func(err error) { settled <- err }) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case err := <-settled:
				if nativeUpdateNetworkError(err) {
					delay = nativeUpdateNetworkRetry
				}
			}
		}
	}()
}

func nativeUpdateNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var network net.Error
	var status *update.CatalogHTTPError
	return errors.As(err, &network) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &status) && status.StatusCode >= 500
}

// A close retires scheduled checks and pauses an owned download. Its partial
// and verified installer remain available to the next run's recovery owner.
func (s *mainState) stopNativeUpdateCenter() {
	s.clearNativeUpdateReadyNotice()
	s.closeNativeUpdateModal(nil)
	s.mu.Lock()
	r := &s.nativeReleases
	r.backgroundSequence++
	background, active := r.backgroundCancel, r.cancel
	r.backgroundCancel = nil
	if r.busy && !r.installing && r.phase == "downloading" {
		r.pauseRequested = true
	}
	if r.installing {
		active = nil
	}
	s.mu.Unlock()
	if background != nil {
		background()
	}
	if active != nil {
		active()
	}
}
