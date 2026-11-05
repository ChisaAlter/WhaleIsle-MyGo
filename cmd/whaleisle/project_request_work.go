package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

func exactRequestWork(card projectflow.RequestCard, selected projectflow.RequestWork) (projectflow.RequestWork, bool) {
	for _, work := range card.Work {
		if work.WorkstreamID == selected.WorkstreamID && work.DelegationRef == selected.DelegationRef && work.WorkerSessionID == selected.WorkerSessionID {
			return work, true
		}
	}
	return projectflow.RequestWork{}, false
}

// The click carries its exact assignment, not just a mutable workstream ID.
func (s *projectController) runProjectRequestWorkAction(win *mygo.Window, state *projectRequestRevertState, key projectRequestRevision, selected projectflow.RequestWork, action string) {
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	card, known := s.requestRevertCardLocked(state, key)
	_, linked := exactRequestWork(card, selected)
	if win == nil || !known || !linked || f.busy || client == nil || f.client != client || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	generation, navigation := f.generation, f.navigation
	s.mu.Unlock()
	s.dshMu.Lock()
	mux := s.dshMux
	s.dshMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		fresh, err := rpc.Request(ctx, card.ProjectID, card.ID)
		var work projectflow.RequestWork
		if err == nil {
			var ok bool
			work, ok = exactRequestWork(fresh, selected)
			if fresh.Revision != key.revision || !ok {
				err = fmt.Errorf("The requirement assignment changed. Refresh before continuing.")
			}
		}
		var detail projectflow.WorkDetail
		var report projectflow.Report
		var binding projectflow.HistoricalBinding
		var window projectflow.HistoricalWindow
		var overview projectflow.Overview
		if err == nil {
			switch action {
			case "process":
				target := projectflow.HistoricalTarget{Assignment: true, ProjectID: card.ProjectID, WorkstreamID: work.WorkstreamID, DelegationRef: work.DelegationRef, RequestID: card.ID, WorkerSessionID: work.WorkerSessionID}
				binding, err = rpc.HistoricalAssignment(ctx, target)
				if err == nil {
					window, err = rpc.ReadHistoricalProcess(ctx, mux, binding)
				}
			case "report":
				var history projectflow.ReportHistory
				history, err = rpc.ReportHistory(ctx, card.ProjectID, work.WorkstreamID, "", work.DelegationRef)
				if err == nil {
					report = history.Items[0]
				}
			case "stop", "summary":
				if work.Current != nil && !*work.Current {
					err = fmt.Errorf("Historical assignments are read-only.")
					break
				}
				detail, err = rpc.WorkDetail(ctx, card.ProjectID, work.WorkstreamID, "")
				if err == nil && (detail.Workstream.CurrentDelegationRef != work.DelegationRef || detail.Workstream.Request.ID != card.ID) {
					err = fmt.Errorf("The current assignment changed. Refresh before continuing.")
				}
				if err == nil && action == "stop" {
					_, err = rpc.Control(ctx, card.ProjectID, "stop", work.WorkstreamID)
					if err == nil {
						overview, err = rpc.TeamOverviewPage(ctx, card.ProjectID, "")
					}
				}
			default:
				err = fmt.Errorf("Unknown requirement work action.")
			}
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if f.client != client || s.dshClient != client || f.generation != generation || f.navigation != navigation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			sourceValid := func(f *projectFlowState) bool {
				cached, ok := s.requestRevertCardLocked(state, key)
				if !ok {
					return false
				}
				_, ok = exactRequestWork(cached, selected)
				return ok
			}
			if !sourceValid(f) {
				s.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			switch action {
			case "process":
				s.closeProjectHistoricalLocked()
				browser := projectflow.NewHistoricalBrowser(fmt.Sprintf("%p", client), generation, binding.Target)
				req, _ := browser.Begin("record")
				browser.CompleteRecord(req, binding, nil)
				req, _ = browser.Begin("process")
				browser.CompleteProcess(req, window, nil)
				f.historical = &projectHistoricalState{browser: browser, sourceValid: sourceValid, generation: generation, navigation: navigation}
				f.historicalGeneration++
			case "report":
				f.reports = &projectflow.ReportBrowser{ProjectID: card.ProjectID, WorkID: work.WorkstreamID, Title: work.Title, Selected: &report, History: projectflow.ReportHistory{Items: []projectflow.Report{report}, Total: 1}, ListCursors: []string{""}, WorkCursors: []string{""}}
			case "stop":
				f.team = &overview
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == card.ProjectID {
						f.listing.Projects[i] = overview.Project
					}
				}
			}
			s.mu.Unlock()
			if action == "summary" {
				s.runProjectRecoveryDraftOwned(win, card.ProjectID, "summary", &detail, sourceValid)
			}
		})
	}()
}
