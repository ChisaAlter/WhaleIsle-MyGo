package theme

import "testing"

func TestColorHex(t *testing.T) {
	c := Color("#151517")
	if c.R != 0x15 || c.G != 0x15 || c.B != 0x17 || c.A != 255 {
		t.Fatalf("hex = %+v", c)
	}
}

func TestColorRGBA(t *testing.T) {
	c := Color("rgba(0, 0, 0, 0.10)")
	if c.R != 0 || c.G != 0 || c.B != 0 || c.A == 255 {
		t.Fatalf("rgba = %+v", c)
	}
	if c.A < 20 || c.A > 30 { // 0.10 * 255 ≈ 26
		t.Fatalf("rgba alpha = %d", c.A)
	}
}

func TestColorBad(t *testing.T) {
	c := Color("not-a-color")
	if c.R != 0 || c.G != 0 || c.B != 0 {
		t.Fatalf("bad color should be black, got %+v", c)
	}
}
