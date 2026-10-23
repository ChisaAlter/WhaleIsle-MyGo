package frame

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func inspectorPickAnchor(target chat.InspectorTarget) *float64 {
	if target.AnchorSeq != nil {
		return target.AnchorSeq
	}
	if target.Seq > 0 {
		return &target.Seq
	}
	return nil
}

func inspectorPickCoordinate(a, b *float64) bool { return a != nil && b != nil && *a == *b }

// A group pick in the log uses its first current member. Placement comes from
// that node; part/call selectors still belong to the picked occurrence.
func (p *SessionInspectorPaneState) inspectorLogPickTarget(target chat.InspectorTarget) chat.InspectorTarget {
	id := target.ID
	if members, ok := p.navigation.Groups[id]; ok {
		target.ID, target.GroupKey = "", id
		id = ""
		if len(members) > 0 {
			id = members[0]
		}
	} else if id == "" && target.GroupKey != "" {
		if members := p.navigation.Groups[target.GroupKey]; len(members) > 0 {
			id = members[0]
		}
	}
	for _, node := range p.navigation.Nodes {
		if id == "" || node.ID != id {
			continue
		}
		position := node.InspectorTarget()
		target.Seq, target.AnchorSeq, target.NodeKind = position.Seq, position.AnchorSeq, position.NodeKind
		if position.Turn != nil {
			target.Turn = position.Turn
		}
		if position.Step != nil {
			target.Step = position.Step
		}
		break
	}
	return target
}

func inspectorPickRowTarget(row session.InspectorRecord) chat.InspectorTarget {
	target := row.ChatTarget
	if target.ID == "" && target.GroupKey == "" && strings.HasPrefix(row.Key, "chat:") {
		target.ID = strings.TrimPrefix(row.Key, "chat:")
	}
	return target
}

func inspectorPickChatRecord(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	firstNode, exactNode, group, step, turn := "", "", "", "", ""
	for _, row := range rows {
		candidate := inspectorPickRowTarget(row)
		if target.ID != "" && candidate.ID == target.ID {
			if firstNode == "" {
				firstNode = row.Key
			}
			if exactNode == "" && candidate.GroupPart == target.GroupPart {
				exactNode = row.Key
			}
		}
		if group == "" && target.GroupKey != "" && candidate.GroupKey == target.GroupKey && candidate.ID == "" {
			group = row.Key
		}
		if inspectorPickCoordinate(target.Turn, candidate.Turn) {
			if turn == "" {
				turn = row.Key
			}
			if step == "" && inspectorPickCoordinate(target.Step, candidate.Step) {
				step = row.Key
			}
		}
	}
	for _, key := range []string{exactNode, firstNode, group, step, turn} {
		if key != "" {
			return key
		}
	}
	return ""
}

type inspectorPickLogEntry struct {
	key, kind string
	seq       *float64
	target    chat.InspectorTarget
}

// Rank the loaded source entries independently of table hierarchy and folding.
// An absent native transient sequence must not become durable event zero.
func inspectorPickLogRoot(entries []inspectorPickLogEntry, target chat.InspectorTarget) string {
	anchorSeq := inspectorPickAnchor(target)
	call, anchor, assistant, step, turn, first := "", "", "", "", "", ""
	for _, entry := range entries {
		candidate := entry.target
		sameTurn := inspectorPickCoordinate(target.Turn, candidate.Turn)
		sameStep := sameTurn && (target.Step == nil || inspectorPickCoordinate(target.Step, candidate.Step))
		if call == "" && target.CallID != "" && entry.kind == "tool/call" && candidate.CallID == target.CallID {
			call = entry.key
		}
		if anchor == "" && inspectorPickCoordinate(anchorSeq, entry.seq) {
			anchor = entry.key
		}
		if target.NodeKind == "assistant-step" && sameStep && (entry.kind == "assistant/message" || entry.kind == "assistant/attempt" || entry.kind == "assistant/live") {
			assistant = entry.key
		}
		if step == "" && target.Step != nil && entry.kind == "step/start" && sameStep {
			step = entry.key
		}
		if turn == "" && entry.kind == "turn/start" && sameTurn {
			turn = entry.key
		}
		if first == "" && sameStep {
			first = entry.key
		}
	}
	for _, key := range []string{call, anchor, assistant, step, turn, first} {
		if key != "" {
			return key
		}
	}
	return ""
}

func inspectorPickLogPart(rows []session.InspectorRecord, key, part string) string {
	if key != "" && part == "reasoning" {
		for _, row := range rows {
			if row.Parent != key || row.Type != "block-start" {
				continue
			}
			var chunk struct{ BlockType string }
			if json.Unmarshal(row.Raw, &chunk) == nil && chunk.BlockType == "reasoning" {
				return row.Key
			}
		}
	}
	return key
}

func inspectorPickLogRecord(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	entries := make([]inspectorPickLogEntry, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(row.Key, "chat:") || strings.Contains(row.Key, "/chunk:") {
			continue
		}
		var seq *float64
		if row.Type != "assistant/live" {
			value := row.Seq
			seq = &value
		}
		entries = append(entries, inspectorPickLogEntry{key: row.Key, kind: row.Type, seq: seq, target: row.ChatTarget})
	}
	return inspectorPickLogPart(rows, inspectorPickLogRoot(entries, target), target.GroupPart)
}

func inspectorPickLiveBlock(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	if target.ID == "" || inspectorPickAnchor(target) != nil {
		return ""
	}
	first := ""
	for _, row := range rows {
		if row.ChatPickID == target.ID {
			if row.Type == "block-start" {
				return row.Key
			}
			if first == "" {
				first = row.Key
			}
		}
	}
	return first
}

func (p *SessionInspectorPaneState) inspectorPreparingPick(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	if inspectorPickAnchor(target) != nil {
		return ""
	}
	id, call := target.ID, target.CallID
	if id == "" && target.GroupKey != "" {
		if members := p.navigation.Groups[target.GroupKey]; len(members) > 0 {
			id = members[0]
		}
	}
	preparing := false
	for _, node := range p.navigation.Nodes {
		if node.ID == id && node.Tool != nil && node.Tool.State == "preparing" {
			if call == "" && target.ID == "" {
				call = node.Tool.CallID
			}
			preparing = call != "" && node.Tool.CallID == call
			break
		}
	}
	if !preparing {
		return ""
	}
	roots := map[string]bool{}
	for _, row := range rows {
		if !strings.Contains(row.Key, "/chunk:") && !strings.HasPrefix(row.Key, "chat:") {
			roots[row.Key] = true
		}
		if row.Type == "tool/call" && row.ChatTarget.CallID == call {
			return row.Key
		}
	}
	for _, row := range rows {
		if row.ChatTarget.CallID != call {
			continue
		}
		if root, _, ok := strings.Cut(row.Key, "/chunk:"); ok && roots[root] {
			return root
		}
	}
	return ""
}

func (p *SessionInspectorPaneState) inspectorPickRecord(rows []session.InspectorRecord, target chat.InspectorTarget) string {
	if p.Mode == "chat-node" {
		if _, ok := p.navigation.Groups[target.ID]; ok {
			target.ID, target.GroupKey = "", target.ID
		}
		return inspectorPickChatRecord(rows, target)
	}
	target = p.inspectorLogPickTarget(target)
	// Native live blocks have process-local identities instead of the original
	// transient sequence. Their actual stream header remains more specific than
	// the Step fallback. ChatID-only forward aliases cannot reverse-pick a block.
	if key := inspectorPickLiveBlock(rows, target); key != "" {
		return key
	}
	if key := p.inspectorPreparingPick(rows, target); key != "" {
		return key
	}
	return inspectorPickLogRecord(rows, target)
}
