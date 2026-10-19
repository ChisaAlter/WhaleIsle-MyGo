package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

// A task's work inspector is available before the separate Team page is opened.
// Read the existing Team projection as part of that one click, with the request
// and transport owner retained throughout. This never runs or retries work.
func (s *projectController) runProjectRequestWorkDetail(win *mygo.Window, projectID, requestID, workID string, owner projectQueueSubmission) {
	if win == nil || projectID == "" || requestID == "" || workID == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if !owner.matches(f, client) || f.busy || !f.visible() || !f.listing.Available || f.requestProjectID != projectID || f.requestCard == nil || f.requestCard.ID != requestID {
		s.mu.Unlock()
		return
	}
	revision := f.requestCard.Revision
	s.stopProjectProcessLocked()
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	f.detailGeneration++
	detailGeneration := f.detailGeneration
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		team, detail, err := readProjectRequestWork(ctx, rpc, projectID, requestID, revision, workID)
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.navigation != owner.Navigation || f.requestGeneration != owner.RequestGeneration || f.detailGeneration != detailGeneration || f.requestProjectID != projectID || f.requestCard == nil || f.requestCard.ID != requestID || f.requestCard.Revision != revision {
				return
			}
			if err != nil {
				f.err = err.Error()
				return
			}
			f.teamID = projectID
			f.team = &team
			f.detail = &detail
			f.process = nil
			f.assetDiff = nil
		})
	}()
}

func readProjectRequestWork(ctx context.Context, rpc projectflow.Client, projectID, requestID string, revision int, workID string) (projectflow.Overview, projectflow.WorkDetail, error) {
	var team projectflow.Overview
	var detail projectflow.WorkDetail
	card, err := rpc.Request(ctx, projectID, requestID)
	if err != nil {
		return team, detail, err
	}
	if card.Revision != revision {
		return team, detail, fmt.Errorf("Requirement changed; refresh task details first")
	}
	linked := false
	for _, work := range append(append([]projectflow.RequestWork{}, card.Work...), card.WorkPreview...) {
		if work.WorkstreamID == workID {
			linked = true
			break
		}
	}
	if !linked {
		return team, detail, fmt.Errorf("Work no longer belongs to the displayed requirement")
	}
	team, err = rpc.TeamOverviewPage(ctx, projectID, "")
	if err != nil {
		return team, detail, err
	}
	detail, err = rpc.WorkDetail(ctx, projectID, workID, "")
	if err == nil && detail.Workstream.Request.ID != requestID {
		err = fmt.Errorf("Work requirement identity does not match")
	}
	return team, detail, err
}
