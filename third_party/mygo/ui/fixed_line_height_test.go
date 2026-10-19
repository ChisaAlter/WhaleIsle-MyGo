package ui

import "testing"

func fixedLineHeightView(c *context) {
	coreColumn(c).Padding(16).AlignItems(Start).Font("Segoe UI, Microsoft YaHei, sans-serif").FontSize(24).FixedLineHeight(18).TextColor(Hex("#000000")).Children(func() {
		coreText(c, "ABC鲸屿\nxyz设置").Label("exact")
		coreText(c, "ABC鲸屿").LineHeight(.5).Label("ordinary")
	})
}

func TestFixedLineHeightInheritsExactFrameAcrossScales(t *testing.T) {
	tt := coreNewTester(fixedLineHeightView, 400, 140)
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		tt.SetScale(scale)
		exact, _ := tt.Find("exact")
		ordinary, _ := tt.Find("ordinary")
		if exact.H != 36 || ordinary.Y != exact.Y+36 || ordinary.H <= 18 {
			t.Errorf("scale %v: exact frame %+v, ordinary frame %+v", scale, exact, ordinary)
		}
	}
}
