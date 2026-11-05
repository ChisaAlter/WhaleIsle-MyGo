package session

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"whaleisle/internal/dsh/api"
)

func parseSubagentCatalogEntries(sessionID string, raw json.RawMessage) ([]SubagentCatalogEntry, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false
	}
	var entries []SubagentCatalogEntry
	if err := json.Unmarshal(raw, &entries); err != nil || entries == nil {
		return nil, false
	}
	filtered := make([]SubagentCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" || id == sessionID {
			continue
		}
		filtered = append(filtered, SubagentCatalogEntry{
			ID:    id,
			Label: strings.TrimSpace(entry.Label),
			Mode:  strings.TrimSpace(entry.Mode),
		})
	}
	return filtered, true
}

func (s *Store) setSubagentCatalogLocked(sessionID string, entries []SubagentCatalogEntry) {
	prev := s.subagentCatalogs[sessionID]
	if reflect.DeepEqual(prev, entries) || (len(prev) == 0 && len(entries) == 0) {
		return
	}
	if len(entries) == 0 {
		delete(s.subagentCatalogs, sessionID)
	} else {
		if s.subagentCatalogs == nil {
			s.subagentCatalogs = make(map[string][]SubagentCatalogEntry)
		}
		s.subagentCatalogs[sessionID] = entries
	}
	s.invalidateViewLocked()
}

func (s *Store) applySubagentCatalogValuesLocked(sessionID string, values json.RawMessage) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || len(values) == 0 || string(values) == "null" {
		return
	}
	var block map[string]json.RawMessage
	if json.Unmarshal(values, &block) != nil || block == nil {
		return
	}
	raw, present := block["subagentCatalog"]
	if !present {
		return
	}
	entries, ok := parseSubagentCatalogEntries(sessionID, raw)
	if !ok {
		return
	}
	s.setSubagentCatalogLocked(sessionID, entries)
}

// RefreshProjections loads session/projections for the selected session and
// applies the title when present (list projections may lag behind follow).
func (s *Store) RefreshProjections(ctx context.Context) {
	s.refreshProjectionsForOwner(ctx, nil)
}

func (s *Store) refreshProjectionsForOwner(ctx context.Context, owner *workspaceSelectedOwner) {
	s.mu.Lock()
	if ctx.Err() != nil || (owner != nil && !owner.currentLocked(s)) {
		s.mu.Unlock()
		return
	}
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		var out api.SessionProjectionsValue
		err := client.Call(ctx, "session/projections", map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		}, &out)
		if err != nil {
			return
		}
		s.mu.Lock()
		if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) || (owner != nil && !owner.currentLocked(s)) {
			s.mu.Unlock()
			return
		}
		if !out.Sequenced() {
			if out.Kind == "migration-required" {
				// The published union carries cached display hints, not a live
				// baseline. Existing cached/live merge rules protect newer owners.
				s.applySessionPresentationValuesLocked(sessionID, "cached", 0, out.Values)
				s.applySessionSubagentIdentityValuesLocked(sessionID, "cached", 0, out.Values)
				s.applyProjectionTitleHintLocked(sessionID, out.Values)
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		s.applyCostBaselineLocked(sessionID, out.AsOfSeq, out.Values)
		s.applySubagentCatalogValuesLocked(sessionID, out.Values)
		s.applySessionPresentationValuesLocked(sessionID, "sequenced", out.AsOfSeq, out.Values)
		s.applySessionSubagentIdentityValuesLocked(sessionID, "sequenced", out.AsOfSeq, out.Values)
		s.applyQueueValuesLocked(out.Values, out.AsOfSeq)
		s.applyQuestionValuesLocked(out.Values, out.AsOfSeq)
		s.applyPermissionValuesLocked(out.Values, out.AsOfSeq)
		s.applyModelValuesLocked(out.Values, out.AsOfSeq)
		s.applyContextValuesLocked(out.Values, out.AsOfSeq)
		s.applyPlanModeValuesLocked(out.Values, out.AsOfSeq)
		s.applyImageLimitValuesLocked(out.Values, out.AsOfSeq)
		s.applyHistoryOutlineValuesLocked(out.Values, out.AsOfSeq)
		s.applyProjectionTitleHintLocked(sessionID, out.Values)
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) applyProjectionTitleHintLocked(sessionID string, values json.RawMessage) {
	title := titleFromProjections(values)
	for i := range s.items {
		if s.items[i].SessionID == sessionID {
			if title != "" && (strings.TrimSpace(s.items[i].Title) == "" || s.items[i].Title == sessionID) {
				s.items[i].Title = title
				s.invalidateViewLocked()
			}
			break
		}
	}
}
