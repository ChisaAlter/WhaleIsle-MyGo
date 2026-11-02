package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

func (s *projectController) bindProjectHistory(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	m.Reports = s.projectFlow.reports
	s.mu.Unlock()
	m.OnReports = func(id string) { s.runProjectHistory(win, id, "list", "", nil) }
	m.OnReportAction = func(action, arg string) {
		if m.Reports != nil {
			s.runProjectHistory(win, m.Reports.ProjectID, action, arg, nil)
		}
	}
	m.OnReportFile = func(path string, open func(string)) {
		if m.Reports != nil {
			s.runProjectHistory(win, m.Reports.ProjectID, "file", path, open)
		}
	}
	m.OnCloseReports = func() {
		s.mu.Lock()
		if !s.projectFlow.busy {
			s.projectFlow.reports = nil
		}
		s.mu.Unlock()
	}
	m.OnReportWork = func() {
		if m.Reports != nil {
			s.runProjectHistory(win, m.Reports.ProjectID, "inspect", "", nil)
		}
	}
}
func (s *projectController) runProjectHistory(win *mygo.Window, id, action, arg string, open func(string)) {
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	original := f.reports
	var next projectflow.ReportBrowser
	if action == "list" {
		next = projectflow.ReportBrowser{ProjectID: id, ListCursors: []string{""}}
	} else {
		if original == nil || original.ProjectID != id {
			s.mu.Unlock()
			return
		}
		next = *original
		next.ListCursors = append([]string(nil), original.ListCursors...)
		next.WorkCursors = append([]string(nil), original.WorkCursors...)
	}
	var file projectflow.Artifact
	switch action {
	case "back-detail":
		next.Selected = nil
		next.Work = nil
		f.reports = &next
		s.mu.Unlock()
		return
	case "back-work":
		next.WorkID = ""
		next.Selected = nil
		next.Work = nil
		f.reports = &next
		s.mu.Unlock()
		return
	case "work":
		found := false
		for _, e := range next.Listing.Results {
			if e.WorkstreamID == arg {
				next.WorkID = arg
				next.Title = e.Title
				found = true
				break
			}
		}
		if !found {
			s.mu.Unlock()
			return
		}
		next.WorkCursors = []string{""}
		next.Selected = nil
		next.Work = nil
	case "list-next":
		if next.Listing.NextCursor == "" {
			s.mu.Unlock()
			return
		}
		next.ListCursors = append(next.ListCursors, next.Listing.NextCursor)
	case "list-prev":
		if len(next.ListCursors) < 2 {
			s.mu.Unlock()
			return
		}
		next.ListCursors = next.ListCursors[:len(next.ListCursors)-1]
	case "work-next":
		if next.History.NextCursor == "" {
			s.mu.Unlock()
			return
		}
		next.WorkCursors = append(next.WorkCursors, next.History.NextCursor)
	case "work-prev":
		if len(next.WorkCursors) < 2 {
			s.mu.Unlock()
			return
		}
		next.WorkCursors = next.WorkCursors[:len(next.WorkCursors)-1]
	case "detail":
		found := false
		for _, r := range next.History.Items {
			if r.DelegationRef == arg {
				found = true
				break
			}
		}
		if !found {
			s.mu.Unlock()
			return
		}
	case "file":
		if next.Selected == nil || open == nil {
			s.mu.Unlock()
			return
		}
		var err error
		file, err = next.Selected.File(next.WorkID, arg)
		if err != nil {
			s.mu.Unlock()
			return
		}
	}
	f.busy = true
	f.err = ""
	next.Error = ""
	next.Notice = ""
	generation := f.generation
	navigation := f.navigation
	sessions := s.sessions
	s.mu.Unlock()
	selected := ""
	if sessions != nil {
		selected = sessions.SelectedSessionID()
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var err error
		var resolved projectflow.ResolvedArtifact
		var overview projectflow.Overview
		switch action {
		case "list", "list-next", "list-prev":
			next.Listing, err = rpc.Reports(ctx, id, next.ListCursors[len(next.ListCursors)-1])
		case "work", "work-next", "work-prev":
			next.History, err = rpc.ReportHistory(ctx, id, next.WorkID, next.WorkCursors[len(next.WorkCursors)-1], "")
		case "detail":
			var page projectflow.ReportHistory
			page, err = rpc.ReportHistory(ctx, id, next.WorkID, "", arg)
			if err == nil {
				r := page.Items[0]
				if len(r.Request) == 0 {
					r.Request = page.Request
				}
				next.Selected = &r
				var work projectflow.WorkDetail
				work, err = rpc.WorkDetail(ctx, id, next.WorkID, "")
				if err == nil {
					next.Work = &work
				}
			}
		case "inspect":
			if next.WorkID == "" || next.Selected == nil {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectSelectReport))
				break
			}
			overview, err = rpc.TeamOverview(ctx, id)
			if err == nil {
				var work projectflow.WorkDetail
				work, err = rpc.WorkDetail(ctx, id, next.WorkID, "")
				next.Work = &work
			}
		case "file":
			resolved, err = rpc.ResolveArtifact(ctx, id, file)
		default:
			err = fmt.Errorf("unsupported report action")
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if f.client != client || s.dshClient != client || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if f.reports != original {
				s.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			if action == "inspect" {
				f.reports = nil
				f.teamID = id
				f.team = &overview
				f.detail = next.Work
				f.process = nil
				f.detailGeneration++
				s.mu.Unlock()
				return
			}
			if action == "file" {
				if !f.visible() || f.navigation != navigation || sessions == nil || sessions.SelectedSessionID() != selected {
					f.err = i18n.T(i18n.KeyCmdNavigationChangedTheFileWasNotOpened)
					s.mu.Unlock()
					return
				}
				next.Notice = i18n.T(i18n.KeyCmdProjectCurrentFileNotice)
				if resolved.ModifiedSinceReport {
					next.Notice = i18n.T(i18n.KeyCmdProjectFileChangedNotice)
				}
				f.reports = &next
				s.mu.Unlock()
				open(resolved.CanonicalPath)
				return
			}
			f.reports = &next
			s.mu.Unlock()
		})
	}()
}
