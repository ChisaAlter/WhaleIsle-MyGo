package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// accessRows returns the tree's options of a select popup, in order.
func accessRows(tree *platform.AccessTree) []platform.AccessNode {
	var rows []platform.AccessNode
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleRow || n.Role == platform.RoleListItem {
			rows = append(rows, n)
		}
	}
	return rows
}

// The API 协议 field of the provider card is a Select, and a card a keyboard or
// screen-reader user cannot fill in is a card they cannot use: the tree must
// carry the current value of the closed select, the expand pattern must open
// it, and each option must report whether it is the chosen one.
func TestSelectCarriesItsValueAndOpensByExpand(t *testing.T) {
	options := []string{"OpenAI Chat Completions", "OpenAI Responses", "Anthropic Messages"}
	picked := options[0]
	tt := coreNewTester(func(c *context) {
		coreSelect(c, &picked, options)
	}, 420, 420)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})

	trigger := accessNode(t, tt.h.access, platform.RolePopUpButton, "")
	if trigger.Value != options[0] {
		t.Fatalf("the closed select reports value %q, want %q", trigger.Value, options[0])
	}

	// Expanding is the opening gesture a client has: a pop-up button exposes
	// the expand pattern, not an invoke one.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: trigger.ID, Action: platform.AccessExpand})
	tt.Frame()
	rows := accessRows(tt.h.access)
	if len(rows) != len(options) {
		t.Fatalf("expanding the select shows %d options, want %d: %+v", len(rows), len(options), tt.Texts())
	}
	chosen := 0
	for i, row := range rows {
		if row.States&platform.AccessSelectable == 0 {
			t.Errorf("option %q is not selectable", row.Label)
		}
		if row.States&platform.AccessChecked != 0 {
			chosen++
			if row.Label != options[0] {
				t.Errorf("the chosen option is %q, want %q", row.Label, options[0])
			}
			if i != 0 {
				t.Errorf("the chosen option is at %d", i)
			}
		}
	}
	if chosen != 1 {
		t.Fatalf("%d options report themselves chosen, want 1", chosen)
	}

	// Choosing another option is a press on it, the act a selectable option's
	// selection pattern performs.
	for _, row := range rows {
		if row.Label == options[1] {
			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: row.ID, Action: platform.AccessPress})
		}
	}
	tt.Frame()
	if picked != options[1] {
		t.Fatalf("pressing the second option chose %q, want %q", picked, options[1])
	}
	trigger = accessNode(t, tt.h.access, platform.RolePopUpButton, "")
	if trigger.Value != options[1] {
		t.Errorf("the closed select reports %q after the choice, want %q", trigger.Value, options[1])
	}
}
