package frame

import (
	"github.com/egoist/mygo/ui"
	"reflect"
	"testing"
	"whaleisle/internal/store/session"
)

func TestInspectorCandidateConfirmationAndCancellation(t *testing.T) {
	rows := []session.InspectorRecord{{Type: "tool/end"}, {Type: "tool/start"}, {Type: "tool/end"}, {Type: "assistant"}}
	if got := inspectorTypeCandidates(rows, " *TOOL* "); !reflect.DeepEqual(got, []string{"tool/end", "tool/start"}) {
		t.Fatal(got)
	}
	p := SessionInspectorPaneState{Filter: "old", draft: "tool", filterOpen: true, filterActive: -1}
	candidates := inspectorTypeCandidates(rows, p.draft)
	p.moveFilterCandidate(-1, len(candidates))
	if p.filterActive != 1 || p.Filter != "old" {
		t.Fatal("Up must wrap without applying")
	}
	p.moveFilterCandidate(1, len(candidates))
	if p.filterActive != 0 {
		t.Fatal("Down must wrap")
	}
	p.confirmFilter(candidates)
	if p.Filter != "tool/end" || p.filterOpen {
		t.Fatal("Enter did not apply candidate")
	}
	p.filterOpen = true
	p.draft = "discard"
	p.cancelFilter()
	if p.Filter != "tool/end" || p.draft != "tool/end" || p.filterOpen {
		t.Fatal("cancel changed filter")
	}
	p.filterOpen = true
	p.draft = "custom"
	p.filterActive = -1
	p.confirmFilter(nil)
	if p.Filter != "custom" {
		t.Fatal("Enter lost free text")
	}
}
func TestInspectorDetailsResizeBoundsAndKeyboard(t *testing.T) {
	p := SessionInspectorPaneState{}
	if p.inspectorDetailsPercent() != 40 {
		t.Fatal("wrong default")
	}
	p.resizeInspectorDetails(-50, 500)
	if p.inspectorDetailsPercent() != 50 {
		t.Fatal("upward drag must grow details")
	}
	p.detailsKey(ui.KeyHome)
	if p.inspectorDetailsPercent() != 15 {
		t.Fatal("Home")
	}
	p.detailsKey(ui.KeyDown)
	if p.inspectorDetailsPercent() != 15 {
		t.Fatal("lower bound")
	}
	p.detailsKey(ui.KeyEnd)
	p.detailsKey(ui.KeyUp)
	if p.inspectorDetailsPercent() != 80 {
		t.Fatal("upper bound")
	}
}
func TestInspectorFilterConfirmationRevealsThenAllowsManualFold(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "parent", Type: "turn"}, {Key: "match", Parent: "parent", Type: "tool"}}
	p := SessionInspectorPaneState{collapsed: map[string]bool{"parent": true}}
	p.applyInspectorFilter(" *TOOL* ")
	if p.Filter != "TOOL" || len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 2 {
		t.Fatal("confirmation did not normalize the query and reveal matching descendants")
	}
	p.collapsed["parent"] = true
	if got := inspectorVisibleRows(rows, p.Filter, p.collapsed); len(got) != 1 || got[0].Key != "parent" {
		t.Fatal("manual fold of filtered context was ignored")
	}
}

func TestInspectorCandidatePopoverKeyboard(t *testing.T) {
	p := SessionInspectorPaneState{}
	rows := []session.InspectorRecord{{Type: "alpha"}, {Type: "beta"}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { inspectorTypeFilter(c, &p, rows) }) }, 500, 400)
	if err := tt.Click("Inspector type filter"); err != nil {
		t.Fatal(err)
	}
	if !p.filterOpen {
		t.Fatal("filter did not open")
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if p.Filter != "alpha" || p.filterOpen {
		t.Fatalf("candidate not applied: %+v", p)
	}
	if err := tt.Click("Inspector type filter"); err != nil {
		t.Fatal(err)
	}
	tt.Type("discard")
	tt.Key(0, ui.KeyEscape)
	if p.filterOpen || p.Filter != "alpha" || p.draft != "alpha" {
		t.Fatal("Escape changed confirmed filter")
	}
}
func TestInspectorDetailsDividerKeyboard(t *testing.T) {
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { inspectorDetailsDivider(c, &p, 400) }) }, 400, 300)
	if err := tt.Click("Inspector details divider"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnd)
	if p.inspectorDetailsPercent() != 80 {
		t.Fatal("End did not reach divider")
	}
	tt.Key(0, ui.KeyDown)
	if p.inspectorDetailsPercent() != 75 {
		t.Fatal("Down did not reach divider")
	}
	tt.Key(0, ui.KeyHome)
	if p.inspectorDetailsPercent() != 15 {
		t.Fatal("Home did not reach divider")
	}
}

func TestInspectorDetailsDragRetainsPointerOrigin(t *testing.T) {
	p := SessionInspectorPaneState{}
	p.dragInspectorDetails(0, 400, true)
	p.dragInspectorDetails(-400, 400, true)
	if p.inspectorDetailsPercent() != 80 {
		t.Fatal("drag clamp")
	}
	p.dragInspectorDetails(400, 800, true)
	if p.inspectorDetailsPercent() != 40 {
		t.Fatal("drag origin drifted after overshoot or resize")
	}
	p.dragInspectorDetails(0, 400, false)
	if p.detailsDragging {
		t.Fatal("drag did not release")
	}
}
