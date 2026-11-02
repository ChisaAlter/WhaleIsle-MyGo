package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
)

// WatchControl opens session/control and records the latest control frame type.
func (s *Store) WatchControl(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	client := s.client
	sessionID := s.selectedID
	changed := s.controlNote != ""
	s.controlNote = ""
	if changed {
		s.invalidateViewLocked()
	}
	notify := s.notify
	if mux == nil || sessionID == "" || ctx.Err() != nil {
		if s.controlCancel != nil {
			s.controlCancel()
			s.controlCancel = nil
		}
		s.controlWatchStreamCtx = nil
		s.mu.Unlock()
		if changed && notify != nil {
			notify()
		}
		return
	}
	if s.controlWatchStreamCtx != nil && s.controlWatchStreamCtx.Err() == nil &&
		s.controlWatchClient == client && s.controlWatchMux == mux &&
		s.controlWatchHost == s.eventsGeneration && s.controlWatchParentCtx == ctx {
		s.mu.Unlock()
		if changed && notify != nil {
			notify()
		}
		return
	}
	if s.controlCancel != nil {
		s.controlCancel()
		s.controlCancel = nil
	}
	fctx, cancel := context.WithCancel(ctx)
	s.controlCancel = cancel
	s.controlWatchStreamCtx = fctx
	s.controlWatchParentCtx = ctx
	s.controlWatchClient = client
	s.controlWatchMux = mux
	s.controlWatchHost = s.eventsGeneration
	s.mu.Unlock()
	if changed && notify != nil {
		notify()
	}
	go s.runControlWatch(fctx, client, mux, "")
}

func (s *Store) controlWatchCurrentLocked(ctx context.Context, client *dsh.Client, mux *dsh.Mux, sessionID string) bool {
	if ctx.Err() != nil || s.client != client || s.mux != mux || s.selectedID == "" {
		return false
	}
	return sessionID == "" || s.selectedID == sessionID
}

func (s *Store) runControlWatch(ctx context.Context, client *dsh.Client, mux *dsh.Mux, sessionID string) {
	defer func() {
		s.mu.Lock()
		if s.controlWatchStreamCtx == ctx {
			s.controlWatchStreamCtx = nil
		}
		s.mu.Unlock()
	}()
	var catalogOwner sidebarCatalogOwner
	s.mu.Lock()
	current := s.controlWatchCurrentLocked(ctx, client, mux, sessionID)
	if current {
		catalogOwner = s.beginSidebarCatalogWatchLocked(ctx, client, mux)
	}
	s.mu.Unlock()
	if current {
		s.fire()
		defer s.finishSidebarCatalogWatch(catalogOwner)
	}
	// Captured readers keep the existing launch contract even after navigation;
	// the receive fence discards their frames. Only the current reader owns IDs.
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "session/control", map[string]any{"args": map[string]any{}})
	if err != nil {
		return
	}
	defer stream.Cancel()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			return
		}
		var head struct {
			Type  string          `json:"type"`
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		_ = json.Unmarshal(raw, &head)
		note := strings.TrimSpace(head.Type)
		if note == "" {
			note = truncateRunes(string(raw), 80)
		}
		var baselineProjections map[string]agentNavigationProjection
		var baselineOK bool
		if head.Type == "baseline" {
			var b struct {
				Projections map[string]agentNavigationProjection `json:"projections"`
			}
			if json.Unmarshal(head.Value, &b) == nil && b.Projections != nil {
				baselineProjections = b.Projections
				baselineOK = true
			}
		}
		s.mu.Lock()
		if !s.controlWatchCurrentLocked(ctx, client, mux, sessionID) {
			s.mu.Unlock()
			return
		}
		switch head.Type {
		case "baseline":
			s.applyControlBaselineMapLocked(catalogOwner, raw, baselineProjections, baselineOK)
		case "projection":
			s.applyControlProjectionKeyLocked(catalogOwner, head.Key, raw)
		default:
			s.applySidebarCatalogControlLocked(catalogOwner, raw)
			s.applySubagentCatalogControlLocked(raw)
			s.applyAgentNavigationControlLocked(raw)
			s.applySessionPresentationControlLocked(raw)
			s.applySessionSubagentIdentityControlLocked(raw)
			s.applyCostControlLocked(raw)
			s.applyAgentMetricsControlLocked(raw)
			s.applyQueueControlLocked(raw)
			s.applyQuestionControlLocked(raw)
			s.applyPermissionControlLocked(raw)
			s.applyModelControlLocked(raw)
			s.applyContextControlLocked(raw)
			s.applyPlanModeControlLocked(raw)
			s.applyImageLimitControlLocked(raw)
			s.applyHistoryOutlineControlLocked(raw)
		}
		if s.controlNote != note {
			s.controlNote = note
			s.invalidateViewLocked()
		}
		s.mu.Unlock()
		s.fire()
	}
}

func (s *Store) applyControlProjectionKeyLocked(catalogOwner sidebarCatalogOwner, key string, raw json.RawMessage) {
	switch key {
	case "subagentCatalog":
		s.applySidebarCatalogControlLocked(catalogOwner, raw)
		s.applySubagentCatalogControlLocked(raw)
		s.applyAgentNavigationControlLocked(raw)
	case "imageLimits":
		s.applyAgentNavigationControlLocked(raw)
		s.applyImageLimitControlLocked(raw)
	case "sessionListMetadata":
		s.applySessionPresentationControlLocked(raw)
	case "subagent":
		s.applySessionSubagentIdentityControlLocked(raw)
	case "billedUsage":
		s.applyCostControlLocked(raw)
	case "tokenUsage", "subagentTiming":
		s.applyAgentMetricsControlLocked(raw)
	case "inbox":
		s.applyQueueControlLocked(raw)
		s.applyQuestionControlLocked(raw)
	case "userQuestions":
		s.applyQuestionControlLocked(raw)
	case "permissions":
		s.applyPermissionControlLocked(raw)
	case "modelSelection":
		s.applyModelControlLocked(raw)
	case "contextPressure", "contextBreakdown":
		s.applyContextControlLocked(raw)
	case "plan":
		s.applyPlanModeControlLocked(raw)
	case "turnOutline":
		s.applyHistoryOutlineControlLocked(raw)
	}
}

func (s *Store) applyControlBaselineMapLocked(catalogOwner sidebarCatalogOwner, raw json.RawMessage, projections map[string]agentNavigationProjection, ok bool) {
	s.invalidateViewLocked()
	if !ok {
		return
	}
	if s.sidebarCatalogCurrentLocked(catalogOwner) {
		n := &s.sidebarCatalog
		next := make(map[string]sessionSidebarChildIDs, len(projections))
		validAll := true
		for id, p := range projections {
			if strings.TrimSpace(id) == "" || !validSidebarCatalogSeq(p.AsOfSeq) {
				validAll = false
				break
			}
			ids, known, valid := decodeSidebarCatalogValues(p.Values)
			if !valid {
				validAll = false
				break
			}
			cut := sessionSidebarChildIDs{IDs: ids, Known: known, Seq: *p.AsOfSeq}
			if old, exists := n.ByParent[id]; exists && (old.Seq > cut.Seq || old.Seq == cut.Seq && known) {
				cut = old
			}
			next[id] = cut
		}
		if validAll {
			n.ByParent, n.Baseline = next, true
		}
	}
	if s.agentNavigationOwnedLocked() {
		s.applyAgentNavigationControlLocked(raw)
	}
	itemIdx := make(map[string]int, len(s.items))
	for i := range s.items {
		itemIdx[s.items[i].SessionID] = i
	}
	for id, p := range projections {
		seq := 0.0
		hasSeq := p.AsOfSeq != nil
		if hasSeq {
			seq = *p.AsOfSeq
		}
		if hasSeq {
			if presentation, known := presentationFromValues(p.Values); known {
				if i, exists := itemIdx[id]; exists {
					next := Item{Presentation: presentation, presentationKnown: true, presentationKind: "sequenced", presentationSeq: seq}
					mergeSessionPresentation(&next, s.items[i])
					s.items[i].Presentation = next.Presentation
					s.items[i].presentationKnown, s.items[i].presentationKind, s.items[i].presentationSeq = next.presentationKnown, next.presentationKind, next.presentationSeq
				}
			}
			if identity, known := subagentIdentityFromValues(p.Values); known && id != "" {
				next := Item{SessionID: id, SubagentIdentity: identity, subagentIdentityKnown: true, subagentIdentityKind: "sequenced", subagentIdentitySeq: seq}
				if i, exists := itemIdx[id]; exists {
					mergeSessionSubagentIdentity(&next, s.items[i])
					s.syncSessionSubagentIdentityLocked(&next)
					s.items[i].SubagentIdentity = next.SubagentIdentity
					s.items[i].subagentIdentityKnown, s.items[i].subagentIdentityKind, s.items[i].subagentIdentitySeq = next.subagentIdentityKnown, next.subagentIdentityKind, next.subagentIdentitySeq
				} else {
					s.syncSessionSubagentIdentityLocked(&next)
				}
			}
			if i, exists := itemIdx[id]; exists && validSidebarCatalogSeq(p.AsOfSeq) {
				next := decodeAgentMetrics(p.Values, seq)
				m := &s.items[i].AgentMetrics
				if !m.Ready || seq >= m.TokenSeq {
					m.Tokens = next.Tokens
					m.TokenSeq = seq
				}
				if !m.Ready || seq >= m.TimingSeq {
					m.Timing = next.Timing
					m.Completed = next.Completed
					m.TimingSeq = seq
				}
				m.Ready = true
			}
		}
		s.applyCostBaselineLocked(id, seq, p.Values)
		s.applySubagentCatalogValuesLocked(id, p.Values)
		s.applyQuestionValuesForSessionLocked(id, p.Values, seq)
	}
}

func (s *Store) applySubagentCatalogControlLocked(raw json.RawMessage) {
	var frame struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionId"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	switch frame.Type {
	case "projection":
		sessionID := strings.TrimSpace(frame.SessionID)
		if frame.Key != "subagentCatalog" || sessionID == "" {
			return
		}
		if entries, ok := parseSubagentCatalogEntries(sessionID, frame.Value); ok {
			s.setSubagentCatalogLocked(sessionID, entries)
		}
	case "baseline":
		var b struct {
			Projections map[string]agentNavigationProjection `json:"projections"`
		}
		if json.Unmarshal(frame.Value, &b) != nil || b.Projections == nil {
			return
		}
		for id, p := range b.Projections {
			s.applySubagentCatalogValuesLocked(id, p.Values)
		}
	}
}
