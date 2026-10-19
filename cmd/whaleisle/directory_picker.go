package main

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) openDirectoryPicker(sessions *session.Store, win *mygo.Window, mode string) {
	if sessions == nil {
		return
	}
	if mode == "" {
		mode = "workspace"
	}
	host := sessions.Snapshot().ComposerHostGeneration
	s.mu.Lock()
	s.dirPickOpen = true
	s.dirPickMode = mode
	s.dirPickHostGeneration = host
	s.dirPickListing = api.DirectoryListing{}
	s.dirPickNewName = ""
	s.mu.Unlock()
	// DirectoryBrowser mounts before its first scan; a slow or failed Host
	// must leave a visible, dismissible loading/error dialog.
	s.requestDirectoryPicker(sessions, win, "", "")
}

func (s *mainState) closeDirectoryPicker(win *mygo.Window) {
	s.mu.Lock()
	s.dirPickRequest++
	if s.dirPickCancel != nil {
		s.dirPickCancel()
		s.dirPickCancel = nil
	}
	s.dirPickOpen = false
	s.dirPickLoading = false
	s.dirPickNewName = ""
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) requestDirectoryPicker(sessions *session.Store, win *mygo.Window, path, createName string) {
	if sessions == nil {
		return
	}
	host := sessions.Snapshot().ComposerHostGeneration
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	if !s.dirPickOpen || s.dirPickHostGeneration != host {
		s.mu.Unlock()
		return
	}
	if s.dirPickCancel != nil {
		s.dirPickCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.dirPickCancel = cancel
	s.dirPickRequest++
	request, generation := s.dirPickRequest, s.bootStatus.Generation
	s.dirPickLoading = true
	s.dirPickErr = ""
	s.dirPickPath = path
	if createName != "" {
		s.dirPickNewName = ""
	}
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	current := func() bool {
		if ctx.Err() != nil || sessions.Snapshot().ComposerHostGeneration != host {
			return false
		}
		s.mu.Lock()
		valid := s.dirPickOpen && s.dirPickRequest == request && s.dirPickHostGeneration == host && s.bootStatus.Generation == generation
		s.mu.Unlock()
		s.dshMu.Lock()
		valid = valid && s.dshClient == client
		s.dshMu.Unlock()
		return valid
	}
	go func() {
		defer cancel()
		var listing api.DirectoryListing
		var err error
		if client == nil {
			err = session.ErrNoClient
		} else if current() {
			// Capture the client once: navigation and create-then-list never
			// resolve the Store again and silently jump to a replacement Host.
			if createName != "" {
				var created api.CreateDirectoryResult
				err = client.Call(ctx, "directoryPicker/createDirectory", map[string]any{"path": path, "name": createName}, &created)
				if err == nil {
					path = strings.TrimSpace(string(created))
				}
			}
			if err == nil && current() {
				s.mu.Lock()
				if s.dirPickOpen && s.dirPickRequest == request {
					s.dirPickPath = path
				}
				s.mu.Unlock()
				args := map[string]any{}
				if path = strings.TrimSpace(path); path != "" {
					args["path"] = path
				}
				err = client.Call(ctx, "directoryPicker/list", args, &listing)
			}
		}
		if !current() {
			return
		}
		s.mu.Lock()
		if s.dirPickOpen && s.dirPickRequest == request && s.bootStatus.Generation == generation {
			s.dirPickLoading = false
			s.dirPickCancel = nil
			if err != nil {
				s.dirPickErr = err.Error()
			} else {
				s.dirPickErr = ""
				s.dirPickListing = listing
			}
		}
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}

// directoryPickerView supplies the original DirectoryBrowser modal budget;
// dirpick.View owns its header, scrolling body and pinned footer.
func (s *mainState) directoryPickerView(c *ui.Context, win *mygo.Window, sessions *session.Store, themeID string, systemDark bool, onPick func(mode, path string)) {
	s.mu.Lock()
	open, host := s.dirPickOpen, s.dirPickHostGeneration
	s.mu.Unlock()
	if !open {
		return
	}
	if sessions == nil || sessions.Snapshot().ComposerHostGeneration != host {
		s.closeDirectoryPicker(win)
		return
	}
	s.mu.Lock()
	listing, loading, errorText := s.dirPickListing, s.dirPickLoading, s.dirPickErr
	mode, name, request, attemptedPath := s.dirPickMode, s.dirPickNewName, s.dirPickRequest, s.dirPickPath
	s.mu.Unlock()
	w, h := c.Size()
	width, height := min(float32(680), max(float32(0), w-48)), min(float32(500), max(float32(0), h-48))
	colors := theme.Resolve(themeID).Active(systemDark)
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Key("directory-picker-dialog").Width(width).Height(height).MinWidth(0).MinHeight(0).Padding(0).Radius(28).Background(chrome.Color(colors.BG)).Children(func() {
			dirpick.View(c, dirpick.Model{
				Listing: listing, ThemeID: themeID, SystemDark: systemDark, Loading: loading, Busy: loading,
				NewName: &name, Err: errorText,
				OnNavigate: func(path string) { s.requestDirectoryPicker(sessions, win, path, "") },
				OnRetry:    func() { s.requestDirectoryPicker(sessions, win, attemptedPath, "") },
				OnCreate: func(parent, value string) {
					if directorypicker.SelectablePath(parent) && strings.TrimSpace(value) != "" {
						s.requestDirectoryPicker(sessions, win, parent, strings.TrimSpace(value))
					}
				},
				OnPick: func(path string) {
					if !directorypicker.SelectablePath(path) || sessions.Snapshot().ComposerHostGeneration != host {
						return
					}
					s.mu.Lock()
					valid := s.dirPickOpen && !s.dirPickLoading && s.dirPickRequest == request
					s.mu.Unlock()
					if valid {
						s.closeDirectoryPicker(win)
						if onPick != nil {
							onPick(mode, path)
						}
					}
				},
				OnClose: func() { s.closeDirectoryPicker(win) },
			})
		})
	})
	if !open {
		s.closeDirectoryPicker(win)
		return
	}
	s.mu.Lock()
	if s.dirPickOpen && s.dirPickRequest == request {
		s.dirPickNewName = name
	}
	s.mu.Unlock()
}
