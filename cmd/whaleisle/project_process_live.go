package main

import (
	"context"
	"github.com/egoist/mygo"
	"whaleisle/internal/projectflow"
)

func (s *mainState) stopProjectProcessLocked() {
	f := &s.projectFlow
	if f.processCancel != nil {
		f.processCancel()
		f.processCancel = nil
	}
	f.processMux = nil
	f.processWatchGeneration++
	if f.process != nil && f.process.Live {
		next := *f.process
		next.Live = false
		f.process = &next
	}
}
func (s *mainState) syncProjectProcessLocked() {
	f := &s.projectFlow
	if f.processCancel != nil && (f.processMux != s.dshMux || !f.expanded || f.detail == nil || f.teamID == "" || f.reports != nil || f.materialKey != "" || f.requestProjectID != "" || f.archiveCandidate != nil) {
		s.stopProjectProcessLocked()
	}
}
func (s *mainState) watchProjectProcess(win *mygo.Window, id string) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := &s.projectFlow
	client, mux := s.dshClient, s.dshMux
	if f.busy || client == nil || mux == nil || f.client != client || !f.expanded || f.teamID != id || f.team == nil || f.detail == nil {
		s.mu.Unlock()
		return
	}
	p, d := f.team.Project, *f.detail
	if _, err := d.ProcessAddress(p); err != nil {
		f.err = err.Error()
		s.mu.Unlock()
		return
	}
	s.stopProjectProcessLocked()
	ctx, cancel := context.WithCancel(context.Background())
	f.processCancel = cancel
	f.processMux = mux
	watch, detailGeneration := f.processWatchGeneration, f.detailGeneration
	f.process = &projectflow.ProcessSnapshot{Live: true}
	f.err = ""
	s.mu.Unlock()
	current := func() bool {
		f := &s.projectFlow
		return s.dshClient == client && s.dshMux == mux && f.client == client && f.expanded && f.teamID == id && f.detailGeneration == detailGeneration && f.processWatchGeneration == watch && f.detail != nil && f.detail.Workstream.ID == d.Workstream.ID && f.detail.Workstream.WorkerSessionID == d.Workstream.WorkerSessionID && f.detail.Workstream.CurrentDelegationRef == d.Workstream.CurrentDelegationRef
	}
	go func() {
		defer cancel()
		err := projectflow.FollowProcess(ctx, mux, p, d, func(next projectflow.ProcessSnapshot) bool {
			s.mu.Lock()
			valid := ctx.Err() == nil && current()
			s.mu.Unlock()
			if !valid {
				return false
			}
			// At most one immutable snapshot waits on the UI queue. The shared
			// mux's bounded stream inbox supplies backpressure while it paints.
			ack := make(chan struct{})
			win.Update(func() {
				defer close(ack)
				s.mu.Lock()
				defer s.mu.Unlock()
				if current() {
					s.projectFlow.process = &next
				}
			})
			select {
			case <-ack:
				return true
			case <-ctx.Done():
				return false
			}
		})
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if !current() {
				return
			}
			f := &s.projectFlow
			f.processCancel = nil
			next := projectflow.ProcessSnapshot{}
			if f.process != nil {
				next = *f.process
			}
			next.Live = false
			if err != nil {
				next.Error = err.Error()
			}
			f.process = &next
		})
	}()
}
