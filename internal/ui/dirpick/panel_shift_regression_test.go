package dirpick

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestDirectoryPathBarHoverZeroShift(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	s := fixtureBrowser()
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{State: &s, OnBeginPath: func() {}})
	}, 680, 500)

	crumbBefore, ok1 := tt.Find("Home")
	editBefore, ok2 := tt.Find("Edit path")
	if !ok1 || !ok2 {
		t.Fatalf("missing path bar controls: %v", tt.Texts())
	}

	tt.Move(editBefore.X+editBefore.W/2, editBefore.Y+editBefore.H/2)
	tt.Frame()

	crumbAfter, _ := tt.Find("Home")
	editAfter, _ := tt.Find("Edit path")
	if crumbAfter != crumbBefore || editAfter != editBefore {
		t.Fatalf("hovering Edit path shifted path bar: crumb %+v -> %+v, edit %+v -> %+v",
			crumbBefore, crumbAfter, editBefore, editAfter)
	}
}
