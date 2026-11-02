package session

import (
	"encoding/json"
	"math"
)

type ContextBreakdown struct {
	SystemTokens  float64 `json:"systemTokens"`
	ToolsTokens   float64 `json:"toolsTokens"`
	MessageTokens float64 `json:"messageTokens"`
}

type ContextOccupancy struct {
	UsedTokens    float64
	ContextWindow float64
	Percent       int
	Breakdown     *ContextBreakdown
}

type contextProjectionValue struct {
	Seq float64
	Raw json.RawMessage
}
type contextProjectionState struct {
	SessionID           string
	Generation          uint64
	Pressure, Breakdown contextProjectionValue
}

func validContextNumber(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 }

func (s *Store) applyContextProjectionLocked(key string, raw json.RawMessage, seq float64) {
	s.invalidateViewLocked()
	if s.selectedID == "" || !validCompactionSeq(seq) {
		return
	}
	p := &s.contextProjection
	if p.SessionID != s.selectedID || p.Generation != s.followGeneration {
		*p = contextProjectionState{SessionID: s.selectedID, Generation: s.followGeneration, Pressure: contextProjectionValue{Seq: -1}, Breakdown: contextProjectionValue{Seq: -1}}
	}
	target := &p.Pressure
	if key == "contextBreakdown" {
		target = &p.Breakdown
	} else if key != "contextPressure" {
		return
	}
	if seq < target.Seq {
		return
	}
	// Null/missing withdraws capability. Invalid values are retained as unavailable,
	// rather than continuing to display a stale reading after a route switch.
	*target = contextProjectionValue{Seq: seq, Raw: append(json.RawMessage(nil), raw...)}
}
func (s *Store) applyContextValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	s.applyContextProjectionLocked("contextPressure", all["contextPressure"], seq)
	s.applyContextProjectionLocked("contextBreakdown", all["contextBreakdown"], seq)
}
func (s *Store) applyContextControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var f struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &f) != nil || f.Type != "projection" || f.SessionID != s.selectedID {
		return
	}
	s.applyContextProjectionLocked(f.Key, f.Value, f.Seq)
}
func (s *Store) contextOccupancyLocked() *ContextOccupancy {
	p := s.contextProjection
	if s.selectedID == "" || p.SessionID != s.selectedID || p.Generation != s.followGeneration {
		return nil
	}
	var pressure struct{ PressureTokens, ProjectedTokens, ContextWindow *float64 }
	if json.Unmarshal(p.Pressure.Raw, &pressure) != nil {
		return nil
	}
	used := pressure.ProjectedTokens
	if used == nil {
		used = pressure.PressureTokens
	}
	if used == nil || pressure.ContextWindow == nil || !validContextNumber(*used) || !validContextNumber(*pressure.ContextWindow) || *pressure.ContextWindow == 0 {
		return nil
	}
	result := &ContextOccupancy{UsedTokens: *used, ContextWindow: *pressure.ContextWindow, Percent: int(math.Min(100, math.Round(*used / *pressure.ContextWindow * 100)))}
	var breakdown struct{ SystemTokens, ToolsTokens, MessageTokens *float64 }
	if json.Unmarshal(p.Breakdown.Raw, &breakdown) == nil && breakdown.SystemTokens != nil && breakdown.ToolsTokens != nil && breakdown.MessageTokens != nil &&
		validContextNumber(*breakdown.SystemTokens) && validContextNumber(*breakdown.ToolsTokens) && validContextNumber(*breakdown.MessageTokens) {
		result.Breakdown = &ContextBreakdown{SystemTokens: *breakdown.SystemTokens, ToolsTokens: *breakdown.ToolsTokens, MessageTokens: *breakdown.MessageTokens}
	}
	return result
}
