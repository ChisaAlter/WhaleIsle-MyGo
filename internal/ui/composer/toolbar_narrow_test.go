package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// The InputBar keeps one toolbar row at every width: at the width the composer
// has beside an open right pane, the permission chip stays on the attach row
// and the model chip shrinks instead of sliding under the send button.
func TestComposerToolbarStaysOneRowWhenNarrow(t *testing.T) {
	m := permissionModelFixture()
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 400, 420)
	permission, ok := tt.Find(i18n.T(i18n.KeyComposerPermissionWorkspace))
	model, okModel := tt.Find("Model catalog")
	send, okSend := tt.Find(i18n.T(i18n.KeyComposerSend))
	if !ok || !okModel || !okSend {
		t.Fatalf("toolbar seats missing: permission=%v model=%v send=%v", ok, okModel, okSend)
	}
	mid := func(r ui.Rect) float32 { return r.Y + r.H/2 }
	if d := mid(permission) - mid(send); d > 4 || d < -4 {
		t.Fatalf("permission chip left the toolbar row: permission=%+v send=%+v", permission, send)
	}
	if d := mid(model) - mid(send); d > 4 || d < -4 {
		t.Fatalf("model chip left the toolbar row: model=%+v send=%+v", model, send)
	}
	if model.X+model.W > send.X {
		t.Fatalf("model chip runs under the send button: model=%+v send=%+v", model, send)
	}
	if permission.X+permission.W > model.X {
		t.Fatalf("permission chip overlaps the model chip: permission=%+v model=%+v", permission, model)
	}
}
