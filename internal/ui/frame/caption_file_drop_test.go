package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestCaptionFileDropScope(t *testing.T) {
	owner := session.New()
	draft := composer.NewDraftState("")
	calls := 0
	m := Model{Store: owner, Draft: &draft.Text, ComposerState: draft, OnAttachComposerFiles: func([]string) { calls++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			MainCaptionFileDrop(c, m, func() { ui.Text(c, "Main caption") })
			ui.Button(c, "Internal dock").Drag(workspaceTabDrag{ID: "synthetic"})
		})
	}, 400, 200)
	if _, ok := tt.Find("Main caption"); !ok {
		t.Fatal("caption content missing")
	}
	src, _ := tt.Find("Internal dock")
	tt.Press(src.X+src.W/2, src.Y+src.H/2)
	tt.Move(100, 10)
	tt.Release(100, 10)
	tt.Frame()
	if calls != 0 {
		t.Fatal("internal dock reached attachment intake")
	}
}
