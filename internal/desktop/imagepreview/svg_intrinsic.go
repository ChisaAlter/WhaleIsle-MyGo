package imagepreview

import (
	"bytes"
	"encoding/xml"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var svgIntrinsicLength = regexp.MustCompile(`^([-+]?(?:[0-9]*\.)?[0-9]+(?:[eE][-+]?[0-9]+)?)([a-zA-Z%]*)$`)

// Browser sizing is separate from the static renderer's admission policy.
// Coordinate/viewBox budgets remain bounded, but percentages and font/CSS
// lengths need no invented font metrics or viewBox to enter an inert <img>.
func svgBrowserDimensions(attrs []xml.Attr) error {
	values := svgIntrinsicAttributes(attrs)
	if view := values["viewbox"]; view != "" {
		fields := strings.Fields(strings.ReplaceAll(view, ",", " "))
		if len(fields) != 4 {
			return errors.New("SVG viewBox is invalid")
		}
		var box [4]float64
		for i, field := range fields {
			n, err := strconv.ParseFloat(field, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1_000_000 {
				return errors.New("SVG viewBox is unbounded")
			}
			box[i] = n
		}
		if err := svgArea(box[2], box[3]); err != nil {
			return err
		}
	}
	w, wk := svgAbsoluteLength(values["width"])
	h, hk := svgAbsoluteLength(values["height"])
	if wk && (w <= 0 || w > 32768) || hk && (h <= 0 || h > 32768) {
		return errors.New("SVG dimensions exceed the pixel budget")
	}
	if wk && hk {
		return svgArea(w, h)
	}
	for _, key := range []string{"width", "height"} {
		if err := svgNumbersBounded(values[key]); err != nil {
			return err
		}
	}
	return nil
}

func svgIntrinsicAttributes(attrs []xml.Attr) map[string]string {
	values := make(map[string]string, len(attrs))
	for _, a := range attrs {
		values[strings.ToLower(a.Name.Local)] = strings.TrimSpace(a.Value)
	}
	return values
}

// CSS absolute units are fixed at 96 CSS px per inch. Relative units, auto,
// percentages, calc() and CSS overrides are resolved by the same browser image
// that paints the seat; they must never be approximated by Go font metrics.
func svgAbsoluteLength(value string) (float64, bool) {
	m := svgIntrinsicLength.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	factor := float64(1)
	switch strings.ToLower(m[2]) {
	case "", "px":
	case "in":
		factor = 96
	case "cm":
		factor = 96 / 2.54
	case "mm":
		factor = 96 / 25.4
	case "q":
		factor = 96 / 101.6
	case "pt":
		factor = 96 / 72.0
	case "pc":
		factor = 16
	default:
		return 0, false
	}
	return n * factor, true
}

// svgIntrinsicHint is provisional layout while the contained browser decodes.
// Only two definite absolute lengths are known without that browser. The
// 300x150 default-object box is a placeholder, never a viewBox-sized claim.
func svgIntrinsicHint(data []byte) (width, height float32, exact bool, err error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, e := d.Token()
		if e != nil {
			return 0, 0, false, e
		}
		root, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		values := svgIntrinsicAttributes(root.Attr)
		w, wk := svgAbsoluteLength(values["width"])
		h, hk := svgAbsoluteLength(values["height"])
		exact = wk && hk && !strings.Contains(strings.ToLower(values["style"]), "width") && !strings.Contains(strings.ToLower(values["style"]), "height")
		if !wk || w <= 0 {
			w = 300
		}
		if !hk || h <= 0 {
			h = 150
		}
		if err := svgArea(w, h); err != nil {
			return 0, 0, false, err
		}
		return float32(w), float32(h), exact, nil
	}
}
