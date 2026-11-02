package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

const MessageKindSystemPrompt MessageKind = "system-prompt"

// Only the original model-facing system/message text is disclosed. Request
// authentication and diagnostics are not retained. Typed request configuration
// and valid tool declarations feed only the separate trajectory inspector, never prompt text.
type systemPromptEvent struct {
	Type      string          `json:"type"`
	Seq       float64         `json:"seq"`
	SurfaceOp json.RawMessage `json:"surfaceOp,omitempty"`
	Data      struct {
		Header       *RequestInspectionHeader `json:"header,omitempty"`
		Turn         *float64                 `json:"turn,omitempty"`
		Step         *float64                 `json:"step,omitempty"`
		Reason       string                   `json:"reason,omitempty"`
		StartsSeries bool                     `json:"startsSeries,omitempty"`
		Message      *struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			} `json:"content"`
		} `json:"message,omitempty"`
	} `json:"data"`
}

type systemPromptCoordEvent struct {
	Type      string          `json:"type"`
	Seq       float64         `json:"seq"`
	SurfaceOp json.RawMessage `json:"surfaceOp,omitempty"`
	Data      struct {
		Turn         *float64 `json:"turn,omitempty"`
		Step         *float64 `json:"step,omitempty"`
		Reason       string   `json:"reason,omitempty"`
		StartsSeries bool     `json:"startsSeries,omitempty"`
	} `json:"data"`
}

func unmarshalSystemPromptEvent(raw json.RawMessage, e *systemPromptEvent) error {
	if typ, ok := peekEventType(raw); ok {
		if typ != "system/message" && typ != "request/header" {
			var c systemPromptCoordEvent
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			e.Type = c.Type
			e.Seq = c.Seq
			e.SurfaceOp = c.SurfaceOp
			e.Data.Turn = c.Data.Turn
			e.Data.Step = c.Data.Step
			e.Data.Reason = c.Data.Reason
			e.Data.StartsSeries = c.Data.StartsSeries
			return nil
		}
		return json.Unmarshal(raw, e)
	}
	if !bytes.Contains(raw, []byte("system/message")) && !bytes.Contains(raw, []byte("request/header")) && !bytes.Contains(raw, []byte(`\u`)) {
		var c systemPromptCoordEvent
		if err := json.Unmarshal(raw, &c); err != nil {
			return err
		}
		e.Type = c.Type
		e.Seq = c.Seq
		e.SurfaceOp = c.SurfaceOp
		e.Data.Turn = c.Data.Turn
		e.Data.Step = c.Data.Step
		e.Data.Reason = c.Data.Reason
		e.Data.StartsSeries = c.Data.StartsSeries
		return nil
	}
	return json.Unmarshal(raw, e)
}

type promptReplacement struct {
	Op    string  `json:"op"`
	Start float64 `json:"startSeq"`
	End   float64 `json:"endSeq"`
}

func promptReplace(raw json.RawMessage) (promptReplacement, bool) {
	var op promptReplacement
	ok := json.Unmarshal(raw, &op) == nil && op.Op == "replace" && assistantSafeInt(op.Start) && assistantSafeInt(op.End) && op.Start >= 0 && op.End >= 0
	return op, ok
}

// Called only for otherwise unretained boundaries. Keep replacement positions
// even when their payload is not a system message (for example compaction).
func sanitizedPromptBoundary(raw json.RawMessage) (json.RawMessage, bool) {
	var e systemPromptEvent
	if unmarshalSystemPromptEvent(raw, &e) != nil {
		return nil, false
	}
	_, replace := promptReplace(e.SurfaceOp)
	if e.Type != "system/message" && e.Type != "request/header" && !replace && e.Data.Turn == nil {
		return nil, false
	}
	if e.Type == "request/header" {
		e.Data.Header = sanitizeRequestInspectionHeader(e.Data.Header)
	} else {
		e.Data.Header = nil
	}
	if e.Type != "system/message" {
		e.Data.Message = nil
	} else if e.Data.Message != nil {
		content := e.Data.Message.Content[:0]
		for _, block := range e.Data.Message.Content {
			if block.Type == "text" {
				content = append(content, block)
			}
		}
		e.Data.Message.Content = content
	}
	out, err := json.Marshal(e)
	return out, err == nil
}

func promptCoordinate(value *float64) float64 {
	if value != nil {
		return *value
	}
	return 0
}

type positionedPrompt struct {
	position, seq, turn, step float64
	text                      string
	update                    bool
}

// Replaying the loaded boundary journal handles live, replay and prepend alike.
// Historical cards are immutable; positional replacements only alter the prompt
// supplied to later request headers, using inherited surface order rather than
// the replacement event's chronological sequence.
func (s *Store) projectSystemPromptsLocked(messages []Message) []Message {
	rows := s.systemPromptRowsCache
	if rows == nil || !s.boundariesVerifiedLocked() {
		rows = s.computeSystemPromptRowsLocked()
		s.systemPromptRowsCache = rows
	}
	if len(rows) == 0 {
		return messages
	}
	out := append([]Message(nil), messages...)
	for _, row := range rows {
		index := len(out)
		for i, msg := range out {
			if msg.Seq >= row.Seq && msg.Seq > 0 {
				index = i
				break
			}
		}
		out = append(out, Message{})
		copy(out[index+1:], out[index:])
		out[index] = row
	}
	return out
}

func (s *Store) computeSystemPromptRowsLocked() []Message {
	seqs := make([]float64, 0, len(s.trajectoryBoundaries))
	for seq := range s.trajectoryBoundaries {
		seqs = append(seqs, seq)
	}
	sort.Float64s(seqs)
	var nodes []positionedPrompt
	replacements := map[float64]float64{}
	var effective *positionedPrompt
	var first float64
	relevant, uncertain := false, false
	var turn, step, turnStart, stepStart float64
	inStep := false
	var previousText string
	var previousTurn, previousStep float64
	hasHeader := false
	lastSystemAppend := false
	rows := make([]Message, 0)
	anchor := func(seq float64, initial bool) float64 {
		if !inStep || (!hasHeader && !initial) || (hasHeader && previousTurn == turn && previousStep == step) {
			return seq
		}
		if step == 1 && turnStart > 0 {
			return turnStart
		}
		if stepStart > 0 {
			return stepStart
		}
		return seq
	}
	add := func(e systemPromptEvent, text string, update bool, at float64) {
		if e.Type == "request/header" {
			if s.systemPromptAnchors == nil {
				s.systemPromptAnchors = map[float64]float64{}
			}
			if previous, ok := s.systemPromptAnchors[e.Seq]; ok {
				at = previous
			} else {
				s.systemPromptAnchors[e.Seq] = at
			}
		}
		eventType := "system-prompt"
		if update {
			eventType = "system-prompt-update"
		}
		rows = append(rows, Message{ID: fmt.Sprintf("system-prompt:%s:%g", e.Type, e.Seq), Kind: MessageKindSystemPrompt, Role: "context", Body: text, Seq: at, TurnSeq: turnStart, EventType: eventType})
	}
	for _, seq := range seqs {
		var e systemPromptEvent
		if unmarshalSystemPromptEvent(s.trajectoryBoundaries[seq], &e) != nil {
			continue
		}
		switch e.Type {
		case "turn/start":
			turn, turnStart, step, stepStart, inStep = promptCoordinate(e.Data.Turn), seq, 0, 0, false
		case "step/start":
			turn, step, stepStart, inStep = promptCoordinate(e.Data.Turn), promptCoordinate(e.Data.Step), seq, true
		case "step/end":
			inStep = false
		case "turn/end":
			turnStart, stepStart, inStep = 0, 0, false
		}
		// Explicit payload coordinates establish a partial-window location even
		// when turn/start or step/start has not been loaded yet.
		if e.Data.Turn != nil && e.Type != "step/end" && e.Type != "turn/end" {
			if turn != *e.Data.Turn {
				turnStart, stepStart, inStep = 0, 0, false
			}
			turn = *e.Data.Turn
			if e.Data.Step != nil {
				if step != *e.Data.Step {
					stepStart = 0
				}
				step, inStep = *e.Data.Step, true
			}
		}
		op, replace := promptReplace(e.SurfaceOp)
		if e.Type == "system/message" || replace {
			if !relevant {
				first, relevant = seq, true
			}
			lastSystemAppend = e.Type == "system/message" && string(e.SurfaceOp) == `"append"`
			if replace {
				_, knownStart := replacements[op.Start]
				_, knownEnd := replacements[op.End]
				uncertain = uncertain || (op.Start < first && !knownStart) || (op.End < first && !knownEnd)
			}
			if uncertain {
				nodes, effective = nil, nil
				replacements = map[float64]float64{}
				continue
			}
			position := seq
			if replace {
				position = op.Start
				if inherited, ok := replacements[op.Start]; ok {
					position = inherited
				}
				end := op.End
				if inherited, ok := replacements[op.End]; ok {
					end = inherited
				}
				kept := nodes[:0]
				for _, n := range nodes {
					if n.position < position || n.position > end {
						kept = append(kept, n)
					}
				}
				nodes = kept
				for replacement, pos := range replacements {
					if pos >= position && pos <= end {
						delete(replacements, replacement)
					}
				}
				replacements[seq] = position
			}
			var introduced *positionedPrompt
			if e.Type == "system/message" {
				n := positionedPrompt{position: position, seq: seq, turn: promptCoordinate(e.Data.Turn), step: promptCoordinate(e.Data.Step)}
				if e.Data.Message != nil {
					for _, b := range e.Data.Message.Content {
						if b.Type == "text" {
							n.text += b.Text
						}
					}
				}
				for _, old := range nodes {
					if old.text != "" && lastSystemAppend {
						n.update = true
					}
				}
				introduced = &n
				nodes = append(nodes, n)
				sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].position < nodes[j].position })
			}
			var surviving *positionedPrompt
			for i := range nodes {
				if nodes[i].text != "" {
					n := nodes[i]
					surviving = &n
				}
			}
			if surviving == nil {
				effective = nil
			} else if effective == nil || surviving.seq != effective.seq {
				n := *surviving
				if introduced == nil || introduced.seq != surviving.seq {
					n.update = false
				}
				effective = &n
			}
			if introduced != nil && lastSystemAppend && introduced.text != "" {
				at := anchor(seq, true)
				// An append owns its own initial card even if a prior header exists.
				if !introduced.update && inStep {
					at = seq
					if stepStart > 0 {
						at = stepStart
					}
					if step == 1 && turnStart > 0 {
						at = turnStart
					}
				}
				if introduced.update {
					at = seq
				}
				add(e, introduced.text, introduced.update, at)
			}
		}
		if e.Type != "request/header" {
			continue
		}
		text := ""
		if effective != nil {
			text = effective.text
		}
		shownByAppend := effective != nil && lastSystemAppend && (effective.update || !hasHeader) && inStep && effective.turn == turn && effective.step == step
		changed := hasHeader && previousText != text && (effective == nil || !effective.update)
		visible := !shownByAppend && (!hasHeader || e.Data.Reason != "change" || e.Data.StartsSeries || changed)
		if visible && text != "" {
			add(e, text, false, anchor(seq, e.Data.Reason == "initial"))
		}
		previousText, previousTurn, previousStep, hasHeader = text, turn, step, true
	}
	return rows
}
