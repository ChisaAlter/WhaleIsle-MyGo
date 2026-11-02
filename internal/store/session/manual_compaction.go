package session

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ManualCompactionView exists only after a correlated surface replacement
// checkpoint has landed. Summary generation by itself is not a committed result.
type ManualCompactionView struct {
	CompactionID       string
	CheckpointSeq      float64
	SummaryEventSeq    *float64
	Summary            string
	ShadowedItemCount  *int
	ShadowedTokenCount *int
}

type manualCompactionFact struct {
	kind, commandID, compactionID string
	seq, time                     float64
	summary                       string
	itemCount, tokenCount         *int
}

func validCompactionSeq(value float64) bool {
	return value >= 0 && value <= 9007199254740991 && value == float64(int64(value))
}

// Parse explicit transaction identities only; never infer ownership from text,
// temporal proximity, a command name, or a numeric sourceEventSeq alone.
func parseManualCompactionFact(raw json.RawMessage) (manualCompactionFact, bool) {
	fact, ok := parseCompactionFact(raw)
	return fact, ok && strings.TrimSpace(fact.commandID) != ""
}

func parseCompactionFact(raw json.RawMessage) (manualCompactionFact, bool) {
	if len(raw) == 0 || !bytes.Contains(raw, []byte("compactionId")) {
		return manualCompactionFact{}, false
	}
	var event struct {
		Type      string  `json:"type"`
		Seq       float64 `json:"seq"`
		Time      float64 `json:"time"`
		SurfaceOp struct {
			Op string `json:"op"`
		} `json:"surfaceOp"`
		Data struct {
			CompactionID    string `json:"compactionId"`
			SourceCommandID string `json:"sourceCommandId"`
			Source          struct {
				Kind            string `json:"kind"`
				CompactionID    string `json:"compactionId"`
				SourceCommandID string `json:"sourceCommandId"`
			} `json:"source"`
			Summary []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"summary"`
			ShadowedSeqs       []float64 `json:"shadowedSeqs"`
			ShadowedTokenCount *float64  `json:"shadowedTokenCount"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil || !validCompactionSeq(event.Seq) {
		return manualCompactionFact{}, false
	}
	fact := manualCompactionFact{
		kind: event.Type, seq: event.Seq, time: event.Time,
		commandID: event.Data.SourceCommandID, compactionID: event.Data.CompactionID,
	}
	switch event.Type {
	case "user/message":
		source := event.Data.Source
		if event.SurfaceOp.Op != "replace" || source.Kind != "compact-checkpoint" {
			return manualCompactionFact{}, false
		}
		fact.kind = "checkpoint"
		fact.commandID, fact.compactionID = source.SourceCommandID, source.CompactionID
	case "compaction/start", "compaction/end":
	case "compaction/summary":
		var text strings.Builder
		for _, block := range event.Data.Summary {
			if block.Type == "text" {
				text.WriteString(block.Text)
			}
		}
		if strings.TrimSpace(text.String()) != "" {
			fact.summary = text.String()
		}
		valid := event.Data.ShadowedSeqs != nil
		for _, seq := range event.Data.ShadowedSeqs {
			valid = valid && validCompactionSeq(seq)
		}
		if valid {
			count := len(event.Data.ShadowedSeqs)
			fact.itemCount = &count
		}
		if n := event.Data.ShadowedTokenCount; n != nil && validCompactionSeq(*n) {
			count := int(*n)
			fact.tokenCount = &count
		}
	default:
		return manualCompactionFact{}, false
	}
	if strings.TrimSpace(fact.compactionID) == "" {
		return manualCompactionFact{}, false
	}
	return fact, true
}

func classifyManualCompactionCheckpoint(raw json.RawMessage) (Message, bool) {
	fact, ok := parseManualCompactionFact(raw)
	if !ok || fact.kind != "checkpoint" {
		return Message{}, false
	}
	return Message{
		Kind: MessageKindCompaction, Role: "system", Seq: fact.seq,
		EventType: "user/message", EventTime: fact.time,
		Raw: append(json.RawMessage(nil), raw...),
	}, true
}

// projectManualCompactionMessages consumes only correlated lifecycle evidence.
// The visible marker sits at its checkpoint; shadowed transcript rows and the
// journal used by paging remain untouched. Truncated history may supply the
// checkpoint before either command record, in which case identity is retained
// and the outcome remains unknown/running until command/done is loaded.
func projectManualCompactionMessages(messages []Message) []Message {
	type locatedFact struct {
		index int
		fact  manualCompactionFact
	}
	groups := make(map[string][]locatedFact)
	commandRows := make(map[string]int)
	for i, msg := range messages {
		if msg.Kind == MessageKindCommand && msg.Command != nil {
			commandRows[msg.Command.CommandID] = i
		}
		if (msg.Kind == "" || msg.Kind == MessageKindCompaction) && len(msg.Raw) != 0 {
			if fact, ok := parseManualCompactionFact(msg.Raw); ok {
				groups[fact.commandID] = append(groups[fact.commandID], locatedFact{i, fact})
			}
		}
	}
	if len(groups) == 0 {
		return messages
	}
	hidden := make(map[int]bool)
	replacements := make(map[int]Message)
	for id, facts := range groups {
		commandIndex, hasCommand := commandRows[id]
		var command CommandView
		if hasCommand {
			command = *messages[commandIndex].Command
		} else {
			command = CommandView{CommandID: id, Name: "compact", State: "running"}
		}
		// A referenced summary is usable only with this command's explicit
		// identity. A missing/mismatched source reference never borrows a nearby
		// summary from another transaction.
		var cited *locatedFact
		if command.State == "success" && command.SourceEventSeq != nil {
			for i := range facts {
				if facts[i].fact.kind == "compaction/summary" && facts[i].fact.seq == *command.SourceEventSeq {
					cited = &facts[i]
					break
				}
			}
		}
		checkpoint := -1
		for i := range facts {
			if facts[i].fact.kind != "checkpoint" || (cited != nil && facts[i].fact.compactionID != cited.fact.compactionID) {
				continue
			}
			if checkpoint < 0 || facts[i].fact.seq > facts[checkpoint].fact.seq {
				checkpoint = i
			}
		}
		// All these are log-only records owned by this one command. Without
		// its command or checkpoint they are evidence, not a completed UI card.
		for _, located := range facts {
			hidden[located.index] = true
		}
		if checkpoint < 0 {
			continue // Keep an existing generic result, including no-op/error text.
		}
		cp := facts[checkpoint]
		if command.Name == "" {
			command.Name = "compact"
		}
		if command.Name != "compact" {
			continue // A different command keeps its complete generic outcome.
		}
		compact := &ManualCompactionView{CompactionID: cp.fact.compactionID, CheckpointSeq: cp.fact.seq}
		var summary *locatedFact
		if command.State == "success" && command.SourceEventSeq != nil {
			if cited != nil && cited.fact.compactionID == cp.fact.compactionID && cited.fact.seq < cp.fact.seq {
				summary = cited
			}
		} else {
			for i := range facts {
				if facts[i].fact.kind == "compaction/summary" && facts[i].fact.compactionID == cp.fact.compactionID && facts[i].fact.seq < cp.fact.seq &&
					(summary == nil || facts[i].fact.seq > summary.fact.seq) {
					summary = &facts[i]
				}
			}
		}
		if summary != nil {
			seq := summary.fact.seq
			compact.SummaryEventSeq = &seq
			compact.Summary = summary.fact.summary
			compact.ShadowedItemCount, compact.ShadowedTokenCount = summary.fact.itemCount, summary.fact.tokenCount
		}
		command.Compaction = compact
		marker := messages[cp.index]
		marker.Kind, marker.Role = MessageKindCommand, "command"
		marker.ID, marker.Command, marker.Body = "command:"+id, &command, command.Text
		if hasCommand {
			hidden[commandIndex] = true
		}
		replacements[cp.index] = marker
	}
	out := make([]Message, 0, len(messages))
	for i, msg := range messages {
		if replacement, ok := replacements[i]; ok {
			out = append(out, replacement)
		} else if !hidden[i] {
			out = append(out, msg)
		}
	}
	return out
}
