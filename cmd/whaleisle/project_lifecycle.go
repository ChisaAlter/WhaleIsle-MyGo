package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func (s *projectController) runProjectLifecycle(win *mygo.Window, id, action string) {
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
		f.err = i18n.T(i18n.KeyCmdProjectHostUnavailableRefreshToRetry)
		s.mu.Unlock()
		return
	}
	var selected projectflow.Project
	for _, p := range f.listing.Projects {
		if p.ID == id {
			selected = p
			break
		}
	}
	if selected.ID == "" || selected.Archiving {
		f.err = i18n.T(i18n.KeyCmdProjectUnavailableOrArchivingRefreshTo)
		s.mu.Unlock()
		return
	}
	if action == "archive" && (f.archiveCandidate == nil || f.archiveCandidate.ID != id) {
		s.mu.Unlock()
		return
	}
	if action == "restore" && selected.Lifecycle != "archived" {
		s.mu.Unlock()
		return
	}
	if action == "prepare" && selected.Lifecycle != "ready" {
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.lifecycleNote = ""
	f.generation++
	generation := f.generation
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var err error
		var candidate *projectflow.Project
		var result projectflow.LifecycleResult
		var listing projectflow.Listing
		var refreshErr error
		mutated := false
		if action == "prepare" {
			var fresh projectflow.Project
			fresh, err = rpc.LifecycleProject(ctx, id)
			if err == nil {
				if fresh.Archiving || fresh.Lifecycle != "ready" {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectLifecycleChanged))
				} else if fresh.HasActiveWork() || fresh.Activity.State == "" {
					candidate = &fresh
				} else {
					result, err = rpc.SetProjectLifecycle(ctx, id, "archive")
					mutated = err == nil
				}
			}
		} else if action == "archive" || action == "restore" {
			result, err = rpc.SetProjectLifecycle(ctx, id, action)
			mutated = err == nil
		} else {
			err = fmt.Errorf("unsupported Project lifecycle action")
		}
		if mutated {
			listing, refreshErr = rpc.List(ctx)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			if candidate != nil {
				f.archiveCandidate = candidate
				return
			}
			f.archiveCandidate = nil
			if mutated {
				if refreshErr == nil {
					f.listing = listing
					f.loaded = true
				} else {
					for i, p := range f.listing.Projects {
						if p.ID == id {
							f.listing.Projects[i] = result.Project
						}
					}
					f.err = i18n.T(i18n.KeyCmdProjectStateUpdatedButListRefreshFaile) + refreshErr.Error()
				}
				// Clear stale snapshots whose controls depend on Project lifecycle.
				if f.teamID == id {
					f.teamID = ""
					f.team = nil
					f.detail = nil
					f.process = nil
					f.detailGeneration++
				}
				if f.requestProjectID == id {
					f.requestProjectID = ""
					f.requestCard = nil
					f.requestGeneration++
				}
				if result.Project.Lifecycle == "archived" {
					f.lifecycleNote = i18n.T(i18n.KeyCmdProjectArchivedWorkHasStoppedConversat)
				} else {
					f.lifecycleNote = i18n.T(i18n.KeyCmdProjectRestoredSendANewInstructionInT)
				}
			}
		})
	}()
}
