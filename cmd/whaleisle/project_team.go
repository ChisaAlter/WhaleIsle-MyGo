package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func (s *projectController) runProjectTeam(win *mygo.Window, id, action, arg string) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailableRefreshToRetry)
		s.mu.Unlock()
		return
	}
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == id {
			known = true
		}
	}
	if !known {
		f.err = i18n.T(i18n.KeyCmdThisProjectIsNotInTheCurrentListRefr)
		s.mu.Unlock()
		return
	}
	s.stopProjectProcessLocked()
	if f.teamID != id {
		f.teamQuery = ""
		f.team = nil
		f.teamNote = ""
	}
	f.detail = nil
	f.process = nil
	f.detailGeneration++
	f.teamID = id
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	prior := f.team
	cursors := []string{""}
	if prior != nil && len(prior.CurrentCursors) > 0 {
		cursors = append([]string(nil), prior.CurrentCursors...)
	}
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var out projectflow.Overview
		var err error
		var note string
		switch action {
		case "inspect", "next", "previous":
			if action == "next" {
				if prior == nil || prior.WorkPage.CurrentNextCursor == nil || arg == "" || arg != *prior.WorkPage.CurrentNextCursor {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectTeamRefreshPaging))
					break
				}
				cursors = append(cursors, arg)
			}
			if action == "previous" {
				if prior == nil || len(cursors) < 2 {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectWorkNoPreviousPage))
					break
				}
				cursors = cursors[:len(cursors)-1]
			}
			out, err = rpc.TeamOverviewPage(ctx, id, cursors[len(cursors)-1])
		case "stop", "pause", "resume":
			if prior == nil || prior.Team == nil || prior.Team.Failure != "" || prior.Project.Lifecycle != "ready" {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectTeamRefreshControl))
				break
			}
			note, err = rpc.Control(ctx, id, action, arg)
			if err == nil {
				out, err = rpc.TeamOverviewPage(ctx, id, cursors[len(cursors)-1])
			}
		default:
			err = fmt.Errorf("unsupported Project Team action")
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.teamID != id {
				return
			}
			if err != nil {
				f.err = err.Error()
				// Keep the last readable page and its navigation on a failed refresh.
				return
			}
			if out.WorkPage.CurrentCursor != nil && *out.WorkPage.CurrentCursor == cursors[len(cursors)-1] {
				out.CurrentCursors = cursors
			}
			f.team = &out
			f.teamNote = note
			for i, p := range f.listing.Projects {
				if p.ID == out.Project.ID {
					f.listing.Projects[i] = out.Project
				}
			}
		})
	}()
}
