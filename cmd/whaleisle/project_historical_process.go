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

type projectHistoricalState struct {
	browser                *projectflow.HistoricalBrowser
	cancel                 context.CancelFunc
	sourceValid            func(*projectFlowState) bool
	generation, navigation uint64
}

// Activity belongs to an admitted cached request revision, independently of the
// controller's most recently opened detail card. Caller holds s.mu.
func (s *projectController) projectActivityHistoricalSourceValidLocked(f *projectFlowState, activity *projectActivityState, view uint64) bool {
	if activity == nil || f.activity != activity || activity.browser.View() != view {
		return false
	}
	_, known := s.requestRevertCardLocked(activity.owner, activity.request)
	return known
}

func (s *projectController) closeProjectHistoricalLocked() {
	f := s.projectFlow
	if h := f.historical; h != nil {
		if h.cancel != nil {
			h.cancel()
			h.cancel = nil
			f.busy = false
		}
		h.browser.Close()
		f.historical = nil
		f.historicalGeneration++
	}
}
func (s *projectController) bindProjectHistorical(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	f := s.projectFlow
	h := f.historical
	if h != nil && (h.generation != f.generation || h.navigation != f.navigation || !h.sourceValid(f)) {
		s.closeProjectHistoricalLocked()
		h = nil
	}
	var view uint64
	if h != nil {
		snapshot := h.browser.Snapshot()
		m.Historical = &snapshot
		view = snapshot.View
	}
	expected, generation, navigation, historicalGeneration := f.client, f.generation, f.navigation, f.historicalGeneration
	reports, delivery, activity := f.reports, f.deliveries, f.activity
	var deliveryToken, activityToken uint64
	if delivery != nil {
		deliveryToken = delivery.browser.NavigationToken()
	}
	if activity != nil {
		activityToken = activity.browser.View()
	}
	s.mu.Unlock()
	openSource := func(source string, terminal *projectflow.DeliveryTerminalRef) {
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		f := s.projectFlow
		if win == nil || client == nil || expected != client || f.client != client || f.generation != generation || f.navigation != navigation || f.busy || !f.visible() || !f.listing.Available || f.historical != nil || f.historicalGeneration != historicalGeneration {
			s.mu.Unlock()
			return
		}
		var target projectflow.HistoricalTarget
		var valid func(*projectFlowState) bool
		switch source {
		case "reports":
			if reports == nil || f.reports != reports || reports.Selected == nil {
				s.mu.Unlock()
				return
			}
			r := *reports.Selected
			target = projectflow.HistoricalTarget{ProjectID: reports.ProjectID, WorkstreamID: reports.WorkID, DelegationRef: r.DelegationRef, ReportID: r.CallID, Report: r}
			valid = func(f *projectFlowState) bool {
				return f.reports == reports && reports.Selected != nil && reports.Selected.DelegationRef == target.DelegationRef && reports.Selected.CallID == target.ReportID
			}
		case "delivery", "delivery-terminal":
			if delivery == nil || f.deliveries != delivery || delivery.browser.NavigationToken() != deliveryToken {
				s.mu.Unlock()
				return
			}
			d := delivery.browser.Snapshot()
			if d.Selected == nil || source == "delivery" && (d.Report == nil || d.ReportRef == nil) {
				s.mu.Unlock()
				return
			}
			if source == "delivery-terminal" {
				found := false
				if terminal != nil {
					for _, ref := range d.Selected.TerminalRefs {
						if ref == *terminal {
							found = true
							break
						}
					}
				}
				if !found {
					s.mu.Unlock()
					return
				}
				target = projectflow.HistoricalTarget{ProjectID: d.Scope.ProjectID, WorkstreamID: terminal.WorkstreamID, DelegationRef: terminal.DelegationRef, RunID: terminal.RunID, Terminal: true, RequestID: d.Selected.RequestID, RequestRevision: d.Selected.RequestRevision}
			} else {
				target = projectflow.HistoricalTarget{ProjectID: d.Scope.ProjectID, WorkstreamID: d.ReportRef.WorkstreamID, DelegationRef: d.ReportRef.DelegationRef, ReportID: d.Report.CallID, Report: *d.Report}
			}
			for _, ref := range d.Selected.TerminalRefs {
				if ref.WorkstreamID == target.WorkstreamID && ref.DelegationRef == target.DelegationRef {
					if target.RunID != "" && target.RunID != ref.RunID {
						f.err = i18n.T(i18n.KeyCmdProjectDeliveryAmbiguousRuns)
						s.mu.Unlock()
						return
					}
					target.RunID = ref.RunID
				}
			}
			valid = func(f *projectFlowState) bool {
				return f.deliveries == delivery && delivery.browser.NavigationToken() == deliveryToken
			}
		case "activity":
			if !s.projectActivityHistoricalSourceValidLocked(f, activity, activityToken) {
				s.mu.Unlock()
				return
			}
			a := activity.browser.Snapshot()
			if a.Selected == nil || a.Report == nil || a.ReportRef == nil {
				s.mu.Unlock()
				return
			}
			target = projectflow.HistoricalTarget{ProjectID: a.Scope.ProjectID, WorkstreamID: a.ReportRef.WorkstreamID, DelegationRef: a.ReportRef.DelegationRef, ReportID: a.Report.CallID, Report: *a.Report}
			if w := a.Selected.Work; w != nil && w.WorkstreamID == target.WorkstreamID && w.DelegationRef == target.DelegationRef {
				target.RunID = w.RunID
				target.WorkerSessionID = w.WorkerSessionID
			}
			if d := a.Selected.Delivery; d != nil {
				for _, ref := range d.TerminalRefs {
					if ref.WorkstreamID == target.WorkstreamID && ref.DelegationRef == target.DelegationRef {
						if target.RunID != "" && target.RunID != ref.RunID {
							f.err = i18n.T(i18n.KeyCmdProjectEventAmbiguousRuns)
							s.mu.Unlock()
							return
						}
						target.RunID = ref.RunID
					}
				}
			}
			valid = func(f *projectFlowState) bool {
				return s.projectActivityHistoricalSourceValidLocked(f, activity, activityToken)
			}
		default:
			s.mu.Unlock()
			return
		}
		s.stopProjectProcessLocked()
		h := &projectHistoricalState{browser: projectflow.NewHistoricalBrowser(fmt.Sprintf("%p", client), generation, target), sourceValid: valid, generation: generation, navigation: navigation}
		f.historical = h
		f.historicalGeneration++
		s.mu.Unlock()
		s.runProjectHistorical(win, h, 0, "record")
	}
	m.OnHistoricalSource = func(source string) { openSource(source, nil) }
	m.OnDeliveryTerminal = func(ref projectflow.DeliveryTerminalRef) { openSource("delivery-terminal", &ref) }
	m.OnHistoricalAction = func(action string) { s.runProjectHistorical(win, h, view, action) }
}
func (s *projectController) runProjectHistorical(win *mygo.Window, h *projectHistoricalState, view uint64, action string) {
	s.dshMu.Lock()
	client, mux := s.dshClient, s.dshMux
	s.dshMu.Unlock()
	s.mu.Lock()
	f := s.projectFlow
	if h == nil || f.historical != h || h.browser.View() != view {
		s.mu.Unlock()
		return
	}
	if action == "close" {
		s.closeProjectHistoricalLocked()
		s.mu.Unlock()
		return
	}
	if client == nil || win == nil || f.client != client || h.browser.Snapshot().Host != fmt.Sprintf("%p", client) || f.generation != h.generation || f.navigation != h.navigation || !h.sourceValid(f) || !f.visible() || f.busy || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	if action == "next" || action == "previous" {
		delta := 1
		if action == "previous" {
			delta = -1
		}
		h.browser.MovePage(delta)
		s.mu.Unlock()
		return
	}
	snapshot := h.browser.Snapshot()
	req, err := h.browser.Begin(action)
	if err != nil {
		f.err = err.Error()
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	h.cancel = cancel
	f.busy = true
	s.mu.Unlock()
	go func() {
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var binding projectflow.HistoricalBinding
		var window projectflow.HistoricalWindow
		var loadErr error
		if action == "record" {
			binding, loadErr = rpc.HistoricalAssignment(ctx, snapshot.Target)
		} else if action == "process" {
			window, loadErr = rpc.ReadHistoricalProcess(ctx, mux, *snapshot.Binding)
		}
		win.Update(func() {
			s.dshMu.Lock()
			current, currentMux := s.dshClient, s.dshMux
			s.dshMu.Unlock()
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if f.historical != h || f.client != client || current != client || !h.browser.Pending(req) {
				return
			}
			if f.generation != h.generation || f.navigation != h.navigation || !h.sourceValid(f) || !f.visible() || action == "process" && currentMux != mux {
				s.closeProjectHistoricalLocked()
				return
			}
			if action == "record" {
				h.browser.CompleteRecord(req, binding, loadErr)
			} else {
				h.browser.CompleteProcess(req, window, loadErr)
			}
			h.cancel = nil
			f.busy = false
		})
	}()
}
