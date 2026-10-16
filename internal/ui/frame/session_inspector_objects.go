package frame

import (
	"encoding/json"
	"fmt"
	"whaleisle/internal/store/session"
)

type inspectorObject struct {
	Key, Label string
	Raw        json.RawMessage
	Links      []string
}

// The graph contains actual native Message objects and loaded event Data.
// It never fabricates React Nodes, missing Turns, or unloaded Step objects.
func inspectorObjects(source session.InspectorSnapshot) map[string]inspectorObject {
	objects := map[string]inspectorObject{}
	eventBySeq := map[float64]string{}
	turnAt := map[float64]string{}
	stepAt := map[float64]string{}
	var turn, step string
	for _, row := range source.Records {
		if row.Parent != "" && row.Seq > 0 && row.Key != fmt.Sprintf("event:%.0f", row.Seq) {
			continue
		}
		if row.Type == "turn/start" {
			turn = row.Key
			step = ""
		}
		if row.Type == "step/start" {
			step = row.Key
		}
		if row.Seq > 0 {
			eventBySeq[row.Seq] = row.Key
			turnAt[row.Seq] = turn
			stepAt[row.Seq] = step
		}
		o := inspectorObject{Key: row.Key, Label: row.Type, Raw: row.Raw}
		var event map[string]json.RawMessage
		if json.Unmarshal(row.Raw, &event) == nil && len(event["data"]) > 0 {
			key := row.Key + "/data"
			objects[key] = inspectorObject{Key: key, Label: "Data", Raw: event["data"]}
			o.Links = append(o.Links, key)
		}
		if turn != "" && turn != row.Key {
			o.Links = append(o.Links, turn)
		}
		if step != "" && step != row.Key {
			o.Links = append(o.Links, step)
		}
		objects[row.Key] = o
		if row.Type == "step/end" {
			step = ""
		}
		if row.Type == "turn/end" {
			turn = ""
			step = ""
		}
	}
	for _, row := range source.Chat {
		o := inspectorObject{Key: row.Key, Label: row.Type, Raw: row.Raw}
		if key := eventBySeq[row.Seq]; key != "" {
			o.Links = append(o.Links, key)
		}
		if key := turnAt[row.Seq]; key != "" {
			o.Links = append(o.Links, key)
		}
		if key := stepAt[row.Seq]; key != "" {
			o.Links = append(o.Links, key)
		}
		objects[row.Key] = o
	}
	return objects
}
