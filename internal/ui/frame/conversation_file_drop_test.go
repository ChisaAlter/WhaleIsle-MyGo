package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/composer"
)

func TestConversationFileTargetIgnoresDockDrag(t *testing.T) {
	draft := composer.NewDraftState("")
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Button(c, "Drag internal tab").Drag(workspaceTabDrag{ID: "synthetic-tab"})
			area := ui.Box(c).Width(400).Height(300)
			if composer.FileDropTarget(area, composer.Model{Draft: &draft.Text, State: draft, OnAttachFiles: func([]string) { calls++ }}) {
				t.Error("internal drag reported native hover")
			}
			area.Children(func() { ui.Text(c, "Conversation history") })
		})
	}, 500, 400)
	src, _ := tt.Find("Drag internal tab")
	tt.Press(src.X+src.W/2, src.Y+src.H/2)
	tt.Move(200, 180)
	tt.Release(200, 180)
	tt.Frame()
	if calls != 0 || draft.Text != "" {
		t.Fatal("Dock payload entered attachments")
	}
}
