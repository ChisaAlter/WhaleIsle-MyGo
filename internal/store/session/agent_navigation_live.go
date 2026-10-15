package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// freezeAgentNavigationView discards transient previews when leaving latest.
// Cached parents and earlier pages never advertise a stopped stream as live.
func freezeAgentNavigationView(v AgentNavigationView) AgentNavigationView {
	v = cloneAgentNavigationView(v)
	v.Process.Live = false
	if len(v.Messages) > 0 {
		kept := v.Messages[:0]
		for _, message := range v.Messages {
			if !message.Streaming {
				kept = append(kept, message)
			}
		}
		v.Messages = kept
		v.Body = WorkflowMemberDisplayText(WorkflowMemberView{Process: v.Process, Messages: kept, PresentationReady: true})
		if v.Body == "" {
			v.Body = WorkflowProcessText(WorkflowMemberView{Process: v.Process})
		}
	}
	return v
}

func (s *Store) agentNavigationLiveCurrentLocked(ctx context.Context, token uint64, client *dsh.Client, mux *dsh.Mux, address AgentSessionAddress) bool {
	n := &s.agentNavigation
	return ctx.Err() == nil && s.agentNavigationOwnedLocked() && n.view.Token == token && n.client == client && n.mux == mux && n.view.Address == address
}

// Only explicit Open/Latest enters this path. Sync after recovery loads a root
// directory, never a child follow. Existing pane/Host/navigation cancellation
// owns this lifetime; no detached observer or automatic prompt/tool is created.
func (s *Store) startAgentNavigationLiveLocked() {
	n := &s.agentNavigation
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel
	n.view.Token++
	n.view.Busy = true
	n.view.Error = ""
	go s.followAgentNavigation(ctx, n.view.Token, n.client, n.mux, n.view.Address)
}

func (s *Store) followAgentNavigation(ctx context.Context, token uint64, client *dsh.Client, mux *dsh.Mux, address AgentSessionAddress) {
	current := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.agentNavigationLiveCurrentLocked(ctx, token, client, mux, address)
	}
	payload := workflowMemberFollowPayload(map[string]any{"kind": address.Kind, "parentSessionId": address.ParentSessionID, "childSessionId": address.ChildSessionID, "mode": address.Mode})
	for current() {
		if err := mux.Dial(ctx); err != nil {
			s.finishAgentNavigationLive(ctx, token, client, mux, address, err)
			return
		}
		if !current() {
			return
		}
		stream, err := mux.Open(ctx, "session/follow", payload)
		if err != nil {
			s.finishAgentNavigationLive(ctx, token, client, mux, address, err)
			return
		}
		catalogCtx, cancelCatalog := context.WithCancel(ctx)
		reducer := newWorkflowMemberStream()
		openingCtx, stopOpening := context.WithTimeout(ctx, 15*time.Second)
		opening, restart := true, false
		for current() {
			receiveCtx := ctx
			if opening {
				receiveCtx = openingCtx
			}
			raw, recvErr := stream.Recv(receiveCtx)
			if recvErr != nil {
				err = recvErr
				break
			}
			if !current() {
				stopOpening()
				cancelCatalog()
				stream.Cancel()
				return
			}
			var baseline *agentNavigationLiveBaseline
			if opening {
				baseline, err = parseAgentNavigationLiveOpening(raw, address)
				if err != nil {
					break
				}
				stopOpening()
				opening = false
			}
			restart, err = reducer.apply(raw)
			if err != nil || restart {
				break
			}
			messages := reducer.messages()
			body := WorkflowMemberDisplayText(WorkflowMemberView{Process: reducer.process, Messages: messages, PresentationReady: true})
			if body == "" {
				body = WorkflowProcessText(WorkflowMemberView{Process: reducer.process})
			}
			s.mu.Lock()
			if !s.agentNavigationLiveCurrentLocked(ctx, token, client, mux, address) {
				s.mu.Unlock()
				stopOpening()
				cancelCatalog()
				stream.Cancel()
				return
			}
			n := &s.agentNavigation
			n.view.Process = reducer.process
			n.view.Messages = messages
			n.view.Body = body
			n.view.Busy = false
			if baseline != nil {
				n.view.Error = ""
				// Original follow carries the authoritative same-cut projection baseline.
				// Use it without blocking stream draining on another RPC or activating anything.
				var values json.RawMessage
				if baseline.Projections != nil {
					values = baseline.Projections.Values
				}
				rows, catalogErr := agentNavigationRows(address.ChildSessionID, values, s.items)
				if catalogErr == nil {
					n.view.Rows = rows
				} else {
					n.view.Error = "Agent directory: " + catalogErr.Error()
				}
			}
			notify := !n.notifyPending
			n.notifyPending = true
			s.mu.Unlock()
			if notify {
				s.fire()
			}
			if baseline != nil && baseline.Projections == nil {
				go s.loadAgentNavigationLiveCatalog(catalogCtx, token, client, mux, address)
			}
		}
		stopOpening()
		cancelCatalog()
		stream.Cancel()
		if !current() {
			return
		}
		if !restart {
			s.finishAgentNavigationLive(ctx, token, client, mux, address, err)
			return
		}
		s.mu.Lock()
		if !s.agentNavigationLiveCurrentLocked(ctx, token, client, mux, address) {
			s.mu.Unlock()
			return
		}
		n := &s.agentNavigation
		n.view = freezeAgentNavigationView(n.view)
		n.view.Busy = true
		n.view.Error = "实时序列不连续，正在重新同步… / Live sequence changed; resynchronizing…"
		notify := !n.notifyPending
		n.notifyPending = true
		s.mu.Unlock()
		if notify {
			s.fire()
		}
		// Same verified reducer-fault recovery policy as Workflow member streams:
		// restart only this logical follow, retaining the complete original address.
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *Store) finishAgentNavigationLive(ctx context.Context, token uint64, client *dsh.Client, mux *dsh.Mux, address AgentSessionAddress, err error) {
	s.mu.Lock()
	if !s.agentNavigationLiveCurrentLocked(ctx, token, client, mux, address) {
		s.mu.Unlock()
		return
	}
	n := &s.agentNavigation
	n.view = freezeAgentNavigationView(n.view)
	n.view.Busy = false
	if err != nil {
		n.view.Error = err.Error()
	}
	cancel := n.cancel
	n.cancel = nil
	notify := !n.notifyPending
	n.notifyPending = true
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if notify {
		s.fire()
	}
}

type agentNavigationLiveBaseline struct {
	Type   string `json:"type"`
	Header struct {
		ID string `json:"id"`
	} `json:"header"`
	Cursor      *float64          `json:"cursor"`
	Records     []json.RawMessage `json:"records"`
	Projections *struct {
		Values json.RawMessage `json:"values"`
	} `json:"projections"`
}

func parseAgentNavigationLiveOpening(raw json.RawMessage, address AgentSessionAddress) (*agentNavigationLiveBaseline, error) {
	var baseline agentNavigationLiveBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return nil, err
	}
	if baseline.Type != "snapshot" || baseline.Header.ID != address.ChildSessionID || baseline.Cursor == nil || *baseline.Cursor < -1 || baseline.Records == nil {
		return nil, fmt.Errorf("Invalid addressed child live snapshot")
	}
	for _, rawRecord := range baseline.Records {
		var record struct {
			Type  string `json:"type"`
			Event struct {
				Seq *float64 `json:"seq"`
			} `json:"event"`
		}
		if json.Unmarshal(rawRecord, &record) != nil || record.Type != "event" || record.Event.Seq == nil || *record.Event.Seq < 0 || *record.Event.Seq > *baseline.Cursor {
			return nil, fmt.Errorf("Child live snapshot crossed its fixed opening cursor")
		}
	}
	return &baseline, nil
}

// Older snapshots without projection baselines get one bounded non-activating
// fallback. It never stalls draining the live carrier and dies with that exact
// logical stream, so a late old baseline cannot overwrite a newer rebaseline.
func (s *Store) loadAgentNavigationLiveCatalog(ctx context.Context, token uint64, client *dsh.Client, mux *dsh.Mux, address AgentSessionAddress) {
	query, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var projection api.SessionProjectionsValue
	err := client.Call(query, "session/projections", map[string]any{"request": map[string]any{"sessionId": address.ChildSessionID}}, &projection)
	s.mu.Lock()
	if !s.agentNavigationLiveCurrentLocked(query, token, client, mux, address) {
		s.mu.Unlock()
		return
	}
	n := &s.agentNavigation
	if err == nil {
		n.view.Rows, err = agentNavigationRows(address.ChildSessionID, projection.Values, s.items)
	}
	if err != nil {
		n.view.Error = "Agent directory: " + err.Error()
	}
	notify := !n.notifyPending
	n.notifyPending = true
	s.mu.Unlock()
	if notify {
		s.fire()
	}
}
