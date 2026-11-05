package frame

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func inspectorCoordinate(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func inspectorTargetLocation(target session.InspectorChatTarget) string {
	if target.Turn == nil {
		return ""
	}
	location := inspectorCoordinate(*target.Turn)
	if target.Step != nil {
		location += "/" + inspectorCoordinate(*target.Step)
	}
	return location
}

// Coordinate-free events and generated presentation rows still belong to real
// loaded lifecycle boundaries. Missing boundaries remain unknown; sequence zero
// is used only when the source explicitly supplies an anchor.
func inspectorRecordTargets(source session.InspectorSnapshot) map[string]session.InspectorChatTarget {
	targets := map[string]session.InspectorChatTarget{}
	var turn, step *float64
	for _, row := range source.Records {
		target := inspectorRecordTarget(row)
		explicitTurn := target.Turn != nil
		switch row.Type {
		case "turn/start":
			turn, step = target.Turn, nil
		case "step/start":
			if target.Turn != nil {
				turn = target.Turn
			}
			step = target.Step
		}
		if target.Turn == nil {
			target.Turn = turn
		}
		if target.Step == nil && !explicitTurn && row.Type != "turn/start" && row.Type != "turn/end" && inspectorPickCoordinate(target.Turn, turn) {
			target.Step = step
		}
		targets[row.Key] = target
		if row.Type == "step/end" {
			step = nil
		}
		if row.Type == "turn/end" {
			turn, step = nil, nil
		}
	}
	return targets
}

func inspectorMessageTargets(source session.InspectorSnapshot, messages []session.Message) map[string]session.InspectorChatTarget {
	bySequence := map[float64]session.InspectorChatTarget{}
	byCall := map[string]session.InspectorChatTarget{}
	records := inspectorRecordTargets(source)
	for _, row := range source.Records {
		target := records[row.Key]
		if target.CallID != "" {
			byCall[target.CallID] = target
		}
		if target.AnchorSeq != nil && !strings.Contains(row.Key, "/chunk:") {
			bySequence[*target.AnchorSeq] = target
		}
	}
	targets := map[string]session.InspectorChatTarget{}
	for _, message := range messages {
		target := message.InspectorTarget()
		if target.Turn == nil && target.AnchorSeq != nil {
			position := bySequence[*target.AnchorSeq]
			target.Turn, target.Step = position.Turn, position.Step
		}
		if target.Turn == nil && message.TurnSeq > 0 {
			target.Turn = bySequence[message.TurnSeq].Turn
		}
		if message.Streaming && message.Tool != nil && message.Tool.State == "preparing" {
			if position, known := byCall[message.Tool.CallID]; known {
				if target.Turn == nil {
					target.Turn = position.Turn
				}
				if target.Step == nil && inspectorPickCoordinate(target.Turn, position.Turn) {
					target.Step = position.Step
				}
			}
		}
		targets[message.ID] = target
	}
	return targets
}

// Inspect the exact projection supplied by Chat, including folded headers. The
// store-only fallback uses its actual Messages until Chat publishes its groups.
func inspectorChatMessages(source session.InspectorSnapshot, navigation *chat.InspectorNavigation) []session.Message {
	if len(navigation.Nodes) > 0 {
		return navigation.Nodes
	}
	messages := make([]session.Message, 0, len(source.Chat))
	for _, row := range source.Chat {
		var message session.Message
		if json.Unmarshal(row.Raw, &message) == nil && message.ID != "" {
			messages = append(messages, message)
		}
	}
	return messages
}

// Group membership determines order and Parent, not the header's presentation
// sequence. Ungrouped messages retain the loaded Chat order around each group.
func inspectorChatRows(source session.InspectorSnapshot, navigation *chat.InspectorNavigation) []session.InspectorRecord {
	messages := inspectorChatMessages(source, navigation)
	if len(messages) == 0 {
		return source.Chat
	}
	byID, ordinal := map[string]session.Message{}, map[string]int{}
	targets := inspectorMessageTargets(source, messages)
	for i, message := range messages {
		byID[message.ID], ordinal[message.ID] = message, i
	}
	if len(navigation.Nodes) == 0 {
		rows := append([]session.InspectorRecord(nil), source.Chat...)
		for i := range rows {
			var message session.Message
			if json.Unmarshal(rows[i].Raw, &message) != nil || message.ID == "" {
				continue
			}
			rows[i].Identity, rows[i].ChatTarget = message.ID, targets[message.ID]
			if location := inspectorTargetLocation(rows[i].ChatTarget); location != "" {
				rows[i].Location = location
			}
			if rows[i].Type == "" {
				rows[i].Type = string(message.Kind)
			}
		}
		return rows
	}
	type groupPosition struct {
		key string
		at  int
	}
	var groups []groupPosition
	grouped := map[string]bool{}
	for key, members := range navigation.Groups {
		at := len(messages)
		for _, id := range members {
			if index, ok := ordinal[id]; ok {
				at = min(at, index)
				grouped[id] = true
			}
		}
		if _, loaded := byID[key]; loaded {
			groups = append(groups, groupPosition{key, at})
			grouped[key] = true
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].at != groups[j].at {
			return groups[i].at < groups[j].at
		}
		return groups[i].key < groups[j].key
	})
	var rows []session.InspectorRecord
	emitted := map[string]int{}
	appendMessage := func(message session.Message, parent string) {
		raw, _ := json.Marshal(message)
		target := targets[message.ID]
		row := session.InspectorRecord{Key: "chat:" + message.ID, Parent: parent, Type: string(message.Kind), Identity: message.ID, Location: inspectorTargetLocation(target), Seq: message.Seq, Raw: raw, ChatID: message.ID, ChatTarget: target}
		if emitted[message.ID] > 0 {
			row.Key += "/member:" + strconv.Itoa(emitted[message.ID])
		}
		emitted[message.ID]++
		rows = append(rows, row)
	}
	next := 0
	appendUngroupedBefore := func(at int) {
		for next < len(messages) && next < at {
			message := messages[next]
			if !grouped[message.ID] {
				appendMessage(message, "")
			}
			next++
		}
	}
	for _, group := range groups {
		appendUngroupedBefore(group.at)
		message := byID[group.key]
		raw, _ := json.Marshal(message)
		key := "chat:" + group.key
		rows = append(rows, session.InspectorRecord{Key: key, Type: "group", Identity: group.key, Raw: raw, Disclosure: "open", ChatTarget: navigation.PickTarget(message)})
		for _, id := range navigation.Groups[group.key] {
			if member, ok := byID[id]; ok {
				appendMessage(member, key)
			}
		}
	}
	appendUngroupedBefore(len(messages))
	return rows
}
