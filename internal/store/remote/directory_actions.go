package remote

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (s *Store) SetDirectoryPath(path string) {
	s.mu.Lock()
	s.view.Directory.InputPath = path
	s.mu.Unlock()
}
func (s *Store) SetDirectoryName(name string) {
	s.mu.Lock()
	s.view.Directory.NewName = name
	s.mu.Unlock()
}
func (s *Store) ShowDirectoryCreate(open bool) {
	s.mu.Lock()
	if !s.view.Busy {
		s.view.Directory.MkdirOpen = open
		s.view.Directory.NewName = ""
		s.view.Directory.CreateError = ""
	}
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}
func (s *Store) BrowseHome()                       { _ = s.directoryAction("home", "") }
func (s *Store) CreateDirectory(name string) error { return s.directoryAction("mkdir", name) }

func (s *Store) directoryAction(operation, name string) error {
	s.mu.Lock()
	if s.client == nil || s.view.Busy || s.view.CurrentID == "" || s.view.Error != "" {
		s.mu.Unlock()
		return fmt.Errorf("Select and refresh a remote machine first")
	}
	captured := s.view.Directory
	path := captured.Path
	if operation == "mkdir" {
		if !captured.CanChoose() {
			s.mu.Unlock()
			return fmt.Errorf("Browse a remote directory first")
		}
		var err error
		path, err = remoteChild(captured.Path, name)
		if err != nil {
			s.mu.Unlock()
			return err
		}
	}
	c, n, g, machineID := s.client, s.notify, s.generation, s.view.CurrentID
	s.view.Busy = true
	if operation == "mkdir" {
		s.view.Directory.CreateError = ""
	} else {
		s.view.Directory.Error = ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		var current struct {
			CurrentID string `json:"currentId"`
		}
		err := request(ctx, c, "GET", "status", nil, &current)
		if err == nil && current.CurrentID != machineID {
			err = fmt.Errorf("Active remote machine changed; refresh before continuing")
		}
		applied := false
		if err == nil && operation == "home" {
			var result struct {
				Home  string `json:"home"`
				Hint  string `json:"hint"`
				Error string `json:"error"`
			}
			err = request(ctx, c, "POST", "home", map[string]any{}, &result)
			if err == nil {
				path = strings.TrimSpace(result.Home)
				if path == "" {
					detail := strings.TrimSpace(result.Error + " " + result.Hint)
					if detail == "" {
						detail = "The remote host did not provide a home directory"
					}
					err = fmt.Errorf("%s", detail)
				}
			}
		}
		if err == nil && operation == "mkdir" {
			err = request(ctx, c, "POST", "fs", map[string]any{"op": "mkdir", "path": path}, nil)
			applied = err == nil
			path = captured.Path
		}
		var listing DirectorySnapshot
		if err == nil {
			err = request(ctx, c, "GET", "ls?path="+url.QueryEscape(path), nil, &listing)
			if err == nil && listing.Missing {
				err = fmt.Errorf("Remote directory is missing")
			}
		}
		s.mu.Lock()
		if s.generation != g {
			s.mu.Unlock()
			return
		}
		s.cancel = nil
		s.view.Busy = false
		if err != nil {
			if operation == "mkdir" && !applied {
				s.view.Directory.CreateError = err.Error()
			} else {
				s.view.Directory.Error = err.Error()
			}
			if applied {
				s.view.Directory.Error += "; folder was created, but refreshing failed"
				s.view.Directory.MkdirOpen = false
			}
		} else {
			listing.Loaded, listing.InputPath = true, listing.Path
			listing.LocalMirror, listing.SelectedPath = captured.LocalMirror, captured.SelectedPath
			s.view.Directory = listing
		}
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
	return nil
}

func (s *Store) SetDirectoryError(message string) {
	s.mu.Lock()
	s.view.Directory.Error = message
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}

func (s *Store) SetDirectoryCreateError(message string) {
	s.mu.Lock()
	s.view.Directory.CreateError = message
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}
