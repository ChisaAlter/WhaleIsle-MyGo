package session

import (
	"bytes"
	"encoding/json"
	"math"
	"sort"
)

type contextTriggerIdentity struct {
	ID     string `json:"id"`
	Source struct {
		Kind string `json:"kind"`
	} `json:"source"`
}

type contextTriggerFact struct {
	Type      string          `json:"type"`
	Seq       float64         `json:"seq"`
	SurfaceOp json.RawMessage `json:"surfaceOp"`
	Data      struct {
		Turn         json.RawMessage          `json:"turn"`
		Step         json.RawMessage          `json:"step"`
		ID           string                   `json:"id"`
		Source       json.RawMessage          `json:"source"`
		Target       string                   `json:"target"`
		Start        float64                  `json:"start"`
		RemovedCount float64                  `json:"removedCount"`
		Inserted     []contextTriggerIdentity `json:"inserted"`
		Outcome      string                   `json:"outcome"`
	} `json:"data"`
}

type contextTriggerInbox struct {
	pending        []contextTriggerIdentity
	currentClaimed map[string]bool
	claimSeq       float64
	claimedHuman   bool
}

// Omitted removedCount means zero, as in the reference Inbox's explicit splice
// argument. A claim contains only identities removed from this loaded window.
func (inbox *contextTriggerInbox) splice(fact contextTriggerFact) {
	d := fact.Data
	startNumber := math.Trunc(d.Start)
	if startNumber < 0 {
		startNumber = math.Max(float64(len(inbox.pending))+startNumber, 0)
	} else {
		startNumber = math.Min(startNumber, float64(len(inbox.pending)))
	}
	start := int(startNumber)
	removedCount := int(math.Min(math.Max(math.Trunc(d.RemovedCount), 0), float64(len(inbox.pending)-start)))
	removed := inbox.pending[start : start+removedCount]
	if d.RemovedCount > 0 && d.Outcome != "canceled" {
		inbox.currentClaimed = make(map[string]bool, len(removed))
		inbox.claimedHuman = false
		for _, identity := range removed {
			inbox.currentClaimed[identity.ID] = true
			inbox.claimedHuman = inbox.claimedHuman || identity.Source.Kind == "user"
		}
		inbox.claimSeq = fact.Seq
	} else {
		for _, identity := range d.Inserted {
			delete(inbox.currentClaimed, identity.ID)
		}
	}
	next := make([]contextTriggerIdentity, 0, len(inbox.pending)-removedCount+len(d.Inserted))
	next = append(next, inbox.pending[:start]...)
	next = append(next, d.Inserted...)
	inbox.pending = append(next, inbox.pending[start+removedCount:]...)
}

func contextTriggerCoordinate(raw json.RawMessage) (float64, bool) {
	var value float64
	ok := json.Unmarshal(raw, &value) == nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) && assistantSafeInt(value) && value >= 0
	return value, ok
}

// contextTriggerWaking folds only the available durable window. Missing claim
// identities remain unknown until an earlier page supplies them.
func contextTriggerWaking(events map[float64]json.RawMessage) map[float64]bool {
	waking := make(map[float64]bool)
	if len(events) > 0 {
		hasSplice := false
		for _, raw := range events {
			if typ, ok := peekEventType(raw); ok {
				if typ == "agent/inbox/spliced" {
					hasSplice = true
					break
				}
				continue
			}
			if bytes.Contains(raw, []byte("agent/inbox/spliced")) || bytes.Contains(raw, []byte(`\u`)) {
				hasSplice = true
				break
			}
		}
		if !hasSplice {
			return waking
		}
		seqs := make([]float64, 0, len(events))
		for seq := range events {
			seqs = append(seqs, seq)
		}
		sort.Float64s(seqs)
		facts := make([]contextTriggerFact, 0, len(seqs))
		turnStarts := make(map[float64]float64)
		for _, seq := range seqs {
			raw := events[seq]
			if typ, ok := peekEventType(raw); ok && isToolTimingEventType(typ) {
				part, _ := readToolLifecycleMemo(seq, raw)
				if !part.jsonValid {
					continue
				}
				var fact contextTriggerFact
				fact.Type = part.eventType
				fact.Seq = part.seq
				fact.SurfaceOp = part.surfaceOp
				if part.turn != nil {
					fact.Data.Turn, _ = json.Marshal(*part.turn)
				}
				if part.step != nil {
					fact.Data.Step, _ = json.Marshal(*part.step)
				}
				facts = append(facts, fact)
				continue
			}
			var fact contextTriggerFact
			if json.Unmarshal(raw, &fact) != nil {
				continue
			}
			facts = append(facts, fact)
			if fact.Type == "turn/start" {
				if turn, ok := contextTriggerCoordinate(fact.Data.Turn); ok {
					turnStarts[turn] = seq
				}
			}
		}
		nextTurn := contextTriggerInbox{claimSeq: -1}
		nextStep := contextTriggerInbox{claimSeq: -1}
		var currentTurn, currentStep float64
		haveTurn, haveStep := false, false
		for _, fact := range facts {
			turn, explicitTurn := contextTriggerCoordinate(fact.Data.Turn)
			step, explicitStep := contextTriggerCoordinate(fact.Data.Step)
			session := bytes.Equal(bytes.TrimSpace(fact.Data.Turn), []byte("null"))
			if fact.Type == "turn/start" {
				currentTurn, haveTurn, haveStep = turn, explicitTurn, false
			} else if fact.Type == "step/start" {
				currentTurn, haveTurn = turn, explicitTurn
				currentStep, haveStep = step, explicitStep
			}
			// Explicit coordinates on any event can establish a partial-window
			// location. Session-scoped events leave the running location intact.
			if !session && explicitTurn {
				if !haveTurn || currentTurn != turn {
					haveStep = false
				}
				currentTurn, haveTurn = turn, true
				if explicitStep {
					currentStep, haveStep = step, true
				}
			}
			if fact.Type == "agent/inbox/spliced" {
				switch fact.Data.Target {
				case "next-turn":
					nextTurn.splice(fact)
				case "next-step":
					nextStep.splice(fact)
				}
			}
			var source struct {
				Kind string `json:"kind"`
			}
			if fact.Type == "user/message" {
				_ = json.Unmarshal(fact.Data.Source, &source)
			}
			if fact.Type == "user/message" && string(fact.SurfaceOp) == `"append"` && source.Kind != "user" {
				locationStep, locationHasStep := currentStep, haveStep
				if explicitStep {
					locationStep, locationHasStep = step, true
				}
				turnStart, hasTurnStart := turnStarts[currentTurn]
				idleSteer := !session && haveTurn && locationHasStep && locationStep == 1 && hasTurnStart &&
					nextStep.claimSeq > turnStart && nextTurn.claimSeq < turnStart && !nextStep.claimedHuman &&
					nextStep.currentClaimed[fact.Data.ID]
				waking[fact.Seq] = nextTurn.currentClaimed[fact.Data.ID] || idleSteer
			}
			if fact.Type == "step/end" && haveTurn && currentTurn == turn && haveStep && currentStep == step {
				haveStep = false
			}
			if fact.Type == "turn/end" && haveTurn && currentTurn == turn {
				haveTurn, haveStep = false, false
			}
		}
	}
	return waking
}

// projectContextTriggersLocked caches drawing facts with the exact retained
// Inspector window. Capturing a changed event or resetting its binding clears
// the cache; ordinary frames only apply its flags to copied Message structs.
func (s *Store) projectContextTriggersLocked(messages []Message) []Message {
	var waking map[float64]bool
	if s.inspector.client == s.client && s.inspector.session == s.selectedID && s.inspector.generation == s.followGeneration {
		if s.inspector.contextWaking == nil {
			s.inspector.contextWaking = contextTriggerWaking(s.inspector.events)
		}
		waking = s.inspector.contextWaking
	}
	out := messages
	copied := false
	for i, message := range messages {
		value := message.Kind == MessageKindContext && message.EventType == "user/message" && waking[message.Seq]
		if message.ContextWaking == value {
			continue
		}
		if !copied {
			out = append([]Message(nil), messages...)
			copied = true
		}
		out[i].ContextWaking = value
	}
	return out
}
