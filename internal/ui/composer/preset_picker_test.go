package composer

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestPresetPickerCaptionsDoNotReplaceIDs(t *testing.T) {
	p := permissionChoices([]session.PermissionOption{{Label: "Same", Value: "a"}, {Label: "Same", Value: "b"}})
	a := agentPresetChoices([]session.AgentPresetOption{{Label: "Same", ID: "a"}, {Label: "Same", ID: "b"}})
	for _, rows := range [][]presetChoice{p, a} {
		if len(rows) != 2 || rows[0].ID != "a" || rows[1].ID != "b" || rows[0].Label != rows[1].Label {
			t.Fatalf("lost identity: %+v", rows)
		}
	}
}
