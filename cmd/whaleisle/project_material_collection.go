package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

type projectMaterialCollectionState struct {
	snapshot projectflow.MaterialCollection
	cancel   context.CancelFunc
}

func (s *projectController) closeProjectMaterialCollectionLocked() {
	f := s.projectFlow
	if d := f.materialCollection; d != nil {
		if d.cancel != nil {
			d.cancel()
			f.busy = false
		}
		f.materialCollection = nil
	}
}
func (s *projectController) bindProjectMaterialCollection(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	d := s.projectFlow.materialCollection
	client, generation, navigation := s.projectFlow.client, s.projectFlow.generation, s.projectFlow.navigation
	if d != nil {
		snapshot := d.snapshot
		m.MaterialCollection = &snapshot
	}
	s.mu.Unlock()
	m.OnMaterialCollection = func(id string) {
		s.mu.Lock()
		f := s.projectFlow
		if win == nil || client == nil || f.client != client || s.dshClient != client || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available {
			s.mu.Unlock()
			return
		}
		title, found := "", false
		for _, p := range f.listing.Projects {
			if p.ID == id {
				title, found = p.Title, true
				break
			}
		}
		if !found {
			s.mu.Unlock()
			return
		}
		s.closeProjectMaterialCollectionLocked()
		s.stopProjectProcessLocked()
		next := &projectMaterialCollectionState{snapshot: projectflow.MaterialCollection{ProjectID: id, Title: title, Cursors: []string{""}}}
		f.materialCollection = next
		s.mu.Unlock()
		s.runProjectMaterialCollection(win, next, "refresh")
	}
	m.OnMaterialCollectionAction = func(action string) { s.runProjectMaterialCollection(win, d, action) }
}
func (s *projectController) runProjectMaterialCollection(win *mygo.Window, d *projectMaterialCollectionState, action string) {
	s.mu.Lock()
	f := s.projectFlow
	if d == nil || f.materialCollection != d {
		s.mu.Unlock()
		return
	}
	if action == "close" {
		s.closeProjectMaterialCollectionLocked()
		s.mu.Unlock()
		return
	}
	client := s.dshClient
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	cursors := append([]string(nil), d.snapshot.Cursors...)
	switch action {
	case "refresh":
	case "next":
		if d.snapshot.Requests.NextCursor == "" {
			s.mu.Unlock()
			return
		}
		cursors = append(cursors, d.snapshot.Requests.NextCursor)
	case "previous":
		if len(cursors) < 2 {
			s.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	default:
		s.mu.Unlock()
		return
	}
	id, cursor := d.snapshot.ProjectID, cursors[len(cursors)-1]
	generation, navigation := f.generation, f.navigation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	d.cancel = cancel
	d.snapshot.Error = ""
	f.busy = true
	s.mu.Unlock()
	go func() {
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		page, err := rpc.MaterialRequestHistory(ctx, id, cursor)
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.materialCollection != d || f.client != client || s.dshClient != client {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				s.closeProjectMaterialCollectionLocked()
				return
			}
			d.cancel = nil
			f.busy = false
			if err != nil {
				d.snapshot.Error = err.Error()
				return
			}
			d.snapshot.Requests, d.snapshot.Cursors, d.snapshot.Loaded = page, cursors, true
		})
	}()
}
