package pluginui

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/egoist/mygo/ui"
)

// RenderNativeInteraction never substitutes a different pending request simply
// because it uses the same slot. Anonymous plan reviews additionally require
// the source's current single pending key and the complete question schema.
func (o *Owner) RenderNativeInteraction(c *ui.Context, slot, agentID, sourceRequestID, callID string) bool {
	if agentID == "" || sourceRequestID == "" && callID == "" {
		return false
	}
	s := o.Snapshot()
	var matched []SessionOutlet
	for _, row := range s.Outlets {
		if row.Slot != slot || row.AgentID != agentID {
			continue
		}
		if strings.HasPrefix(sourceRequestID, "questions:") {
			if slot != "conversation.plan-review.actions" || row.PendingKey == "" || !strings.HasPrefix(row.SourceRequestID, "questions:") {
				continue
			}
			var expected, actual any
			if json.Unmarshal([]byte(strings.TrimPrefix(sourceRequestID, "questions:")), &expected) != nil || json.Unmarshal([]byte(strings.TrimPrefix(row.SourceRequestID, "questions:")), &actual) != nil || !reflect.DeepEqual(expected, actual) {
				continue
			}
		} else if sourceRequestID != "" {
			if row.SourceRequestID != sourceRequestID {
				continue
			}
		} else if row.CallID != callID {
			continue
		}
		matched = append(matched, row)
	}
	s.Outlets = matched
	return o.renderNativeSnapshot(c, s, slot, "", -1, -1, "")
}
