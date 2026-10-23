package chat

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func inspectorPickerFixture(t *testing.T) (*ui.Tester, *InspectorNavigation, *ui.ScrollState, *ui.ListState, *int, *bool) {
	t.Helper()
	n := &InspectorNavigation{SessionID: "s"}
	list, scroll := &ui.ListState{}, &ui.ScrollState{}
	clicks, show := 0, true
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}, ThemeID: theme.IDDeepseekLight}
	for i := range 12 {
		m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: fmt.Sprint(i), Seq: float64(i), Role: "user", Kind: session.MessageKindText, Body: "node " + fmt.Sprint(i)})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.TextInput(c.Key("composer"), new(string)).Label("Outside composer").Height(30).FillWidth()
			if !show {
				n.CancelPicking()
				return
			}
			prepareSessionInspector(m, m.Snapshot.Messages, nil, list)
			view := ui.List(c, list, len(m.Snapshot.Messages)).Fill().TrackScroll(scroll)
			inspectorChatViewport(c, m, view, list)
			view.Rows(func(row ui.ListRow) {
				id := m.Snapshot.Messages[row.Index].ID
				box := ui.Column(row.Context).FillWidth().Height(60).Padding(0, 20)
				inspectorChatRowGeometry(row.Context, m, m.Snapshot.Messages[row.Index], box, 0, 20, 10, 20)
				box.Children(func() {
					if ui.Button(row.Context, "node "+id).Height(30).Width(90).Clicked() {
						clicks++
					}
				})
			})
			inspectorChatPickerOverlay(c, m, theme.Default().Light, list, scroll)
			ui.Box(c).Height(60).FillWidth().Label("Composer seat")
		})
	}, 300, 330)
	t.Cleanup(n.CancelPicking)
	return tt, n, scroll, list, &clicks, &show
}

func TestInspectorPickerMaskDoesNotReflowAndBlocksUnderlyingActions(t *testing.T) {
	tt, n, _, _, clicks, _ := inspectorPickerFixture(t)
	before, _ := tt.Find("node 1")
	if err := tt.Click("Outside composer"); err != nil {
		t.Fatal(err)
	}
	n.Picking = true
	tt.Frame()
	after, _ := tt.Find("node 1")
	if before != after || tt.HasText("Pick this chat node") {
		t.Fatalf("mask changed row layout: before=%+v after=%+v", before, after)
	}
	x, y := after.X+after.W/2, after.Y+after.H/2
	tt.Move(x, y)
	if n.picker.previewID != "1" || n.picker.preview != n.reveal.rows["1"] || tt.Cursor() != ui.CursorCrosshair {
		t.Fatalf("missing clipped hover: %+v cursor=%v", n.picker, tt.Cursor())
	}
	tt.RightClickAt(x, y)
	if n.Picked != nil || !n.Picking || *clicks != 0 || len(tt.Menu()) > 0 {
		t.Fatal("secondary action escaped the mask")
	}
	tt.ClickAt(x, y)
	if n.Picked == nil || n.Picked.ID != "1" || *clicks != 0 || !tt.Focused("Outside composer") {
		t.Fatalf("pick activated or stole focus: picked=%+v clicks=%d focused=%v", n.Picked, *clicks, tt.Focused("Outside composer"))
	}
	// The frame's row matcher, not the mask, decides if this is accepted.
	n.CancelPicking()
	tt.Frame()
	tt.ClickAt(x, y)
	if *clicks != 1 {
		t.Fatal("successful teardown left underlying controls blocked")
	}
}

func TestInspectorPickerPreviewClearsOnLeaveScrollAndResize(t *testing.T) {
	tt, n, scroll, _, _, _ := inspectorPickerFixture(t)
	n.Picking = true
	tt.Frame()
	tt.Move(80, 105)
	if n.picker.previewID != "1" {
		t.Fatal("missing initial preview")
	}
	tt.Scroll(80, 105, 0, 30.5)
	if scroll.Y != 30.5 || !n.Picking || n.picker.preview.W > 0 {
		t.Fatalf("wheel not forwarded/preview retained: scroll=%+v picker=%+v", scroll, n.picker)
	}
	tt.Move(81, 105)
	if n.picker.preview.W <= 0 || n.picker.preview.Y < n.picker.area.Y {
		t.Fatal("new pointer move did not use newly clipped rows")
	}
	tt.SetSize(340, 330)
	if n.picker.preview.W > 0 || n.picker.area.W != 308 {
		t.Fatalf("resize retained preview or stale mask: %+v", n.picker)
	}
	tt.Move(80, 105)
	tt.Move(5, 105)
	if n.picker.preview.W > 0 || !n.Picking {
		t.Fatal("leaving mask should hide preview and keep gesture")
	}
}

func TestInspectorPickerEscapePrecedesComposerAndBlurCannotReleasePick(t *testing.T) {
	tt, n, _, _, _, show := inspectorPickerFixture(t)
	if err := tt.Click("Outside composer"); err != nil {
		t.Fatal(err)
	}
	for _, mods := range []ui.Modifiers{0, ui.Shift, ui.Ctrl, ui.Alt, ui.Super, ui.Alt | ui.Shift} {
		n.Picking = true
		tt.Frame()
		tt.Key(mods, ui.KeyEscape)
		if n.Picking || n.Picked != nil || !tt.Focused("Outside composer") {
			t.Fatalf("Escape did not cancel from composer focus with modifiers %v", mods)
		}
	}
	n.Picking = true
	tt.Frame()
	tt.Press(80, 105)
	if !n.picker.pressed {
		t.Fatal("mask did not latch primary press")
	}
	tt.SetFocused(false)
	if n.Picking || n.Picked != nil || n.picker.pressed {
		t.Fatal("native blur's synthetic release picked a node")
	}
	tt.SetFocused(true)
	n.Picking = true
	tt.Frame()
	*show = false
	tt.Frame()
	if n.Picking || n.Picked != nil || n.picker.preview.W > 0 {
		t.Fatal("hidden Chat retained its gesture")
	}
}

func TestInspectorPickerClearsPreviewWhenSameViewportRowMovesOrDisappears(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s", Picking: true}
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s"}}
	box := ui.Rect{X: 16, Y: 40, W: 268, H: 40}
	n.reveal.viewport = ui.Rect{W: 300, H: 300}
	n.reveal.flow = ui.Rect{W: 300, H: 300}
	n.reveal.rows = map[string]ui.Rect{"node": box}
	n.picker.area = inspectorChatPickArea(m, &n.reveal)
	n.picker.previewID, n.picker.preview = "node", box
	tt := ui.NewTester(func(c *ui.Context) { inspectorChatPickGeometry(c, m) }, 300, 300)
	if n.picker.preview != box {
		t.Fatal("unchanged row lost preview")
	}
	n.reveal.rows["node"] = ui.Rect{X: 16, Y: 60, W: 268, H: 40}
	tt.Frame()
	if n.picker.previewID != "" || n.picker.preview.H > 0 || !n.Picking {
		t.Fatal("same-area content reflow retained the stale preview")
	}
	n.picker.previewID, n.picker.preview = "node", n.reveal.rows["node"]
	delete(n.reveal.rows, "node")
	tt.Frame()
	if n.picker.previewID != "" || n.picker.preview.H > 0 {
		t.Fatal("removed row retained the stale preview")
	}
}

func TestInspectorPickerHitTestUsesOnlyPaintedOccurrences(t *testing.T) {
	tt, n, _, _, _, _ := inspectorPickerFixture(t)
	n.Picking = true
	tt.Frame()
	painted := len(n.picker.rows)
	if painted == 0 || painted >= len(n.Nodes) {
		t.Fatalf("picker retained offscreen messages: painted=%d all=%d", painted, len(n.Nodes))
	}
	// The full-history projection may be replaced independently. A gesture
	// belongs to the source occurrence actually painted in the current frame.
	n.Nodes = nil
	node, _, ok := n.pickAt(80, 105)
	if !ok || node.ID != "1" {
		t.Fatalf("painted occurrence lost its identity: %+v %v", node, ok)
	}
}

func TestInspectorPickerBindingChangeCancelsHeldGesture(t *testing.T) {
	tt, n, _, list, _, _ := inspectorPickerFixture(t)
	n.Picking = true
	tt.Frame()
	tt.Press(80, 105)
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "replacement"}}
	prepareSessionInspector(m, nil, nil, list)
	if n.Picking || n.picker.pressed || n.Picked != nil || n.picker.area.W > 0 {
		t.Fatal("replacement history inherited the old gesture")
	}
}

func TestInspectorPickerDoesNotInventTargetsInWhitespace(t *testing.T) {
	tt, n, _, _, _, _ := inspectorPickerFixture(t)
	n.Picking = true
	tt.Frame()
	// Within the flow, in the vertical spacing between message contents.
	tt.ClickAt(80, 85)
	if n.Picked != nil || !n.Picking {
		t.Fatalf("unmatched content invented a pick: %+v", n.Picked)
	}
}

type inspectorPickSurfaceProbe struct{ clip ui.Rect }

func (p *inspectorPickSurfaceProbe) Present(_, _, _, _, x, y, w, h int, _ string, _ map[string]string) error {
	p.clip = ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: float32(h)}
	return nil
}

func TestInspectorPickerClipsNativeChildUntilGestureEnds(t *testing.T) {
	n := &InspectorNavigation{}
	probe := &inspectorPickSurfaceProbe{}
	tt := ui.NewTester(func(c *ui.Context) { visualReplySurfaceSeat(c, probe, theme.IDDeepseekLight, false, 120, n) }, 300, 200)
	if probe.clip.W <= 0 {
		t.Fatal("visible native guest was hidden")
	}
	n.Picking = true
	tt.Frame()
	if probe.clip.W > 0 || probe.clip.H > 0 {
		t.Fatalf("native guest escaped pick mask: %+v", probe.clip)
	}
	n.CancelPicking()
	tt.Frame()
	if probe.clip.W <= 0 {
		t.Fatal("picker teardown did not restore guest")
	}
}

func TestInspectorProductionChatPickerUsesGroupIdentityAndKeepsLayout(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s"}
	m := Model{Inspector: n, List: &ui.ListState{}, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: []session.Message{
		{ID: "user", Role: "user", Kind: session.MessageKindText, Body: "Question"},
		{ID: "reason", Role: "assistant", Kind: session.MessageKindThinking, Body: "Details"},
		{ID: "answer", Role: "assistant", Kind: session.MessageKindText, Body: "Final answer"},
	}}, SuppressStatus: true, ThemeID: theme.IDDeepseekLight, FullColumnViewport: true, ContentWidth: 400}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 600, 450)
	before := n.reveal.rows["answer"]
	n.Picking = true
	tt.Frame()
	if before != n.reveal.rows["answer"] || tt.HasText("Pick this chat node") {
		t.Fatal("production picker changed message height")
	}
	var group string
	for id := range n.Groups {
		if n.reveal.rows[id].H > 0 {
			group = id
			break
		}
	}
	if group == "" {
		t.Fatal("production fixture did not contain a painted group")
	}
	r := n.reveal.rows[group]
	tt.Move(r.X+10, r.Y+r.H/2)
	tt.ClickAt(r.X+10, r.Y+r.H/2)
	if n.Picked == nil || n.Picked.GroupKey != group || n.Picked.ID != "" || n.Picked.AnchorSeq != nil {
		t.Fatalf("group header borrowed first member identity: %+v", n.Picked)
	}
}
