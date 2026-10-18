package webview

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"math"
	"strings"
	"testing"
)

func sampleB17Annotation() BrowserAnnotation {
	return BrowserAnnotation{Regions: []AnnotationRegion{{Rect: AnnotationRect{0, 0, 10, 10}}}, Crop: AnnotationRect{1, 1, 2, 2}, ViewportWidth: 10, ViewportHeight: 10}
}

func TestB17PayloadAndCrop(t *testing.T) {
	var a BrowserAnnotation
	err := json.Unmarshal([]byte(`{"pageUrl":"https://example.test","comment":"note","regions":[{"id":"r1","rect":{"x":0,"y":0,"width":10,"height":10}}],"crop":{"x":1,"y":1,"width":2,"height":2},"viewportWidth":10,"viewportHeight":10}`), &a)
	if err != nil || a.validate() != nil || a.PageURL != "https://example.test" || a.Comment != "note" {
		t.Fatalf("decode: %v / %+v", err, a)
	}
	changes := []func(*BrowserAnnotation){
		func(a *BrowserAnnotation) { a.Crop.Width = -1 },
		func(a *BrowserAnnotation) { a.ViewportWidth = 0 },
		func(a *BrowserAnnotation) { a.Crop.X = math.NaN() },
		func(a *BrowserAnnotation) { a.Regions = make([]AnnotationRegion, 21) },
		func(a *BrowserAnnotation) { a.Comment = strings.Repeat("x", 16385) },
		func(a *BrowserAnnotation) { a.Strokes = []AnnotationStroke{{Points: make([]AnnotationPoint, 1025)}} },
	}
	for n, change := range changes {
		v := sampleB17Annotation()
		change(&v)
		if v.validate() == nil {
			t.Fatalf("accepted invalid payload %d", n)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 30))); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		r    AnnotationRect
		w, h int
	}{
		{AnnotationRect{1, 1, 2, 2}, 4, 6},
		{AnnotationRect{-1, -1, 3, 3}, 4, 6},
		{AnnotationRect{9, 9, 5, 5}, 2, 3},
	} {
		v := sampleB17Annotation()
		v.Crop = tc.r
		data, err := cropBrowserAnnotation(buf.Bytes(), v)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width != tc.w || cfg.Height != tc.h {
			t.Fatalf("crop %+v => %+v %v", tc, cfg, err)
		}
	}
	a.Crop = AnnotationRect{30, 30, 2, 2}
	if _, err := cropBrowserAnnotation(buf.Bytes(), a); err == nil {
		t.Fatal("accepted off-viewport crop")
	}
	if _, err := cropBrowserAnnotation([]byte("not png"), sampleB17Annotation()); err == nil {
		t.Fatal("accepted non-PNG")
	}
}

func TestB17RendererKeepsStylesUntilNativeCleanup(t *testing.T) {
	start := strings.Index(browserAnnotationScript, "  const submitAnnotation =")
	end := strings.Index(browserAnnotationScript, "  submit.addEventListener")
	if start < 0 || end <= start {
		t.Fatal("missing editor submission")
	}
	submit := browserAnnotationScript[start:end]
	if strings.Contains(submit, "restoreStyles") || strings.Contains(submit, "teardown(") {
		t.Fatal("renderer restored CSS before native capture")
	}
	hide, done := strings.Index(submit, "editor.style.display = 'none'"), strings.Index(submit, "state.done = true")
	if hide < 0 || done < hide {
		t.Fatal("completion before editor chrome hidden")
	}
	if !strings.Contains(browserAnnotationScript, "if (pendingCapture) { event.preventDefault(); event.stopImmediatePropagation(); return; }") {
		t.Fatal("Escape could restore CSS during native capture")
	}
	for _, forbidden := range []string{"ipcRenderer", "chrome.webview", "outerHTML"} {
		if strings.Contains(browserAnnotationScript, forbidden) {
			t.Fatalf("unexpected renderer capability %q", forbidden)
		}
	}
}
