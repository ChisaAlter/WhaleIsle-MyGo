package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

// The input/approval stop action must not navigate to a Team observer. It
// submits one Project-level stop and refreshes authoritative activity in place.
func (s *projectController) runProjectStop(win *mygo.Window, id string) {
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == id && p.Lifecycle == "ready" && !p.Archiving {
			known = true
			break
		}
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available || !known {
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		note, err := rpc.Control(ctx, id, "stop", "")
		var listing projectflow.Listing
		var refreshErr error
		if err == nil {
			listing, refreshErr = rpc.List(ctx)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			f.lifecycleNote = note
			if refreshErr != nil {
				f.refreshNote = "Stop accepted; refresh failed: " + refreshErr.Error()
				return
			}
			f.listing = listing
			f.loaded = true
		})
	}()
}
