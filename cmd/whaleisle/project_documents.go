package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

type projectDocumentState struct {
	browser *projectflow.DocumentBrowser
	cancel  context.CancelFunc
}

func (s *projectController) closeProjectDocumentsLocked() {
	f := s.projectFlow
	if d := f.documents; d != nil {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Close()
		f.documents = nil
	}
}
func (s *projectController) bindProjectDocuments(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	d := s.projectFlow.documents
	expectedClient, expectedGeneration, expectedNavigation := s.projectFlow.client, s.projectFlow.generation, s.projectFlow.navigation
	if d != nil {
		snapshot := d.browser.Snapshot()
		m.Documents = &snapshot
	}
	s.mu.Unlock()
	m.OnDocuments = func(id string) {
		s.openProjectDocuments(win, id, expectedClient, expectedGeneration, expectedNavigation)
	}
	m.OnDocumentAction = func(action, path string) { s.runProjectDocuments(win, d, action, path) }
}
func (s *projectController) openProjectDocuments(win *mygo.Window, id string, expectedClient *dsh.Client, expectedGeneration, expectedNavigation uint64) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if win == nil || client == nil || client != expectedClient || f.client != expectedClient || f.generation != expectedGeneration || f.navigation != expectedNavigation || f.busy || !f.visible() || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	title := ""
	found := false
	for _, p := range f.listing.Projects {
		if p.ID == id {
			title = p.Title
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return
	}
	s.closeProjectDocumentsLocked()
	s.stopProjectProcessLocked()
	scope := projectflow.DocumentScope{Host: fmt.Sprintf("%p", client), ProjectID: id, Generation: f.generation}
	d := &projectDocumentState{browser: projectflow.NewDocumentBrowser(scope, title)}
	f.documents = d
	s.mu.Unlock()
	s.runProjectDocuments(win, d, "list", "")
}
func (s *projectController) runProjectDocuments(win *mygo.Window, d *projectDocumentState, action, documentPath string) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if d == nil || f.documents != d {
		s.mu.Unlock()
		return
	}
	if action == "close" {
		s.closeProjectDocumentsLocked()
		s.mu.Unlock()
		return
	}
	if action == "back" {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Back()
		s.mu.Unlock()
		return
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	snapshot := d.browser.Snapshot()
	if snapshot.Scope.Host != fmt.Sprintf("%p", client) {
		s.mu.Unlock()
		return
	}
	var request projectflow.DocumentRequest
	var err error
	switch action {
	case "list":
		request, err = d.browser.BeginList()
	case "read":
		request, err = d.browser.BeginRead(documentPath)
	default:
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	d.cancel = cancel
	f.busy = true
	generation, navigation := f.generation, f.navigation
	s.mu.Unlock()
	go func() {
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var docs []projectflow.DocumentEntry
		var content projectflow.DocumentContent
		var loadErr error
		if action == "list" {
			docs, loadErr = rpc.Documents(ctx, request.ProjectID())
		} else {
			content, loadErr = rpc.ReadDocument(ctx, request.ProjectID(), request.Entry())
		}
		win.Update(func() {
			s.dshMu.Lock()
			currentClient := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.documents != d || f.client != client || currentClient != client || !d.browser.Pending(request) {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				s.closeProjectDocumentsLocked()
				return
			}
			if action == "list" {
				d.browser.CompleteList(request, docs, loadErr)
			} else {
				d.browser.CompleteRead(request, content, loadErr)
			}
			d.cancel = nil
			f.busy = false
		})
	}()
}
