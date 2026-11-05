package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestParityCanonicalComposerSlotsRetainNativeFallback(t *testing.T) {
	d := NewDraftState("native draft")
	seen := map[string]int{}
	m := Model{Draft: &d.Text, State: d, Snapshot: session.ViewState{ClientReady: true, SelectedID: "s"}, OnSubmit: func() {}, RenderSlot: func(c *ui.Context, name string, native func()) {
		seen[name]++
		if name == "conversation.input.model" {
			ui.Text(c, "Contributed model action")
			return
		}
		native()
	}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 300)
	for _, name := range []string{"conversation.composer", "conversation.composer.bar", "conversation.composer.dock", "conversation.input.dock", "conversation.input.left", "conversation.input.right", "conversation.input.activity", "conversation.input.plan", "conversation.input.permission", "conversation.input.managed", "conversation.input.attachments", "conversation.input.overlay", "conversation.input.model"} {
		if seen[name] == 0 {
			t.Fatalf("missing native seat %s", name)
		}
	}
	if _, editor := tt.Find("Composer draft"); !editor || d.Text != "native draft" || !tt.HasText("Contributed model action") {
		t.Fatalf("slot erased native draft or contribution: %q", tt.Texts())
	}
	seen = map[string]int{}
	m.RenderSlot = func(c *ui.Context, name string, native func()) {
		seen[name]++
		if name == "conversation.composer" {
			ui.Text(c, "Explicit third-party composer")
			return
		}
		native()
	}
	tt.Frame()
	_, editor := tt.Find("Composer draft")
	if !tt.HasText("Explicit third-party composer") || editor || d.Text != "native draft" || seen["conversation.input.left"] != 0 {
		t.Fatal("root chain winner did not own only its explicit replacement")
	}
}
