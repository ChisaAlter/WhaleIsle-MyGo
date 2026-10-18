package session

import "encoding/json"

// Only durable goal/running edges and reconnect snapshots require a live read.
// Token and assistant-stream frames never trigger goals/get.
func goalRefreshFollowFrame(raw json.RawMessage) bool {
	var frame struct {
		Type  string `json:"type"`
		Event struct {
			Type string `json:"type"`
		} `json:"event"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		return false
	}
	if frame.Type == "snapshot" {
		return true
	}
	if frame.Type != "event" {
		return false
	}
	switch frame.Event.Type {
	case "goal/change", "turn/start", "turn/end":
		return true
	}
	return false
}
