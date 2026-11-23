package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyPluginIconsAreNativeVector(t *testing.T) {
	files, err := filepath.Glob("testdata/plugin-icons/*.svg")
	if err != nil || len(files) != 14 {
		t.Fatalf("want the 14 built-in plugin icons, got %d (%v)", len(files), err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		info := ClassifySVG(data)
		if info.Class != SVGNativeVector || info.NeedsFonts {
			t.Errorf("%s: class=%v fonts=%v, want native vector", filepath.Base(file), info.Class, info.NeedsFonts)
		}
		if info.Width != 36 || info.Height != 36 {
			t.Errorf("%s: intrinsic %vx%v, want 36x36", filepath.Base(file), info.Width, info.Height)
		}
	}
}

func TestClassifySVG(t *testing.T) {
	cases := []struct {
		name  string
		svg   string
		class SVGClass
		fonts bool
	}{
		{"text", `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="20"><text x="2" y="15">Hi</text></svg>`, SVGRaster, true},
		{"tspan", `<svg viewBox="0 0 10 10"><text><tspan>a</tspan></text></svg>`, SVGRaster, true},
		{"smil", `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"><animate attributeName="x" from="0" to="5" dur="1s"/></rect></svg>`, SVGRaster, false},
		{"filter", `<svg width="10" height="10"><filter id="f"><feGaussianBlur stdDeviation="2"/></filter><rect width="10" height="10" filter="url(#f)"/></svg>`, SVGRaster, false},
		{"foreignObject", `<svg width="10" height="10"><foreignObject width="10" height="10"><div xmlns="http://www.w3.org/1999/xhtml">x</div></foreignObject><rect width="1" height="1"/></svg>`, SVGRaster, false},
		{"css animation", `<svg width="10" height="10"><style>@keyframes k{from{opacity:0}}</style><rect width="10" height="10"/></svg>`, SVGRaster, false},
		{"native css", `<svg width="10" height="10"><style>.a{fill:#f00}</style><rect class="a" width="10" height="10"/></svg>`, SVGNativeVector, false},
		{"gradient", `<svg viewBox="0 0 4 2"><defs><linearGradient id="g"><stop offset="0" stop-color="#000"/></linearGradient></defs><path d="M0 0h4v2z" fill="url(#g)"/></svg>`, SVGNativeVector, false},
		{"dtd", `<!DOCTYPE svg [<!ENTITY a "b">]><svg width="1" height="1"/>`, SVGUnsupported, false},
		{"not svg", `<html/>`, SVGUnsupported, false},
		{"broken", `<svg width="1" height="1"><rect>`, SVGUnsupported, false},
		{"huge", `<svg width="100000" height="1"/>`, SVGUnsupported, false},
		{"unbounded path", `<svg width="1" height="1"><path d="M0 0L1e9 0"/></svg>`, SVGUnsupported, false},
	}
	for _, tc := range cases {
		info := ClassifySVG([]byte(tc.svg))
		if info.Class != tc.class || info.NeedsFonts != tc.fonts {
			t.Errorf("%s: got %v fonts=%v, want %v fonts=%v", tc.name, info.Class, info.NeedsFonts, tc.class, tc.fonts)
		}
	}
}

func TestClassifySVGIntrinsicSize(t *testing.T) {
	for _, tc := range []struct {
		svg  string
		w, h float64
	}{
		{`<svg width="2in" height="1in"/>`, 192, 96},
		{`<svg viewBox="0 0 40 20"/>`, 40, 20},
		{`<svg width="80" viewBox="0 0 40 20"/>`, 80, 40},
		{`<svg width="100%"/>`, 300, 150},
	} {
		info := ClassifySVG([]byte(tc.svg))
		if info.Width != tc.w || info.Height != tc.h {
			t.Errorf("%s: %vx%v, want %vx%v", tc.svg, info.Width, info.Height, tc.w, tc.h)
		}
	}
}

func TestPreparedInjectsDefaultsWithoutReparse(t *testing.T) {
	data := []byte(`<svg width="10" height="10"><text>a</text></svg>`)
	got := string(ClassifySVG(data).Prepared(data, "#ff0000"))
	if want := `font-size="16" color="#ff0000">`; !contains(got, want) || !contains(got, `font-family="`) {
		t.Fatalf("defaults missing: %s", got)
	}
	selfClosing := []byte(`<svg width="10" height="10" font-family="x" font-size="3"/>`)
	if got := string(ClassifySVG(selfClosing).Prepared(selfClosing, "red")); got != `<svg width="10" height="10" font-family="x" font-size="3" color="red"/>` {
		t.Fatalf("self-closing root: %s", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
