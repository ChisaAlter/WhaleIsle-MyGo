package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The final-placement overlay covers the whole window to run the end-of-frame
// placement. It must only paint: when it took the pointer, every mouse press in
// the main window stopped at it and nothing could be clicked.
func TestPluginFinalPlacementOverlayDoesNotTakeThePointer(t *testing.T) {
	s := &mainState{}
	clicked := false
	tt := ui.NewTester(func(c *ui.Context) {
		if ui.Button(c, "Under the overlay").Size(160, 40).Clicked() {
			clicked = true
		}
		s.endPluginRichFrame(c)
	}, 400, 300)
	if err := tt.Click("Under the overlay"); err != nil {
		t.Fatal(err)
	}
	if !clicked {
		t.Fatal("a press on a control under the final-placement overlay never reached it")
	}
}
