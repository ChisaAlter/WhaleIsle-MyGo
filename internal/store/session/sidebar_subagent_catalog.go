package session

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strings"

	"whaleisle/internal/dsh"
)

// SidebarSubagentActivity is an observed running count from explicit parent
// catalog IDs. Known requires both that catalog and every child's roster status.
// With Known false Count is only a lower bound, never an authoritative zero.
type SidebarSubagentActivity struct {
	Count int
	Known bool
}

// Keep only IDs and their cut for passive status. Agent navigation continues to
// own labels, paths and catalog metadata only for explicitly opened panes.
type sessionSidebarChildIDs struct {
	IDs   []string
	Seq   float64
	Known bool
}
type sidebarCatalogOwner struct{ Epoch, Host uint64 }
type sidebarCatalogState struct {
	Owner    sidebarCatalogOwner
	Client   *dsh.Client
	Mux      *dsh.Mux
	Ctx      context.Context
	Baseline bool
	ByParent map[string]sessionSidebarChildIDs
}

func validSidebarCatalogSeq(seq *float64) bool {
	return seq != nil && !math.IsNaN(*seq) && !math.IsInf(*seq, 0) && *seq >= -1 && math.Trunc(*seq) == *seq
}

// Missing key is baseline absence; null/malformed data must not erase a good cut.
func decodeSidebarCatalogValues(values json.RawMessage) ([]string, bool, bool) {
	trimmed := bytes.TrimSpace(values)
	if len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' &&
		!bytes.Contains(trimmed, []byte(`"subagentCatalog"`)) && json.Valid(trimmed) {
		return nil, false, true
	}
	var block map[string]json.RawMessage
	if json.Unmarshal(values, &block) != nil || block == nil {
		return nil, false, false
	}
	raw, present := block["subagentCatalog"]
	if !present {
		return nil, false, true
	}
	var entries []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &entries) != nil || entries == nil {
		return nil, false, false
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) == "" {
			return nil, false, false
		}
		// Original runningChildCount reduces catalog entries; retain multiplicity.
		ids = append(ids, entry.ID)
	}
	return ids, true, true
}

// Every existing control-watch entry point uses this one receiver; no new read
// or subscription is started for sidebar presentation.
func (s *Store) beginSidebarCatalogWatchLocked(ctx context.Context, client *dsh.Client, mux *dsh.Mux) sidebarCatalogOwner {
	owner := sidebarCatalogOwner{Epoch: s.sidebarCatalog.Owner.Epoch + 1, Host: s.eventsGeneration}
	s.sidebarCatalog = sidebarCatalogState{Owner: owner, Client: client, Mux: mux, Ctx: ctx}
	s.invalidateViewLocked()
	return owner
}
func (s *Store) sidebarCatalogCurrentLocked(owner sidebarCatalogOwner) bool {
	n := &s.sidebarCatalog
	return s.agentNavigationOwner == nil && s.agentRetainedRead == nil && owner == n.Owner && owner.Host == s.eventsGeneration &&
		n.Ctx != nil && n.Ctx.Err() == nil && n.Client == s.client && n.Mux == s.mux
}
func (s *Store) finishSidebarCatalogWatch(owner sidebarCatalogOwner) {
	s.mu.Lock()
	if s.sidebarCatalog.Owner != owner {
		s.mu.Unlock()
		return
	}
	s.resetSidebarCatalogLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) resetSidebarCatalogLocked() {
	// Never reuse an epoch: old cleanup cannot clear a replacement owner's data.
	epoch := s.sidebarCatalog.Owner.Epoch + 1
	s.sidebarCatalog = sidebarCatalogState{Owner: sidebarCatalogOwner{Epoch: epoch, Host: s.eventsGeneration}}
	s.invalidateViewLocked()
}

// This Host-wide status sink runs before the explicitly opened-pane router.
func (s *Store) applySidebarCatalogControlLocked(owner sidebarCatalogOwner, raw json.RawMessage) {
	if !s.sidebarCatalogCurrentLocked(owner) {
		return
	}
	var frame struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionId"`
		Key       string          `json:"key"`
		Seq       *float64        `json:"seq"`
		Value     json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	n := &s.sidebarCatalog
	switch frame.Type {
	case "baseline":
		var b struct {
			Projections map[string]agentNavigationProjection `json:"projections"`
		}
		if json.Unmarshal(frame.Value, &b) != nil || b.Projections == nil {
			return
		}
		// Stage the whole baseline atomically, including scope removals. Invalid
		// envelope data cannot turn a prior good result into a known zero.
		next := make(map[string]sessionSidebarChildIDs, len(b.Projections))
		for id, p := range b.Projections {
			if strings.TrimSpace(id) == "" || !validSidebarCatalogSeq(p.AsOfSeq) {
				return
			}
			ids, known, valid := decodeSidebarCatalogValues(p.Values)
			if !valid {
				return
			}
			cut := sessionSidebarChildIDs{IDs: ids, Known: known, Seq: *p.AsOfSeq}
			if old, ok := n.ByParent[id]; ok && (old.Seq > cut.Seq || old.Seq == cut.Seq && known) {
				cut = old
			}
			// A same-cut whole baseline may clear an omitted key, as the existing
			// navigation seed does; an equal-seq present key cannot overwrite it.
			next[id] = cut
		}
		n.ByParent, n.Baseline = next, true
		s.invalidateViewLocked()
	case "projection":
		if !n.Baseline || frame.Key != "subagentCatalog" || strings.TrimSpace(frame.SessionID) == "" || !validSidebarCatalogSeq(frame.Seq) {
			return
		}
		values, err := json.Marshal(map[string]json.RawMessage{"subagentCatalog": frame.Value})
		if err != nil {
			return
		}
		ids, known, valid := decodeSidebarCatalogValues(values)
		if !valid || !known {
			return
		}
		old, has := n.ByParent[frame.SessionID]
		if has && *frame.Seq <= old.Seq {
			return
		}
		if n.ByParent == nil {
			n.ByParent = map[string]sessionSidebarChildIDs{}
		}
		n.ByParent[frame.SessionID] = sessionSidebarChildIDs{IDs: ids, Known: true, Seq: *frame.Seq}
		s.invalidateViewLocked()
	}
}

func (s *Store) sidebarSubagentActivityLocked() map[string]SidebarSubagentActivity {
	n := &s.sidebarCatalog
	if !s.sidebarCatalogCurrentLocked(n.Owner) || !n.Baseline {
		return nil
	}
	running := make(map[string]bool, len(s.items))
	for _, item := range s.items {
		running[item.SessionID] = item.Running
	}
	out := make(map[string]SidebarSubagentActivity, len(n.ByParent))
	for parent, cut := range n.ByParent {
		status := SidebarSubagentActivity{Known: cut.Known}
		if cut.Known {
			for _, id := range cut.IDs {
				live, exists := running[id]
				if !exists {
					status.Known = false
				} else if live {
					status.Count++
				}
			}
		}
		out[parent] = status
	}
	return out
}
func cloneSidebarSubagentActivity(in map[string]SidebarSubagentActivity) map[string]SidebarSubagentActivity {
	if in == nil {
		return nil
	}
	out := make(map[string]SidebarSubagentActivity, len(in))
	for id, status := range in {
		out[id] = status
	}
	return out
}
