package settings

import (
	"github.com/egoist/mygo/ui"
	"testing"
	settingsstore "whaleisle/internal/store/settings"
)

func TestSettingsModalReturnsWorkspaceFocus(t *testing.T) {
	open := false
	closed, background := 0, 0
	store := settingsstore.New()
	draft := "preserved workspace draft"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if ui.Button(c, "Open settings").Clicked() {
				open = true
			}
			if ui.Button(c, "Workspace action").Clicked() {
				background++
			}
			ui.TextInput(c, &draft).Label("Workspace draft")
		})
		if open {
			ModalView(c, Model{Store: store, OnClose: func() { closed++; open = false }})
		}
	}, 1000, 760)
	tt.Click("Open settings")
	tt.Key(0, ui.KeyEscape)
	if open || closed != 1 || !tt.Focused("Open settings") {
		t.Fatalf("dismissal did not restore opener: open=%v closed=%d", open, closed)
	}
	if draft != "preserved workspace draft" || background != 0 {
		t.Fatal("underlying workspace changed")
	}
}
func TestSettingsModalFitsViewport(t *testing.T) {
	for _, size := range [][2]float32{{1400, 1000}, {600, 400}, {20, 20}} {
		w, h := settingsModalSize(size[0], size[1])
		if w < 0 || h < 0 || w > size[0] || h > size[1] {
			t.Fatalf("invalid modal dimensions %v %v", w, h)
		}
	}
}
