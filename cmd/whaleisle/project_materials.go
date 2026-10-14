package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

func (s *mainState) bindProjectMaterials(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	f := &s.projectFlow
	m.Material = f.materials[f.materialKey]
	if m.Material != nil {
		for _, p := range f.listing.Projects {
			if p.ID == m.Material.ProjectID {
				m.MaterialMutable = p.Lifecycle == "ready" && !p.Archiving
			}
		}
	}
	s.mu.Unlock()
	m.OnMaterial = func(id, path string) { s.runProjectMaterial(win, id, path, "open") }
	m.OnSaveMaterial = func() {
		if m.Material != nil {
			s.runProjectMaterial(win, m.Material.ProjectID, m.Material.Path, "save")
		}
	}
	m.OnReloadMaterial = func() {
		if m.Material != nil {
			s.runProjectMaterial(win, m.Material.ProjectID, m.Material.Path, "reload")
		}
	}
	m.OnDiscardMaterial = func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if d := s.projectFlow.materials[s.projectFlow.materialKey]; d != nil && !s.projectFlow.busy {
			d.Draft = d.Saved
			d.Error = ""
			d.Notice = "Draft discarded; reload to fetch the latest saved text."
		}
	}
	m.OnCloseMaterial = func() { s.mu.Lock(); s.projectFlow.materialKey = ""; s.mu.Unlock() }
}
func (s *mainState) runProjectMaterial(win *mygo.Window, id, path, action string) {
	s.mu.Lock()
	f := &s.projectFlow
	client := s.dshClient
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	if path != "notes.md" && path != "preferences.md" {
		s.mu.Unlock()
		return
	}
	key := id + "\x00" + path
	if f.materials == nil {
		f.materials = map[string]*projectflow.MaterialDraft{}
	}
	d := f.materials[key]
	if d == nil {
		d = &projectflow.MaterialDraft{ProjectID: id, Path: path}
		f.materials[key] = d
	}
	f.materialKey = key
	if action == "open" && d.Loaded {
		s.mu.Unlock()
		return
	}
	if action == "reload" && d.Dirty() {
		d.Error = "Discard or save this draft before reloading."
		s.mu.Unlock()
		return
	}
	if action == "save" {
		allowed := false
		for _, p := range f.listing.Projects {
			if p.ID == id {
				allowed = p.Lifecycle == "ready" && !p.Archiving
			}
		}
		if !allowed || !d.Dirty() {
			s.mu.Unlock()
			return
		}
	}
	draft := d.Draft
	generation := f.generation
	f.busy = true
	d.Error = ""
	d.Notice = ""
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var material projectflow.Material
		var err error
		if action == "save" {
			material, err = rpc.WriteMaterial(ctx, id, path, draft)
		} else {
			material, err = rpc.ReadMaterial(ctx, id, path)
		}
		s.mu.Lock()
		f := &s.projectFlow
		if f.client != client || s.dshClient != client || f.generation != generation {
			s.mu.Unlock()
			return
		}
		f.busy = false
		d.Apply(material, err)
		if err == nil && action == "save" {
			d.Notice = "Saved to Project."
		}
		s.mu.Unlock()
		win.Update(func() {})
	}()
}
