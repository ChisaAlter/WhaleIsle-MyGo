package ui_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMemoPublic(t *testing.T) {
	version, builds, clicks := uint64(1), 0, 0
	label := "First"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Memo(version).Children(func() {
			builds++
			ui.Text(c, label)
			if ui.Button(c, "Press").Clicked() {
				clicks++
			}
		})
	}, 300, 200)
	tt.Frame()
	tt.Frame()
	if builds != 1 {
		t.Fatalf("built %d times", builds)
	}
	label, version = "Second", 2
	tt.Frame()
	if !tt.HasText("Second") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("Press"); err != nil || clicks != 1 {
		t.Fatalf("click: %v, %d", err, clicks)
	}
}
