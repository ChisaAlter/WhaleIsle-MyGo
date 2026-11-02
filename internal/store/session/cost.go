package session

import (
	"encoding/json"
	"whaleisle/internal/store/sessioncost"
)

type costProjection struct {
	Seq   float64
	Value *sessioncost.Projection
}

func (s *Store) applyCostBaselineLocked(id string, seq float64, values json.RawMessage) {
	s.invalidateViewLocked()
	if id == "" {
		return
	}
	if s.costProjections == nil {
		s.costProjections = map[string]costProjection{}
	}
	old, ok := s.costProjections[id]
	if ok && seq < old.Seq {
		return
	}
	s.costProjections[id] = costProjection{seq, sessioncost.Decode(values)}
}
func (s *Store) applyCostControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var frame struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	if frame.Type == "projection" && frame.Key == "billedUsage" {
		values, _ := json.Marshal(map[string]json.RawMessage{"billedUsage": frame.Value})
		s.applyCostBaselineLocked(frame.SessionID, frame.Seq, values)
	}
	if frame.Type == "baseline" {
		var baseline struct {
			Projections map[string]struct {
				AsOfSeq float64
				Values  json.RawMessage
			}
		}
		if json.Unmarshal(frame.Value, &baseline) == nil {
			for id, p := range baseline.Projections {
				s.applyCostBaselineLocked(id, p.AsOfSeq, p.Values)
			}
		}
	}
}
func (s *Store) costSourcesLocked() []sessioncost.Source {
	out := make([]sessioncost.Source, 0, len(s.items)+1)
	found := false
	for _, item := range s.items {
		p := item.BilledUsage
		if live, ok := s.costProjections[item.SessionID]; ok && (item.BilledUsageKind != "sequenced" || live.Seq >= item.BilledUsageSeq) {
			p = live.Value
		}
		out = append(out, sessioncost.Source{ID: item.SessionID, Parent: item.ParentID, Origin: item.Origin, Name: item.Title, Usage: p})
		found = found || item.SessionID == s.selectedID
	}
	if !found && s.selectedID != "" {
		out = append(out, sessioncost.Source{ID: s.selectedID, Usage: s.costProjections[s.selectedID].Value})
	}
	return out
}
