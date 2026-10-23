package ui

import "testing"

func TestContextDeviceScaleMatchesCurrentLayoutAndPaint(t *testing.T) {
	for _, pixel := range []bool{false, true} {
		var built, painted float32
		view := func(c *Context) {
			built = c.DeviceScale()
			Box(c).Fill().Draw(func(p *Painter, _ Rect) { painted = p.Scale() })
		}
		var tt *Tester
		if pixel {
			tt = newPixelTester(view, 300, 200, 1.5)
		} else {
			tt = NewTester(view, 300, 200)
		}
		for _, scale := range []float32{1, 1.25, 1.5, 2, 3, 0, -1} {
			tt.SetScale(scale)
			want := scale
			if want <= 0 {
				want = 1
			}
			if built != want || painted != want {
				t.Fatalf("pixel=%v host scale=%v build=%v paint=%v want=%v", pixel, scale, built, painted, want)
			}
		}
	}
}
