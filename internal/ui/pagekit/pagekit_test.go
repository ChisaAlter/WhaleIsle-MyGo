package pagekit

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSkeletonRowsSettlesWithoutAnimationLoop(t *testing.T) {
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Column(c).Fill().Children(func() {
			SkeletonRows(c, 3, "Loading plugins")
		})
	}, 640, 480)
	if _, ok := tt.Find("Loading plugins"); !ok {
		t.Fatal("SkeletonRows missing status label")
	}
	if builds > 2 {
		t.Fatalf("SkeletonRows triggered continuous animation rebuilds: builds=%d, want <= 2", builds)
	}
}
