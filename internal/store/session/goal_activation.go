package session

import "encoding/json"

func goalRefreshForParsedFrame(parsed parsedFollowFrame) bool {
	if !parsed.kindOK {
		return false
	}
	if parsed.kind == "snapshot" {
		return true
	}
	if parsed.kind != "event" || !parsed.eventHeadOK {
		return false
	}
	switch parsed.eventHead.Type {
	case "goal/change", "turn/start", "turn/end":
		return true
	}
	return false
}

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
