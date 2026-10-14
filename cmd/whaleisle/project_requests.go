package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

func (s *mainState) runProjectRequests(win *mygo.Window, id, action, arg string) {
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
	var project projectflow.Project
	for _, p := range f.listing.Projects {
		if p.ID == id {
			project = p
			break
		}
	}
	if project.ID == "" {
		f.err = resumeText("Project 已不在当前列表，请刷新。", "Project is no longer in the current list; refresh to retry.")
		s.mu.Unlock()
		return
	}
	if (action == "archive" || action == "restore") && (project.Lifecycle != "ready" || project.Archiving) {
		f.err = resumeText("恢复 Project 后才能修改要求卡片。", "Restore the Project before changing request cards.")
		s.mu.Unlock()
		return
	}
	if f.requestProjectID != id {
		f.requestPage = projectflow.RequestPage{}
		f.requestCard = nil
		f.requestLoaded = false
		f.requestCursors = nil
		f.requestKind = "current"
	}
	f.requestProjectID = id
	kind := f.requestKind
	cursors := append([]string(nil), f.requestCursors...)
	switch action {
	case "list":
		kind = arg
		cursors = []string{""}
	case "next":
		if f.requestPage.NextCursor == nil {
			s.mu.Unlock()
			return
		}
		cursors = append(cursors, *f.requestPage.NextCursor)
	case "previous":
		if len(cursors) <= 1 {
			s.mu.Unlock()
			return
		}
		cursors = cursors[:len(cursors)-1]
	case "archive", "restore":
		cursors = []string{""}
	case "get":
	default:
		s.mu.Unlock()
		return
	}
	if len(cursors) == 0 {
		cursors = []string{""}
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	f.requestGeneration++
	requestGeneration := f.requestGeneration
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var page projectflow.RequestPage
		var card projectflow.RequestCard
		var err error
		mutated := false
		if action == "get" {
			card, err = rpc.Request(ctx, id, arg)
		} else {
			if action == "archive" || action == "restore" {
				err = rpc.SetRequestArchived(ctx, id, arg, action == "archive")
				mutated = err == nil
			}
			if err == nil {
				page, err = rpc.Requests(ctx, id, kind, cursors[len(cursors)-1])
			}
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := &s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.requestProjectID != id || f.requestGeneration != requestGeneration {
				return
			}
			if err != nil {
				f.err = err.Error()
				if mutated {
					f.requestCard = nil
					f.requestPage = projectflow.RequestPage{}
					f.requestLoaded = false
					f.err = fmt.Sprintf("%s: %s", resumeText("卡片已更新，但列表刷新失败；请刷新，勿重复操作", "Card updated, but list refresh failed; refresh without repeating the mutation"), err)
				}
				return
			}
			if action == "get" {
				f.requestCard = &card
				return
			}
			f.requestCard = nil
			f.requestKind = kind
			f.requestPage = page
			f.requestCursors = cursors
			f.requestLoaded = true
		})
	}()
}
