package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

func (s *mainState) runProjectTeam(win *mygo.Window, id, action, arg string) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := &s.projectFlow
	client := s.dshClient
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = resumeText("Project Host 不可用，请刷新。", "Project Host is unavailable; refresh to retry.")
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
		f.err = resumeText("此 Project 已不在当前列表，请刷新。", "This Project is not in the current list; refresh to retry.")
		s.mu.Unlock()
		return
	}
	if f.teamID != id {
		f.team = nil
		f.teamNote = ""
	}
	f.teamID = id
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	prior := f.team
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var out projectflow.Overview
		var err error
		var note string
		switch action {
		case "inspect":
			out, err = rpc.TeamOverview(ctx, id)
		case "next":
			if prior == nil {
				err = fmt.Errorf("refresh Team before paging")
				break
			}
			var page projectflow.WorkPage
			page, err = rpc.CurrentWork(ctx, id, arg)
			if err == nil {
				out = *prior
				out.Workstreams = page.Workstreams
				out.Workers = page.Workers
				out.WorkPage.CurrentNextCursor = page.NextCursor
				out.WorkPage.CurrentTotal = page.Total
			}
		case "stop", "pause", "resume":
			if prior == nil || prior.Team == nil || prior.Team.Failure != "" || prior.Project.Lifecycle != "ready" {
				err = fmt.Errorf("refresh a ready Project Team before controlling work")
				break
			}
			note, err = rpc.Control(ctx, id, action, arg)
			if err == nil {
				out, err = rpc.TeamOverview(ctx, id)
			}
		default:
			err = fmt.Errorf("unsupported Project Team action")
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := &s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.teamID != id {
				return
			}
			if err != nil {
				f.err = err.Error()
				f.team = nil
				return
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
