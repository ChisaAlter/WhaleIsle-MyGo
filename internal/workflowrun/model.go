// Package workflowrun projects durable workflow records without executing tools.
package workflowrun

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
)

type Member struct {
	Seq                    float64
	Label, ChildID, Status string
}
type Phase struct {
	Key     string
	Name    *string
	Members []Member
}
type Run struct {
	ID, Name, Status string
	AnchorSeq        float64
	Phases           []Phase
	Events           []json.RawMessage
	EventSeqs        []float64
}
type event struct {
	Type string  `json:"type"`
	Seq  float64 `json:"seq"`
	Data struct {
		RunID      string   `json:"runId"`
		Name       *string  `json:"name"`
		Seq        *float64 `json:"seq"`
		Label      *string  `json:"label"`
		ChildID    string   `json:"childId"`
		Phase      *string  `json:"phase"`
		StopReason string   `json:"stopReason"`
		Outcome    string   `json:"outcome"`
		Turn       *float64 `json:"turn"`
		Step       *float64 `json:"step"`
	} `json:"data"`
}

func Boundary(raw json.RawMessage) bool {
	var e event
	if json.Unmarshal(raw, &e) != nil {
		return false
	}
	switch e.Type {
	case "turn/start", "turn/end", "step/start", "step/end":
		return true
	}
	return false
}
func validNumber(n float64) bool {
	return n >= 0 && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n
}
func phaseKey(p *string) string {
	if p == nil {
		return "missing"
	}
	return "value:" + strconv.Itoa(len(*p)) + ":" + *p
}
func stopStatus(s string) string {
	switch s {
	case "completed", "cancelled":
		return s
	case "error":
		return "failed"
	}
	return ""
}
func outcomeStatus(s string) string {
	switch s {
	case "completed", "cancelled", "failed":
		return s
	}
	return ""
}

// Fold uses exact run/member identities and the run-start's turn/step scope.
// Missing starts and malformed/unknown records remain unclaimed for raw display.
// It never infers completion from a disconnected stream or a Session running bit.
func Fold(raws []json.RawMessage) []Run {
	type record struct {
		e   event
		raw json.RawMessage
	}
	records := make([]record, 0, len(raws))
	seen := map[float64]bool{}
	for _, raw := range raws {
		var e event
		if json.Unmarshal(raw, &e) != nil || !validNumber(e.Seq) || seen[e.Seq] {
			continue
		}
		seen[e.Seq] = true
		records = append(records, record{e, raw})
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].e.Seq < records[j].e.Seq })
	type location struct {
		turn, step       float64
		hasTurn, hasStep bool
	}
	type state struct {
		run      Run
		location location
		stopped  bool
		members  map[float64][2]int
		phases   map[string]int
	}
	var current location
	states := []*state{}
	byID := map[string]*state{}
	closedTurns := map[float64]bool{}
	closedSteps := map[[2]float64]bool{}
	for _, rec := range records {
		e := rec.e
		d := e.Data
		switch e.Type {
		case "turn/start":
			if d.Turn != nil {
				current = location{turn: *d.Turn, hasTurn: true}
			}
			continue
		case "turn/end":
			if d.Turn != nil {
				closedTurns[*d.Turn] = true
				if current.hasTurn && current.turn == *d.Turn {
					current = location{}
				}
			}
			continue
		case "step/start":
			if d.Turn != nil && d.Step != nil {
				current = location{*d.Turn, *d.Step, true, true}
			}
			continue
		case "step/end":
			if d.Turn != nil && d.Step != nil {
				closedSteps[[2]float64{*d.Turn, *d.Step}] = true
				if current.hasStep && current.turn == *d.Turn && current.step == *d.Step {
					current.hasStep = false
				}
			}
			continue
		}
		if d.RunID == "" {
			continue
		}
		s := byID[d.RunID]
		accepted := false
		switch e.Type {
		case "tool-workflow/run-start":
			if s != nil || d.Name == nil {
				continue
			}
			s = &state{run: Run{ID: d.RunID, Name: *d.Name, Status: "running", AnchorSeq: e.Seq}, location: current, members: map[float64][2]int{}, phases: map[string]int{}}
			byID[d.RunID] = s
			states = append(states, s)
			accepted = true
		case "tool-workflow/agent-start":
			if s == nil || d.Seq == nil || !validNumber(*d.Seq) || d.Label == nil || d.ChildID == "" {
				continue
			}
			if _, exists := s.members[*d.Seq]; exists {
				continue
			}
			key := phaseKey(d.Phase)
			pi, exists := s.phases[key]
			if !exists {
				pi = len(s.run.Phases)
				s.phases[key] = pi
				var name *string
				if d.Phase != nil {
					v := *d.Phase
					name = &v
				}
				s.run.Phases = append(s.run.Phases, Phase{Key: key, Name: name})
			}
			mi := len(s.run.Phases[pi].Members)
			s.members[*d.Seq] = [2]int{pi, mi}
			s.run.Phases[pi].Members = append(s.run.Phases[pi].Members, Member{*d.Seq, *d.Label, d.ChildID, "running"})
			accepted = true
		case "tool-workflow/agent-end":
			if s == nil || d.Seq == nil {
				continue
			}
			at, exists := s.members[*d.Seq]
			status := outcomeStatus(d.Outcome)
			if !exists || status == "" {
				continue
			}
			s.run.Phases[at[0]].Members[at[1]].Status = status
			accepted = true
		case "tool-workflow/run-end":
			status := stopStatus(d.StopReason)
			if s == nil || status == "" {
				continue
			}
			s.run.Status = status
			s.stopped = true
			accepted = true
		}
		if accepted {
			s.run.EventSeqs = append(s.run.EventSeqs, e.Seq)
			s.run.Events = append(s.run.Events, append(json.RawMessage(nil), rec.raw...))
		}
	}
	out := make([]Run, 0, len(states))
	for _, s := range states {
		l := s.location
		interrupted := !s.stopped && (l.hasTurn && closedTurns[l.turn] || l.hasStep && closedSteps[[2]float64{l.turn, l.step}])
		if interrupted {
			s.run.Status = "interrupted"
			for pi := range s.run.Phases {
				for mi := range s.run.Phases[pi].Members {
					m := &s.run.Phases[pi].Members[mi]
					if m.Status == "running" {
						m.Status = "interrupted"
					}
				}
			}
		}
		out = append(out, s.run)
	}
	return out
}

// OneShotAddress requires a parent-owned catalog entry, never a bare child ID.
func OneShotAddress(parent, child string, values json.RawMessage) (map[string]any, bool) {
	var p struct {
		Catalog []struct {
			ID   string `json:"id"`
			Mode string `json:"mode"`
		} `json:"subagentCatalog"`
	}
	if parent == "" || child == "" || json.Unmarshal(values, &p) != nil {
		return nil, false
	}
	for _, entry := range p.Catalog {
		if entry.ID == child && entry.Mode == "one-shot" {
			return map[string]any{"kind": "subagent", "parentSessionId": parent, "childSessionId": child, "mode": "one-shot"}, true
		}
	}
	return nil, false
}
