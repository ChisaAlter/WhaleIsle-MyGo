package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

type projectActivityState struct {
	browser *projectflow.ActivityBrowser
	cancel  context.CancelFunc
	owner   *projectRequestRevertState
	request projectRequestRevision
}

func (s *projectController) closeProjectActivityLocked() {
	f := s.projectFlow
	if a := f.activity; a != nil {
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
			f.busy = false
		}
		a.browser.Close()
		f.activity = nil
	}
}
func (s *projectController) bindProjectActivity(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	f := s.projectFlow
	a := f.activity
	if a != nil {
		if _, known := s.requestRevertCardLocked(a.owner, a.request); !known {
			s.closeProjectActivityLocked()
			a = nil
		}
	}
	var view uint64
	if a != nil {
		snapshot := a.browser.Snapshot()
		m.Activity = &snapshot
		view = snapshot.View
	}
	state, card := s.requestRevertStateLocked(), f.requestCard
	s.mu.Unlock()
	m.OnRequestActivity = func() {
		if card != nil {
			s.openProjectRequestActivity(win, state, projectRequestRevision{card.ID, card.Revision})
		}
	}
	m.OnActivityAction = func(action, arg string) {
		s.runProjectActivity(win, a, view, action, arg, projectflow.DeliveryReportRef{})
	}
	m.OnActivityReport = func(key string, ref projectflow.DeliveryReportRef) {
		s.runProjectActivity(win, a, view, "report", key, ref)
	}
}
func (s *projectController) openProjectRequestActivity(win *mygo.Window, state *projectRequestRevertState, key projectRequestRevision) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	card, known := s.requestRevertCardLocked(state, key)
	if win == nil || !known || client != state.client || f.busy {
		s.mu.Unlock()
		return
	}
	s.closeProjectActivityLocked()
	s.stopProjectProcessLocked()
	a := &projectActivityState{browser: projectflow.NewActivityBrowser(projectflow.ActivityScope{Host: fmt.Sprintf("%p", client), ProjectID: card.ProjectID, RequestID: card.ID, Generation: f.generation}), owner: state, request: key}
	f.activity = a
	s.mu.Unlock()
	s.runProjectActivity(win, a, 0, "refresh", "", projectflow.DeliveryReportRef{})
}
func (s *projectController) runProjectActivity(win *mygo.Window, a *projectActivityState, view uint64, action, arg string, ref projectflow.DeliveryReportRef) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if a == nil || f.activity != a || a.browser.View() != view {
		s.mu.Unlock()
		return
	}
	if action == "close" {
		s.closeProjectActivityLocked()
		s.mu.Unlock()
		return
	}
	if action == "back-list" || action == "back-report" {
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
			f.busy = false
		}
		if action == "back-list" {
			a.browser.BackList()
		} else {
			a.browser.BackReport()
		}
		s.mu.Unlock()
		return
	}
	snapshot := a.browser.Snapshot()
	_, known := s.requestRevertCardLocked(a.owner, a.request)
	if win == nil || !known || snapshot.Scope.Host != fmt.Sprintf("%p", client) || f.busy {
		s.mu.Unlock()
		return
	}
	switch action {
	case "filter":
		_ = a.browser.Filter(arg)
		s.mu.Unlock()
		return
	case "next":
		_ = a.browser.MovePage(1)
		s.mu.Unlock()
		return
	case "previous":
		_ = a.browser.MovePage(-1)
		s.mu.Unlock()
		return
	case "select":
		_ = a.browser.Select(arg)
		s.mu.Unlock()
		return
	}
	var req projectflow.ActivityRequest
	var err error
	if action == "refresh" {
		req, err = a.browser.BeginRefresh()
	} else if action == "report" {
		req, err = a.browser.BeginReport(arg, ref)
	} else {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.cancel = cancel
	f.busy = true
	navigation := f.navigation
	s.mu.Unlock()
	go func() {
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var card projectflow.RequestActivityCard
		var report projectflow.Report
		var loadErr error
		if action == "refresh" {
			card, loadErr = rpc.RequestActivity(ctx, snapshot.Scope.ProjectID, snapshot.Scope.RequestID)
		} else {
			report, loadErr = rpc.ActivityReport(ctx, req.Event(), req.Reference())
		}
		win.Update(func() {
			s.dshMu.Lock()
			current := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.activity != a || f.client != client || current != client || !a.browser.Pending(req) {
				return
			}
			if _, known := s.requestRevertCardLocked(a.owner, a.request); !known || f.navigation != navigation {
				s.closeProjectActivityLocked()
				return
			}
			if action == "refresh" {
				a.browser.CompleteRefresh(req, card, loadErr)
			} else {
				a.browser.CompleteReport(req, report, loadErr)
			}
			a.cancel = nil
			f.busy = false
		})
	}()
}
