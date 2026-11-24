package ui

import "testing"

// screenHost is a pixel host whose screen's work area is landscape.
type screenHost struct{ pixelHost }

func (h screenHost) workArea() (bool, bool) { return true, true }

// Size is in DIPs on every surface, at every scale, and DevicePixelSize in
// device pixels; DisplayLandscape is the host's, unknown in a Tester.
func TestSizeContract(t *testing.T) {
	for _, tc := range scaledWindows {
		var w, h, pw, ph float32
		var landscape, known bool
		view := func(c *Context) {
			w, h = c.Size()
			pw, ph = c.DevicePixelSize()
			landscape, known = c.DisplayLandscape()
			Text(c, "x")
		}
		tt := newPixelTester(view, tc.pixels[0], tc.pixels[1], tc.scale)
		if !sameLength(w, tc.wantDIP[0]) || !sameLength(h, tc.wantDIP[1]) {
			t.Errorf("%s: Size %vx%v, want %v", tc.name, w, h, tc.wantDIP)
		}
		if int(pw) != tc.pixels[0] || int(ph) != tc.pixels[1] {
			t.Errorf("%s: DevicePixelSize %vx%v, want %v", tc.name, pw, ph, tc.pixels)
		}
		if known {
			t.Errorf("%s: a Tester knows the screen's orientation", tc.name)
		}
		tt.rt.host = screenHost{tt.rt.host.(pixelHost)}
		tt.Frame()
		if !landscape || !known {
			t.Errorf("%s: the host's landscape screen reads %v, %v", tc.name, landscape, known)
		}

		headless := NewTester(view, tc.pixels[0], tc.pixels[1])
		headless.SetScale(tc.scale)
		if w != float32(tc.pixels[0]) || h != float32(tc.pixels[1]) {
			t.Errorf("%s: headless Size %vx%v", tc.name, w, h)
		}
		if !sameLength(pw, float32(tc.pixels[0])*tc.scale) || !sameLength(ph, float32(tc.pixels[1])*tc.scale) {
			t.Errorf("%s: headless DevicePixelSize %vx%v", tc.name, pw, ph)
		}
	}
}
