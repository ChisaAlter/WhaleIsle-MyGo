package projectflow

import (
	"encoding/json"
	"fmt"
)

// ProcessRecordLimit bounds a long-running observer without removing the
// session/page path to older durable records.
const ProcessRecordLimit = 400

// FoldProcessFrame follows the pinned session/follow snapshot/event contract.
// It never activates an Agent or interprets event contents as instructions.
func FoldProcessFrame(current ProcessSnapshot, raw []byte) (ProcessSnapshot, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return current, err
	}
	switch head.Type {
	case "snapshot":
		next, err := ParseProcessSnapshot(raw)
		if err != nil {
			return current, err
		}
		next.Live = true
		if len(next.Records) > ProcessRecordLimit {
			next.Records = append([]ProcessRecord(nil), next.Records[len(next.Records)-ProcessRecordLimit:]...)
			next.Truncated = true
			next.HasMore = true
		}
		return next, nil
	case "event":
		if current.Type != "snapshot" {
			return current, fmt.Errorf("member event arrived before its opening snapshot")
		}
		var record ProcessRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return current, err
		}
		if record.Event.Seq <= 0 || record.Event.Type == "" {
			return current, fmt.Errorf("invalid member process event")
		}
		if record.Event.Seq <= current.Cursor {
			return current, nil
		}
		current.Records = append(append([]ProcessRecord(nil), current.Records...), record)
		current.Cursor = record.Event.Seq
		current.Live = true
		current.Error = ""
		if len(current.Records) > ProcessRecordLimit {
			current.Records = current.Records[len(current.Records)-ProcessRecordLimit:]
			current.Truncated = true
			current.HasMore = true
		}
		return current, nil
	case "assistant-stream":
		// This observer does not request assistantStream. Durable event commits
		// remain the history/pagination authority if an optional frame arrives.
		return current, nil
	default:
		return current, fmt.Errorf("unknown member process frame %q", head.Type)
	}
}
