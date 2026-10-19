package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/previewprocess"
)

// Project pages use the same authorized bytes, isolated helper and native seat
// as published chat replies. There is no document-to-application capability.
func (s *projectController) startProjectVisualInteractiveLocked(win *mygo.Window, state *projectVisualResultsState, owner projectVisualOwner, page *projectVisualPage) {
	if page.interactiveCancel != nil {
		page.interactiveCancel()
	}
	if page.captureCancel != nil {
		page.captureCancel()
	}
	page.captureToken++
	page.interactiveToken++
	ctx, cancel := context.WithCancel(page.ctx)
	page.interactiveCancel = cancel
	token, width, dark, html := page.interactiveToken, page.width, page.dark, page.html
	page.handle = nil
	page.state.Surface, page.state.Bitmap = nil, nil
	page.state.Loading, page.state.Error, page.state.InteractiveError = true, "", ""
	go func() {
		defer cancel()
		settle := time.NewTimer(120 * time.Millisecond)
		defer settle.Stop()
		select {
		case <-ctx.Done():
			return
		case <-settle.C:
		}
		if !s.projectVisualCurrent(owner, state, page) {
			return
		}
		exe, err := os.Executable()
		var surface *publishedHTMLSurface
		if err == nil {
			appearance := "light"
			if dark {
				appearance = "dark"
			}
			var handle *previewprocess.Session
			handle, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: html, Width: width, Appearance: appearance, Inline: true})
			if err == nil {
				surface = newPublishedHTMLSurface(handle, win)
			}
		}
		if err == nil {
			s.dshMu.Lock()
			client := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			installed := ctx.Err() == nil && client == owner.client && s.projectFlow.visualResults == state && state.pages[owner] == page && page.interactiveToken == token && s.projectVisualCurrentLocked(owner)
			if installed {
				page.handle, page.state.Surface, page.state.Loading = surface, surface, false
			}
			s.mu.Unlock()
			if !installed {
				_ = surface.Close()
				return
			}
			win.Update(func() {})
			// Reuse the seat's paint lease for scrolling/virtualization. Also
			// expire the process when its authoritative Project owner changes,
			// even if that change removes the entire projects view from layout.
			tick := time.NewTicker(250 * time.Millisecond)
		wait:
			for {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					break wait
				case <-surface.Done():
					err = fmt.Errorf("isolated published page closed")
					break wait
				case <-tick.C:
					if !s.projectVisualCurrent(owner, state, page) {
						err = fmt.Errorf("published Project page changed")
						break wait
					}
				}
			}
			tick.Stop()
			if closeErr := surface.Close(); closeErr != nil {
				err = closeErr
			}
		}
		if ctx.Err() != nil {
			return
		}
		win.Update(func() {
			s.dshMu.Lock()
			client := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			if client != owner.client || s.projectFlow.visualResults != state || state.pages[owner] != page || page.interactiveToken != token || !s.projectVisualCurrentLocked(owner) {
				return
			}
			page.handle, page.state.Surface = nil, nil
			page.state.InteractiveError = err.Error()
			// This is an explicit fallback: the UI labels the PNG as static,
			// never reports a failed interactive helper as an active surface.
			s.startProjectVisualCaptureLocked(win, state, owner, page)
		})
	}()
}
