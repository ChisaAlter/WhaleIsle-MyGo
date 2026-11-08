package composer

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
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

func TestPresetSelectKeyboardNavigationOverridesStationaryHover(t *testing.T) {
	var options []presetChoice
	for i := 0; i < 12; i++ {
		options = append(options, presetChoice{ID: fmt.Sprintf("id-%d", i), Label: fmt.Sprintf("Option %02d", i)})
	}
	picked := ""
	tt := ui.NewTester(func(c *ui.Context) {
		presetSelect(c, "Option 00", options, func(id string) { picked = id }, testColors(), presetOptions{})
	}, 400, 500)
	if err := tt.Click("Option 00"); err != nil {
		t.Fatal(err)
	}
	r0, ok := tt.Find("Option 00")
	if !ok {
		t.Fatal("Option 00 not visible")
	}
	// Hover over Option 00 and keep the pointer stationary while navigating with KeyDown.
	tt.Move(r0.X+10, r0.Y+r0.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if picked != "id-2" {
		t.Fatalf("stationary hover overrode keyboard navigation: picked %q, want id-2", picked)
	}
}
