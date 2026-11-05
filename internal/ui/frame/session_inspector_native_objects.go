package frame

import (
	"encoding/json"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

// These references describe loaded native objects. They are navigation values,
// not serialized stand-ins for web Nodes, React values, or missing history.
type inspectorReference struct{ Key string }
type inspectorValueField struct {
	Name  string
	Value any
}
type inspectorTranscriptGroup struct {
	Header  *session.Message
	Members []inspectorReference
}
type inspectorLoadedTurn struct {
	Turn       float64
	Start, End json.RawMessage
	Steps      []inspectorReference
	Events     map[float64]json.RawMessage
}
type inspectorLoadedStep struct {
	Turn, Step float64
	Start, End json.RawMessage
	Events     map[float64]json.RawMessage
}
type inspectorLoadedLocationData struct {
	Start, End json.RawMessage
	Events     map[float64]json.RawMessage
}

func inspectorObjectJSON(value any) (json.RawMessage, string) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err.Error()
	}
	return raw, ""
}

func inspectorEventData(raw json.RawMessage) json.RawMessage {
	for _, field := range inspectorFields(raw) {
		if field.name == "data" {
			return field.raw
		}
	}
	return nil
}

func inspectorRecordTarget(row session.InspectorRecord) session.InspectorChatTarget {
	target := row.ChatTarget
	var event struct {
		Seq  *float64
		Data struct{ Turn, Step *float64 }
	}
	if json.Unmarshal(row.Raw, &event) == nil {
		if target.AnchorSeq == nil {
			target.AnchorSeq = event.Seq
		}
		if target.Turn == nil {
			target.Turn = event.Data.Turn
		}
		if target.Step == nil {
			target.Step = event.Data.Step
		}
	}
	return target
}

func inspectorNativeObjects(source session.InspectorSnapshot, navigation *chat.InspectorNavigation) map[string]inspectorObject {
	objects := inspectorObjects(source)
	for _, row := range source.Chat {
		o := objects[row.Key]
		o.Kind, o.Identity, o.RowKey, o.Target, o.Value = "data", inspectorRecordIdentity(row), row.Key, row.ChatTarget, row.Raw
		objects[row.Key] = o
	}
	turns, steps := map[string]*inspectorLoadedTurn{}, map[string]*inspectorLoadedStep{}
	eventBySeq := map[float64]string{}
	turnTarget, stepTarget := map[string]session.InspectorChatTarget{}, map[string]session.InspectorChatTarget{}
	recordTargets := inspectorRecordTargets(source)
	for _, row := range source.Records {
		o := objects[row.Key]
		o.Key, o.Label, o.Kind, o.Identity, o.RowKey = row.Key, row.Type, "event", inspectorRecordIdentity(row), row.Key
		o.Raw, o.Value, o.Target = row.Raw, row.Raw, recordTargets[row.Key]
		objects[row.Key] = o
		if raw := inspectorEventData(row.Raw); len(raw) > 0 {
			objects[row.Key+"/data"] = inspectorObject{Key: row.Key + "/data", Label: "Event data", Kind: "eventData", Identity: o.Identity, RowKey: row.Key, Raw: raw, Value: raw, Target: o.Target}
			o.Fields = []inspectorValueField{{"Data", inspectorReference{row.Key + "/data"}}}
			objects[row.Key] = o
		}
		if strings.Contains(row.Key, "/chunk:") {
			continue
		}
		target := o.Target
		if target.AnchorSeq != nil {
			eventBySeq[*target.AnchorSeq] = row.Key
		}
		if target.Turn == nil {
			continue
		}
		turnKey := "turn:" + inspectorCoordinate(*target.Turn)
		if row.Type == "turn/start" {
			turns[turnKey] = &inspectorLoadedTurn{Turn: *target.Turn, Start: row.Raw, Events: map[float64]json.RawMessage{}}
			turnTarget[turnKey] = session.InspectorChatTarget{Turn: target.Turn, AnchorSeq: target.AnchorSeq}
		}
		turn := turns[turnKey]
		if turn != nil {
			turn.Events[row.Seq] = row.Raw
			if row.Type == "turn/end" {
				turn.End = row.Raw
			}
		}
		if target.Step == nil {
			continue
		}
		stepKey := "step:" + inspectorTargetLocation(target)
		if row.Type == "step/start" {
			steps[stepKey] = &inspectorLoadedStep{Turn: *target.Turn, Step: *target.Step, Start: row.Raw, Events: map[float64]json.RawMessage{}}
			stepTarget[stepKey] = session.InspectorChatTarget{Turn: target.Turn, Step: target.Step, AnchorSeq: target.AnchorSeq}
			if turn != nil {
				turn.Steps = append(turn.Steps, inspectorReference{stepKey})
			}
		}
		if step := steps[stepKey]; step != nil {
			step.Events[row.Seq] = row.Raw
			if row.Type == "step/end" {
				step.End = row.Raw
			}
		}
	}
	addLocation := func(key, kind, label string, value any, data inspectorLoadedLocationData, target session.InspectorChatTarget) {
		dataKey := key + "/data"
		identity := strings.TrimPrefix(key, kind+":")
		raw, failure := inspectorObjectJSON(value)
		objects[key] = inspectorObject{Key: key, Kind: kind, Label: label, Identity: identity, Raw: raw, JSONError: failure, Value: value, Target: target, Links: []string{dataKey}, Fields: []inspectorValueField{{"Data", inspectorReference{dataKey}}}}
		raw, failure = inspectorObjectJSON(data)
		objects[dataKey] = inspectorObject{Key: dataKey, Kind: kind + "Data", Label: label + " data", Identity: identity, Raw: raw, JSONError: failure, Value: data, Target: target}
	}
	for key, turn := range turns {
		addLocation(key, "turn", "Loaded Turn", turn, inspectorLoadedLocationData{inspectorEventData(turn.Start), inspectorEventData(turn.End), turn.Events}, turnTarget[key])
	}
	for key, step := range steps {
		addLocation(key, "step", "Loaded Step", step, inspectorLoadedLocationData{inspectorEventData(step.Start), inspectorEventData(step.End), step.Events}, stepTarget[key])
		o := objects[key]
		parent := "turn:" + inspectorCoordinate(step.Turn)
		if _, ok := objects[parent]; ok {
			o.Fields = append(o.Fields, inspectorValueField{"Parent Turn", inspectorReference{parent}})
			o.Links = append(o.Links, parent)
		}
		objects[key] = o
	}
	messages := inspectorChatMessages(source, navigation)
	messageTargets := inspectorMessageTargets(source, messages)
	for i := range messages {
		message := &messages[i]
		key, target := "chat:"+message.ID, messageTargets[message.ID]
		o := objects[key]
		o.Key, o.Label, o.Kind, o.Identity, o.RowKey = key, "Message", "node", message.ID, key
		o.Raw, o.JSONError = inspectorObjectJSON(message)
		o.Value, o.Target = message, target
		dataKey := key + "/data"
		data := inspectorEventData(message.Raw)
		if len(data) == 0 {
			data = message.ContentBlock
		}
		objects[dataKey] = inspectorObject{Key: dataKey, Label: "Message data", Kind: "nodeData", Identity: message.ID, RowKey: key, Raw: data, Value: data, Target: target}
		o.Fields = []inspectorValueField{{"Data", inspectorReference{dataKey}}}
		if target.AnchorSeq != nil {
			if event, loaded := objects[eventBySeq[*target.AnchorSeq]]; loaded {
				o.Fields = append(o.Fields, inspectorValueField{"Source event", inspectorReference{event.Key}})
			}
		}
		o.Links = append(o.Links, dataKey)
		if target.Turn != nil {
			turnKey := "turn:" + inspectorCoordinate(*target.Turn)
			if _, ok := objects[turnKey]; ok {
				o.Fields = append(o.Fields, inspectorValueField{"Turn", inspectorReference{turnKey}})
				o.Links = append(o.Links, turnKey)
			}
			if target.Step != nil {
				stepKey := "step:" + inspectorTargetLocation(target)
				if _, ok := objects[stepKey]; ok {
					o.Fields = append(o.Fields, inspectorValueField{"Step", inspectorReference{stepKey}})
					o.Links = append(o.Links, stepKey)
				}
			}
		}
		objects[key] = o
	}
	for key, members := range navigation.Groups {
		groupKey := "chat:" + key
		o, ok := objects[groupKey]
		if !ok {
			continue
		}
		header, _ := o.Value.(*session.Message)
		group := inspectorTranscriptGroup{Header: header}
		for _, id := range members {
			childKey := "chat:" + id
			if child, loaded := objects[childKey]; loaded {
				group.Members = append(group.Members, inspectorReference{childKey})
				o.Links = append(o.Links, childKey)
				child.Fields = append(child.Fields, inspectorValueField{"Group " + key, inspectorReference{groupKey}})
				child.Links = append(child.Links, groupKey)
				objects[childKey] = child
			}
		}
		o.Kind, o.Label, o.Value = "group", "Transcript Group", group
		o.Target = session.InspectorChatTarget{GroupKey: key}
		o.Raw, o.JSONError = inspectorObjectJSON(group)
		dataKey := groupKey + "/data"
		raw, _ := json.Marshal(members)
		objects[dataKey] = inspectorObject{Key: dataKey, Label: "Group data", Kind: "groupData", Identity: key, RowKey: groupKey, Raw: raw, Value: members, Target: o.Target}
		o.Fields = []inspectorValueField{{"Data", inspectorReference{dataKey}}}
		objects[groupKey] = o
	}
	for _, row := range source.Chat {
		if _, exists := objects[row.Key]; exists || row.ChatTarget.ID == "" {
			continue
		}
		if object, exists := objects["chat:"+row.ChatTarget.ID]; exists {
			object.Key, object.RowKey = row.Key, row.Key
			objects[row.Key] = object
		}
	}
	return objects
}
