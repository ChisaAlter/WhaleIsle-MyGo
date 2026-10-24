package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func (s *projectController) runProjectRequests(win *mygo.Window, id, action, arg string, queueChanges ...projectQueueSubmission) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if action == "queue" && (len(queueChanges) != 1 || !queueChanges[0].matches(f, client)) {
		s.mu.Unlock()
		return
	}
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailableRefreshToRetry)
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
		f.err = i18n.T(i18n.KeyCmdProjectIsNoLongerInTheCurrentListRef)
		s.mu.Unlock()
		return
	}
	if (action == "archive" || action == "restore" || action == "queue") && (project.Lifecycle != "ready" || project.Archiving) {
		f.err = i18n.T(i18n.KeyCmdRestoreTheProjectBeforeChangingRequest)
		s.mu.Unlock()
		return
	}
	if f.requestProjectID != id {
		s.closeProjectVisualResultsLocked()
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
	case "queue":
		if project.Queue == nil || len(queueChanges) != 1 || queueChanges[0].Change.RequestID != arg {
			s.mu.Unlock()
			return
		}
	default:
		s.mu.Unlock()
		return
	}
	if len(cursors) == 0 {
		cursors = []string{""}
	}
	f.busy = true
	f.err = ""
	f.requestRevertError = ""
	f.generation++
	generation := f.generation
	f.requestGeneration++
	requestGeneration := f.requestGeneration
	selectedCard := f.requestCard
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var page projectflow.RequestPage
		var card projectflow.RequestCard
		var err error
		mutated := false
		var queueResult *projectflow.RequestQueueResult
		if action == "get" {
			card, err = rpc.Request(ctx, id, arg)
		} else {
			if action == "queue" {
				result, queueErr := rpc.ChangeRequestQueue(ctx, id, queueChanges[0].Change)
				err = queueErr
				if err == nil {
					queueResult = &result
					mutated = true
				}
			}
			if action == "archive" || action == "restore" {
				err = rpc.SetRequestArchived(ctx, id, arg, action == "archive")
				mutated = err == nil
			}
			if err == nil {
				page, err = rpc.Requests(ctx, id, kind, cursors[len(cursors)-1])
				if err == nil && action == "queue" && selectedCard != nil {
					card, err = rpc.Request(ctx, id, selectedCard.ID)
				}
			}
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.requestProjectID != id || f.requestGeneration != requestGeneration {
				return
			}
			if queueResult != nil {
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == id {
						f.listing.Projects[i].Queue = &queueResult.ProjectQueue
					}
				}
			}
			if err != nil {
				f.err = err.Error()
				if mutated {
					f.requestCard = nil
					f.requestPage = projectflow.RequestPage{}
					f.requestLoaded = false
					f.err = fmt.Sprintf("%s: %s", i18n.T(i18n.KeyCmdCardUpdatedButListRefreshFailedRefresh), err)
				}
				return
			}
			f.assetDiff = nil
			f.assetNote = ""
			if action == "get" {
				f.requestCard = &card
				s.rememberProjectRequestCardLocked(f.requestCard)
				return
			}
			f.requestCard = nil
			if action == "queue" && selectedCard != nil {
				f.requestCard = &card
				s.rememberProjectRequestCardLocked(f.requestCard)
			}
			f.requestKind = kind
			f.requestPage = page
			f.requestCursors = cursors
			f.requestLoaded = true
		})
	}()
}
