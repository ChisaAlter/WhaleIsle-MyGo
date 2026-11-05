package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

// The four inventories have independent replacement-page cursors. Their owner
// is the same retained requirements view, not the last card opened in a modal.
func (s *projectController) bindRequirementCatalog(win *mygo.Window, m *projects.Model, o *projects.ObserverState) {
	o.OnLoadCatalog = func(kind string, delta int) { s.loadRequirementCatalog(win, o, kind, delta) }
	queue := m.OnRequestQueue
	m.OnRequestQueue = func(id, action string) {
		if queue != nil {
			o.RefreshCatalog()
			queue(id, action)
		}
	}
	archive := m.OnRequestArchive
	m.OnRequestArchive = func(id string, on bool) {
		if archive != nil {
			o.RefreshCatalog()
			archive(id, on)
		}
	}
	original := m.RequestCardActions
	if original != nil {
		m.RequestCardActions = func(id string, revision int) projects.RequestCardActions {
			actions := original(id, revision)
			dispatch := actions.WorkAction
			if dispatch != nil {
				actions.WorkAction = func(action string, work projectflow.RequestWork) {
					if action == "stop" {
						o.RefreshCatalog()
					}
					dispatch(action, work)
				}
			}
			return actions
		}
	}
}

func (s *projectController) loadRequirementCatalog(win *mygo.Window, o *projects.ObserverState, kind string, delta int) {
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if win == nil || o == nil || client == nil || f.client != client || f.busy || !f.listing.Available || o.Host == "" {
		s.mu.Unlock()
		return
	}
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == o.ProjectID && p.CoordinatorSessionID == o.CoordinatorSessionID {
			known = true
		}
	}
	if !known {
		s.mu.Unlock()
		return
	}
	cursors := append([]string(nil), o.Cursors[kind]...)
	if len(cursors) == 0 {
		cursors = []string{""}
	}
	if delta > 0 {
		page := o.Pages[kind]
		if page.NextCursor == nil {
			s.mu.Unlock()
			return
		}
		cursors = append(cursors, *page.NextCursor)
	}
	if delta < 0 {
		if len(cursors) < 2 {
			s.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	}
	if delta == 0 {
		cursors = []string{""}
	}
	f.requestProjectID = o.ProjectID
	f.busy = true
	f.err = ""
	generation, navigation := f.generation, f.navigation
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		page, err := rpc.Requests(ctx, o.ProjectID, kind, cursors[len(cursors)-1])
		var overview projectflow.Overview
		if err == nil && kind == "current" {
			overview, err = rpc.TeamOverviewPage(ctx, o.ProjectID, "")
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.client != client || s.dshClient != client || f.generation != generation || f.navigation != navigation || f.requestProjectID != o.ProjectID {
				return
			}
			f.busy = false
			if err != nil {
				o.CatalogErrors[kind] = err.Error()
				f.err = err.Error()
				return
			}
			o.Pages[kind], o.Cursors[kind], o.CatalogLoaded[kind] = page, cursors, true
			delete(o.CatalogErrors, kind)
			f.requestKind, f.requestPage, f.requestCursors, f.requestLoaded = kind, page, cursors, true
			if kind == "current" {
				f.team = &overview
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == o.ProjectID {
						f.listing.Projects[i] = overview.Project
					}
				}
			}
		})
	}()
}
