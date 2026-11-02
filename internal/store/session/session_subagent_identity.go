package session

import (
	"bytes"
	"encoding/json"
)

// SessionSubagentIdentity is the Host's child-owned `subagent` projection,
// folded from its own descriptor suffix. Origin and parent catalog rows do not
// prove this identity and must never supply or guess its lifecycle mode.
type SessionSubagentIdentity struct {
	Mode  string  `json:"mode"`
	Label string  `json:"label,omitempty"`
	Seq   float64 `json:"seq"`
}

type sessionSubagentIdentityState struct {
	value *SessionSubagentIdentity
	kind  string
	seq   float64
}

func subagentIdentityValue(raw json.RawMessage) (*SessionSubagentIdentity, bool) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, true
	}
	var identity SessionSubagentIdentity
	if json.Unmarshal(raw, &identity) != nil {
		return nil, false
	}
	if identity.Mode != "one-shot" && identity.Mode != "continuable" {
		return nil, true // An explicit unknown identity cannot retain authority.
	}
	return &identity, true
}

func subagentIdentityFromValues(raw json.RawMessage) (*SessionSubagentIdentity, bool) {
	if !bytes.Contains(raw, []byte(`"subagent"`)) {
		return nil, false
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil, false
	}
	value, present := values["subagent"]
	if !present {
		return nil, false // Sparse availability summaries preserve known facts.
	}
	return subagentIdentityValue(value)
}

func mergeSessionSubagentIdentity(next *Item, old Item) {
	if !next.subagentIdentityKnown || old.subagentIdentityKnown &&
		(old.subagentIdentityKind == "sequenced" && next.subagentIdentityKind == "cached" ||
			old.subagentIdentityKind == next.subagentIdentityKind && old.subagentIdentitySeq > next.subagentIdentitySeq) {
		next.SubagentIdentity = old.SubagentIdentity
		next.subagentIdentityKnown, next.subagentIdentityKind, next.subagentIdentitySeq = old.subagentIdentityKnown, old.subagentIdentityKind, old.subagentIdentitySeq
	}
}

// Control baselines can precede the roster. The source keeps projections by
// session independently, so an availability-only row must see that same value.
// The row is observed on every list refresh; only a changed identity advances the
// Snapshot revision.
func (s *Store) syncSessionSubagentIdentityLocked(item *Item) {
	beforeValue := item.SubagentIdentity
	beforeKnown, beforeKind, beforeSeq := item.subagentIdentityKnown, item.subagentIdentityKind, item.subagentIdentitySeq
	if cached, exists := s.subagentIdentities[item.SessionID]; exists {
		mergeSessionSubagentIdentity(item, Item{SubagentIdentity: cached.value, subagentIdentityKnown: true, subagentIdentityKind: cached.kind, subagentIdentitySeq: cached.seq})
	}
	if item.subagentIdentityKnown {
		if s.subagentIdentities == nil {
			s.subagentIdentities = map[string]sessionSubagentIdentityState{}
		}
		s.subagentIdentities[item.SessionID] = sessionSubagentIdentityState{value: item.SubagentIdentity, kind: item.subagentIdentityKind, seq: item.subagentIdentitySeq}
	}
	if beforeValue != item.SubagentIdentity || beforeKnown != item.subagentIdentityKnown || beforeKind != item.subagentIdentityKind || beforeSeq != item.subagentIdentitySeq {
		s.invalidateViewLocked()
	}
}

func (s *Store) applySessionSubagentIdentityValuesLocked(id, kind string, seq float64, raw json.RawMessage) {
	if identity, known := subagentIdentityFromValues(raw); known {
		s.applySessionSubagentIdentityLocked(id, kind, seq, identity)
	}
}

func (s *Store) applySessionSubagentIdentityLocked(id, kind string, seq float64, identity *SessionSubagentIdentity) {
	// Accepted control data is authoritative even when it clears a row, so the
	// view advances here; only the per-row list refresh below is change-aware.
	s.invalidateViewLocked()
	if id == "" {
		return
	}
	next := Item{SessionID: id, SubagentIdentity: identity, subagentIdentityKnown: true, subagentIdentityKind: kind, subagentIdentitySeq: seq}
	for i := range s.items {
		if s.items[i].SessionID != id {
			continue
		}
		mergeSessionSubagentIdentity(&next, s.items[i])
		s.syncSessionSubagentIdentityLocked(&next)
		s.items[i].SubagentIdentity = next.SubagentIdentity
		s.items[i].subagentIdentityKnown, s.items[i].subagentIdentityKind, s.items[i].subagentIdentitySeq = next.subagentIdentityKnown, next.subagentIdentityKind, next.subagentIdentitySeq
		return
	}
	s.syncSessionSubagentIdentityLocked(&next)
}

func (s *Store) applySessionSubagentIdentityControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var frame struct {
		Type, SessionID, Key string
		Seq                  *float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	if frame.Type == "projection" && frame.Key == "subagent" && frame.Seq != nil {
		if identity, known := subagentIdentityValue(frame.Value); known {
			s.applySessionSubagentIdentityLocked(frame.SessionID, "sequenced", *frame.Seq, identity)
		}
	}
	if frame.Type == "baseline" {
		var baseline struct {
			Projections map[string]agentNavigationProjection
		}
		if json.Unmarshal(frame.Value, &baseline) == nil {
			for id, projection := range baseline.Projections {
				if projection.AsOfSeq != nil {
					s.applySessionSubagentIdentityValuesLocked(id, "sequenced", *projection.AsOfSeq, projection.Values)
				}
			}
		}
	}
}

func (s *Store) resetSessionSubagentIdentitiesLocked() {
	s.invalidateViewLocked()
	s.subagentIdentities = nil
	for i := range s.items {
		s.items[i].SubagentIdentity = nil
		s.items[i].subagentIdentityKnown, s.items[i].subagentIdentityKind, s.items[i].subagentIdentitySeq = false, "", 0
	}
}
