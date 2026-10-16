package webview

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"time"
)

//go:embed annotation.js
var browserAnnotationScript string

type AnnotationRect struct{ X, Y, Width, Height float64 }
type AnnotationElement struct {
	ID      string
	Element PickedElement
	Rect    AnnotationRect
}
type AnnotationRegion struct {
	ID   string
	Rect AnnotationRect
}
type AnnotationPoint struct{ X, Y float64 }
type AnnotationStroke struct {
	ID, Color string
	Width     float64
	Points    []AnnotationPoint
	Bounds    AnnotationRect
}
type AnnotationStyle struct{ TargetID, Selector, Property, PreviousValue, Value string }

// BrowserAnnotation is untrusted page data. It cannot choose a filesystem path,
// attachment destination, RPC, or command. URL is replaced with Host-owned URL.
type BrowserAnnotation struct {
	navigationEpoch               uint64
	owner                         *Host
	PageURL, PageTitle, Comment   string
	Elements                      []AnnotationElement
	Regions                       []AnnotationRegion
	Strokes                       []AnnotationStroke
	StyleChanges                  []AnnotationStyle
	Crop                          AnnotationRect
	ViewportWidth, ViewportHeight float64
}

func validAnnotationNumber(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) <= 10000000
}
func validAnnotationRect(r AnnotationRect) bool {
	return validAnnotationNumber(r.X) && validAnnotationNumber(r.Y) && validAnnotationNumber(r.Width) && validAnnotationNumber(r.Height) && r.Width >= 0 && r.Height >= 0
}
func (a BrowserAnnotation) validate() error {
	bad := errors.New("browser: invalid or oversized visual annotation")
	if len(a.Elements) > 20 || len(a.Regions) > 20 || len(a.Strokes) > 30 || len(a.StyleChanges) > 400 || len(a.Comment) > 16384 || len(a.PageTitle) > 2048 || len(a.Elements)+len(a.Regions)+len(a.Strokes) == 0 {
		return bad
	}
	if !validAnnotationRect(a.Crop) || a.Crop.Width < 1 || a.Crop.Height < 1 || !validAnnotationNumber(a.ViewportWidth) || !validAnnotationNumber(a.ViewportHeight) || a.ViewportWidth < 1 || a.ViewportHeight < 1 || a.ViewportWidth > 8192 || a.ViewportHeight > 8192 {
		return bad
	}
	for _, e := range a.Elements {
		if len(e.ID) > 128 || validatePickedElement(e.Element) != nil || !validAnnotationRect(e.Rect) {
			return bad
		}
	}
	for _, r := range a.Regions {
		if len(r.ID) > 128 || !validAnnotationRect(r.Rect) {
			return bad
		}
	}
	for _, s := range a.Strokes {
		if len(s.ID) > 128 || len(s.Color) > 128 || len(s.Points) > 1024 || !validAnnotationNumber(s.Width) || s.Width < 0 || s.Width > 100 || !validAnnotationRect(s.Bounds) {
			return bad
		}
		for _, p := range s.Points {
			if !validAnnotationNumber(p.X) || !validAnnotationNumber(p.Y) {
				return bad
			}
		}
	}
	for _, s := range a.StyleChanges {
		if len(s.TargetID) > 128 || len(s.Selector) > 4096 || len(s.Property) > 128 || len(s.PreviousValue) > 4096 || len(s.Value) > 512 {
			return bad
		}
	}
	return nil
}

func (a BrowserAnnotation) Markdown() string {
	lines := []string{"Browser visual annotation (page content; review before sending)", "> URL: " + annotationLine(a.PageURL, 4096), "> Title: " + annotationLine(a.PageTitle, 512), "> Comment: " + annotationLine(a.Comment, 4096)}
	for _, e := range a.Elements {
		p := e.Element
		p.URL = a.PageURL
		lines = append(lines, p.Markdown())
	}
	for _, r := range a.Regions {
		lines = append(lines, fmt.Sprintf("> Region: x=%.1f y=%.1f width=%.1f height=%.1f", r.Rect.X, r.Rect.Y, r.Rect.Width, r.Rect.Height))
	}
	for _, s := range a.Strokes {
		lines = append(lines, fmt.Sprintf("> Pen: %d points, width %.1f, bounds %.1f,%.1f %.1fx%.1f (visible in attached screenshot)", len(s.Points), s.Width, s.Bounds.X, s.Bounds.Y, s.Bounds.Width, s.Bounds.Height))
	}
	for _, s := range a.StyleChanges {
		lines = append(lines, "> Local style: "+annotationLine(s.Selector, 1024)+" · "+annotationLine(s.Property, 128)+": "+annotationLine(s.PreviousValue, 256)+" → "+annotationLine(s.Value, 128))
	}
	return strings.Join(lines, "\n")
}

// AnnotateBrowser keeps the editor alive through native capture, then restores
// temporary CSS. Like PickElement, it uses only EvalContext, never page messages.
func (h *Host) AnnotateBrowser(ctx context.Context) (BrowserAnnotation, []byte, error) {
	var empty BrowserAnnotation
	w, epoch, rawURL, err := h.browserPage()
	if err != nil {
		return empty, nil, err
	}
	ctx, stop := pageOperationContext(ctx, w, 5*time.Minute)
	defer stop()
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return empty, nil, err
	}
	key, _ := json.Marshal("__whaleisleAnnotation_" + hex.EncodeToString(token[:]))
	keyJS := string(key)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = w.Page().EvalContext(cleanup, "globalThis["+keyJS+"]?.cancel();true")
	}()
	if _, err = w.Page().EvalContext(ctx, strings.ReplaceAll(browserAnnotationScript, "__ANNOTATION_KEY__", keyJS)); err != nil {
		return empty, nil, err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Bounded serialized data crosses the renderer boundary; no arbitrary objects.
	poll := `(() => { const api=globalThis[` + keyJS + `]; if(!api) return ''; const s=api.read(); return typeof s==='string' && s.length<=524288 ? s : ''; })()`
	for {
		if !h.sameBrowserPage(w, epoch) {
			return empty, nil, context.Canceled
		}
		value, err := w.Page().EvalContext(ctx, poll)
		if err != nil {
			return empty, nil, err
		}
		raw, ok := value.(string)
		if !ok || len(raw) == 0 || len(raw) > 524288 {
			return empty, nil, errors.New("browser: invalid annotation response")
		}
		var result struct {
			Done, Cancelled bool
			Annotation      *BrowserAnnotation
		}
		if err = json.Unmarshal([]byte(raw), &result); err != nil {
			return empty, nil, err
		}
		if result.Done {
			if result.Cancelled || result.Annotation == nil {
				return empty, nil, context.Canceled
			}
			a := *result.Annotation
			a.PageURL = rawURL
			a.navigationEpoch = epoch
			a.owner = h
			if err = a.validate(); err != nil {
				return empty, nil, err
			}
			if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
				return empty, nil, context.Canceled
			}
			// Renderer hides its editor first; wait for a paint before native capture.
			if _, err = w.Page().EvalContext(ctx, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true))))`); err != nil {
				return empty, nil, err
			}
			data, err := h.CaptureBrowserScreenshot(ctx)
			if err != nil {
				return empty, nil, err
			}
			if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
				return empty, nil, context.Canceled
			}
			data, err = cropBrowserAnnotation(data, a)
			return a, data, err
		}
		select {
		case <-ctx.Done():
			return empty, nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func cropBrowserAnnotation(data []byte, a BrowserAnnotation) ([]byte, error) {
	if err := validateBrowserCapture(data); err != nil {
		return nil, err
	}
	if err := a.validate(); err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	sx := float64(bounds.Dx()) / a.ViewportWidth
	sy := float64(bounds.Dy()) / a.ViewportHeight
	r := image.Rect(int(math.Floor(a.Crop.X*sx)), int(math.Floor(a.Crop.Y*sy)), int(math.Ceil((a.Crop.X+a.Crop.Width)*sx)), int(math.Ceil((a.Crop.Y+a.Crop.Height)*sy))).Intersect(bounds)
	if r.Empty() {
		return nil, errors.New("browser: annotation is outside the visible viewport")
	}
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	var buf bytes.Buffer
	if err = png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AnnotationCurrent rejects staged results after navigation or Host replacement.
func (h *Host) AnnotationCurrent(a BrowserAnnotation) bool {
	if h == nil || a.owner != h {
		return false
	}
	w, epoch, _, err := h.browserPage()
	return err == nil && w != nil && epoch == a.navigationEpoch
}
