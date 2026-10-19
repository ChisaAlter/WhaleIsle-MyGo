package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

// A card's read-only changes route is available from the authoritative list;
// it need not first become the controller's currently opened detail card.
func (s *projectController) runProjectRequestChanges(win *mygo.Window, projectID, requestID string, owner projectQueueSubmission) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f, client := s.projectFlow, s.dshClient
	if !owner.matches(f, client) || f.busy || !f.visible() || !f.listing.Available || f.requestProjectID != projectID {
		s.mu.Unlock()
		return
	}
	known := f.requestCard != nil && f.requestCard.ID == requestID && f.requestCard.ProjectID == projectID
	for _, card := range f.requestPage.Items {
		known = known || card.ID == requestID && card.ProjectID == projectID
	}
	for _, project := range f.listing.Projects {
		if project.ID == projectID && project.Queue != nil && project.Queue.Active != nil {
			known = known || project.Queue.Active.ID == requestID
		}
	}
	if !known {
		s.mu.Unlock()
		return
	}
	f.busy, f.err, f.requestRevertError = true, "", ""
	f.generation++
	f.requestGeneration++
	generation, navigation, requestGeneration := f.generation, f.navigation, f.requestGeneration
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		card, err := rpc.Request(ctx, projectID, requestID)
		var changes projectflow.RequestChanges
		if err == nil {
			changes, err = rpc.RequestChanges(ctx, projectID, requestID)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.navigation != navigation || f.requestGeneration != requestGeneration || f.requestProjectID != projectID {
				return
			}
			if err != nil {
				f.err = err.Error()
				return
			}
			f.requestCard = &card
			f.assetDiff = &projectflow.WorkDiff{Requirement: &changes}
			f.assetNote = ""
		})
	}()
	win.Update(func() {})
}

// The confirmation retains the displayed revision; conflict checks belong to
// the Host, which rejects a revert that would overwrite newer edits.
func (s *projectController) revertProjectRequest(win *mygo.Window, projectID, requestID string, revision int, owner projectQueueSubmission) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f, client := s.projectFlow, s.dshClient
	card := f.requestCard
	if !owner.matches(f, client) || f.busy || !f.visible() || !f.listing.Available || f.requestProjectID != projectID || card == nil || card.ID != requestID || card.Revision != revision || card.Changes == nil || !card.Changes.Revertible || card.Changes.RevertedAt != nil {
		s.mu.Unlock()
		return
	}
	allowed := false
	for _, project := range f.listing.Projects {
		if project.ID == projectID {
			allowed = project.Lifecycle == "ready" && !project.Archiving
		}
	}
	if !allowed {
		s.mu.Unlock()
		return
	}
	f.busy, f.requestRevertError = true, ""
	f.generation++
	generation, navigation, requestGeneration := f.generation, f.navigation, f.requestGeneration
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		err := rpc.RevertRequest(ctx, projectID, requestID, revision)
		var fresh projectflow.RequestCard
		committed := err == nil
		if committed {
			fresh, err = rpc.Request(ctx, projectID, requestID)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.navigation != navigation || f.requestGeneration != requestGeneration || f.requestProjectID != projectID || f.requestCard == nil || f.requestCard.ID != requestID || f.requestCard.Revision != revision {
				return
			}
			if err != nil {
				f.requestRevertError = err.Error()
				if committed {
					f.requestCard = nil
					f.err = resumeText("改动已撤销，但读取结果失败；请刷新，不要重复撤销。", "Changes were reverted, but refreshing failed; refresh without repeating the revert.") + " " + err.Error()
				}
				return
			}
			f.requestCard = &fresh
			for i := range f.requestPage.Items {
				if f.requestPage.Items[i].ID == requestID {
					f.requestPage.Items[i] = fresh
				}
			}
			f.assetDiff = nil
			f.assetNote = resumeText("已撤销此要求的改动。", "This requirement’s changes were reverted.")
		})
	}()
	win.Update(func() {})
}
