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

type projectDeliveryState struct {
	browser *projectflow.DeliveryBrowser
	cancel  context.CancelFunc
}

func (s *projectController) closeProjectDeliveriesLocked() {
	f := s.projectFlow
	if d := f.deliveries; d != nil {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		d.browser.Close()
		f.deliveries = nil
	}
}
func (s *projectController) bindProjectDeliveries(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	f := s.projectFlow
	d := f.deliveries
	expected, generation, navigation := f.client, f.generation, f.navigation
	if d != nil {
		snap := d.browser.Snapshot()
		m.Deliveries = &snap
	}
	s.mu.Unlock()
	m.OnDeliveries = func(id string) { s.openProjectDeliveries(win, id, expected, generation, navigation) }
	m.OnDeliveryAction = func(action, arg string) { s.runProjectDeliveries(win, d, action, arg, projectflow.DeliveryReportRef{}) }
	m.OnDeliveryReport = func(id string, ref projectflow.DeliveryReportRef) { s.runProjectDeliveries(win, d, "report", id, ref) }
}
func (s *projectController) openProjectDeliveries(win *mygo.Window, id string, expected *dsh.Client, generation, navigation uint64) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if win == nil || client == nil || client != expected || f.client != expected || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available {
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
	s.closeProjectDeliveriesLocked()
	s.stopProjectProcessLocked()
	d := &projectDeliveryState{browser: projectflow.NewDeliveryBrowser(projectflow.DeliveryScope{Host: fmt.Sprintf("%p", client), ProjectID: id, Generation: generation}, title)}
	f.deliveries = d
	s.mu.Unlock()
	s.runProjectDeliveries(win, d, "refresh", "", projectflow.DeliveryReportRef{})
}
func (s *projectController) runProjectDeliveries(win *mygo.Window, d *projectDeliveryState, action, arg string, ref projectflow.DeliveryReportRef) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if d == nil || f.deliveries != d {
		s.mu.Unlock()
		return
	}
	if action == "close" {
		s.closeProjectDeliveriesLocked()
		s.mu.Unlock()
		return
	}
	if action == "back-list" || action == "back-report" {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
			f.busy = false
		}
		if action == "back-list" {
			d.browser.BackList()
		} else {
			d.browser.BackReport()
		}
		s.mu.Unlock()
		return
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.visible() || !f.listing.Available || d.browser.Snapshot().Scope.Host != fmt.Sprintf("%p", client) {
		s.mu.Unlock()
		return
	}
	if action == "select" {
		_ = d.browser.Select(arg)
		s.mu.Unlock()
		return
	}
	var req projectflow.DeliveryRequest
	var err error
	if action == "report" {
		req, err = d.browser.BeginReport(arg, ref)
	} else {
		req, err = d.browser.BeginPage(action)
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
		var page projectflow.DeliveryHistoryPage
		var report projectflow.Report
		var loadErr error
		if action == "report" {
			report, loadErr = rpc.DeliveryReport(ctx, req.ProjectID(), req.Delivery(), req.Reference())
		} else {
			page, loadErr = rpc.DeliveryHistory(ctx, req.ProjectID(), req.Cursor())
		}
		win.Update(func() {
			s.dshMu.Lock()
			current := s.dshClient
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.deliveries != d || f.client != client || current != client || !d.browser.Pending(req) {
				return
			}
			if f.generation != generation || f.navigation != navigation || !f.visible() {
				s.closeProjectDeliveriesLocked()
				return
			}
			if action == "report" {
				d.browser.CompleteReport(req, report, loadErr)
			} else {
				d.browser.CompletePage(req, page, loadErr)
			}
			d.cancel = nil
			f.busy = false
		})
	}()
}
