package session

import (
	"encoding/json"
	"strings"
)

func parseAutomaticCompactionFact(raw json.RawMessage) (manualCompactionFact, bool) {
	fact, ok := parseCompactionFact(raw)
	return fact, ok && strings.TrimSpace(fact.commandID) == ""
}

func classifyAutomaticCompactionCheckpoint(raw json.RawMessage) (Message, bool) {
	fact, ok := parseAutomaticCompactionFact(raw)
	if !ok || fact.kind != "checkpoint" {
		return Message{}, false
	}
	return Message{Kind: MessageKindCompaction, Role: "system", Seq: fact.seq, EventTime: fact.time, EventType: "user/message", Raw: append(json.RawMessage(nil), raw...)}, true
}

// Automatic lifecycle records are log evidence until replacement lands. Identity
// is compactionId, never proximity; manual command-owned events cannot join.
func projectAutomaticCompactionMessages(messages []Message) []Message {
	type group struct {
		checkpoint    int
		checkpointSeq float64
		summary       *manualCompactionFact
	}
	groups := map[string]*group{}
	facts := map[int]manualCompactionFact{}
	for i, msg := range messages {
		if (msg.Kind != "" && msg.Kind != MessageKindCompaction) || len(msg.Raw) == 0 {
			continue
		}
		fact, ok := parseAutomaticCompactionFact(msg.Raw)
		if !ok {
			continue
		}
		facts[i] = fact
		g := groups[fact.compactionID]
		if g == nil {
			g = &group{checkpoint: -1}
			groups[fact.compactionID] = g
		}
		if fact.kind == "checkpoint" && (g.checkpoint < 0 || fact.seq > g.checkpointSeq) {
			g.checkpoint, g.checkpointSeq = i, fact.seq
		}
		if fact.kind == "compaction/summary" && (g.summary == nil || fact.seq > g.summary.seq) {
			f := fact
			g.summary = &f
		}
	}
	if len(facts) == 0 {
		return messages
	}
	out := make([]Message, 0, len(messages))
	for i, msg := range messages {
		fact, ok := facts[i]
		if !ok {
			out = append(out, msg)
			continue
		}
		g := groups[fact.compactionID]
		if g.checkpoint != i {
			continue
		}
		compact := &ManualCompactionView{CompactionID: fact.compactionID, CheckpointSeq: fact.seq}
		if summary := g.summary; summary != nil {
			seq := summary.seq
			compact.SummaryEventSeq = &seq
			compact.Summary = summary.summary
			compact.ShadowedItemCount, compact.ShadowedTokenCount = summary.itemCount, summary.tokenCount
		}
		msg.Kind, msg.Role = MessageKindCompaction, "system"
		msg.ID = "compaction:" + fact.compactionID
		msg.Body = compact.Summary
		msg.AutomaticCompaction = compact
		out = append(out, msg)
	}
	return out
}
