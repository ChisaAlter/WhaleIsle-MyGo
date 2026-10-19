package imagepreview

import (
	"bytes"
	"encoding/base64"
	"math"
	"strings"
	"testing"
)

func TestSVGImageIntrinsicUnitsAndProvisionalDimensions(t *testing.T) {
	for _, tc := range []struct {
		name, dimensions string
		width, height    float32
	}{
		{"inch", `width="1in" height=".5in"`, 96, 48},
		{"cm", `width="2.54cm" height="1.27cm"`, 96, 48},
		{"mm", `width="25.4mm" height="12.7mm"`, 96, 48},
		{"quarter-mm", `width="101.6Q" height="50.8q"`, 96, 48},
		{"point", `width="72pt" height="36pt"`, 96, 48},
		{"pica", `width="6pc" height="3pc"`, 96, 48},
		{"font", `width="10em" height="5ex"`, 300, 150},
		{"root-font", `width="10rem" height="5rem"`, 300, 150},
		{"percent", `width="100%" height="100%"`, 300, 150},
		{"absent", ``, 300, 150},
		{"viewbox-is-not-natural-size", `viewBox="0 0 800 800"`, 300, 150},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" ` + tc.dimensions + `><animate attributeName="opacity" dur="1s" values="0;1"/></svg>`)
			w, h, err := SVGImageSize(data)
			if err != nil || math.Abs(float64(w-tc.width)) > .001 || math.Abs(float64(h-tc.height)) > .001 {
				t.Fatalf("hint %vx%v, %v; want %vx%v", w, h, err, tc.width, tc.height)
			}
			if !NeedsBrowserSVG(data) {
				t.Fatal("animated/relative SVG lost the image browser route")
			}
			if _, err := BuildSVGImage("diagram.svg", data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSVGImageKeepsOriginalBytesInsideInertImage(t *testing.T) {
	data := []byte(`<svg width="10em" height="5ex"><script>parent.pwned=true</script><foreignObject><div xmlns="http://www.w3.org/1999/xhtml">text</div></foreignObject><image href="https://example.invalid/private.png"/></svg>`)
	markup, err := SVGSurfaceHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = `src="data:image/svg+xml;base64,`
	_, tail, ok := strings.Cut(markup, prefix)
	if !ok {
		t.Fatal("missing data image")
	}
	encoded, _, _ := strings.Cut(tail, `"`)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || !bytes.Equal(decoded, data) {
		t.Fatal("image bytes changed", err)
	}
	if strings.Contains(markup, "<svg") || strings.Contains(markup, "<script>") || strings.Contains(markup, "https://example.invalid") {
		t.Fatal("SVG escaped its image payload")
	}
	for _, policy := range []string{"script-src 'none'", "connect-src 'none'", "img-src data:", "frame-src 'none'"} {
		if !strings.Contains(markup, policy) {
			t.Fatalf("missing %s", policy)
		}
	}
}

func TestSVGImageBrowserGateRetainsBudgets(t *testing.T) {
	for _, source := range []string{
		`<svg width="40000px" height="1px"/>`, `<svg width="6000px" height="6000px"/>`,
		`<svg width="1in" height="0in"/>`, `<svg viewBox="0 0 9999999 1"/>`,
		`<!DOCTYPE svg [<!ENTITY e "x">]><svg>&e;</svg>`, `<svg/><svg/>`,
	} {
		if _, _, err := SVGImageSize([]byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
	if _, err := BuildSVGImage("diagram.html", []byte(`<svg/>`)); err == nil {
		t.Fatal("accepted a non-SVG file")
	}
}
