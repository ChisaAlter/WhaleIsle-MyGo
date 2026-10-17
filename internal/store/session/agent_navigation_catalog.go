package session

import (
	"encoding/json"
	"fmt"
	"math"
)

// Only the open pane/path retains catalog metadata. Closing or changing Host
// discards it with agentNavigation; it is never a global child/session cache.
type agentNavigationCatalog struct {
	values    json.RawMessage // only this key, including {} for capability-absent
	seq       float64
	sequenced bool
	ready     bool
}
type agentNavigationProjection struct {
	AsOfSeq *float64        `json:"asOfSeq"`
	Values  json.RawMessage `json:"values"`
}

// Host computes the whole value. No client-side catalog event fold is allowed.
// Match ProjectionValueStore: higher seq wins; a same-cut seed may clear an
// omitted key, but cannot replace a present key at an equal/older watermark.
func seedAgentNavigationCatalog(v *AgentNavigationView, values json.RawMessage, seq *float64, items []Item) error {
	var block map[string]json.RawMessage
	if json.Unmarshal(values, &block) != nil || block == nil {
		return fmt.Errorf("invalid catalog projection values")
	}
	value, present := block["subagentCatalog"]
	if present {
		var entries []json.RawMessage
		if json.Unmarshal(value, &entries) != nil || entries == nil {
			return fmt.Errorf("invalid subagent catalog value")
		}
	}
	if seq != nil && (math.IsNaN(*seq) || math.IsInf(*seq, 0) || *seq < -1 || math.Trunc(*seq) != *seq) {
		return fmt.Errorf("invalid catalog watermark")
	}
	old := v.catalog
	if old.sequenced && (seq == nil || *seq < old.seq || (present && *seq == old.seq)) {
		return nil
	}
	own := json.RawMessage(`{}`)
	if present {
		own, _ = json.Marshal(map[string]json.RawMessage{"subagentCatalog": value})
	}
	rows, err := agentNavigationRows(v.SessionID, own, items)
	if err != nil {
		return err
	}
	v.catalog = agentNavigationCatalog{values: own, ready: true, sequenced: seq != nil && present}
	if seq != nil {
		v.catalog.seq = *seq
	}
	v.Rows = rows
	return nil
}

// Re-derive status and fallback labels from the already-live list. A catalog
// label keeps priority, and an authoritative [] never falls back to lineage.
func (s *Store) refreshAgentNavigationRowsLocked(v *AgentNavigationView) {
	if !v.catalog.ready {
		return
	}
	if rows, err := agentNavigationRows(v.SessionID, v.catalog.values, s.items); err == nil {
		v.Rows = rows
	}
}

// Called only by the existing Host/client-fenced session/control receiver.
// It updates locations the user already opened, not unseen descendants. No
// requests, streams, prompts or permission/question actions originate here.
func (s *Store) applyAgentNavigationControlLocked(raw json.RawMessage) {
	if !s.agentNavigationOwnedLocked() {
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
	blocks := map[string]agentNavigationProjection{}
	switch frame.Type {
	case "projection":
		if (frame.Key != "subagentCatalog" && frame.Key != "imageLimits") || frame.SessionID == "" || frame.Seq == nil {
			return
		}
		values, err := json.Marshal(map[string]json.RawMessage{frame.Key: frame.Value})
		if err != nil {
			return
		}
		blocks[frame.SessionID] = agentNavigationProjection{AsOfSeq: frame.Seq, Values: values}
	case "baseline":
		var baseline struct {
			Projections map[string]agentNavigationProjection `json:"projections"`
		}
		if json.Unmarshal(frame.Value, &baseline) != nil {
			return
		}
		blocks = baseline.Projections
	default:
		return
	}
	apply := func(v *AgentNavigationView) {
		projection, ok := blocks[v.SessionID]
		if !ok || projection.AsOfSeq == nil {
			return
		}
		if frame.Type == "baseline" || frame.Key == "imageLimits" {
			seedChildImageLimits(v, projection.Values, projection.AsOfSeq)
		}
		if frame.Type == "baseline" || frame.Key == "subagentCatalog" {
			_ = seedAgentNavigationCatalog(v, projection.Values, projection.AsOfSeq, s.items)
		}
	}
	n := &s.agentNavigation
	apply(&n.view)
	for _, b := range n.branches {
		apply(&b.view)
	}
	for i := range n.parents {
		apply(&n.parents[i])
	}
}

// Counts describe the known root catalog, never an invented recursive total.
func (s *Store) agentRootCountsLocked(v *AgentNavigationView) {
	root := *v
	if len(s.agentNavigation.parents) > 0 {
		root = s.agentNavigation.parents[0]
		s.refreshAgentNavigationRowsLocked(&root)
	}
	v.RootCountsKnown = root.catalog.ready
	if !v.RootCountsKnown {
		return
	}
	v.RootChildren = len(root.Rows)
	for _, row := range root.Rows {
		if row.Address.Mode == "external" {
			continue
		}
		if !row.ActivityKnown {
			v.RootActivityUnknown++
		} else if row.Running {
			v.RootRunning++
		}
	}
}
