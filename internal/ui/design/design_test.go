package design

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPalettesMatchSpec(t *testing.T) {
	l, d := Base(Light), Base(Dark)
	if l.Background != (ui.Color{R: 255, G: 255, B: 255, A: 255}) || d.Background != (ui.Color{R: 0x14, G: 0x14, B: 0x14, A: 255}) {
		t.Fatalf("background light %v dark %v", l.Background, d.Background)
	}
	if l.Border != (ui.Color{A: 0x14}) || l.Input != (ui.Color{A: 0x1a}) {
		t.Fatalf("translucent tokens lost alpha: border %v input %v", l.Border, l.Input)
	}
	if d.Ring != (ui.Color{R: 0x73, G: 0x73, B: 0x73, A: 255}) {
		t.Fatalf("dark ring %v", d.Ring)
	}
}

func TestAccentOverrideIsReadable(t *testing.T) {
	for _, accent := range []ui.Color{
		ui.Hex("#4176e6"), ui.Hex("#7aaaff"), ui.Hex("#ffff00"), ui.Hex("#777777"), ui.Hex("#000000"), ui.Hex("#ffffff"), ui.Hex("#e11d48"),
	} {
		for _, s := range []Scheme{Light, Dark} {
			tk := Resolve(s, accent)
			if tk.Primary != accent || tk.Ring != accent {
				t.Fatalf("accent %v not applied: %v", accent, tk.Primary)
			}
			if c := Contrast(tk.PrimaryForeground, tk.Primary); c < 4.5 {
				t.Fatalf("accent %v foreground %v contrast %.2f", accent, tk.PrimaryForeground, c)
			}
		}
	}
	if tk := Resolve(Light, ui.Transparent); tk.Primary != cossLight.Primary {
		t.Fatalf("transparent accent should keep coss primary")
	}
}

func TestGlassSwitchDefaultsOff(t *testing.T) {
	if GlassEnabled() {
		t.Fatal("glass must default off")
	}
	tk := Lookup(Light, ui.Hex("#4176e6"))
	if tk.Glass || tk.Card.A != 255 {
		t.Fatalf("opaque surfaces expected, got %+v", tk.Card)
	}
	SetGlass(true, func(_ Scheme, p *Palette) { p.Card = p.Card.Alpha(0.5) })
	defer SetGlass(false, nil)
	if tk := Lookup(Light, ui.Hex("#4176e6")); !tk.Glass || tk.Card.A == 255 {
		t.Fatalf("glass filter not applied")
	}
}

func TestBezier(t *testing.T) {
	for _, x := range []float32{0, 0.25, 0.5, 0.75, 1} {
		if v := CurveLinear.At(x); v != x {
			t.Fatalf("linear %v → %v", x, v)
		}
	}
	// CSS `ease` (0.25, 0.1, 0.25, 1) is 0.8024 at its midpoint.
	if v := (CubicBezier{0.25, 0.1, 0.25, 1}).At(0.5); v < 0.80 || v > 0.805 {
		t.Fatalf("ease(0.5) = %v", v)
	}
	if v := CurveDrawer.At(0.2); v < 0.5 {
		t.Fatalf("drawer curve should be front-loaded, got %v", v)
	}
}

func TestMotionTables(t *testing.T) {
	if tr := TransitionFor("button/button"); tr.Props != "box-shadow" || tr.Duration != DurationDefault {
		t.Fatalf("button transition %+v", tr)
	}
	if d := EnterOf("dialog/dialog-popup"); !d.HasOpacity || d.Scale != 0.98 {
		t.Fatalf("dialog enter %+v", d)
	}
	if d := EnterOf("sheet/sheet-popup[when:side === \"right\"]"); d.TranslateX != 32 {
		t.Fatalf("sheet right enter %+v", d)
	}
}

func TestForReadsTheme(t *testing.T) {
	var got *Tokens
	tt := ui.NewTester(func(c *ui.Context) {
		th := *c.Theme()
		th.Dark, th.Accent = true, ui.Hex("#7aaaff")
		c.SetTheme(&th)
		got = For(c)
	}, 50, 50)
	_ = tt
	if got == nil || !got.IsDark() || got.Primary != ui.Hex("#7aaaff") {
		t.Fatalf("For: %+v", got)
	}
}
