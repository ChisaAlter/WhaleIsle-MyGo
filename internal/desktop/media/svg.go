package media

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image"
	"io"
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"

	"whaleisle/internal/desktop/imagepreview"
)

// SVGClass is the one decision about how an SVG is drawn.
type SVGClass int

const (
	// SVGUnsupported documents are malformed, over budget or use DTDs; they
	// show the caller's placeholder.
	SVGUnsupported SVGClass = iota
	// SVGNativeVector documents use only what MyGo's GPU vector renderer
	// draws (ui.ParseSVG): no worker raster, no fonts, sharp at any size.
	SVGNativeVector
	// SVGRaster documents need resvg (text, filters, embedded images,
	// patterns, markers, CSS the native path lacks, or SMIL/foreignObject,
	// which render as their static first frame).
	SVGRaster
)

func (c SVGClass) String() string {
	switch c {
	case SVGNativeVector:
		return "native"
	case SVGRaster:
		return "raster"
	}
	return "unsupported"
}

// SVGInfo is the result of ClassifySVG's single XML pass.
type SVGInfo struct {
	Class SVGClass
	// NeedsFonts is set for raster documents that contain text.
	NeedsFonts bool
	// Width and Height are the intrinsic size in CSS px (absolute
	// width/height, else the viewBox, else the 300x150 object default).
	Width, Height float64
	// rootEnd is the offset of the root start tag's closing '>' (or "/>"),
	// where default presentation attributes can be injected without a
	// second parse. hasFamily/hasSize/hasColor record root overrides.
	rootEnd                      int
	hasFamily, hasSize, hasColor bool
}

const (
	svgNS   = "http://www.w3.org/2000/svg"
	xlinkNS = "http://www.w3.org/1999/xlink"
	xmlNS   = "http://www.w3.org/XML/1998/namespace"

	maxSVGTokens = 40_000
	maxSVGDepth  = 64
	maxSVGNodes  = 10_000
	maxSVGAttr   = 64 << 10
	maxSVGCoord  = 32768.0
)

// ClassifySVG reads the document once. It never renders and never touches
// the filesystem or network.
func ClassifySVG(data []byte) SVGInfo {
	info := SVGInfo{Width: 300, Height: 150}
	if len(data) == 0 || len(data) > imagepreview.MaxSVGBytes {
		return info
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	native := true
	var style strings.Builder
	inStyle := false
	ignored := 0
	depth, nodes, tokens := 0, 0, 0
	seen, closed := false, false
	var embedded int64
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return SVGInfo{Width: 300, Height: 150}
		}
		tokens++
		if tokens > maxSVGTokens {
			return SVGInfo{Width: 300, Height: 150}
		}
		switch t := token.(type) {
		case xml.Directive:
			return SVGInfo{Width: 300, Height: 150} // DTDs and entities
		case xml.ProcInst:
			if t.Target != "xml" || seen {
				return SVGInfo{Width: 300, Height: 150}
			}
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) != 0 {
				return SVGInfo{Width: 300, Height: 150}
			}
			if inStyle && ignored == 0 {
				style.Write(t)
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return SVGInfo{Width: 300, Height: 150}
			}
			if depth == 0 {
				closed = true
			}
			if ignored > 0 {
				ignored--
			} else if t.Name.Local == "style" {
				inStyle = false
				if native && !nativeCSS(style.String(), false) {
					native = false
				}
			}
		case xml.StartElement:
			if closed || depth >= maxSVGDepth || nodes >= maxSVGNodes {
				return SVGInfo{Width: 300, Height: 150}
			}
			depth++
			nodes++
			name := t.Name.Local
			if !seen {
				if name != "svg" || t.Name.Space != "" && t.Name.Space != svgNS {
					return SVGInfo{Width: 300, Height: 150}
				}
				if !info.root(t.Attr) {
					return SVGInfo{Width: 300, Height: 150}
				}
				seen = true
				info.rootEnd = int(d.InputOffset()) - 1
				if info.rootEnd > 0 && info.rootEnd < len(data) && data[info.rootEnd-1] == '/' {
					info.rootEnd--
				}
			}
			for _, a := range t.Attr {
				if len(a.Value) > maxSVGAttr && !(a.Name.Local == "href" && strings.HasPrefix(strings.TrimSpace(a.Value), "data:")) {
					return SVGInfo{Width: 300, Height: 150}
				}
				key := strings.ToLower(a.Name.Local)
				if numericAttribute(key) && !numbersBounded(a.Value) {
					return SVGInfo{Width: 300, Height: 150}
				}
			}
			if ignored > 0 {
				ignored++
				continue
			}
			if t.Name.Space != "" && t.Name.Space != svgNS {
				// Foreign content (editor metadata, XHTML in foreignObject's
				// subtree) is skipped by resvg, never by the native parser.
				native = false
				ignored = 1
				continue
			}
			switch name {
			case "title", "desc", "metadata":
				ignored = 1 // nonpainting metadata may carry other namespaces
				continue
			case "svg", "g", "a", "switch", "defs", "symbol", "use", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "linearGradient", "radialGradient", "stop", "clipPath", "mask":
			case "style":
				inStyle = true
				style.Reset()
			case "text", "tspan", "textPath":
				native = false
				info.NeedsFonts = true
			case "image", "feImage":
				native = false
				for _, a := range t.Attr {
					if a.Name.Local == "href" && !strings.HasPrefix(strings.TrimSpace(a.Value), "data:") && !strings.HasPrefix(strings.TrimSpace(a.Value), "#") {
						// External files and URLs are never resolved.
						return SVGInfo{Width: 300, Height: 150}
					}
					if a.Name.Local == "href" && strings.HasPrefix(strings.TrimSpace(a.Value), "data:") {
						count, ok := embeddedPixels(strings.TrimSpace(a.Value))
						if !ok {
							return SVGInfo{Width: 300, Height: 150}
						}
						embedded += count
						if embedded > imagepreview.MaxPixels {
							return SVGInfo{Width: 300, Height: 150}
						}
					}
				}
			case "foreignObject":
				// resvg draws no foreign content; the rest is static artwork.
				native = false
				ignored = 1
				continue
			default:
				// Filters, patterns, markers, SMIL (first frame), scripts
				// (never executed: resvg has no script engine).
				native = false
			}
			if native {
				for _, a := range t.Attr {
					if !nativeAttribute(name, a) {
						native = false
						break
					}
				}
			}
		}
	}
	if !seen || !closed || depth != 0 {
		return SVGInfo{Width: 300, Height: 150}
	}
	if native {
		info.Class = SVGNativeVector
		info.NeedsFonts = false
	} else {
		info.Class = SVGRaster
	}
	return info
}

// root reads the root's intrinsic size and presentation overrides.
func (info *SVGInfo) root(attrs []xml.Attr) bool {
	var width, height, view string
	for _, a := range attrs {
		switch strings.ToLower(a.Name.Local) {
		case "width":
			width = strings.TrimSpace(a.Value)
		case "height":
			height = strings.TrimSpace(a.Value)
		case "viewbox":
			view = strings.TrimSpace(a.Value)
		case "font-family":
			info.hasFamily = true
		case "font-size":
			info.hasSize = true
		case "color":
			info.hasColor = true
		case "style":
			lower := strings.ToLower(a.Value)
			info.hasFamily = info.hasFamily || strings.Contains(lower, "font-family")
			info.hasSize = info.hasSize || strings.Contains(lower, "font-size")
			info.hasColor = info.hasColor || strings.Contains(lower, "color")
		}
	}
	var vw, vh float64
	if view != "" {
		fields := strings.Fields(strings.ReplaceAll(view, ",", " "))
		if len(fields) != 4 {
			return false
		}
		var box [4]float64
		for i, field := range fields {
			n, err := strconv.ParseFloat(field, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1_000_000 {
				return false
			}
			box[i] = n
		}
		if !area(box[2], box[3]) {
			return false
		}
		vw, vh = box[2], box[3]
	}
	w, wk := absoluteLength(width)
	h, hk := absoluteLength(height)
	if wk && !area(w, 1) || hk && !area(1, h) {
		return false
	}
	switch {
	case wk && hk:
		info.Width, info.Height = w, h
	case wk && vw > 0:
		info.Width, info.Height = w, w*vh/vw
	case hk && vh > 0:
		info.Width, info.Height = h*vw/vh, h
	case vw > 0:
		info.Width, info.Height = vw, vh
	}
	return area(info.Width, info.Height)
}

// Prepared returns the bytes resvg renders: the browser's initial serif
// 16px font when the root sets none (text documents only), and currentColor
// from the requested appearance. It reuses rootEnd, never re-parsing.
func (info SVGInfo) Prepared(data []byte, color string) []byte {
	if info.rootEnd <= 0 || info.rootEnd >= len(data) {
		return data
	}
	var extra string
	if info.NeedsFonts && !info.hasFamily {
		name := "DejaVu Serif"
		switch runtime.GOOS {
		case "windows":
			name = "Times New Roman"
		case "darwin":
			name = "Times"
		}
		extra += ` font-family="` + name + `"`
	}
	if info.NeedsFonts && !info.hasSize {
		extra += ` font-size="16"`
	}
	if color != "" && !info.hasColor && safeColor(color) {
		extra += ` color="` + color + `"`
	}
	if extra == "" {
		return data
	}
	out := make([]byte, 0, len(data)+len(extra))
	out = append(out, data[:info.rootEnd]...)
	out = append(out, extra...)
	return append(out, data[info.rootEnd:]...)
}

func safeColor(s string) bool {
	for _, r := range s {
		if !(r == '#' || r == '(' || r == ')' || r == ',' || r == '.' || r == ' ' || r == '%' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return len(s) <= 64
}

func area(w, h float64) bool {
	return !(math.IsNaN(w) || math.IsNaN(h) || math.IsInf(w, 0) || math.IsInf(h, 0) ||
		w <= 0 || h <= 0 || w > maxSVGCoord || h > maxSVGCoord || w*h > float64(imagepreview.MaxPixels))
}

// absoluteLength resolves CSS absolute units at 96 px per inch. Relative
// units and percentages report false and fall back to the viewBox.
func absoluteLength(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	end := 0
	for end < len(value) && (value[end] >= '0' && value[end] <= '9' || value[end] == '.' || value[end] == '-' || value[end] == '+' || value[end] == 'e' && end > 0 && end+1 < len(value) && (value[end+1] >= '0' && value[end+1] <= '9' || value[end+1] == '-')) {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(value[:end], 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	factor := 1.0
	switch strings.ToLower(value[end:]) {
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

func numericAttribute(key string) bool {
	switch key {
	case "d", "points", "transform", "gradienttransform", "patterntransform",
		"x", "y", "x1", "x2", "y1", "y2", "cx", "cy", "r", "rx", "ry", "dx", "dy",
		"width", "height", "font-size", "stroke-width", "stroke-dasharray", "stroke-dashoffset",
		"stddeviation", "radius", "scale", "basefrequency", "numoctaves", "kernelmatrix", "order", "divisor", "bias",
		"surfacescale", "diffuseconstant", "specularconstant", "specularexponent", "azimuth", "elevation":
		return true
	}
	return false
}

// numbersBounded scans numbers without a regexp or allocation: every one
// must be finite and within the coordinate budget the renderers accept.
func numbersBounded(value string) bool {
	for i := 0; i < len(value); {
		c := value[i]
		if !(c >= '0' && c <= '9' || c == '.' || c == '-' || c == '+') {
			i++
			continue
		}
		j := i
		if value[j] == '-' || value[j] == '+' {
			j++
		}
		digits := false
		for j < len(value) && value[j] >= '0' && value[j] <= '9' {
			j++
			digits = true
		}
		if j < len(value) && value[j] == '.' {
			j++
			for j < len(value) && value[j] >= '0' && value[j] <= '9' {
				j++
				digits = true
			}
		}
		if !digits {
			i = max(j, i+1)
			continue
		}
		if j < len(value) && (value[j] == 'e' || value[j] == 'E') {
			k := j + 1
			if k < len(value) && (value[k] == '-' || value[k] == '+') {
				k++
			}
			if k < len(value) && value[k] >= '0' && value[k] <= '9' {
				for k < len(value) && value[k] >= '0' && value[k] <= '9' {
					k++
				}
				j = k
			}
		}
		n, err := strconv.ParseFloat(value[i:j], 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > maxSVGCoord {
			return false
		}
		i = j
	}
	return true
}

// embeddedPixels bounds a data: raster inside an SVG by its header only.
func embeddedPixels(raw string) (int64, bool) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return 0, false
	}
	switch raw[:comma] {
	case "data:image/png;base64", "data:image/jpeg;base64", "data:image/gif;base64", "data:image/webp;base64":
	default:
		return 0, false
	}
	payload := raw[comma+1:]
	// A header never needs more than the first 64 KiB to report its size.
	if len(payload) > 88<<10 {
		payload = payload[:88<<10]
	}
	payload = payload[:len(payload)/4*4]
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return 0, false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width) > imagepreview.MaxPixels/int64(cfg.Height) {
		return 0, false
	}
	return int64(cfg.Width) * int64(cfg.Height), true
}

func nativeAttribute(element string, attr xml.Attr) bool {
	if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
		return true
	}
	if attr.Name.Space == xmlNS {
		return attr.Name.Local == "space" || attr.Name.Local == "lang"
	}
	if attr.Name.Space != "" && attr.Name.Space != xlinkNS {
		return true // editor metadata is ignored by both renderers
	}
	name := attr.Name.Local
	if name == "style" {
		return nativeCSS(attr.Value, true)
	}
	if nativeProperty(name) {
		return nativeValue(attr.Value)
	}
	switch name {
	case "href":
		return element == "a" || strings.HasPrefix(strings.TrimSpace(attr.Value), "#")
	case "id", "class", "version", "baseProfile", "role", "focusable", "tabindex":
		return true
	case "x", "y", "x1", "x2", "y1", "y2", "cx", "cy", "r", "rx", "ry", "width", "height", "d", "points", "viewBox", "preserveAspectRatio", "transform", "gradientTransform", "gradientUnits", "spreadMethod", "fx", "fy", "fr", "offset", "clipPathUnits", "maskUnits", "maskContentUnits":
		return true
	case "type":
		return element == "style" && (attr.Value == "" || attr.Value == "text/css")
	default:
		return strings.HasPrefix(name, "aria-") || strings.HasPrefix(name, "data-")
	}
}

// nativeProperty mirrors third_party/mygo/internal/svg/style.go.
func nativeProperty(name string) bool {
	switch name {
	case "fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-opacity", "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-dasharray", "stroke-dashoffset", "opacity", "color", "display", "visibility", "paint-order", "clip-path", "clip-rule", "mask", "mask-type", "stop-color", "stop-opacity":
		return true
	}
	return false
}

func nativeCSS(source string, inline bool) bool {
	p := css.NewParser(parse.NewInputString(source), inline)
	for {
		grammar, _, data := p.Next()
		switch grammar {
		case css.ErrorGrammar:
			return p.Err() == io.EOF
		case css.CommentGrammar, css.EndRulesetGrammar:
		case css.QualifiedRuleGrammar, css.BeginRulesetGrammar:
			var selector strings.Builder
			for _, token := range p.Values() {
				selector.Write(token.Data)
			}
			for _, part := range strings.Split(selector.String(), ",") {
				if !simpleSelector(strings.TrimSpace(part)) {
					return false
				}
			}
		case css.DeclarationGrammar:
			if !nativeProperty(strings.ToLower(string(data))) {
				return false
			}
			var value strings.Builder
			for _, token := range p.Values() {
				value.Write(token.Data)
			}
			if !nativeValue(value.String()) {
				return false
			}
		default:
			return false // at-rules, animation, variables and nested CSS
		}
	}
}

// simpleSelector: MyGo matches one compound type/id/class selector.
func simpleSelector(source string) bool {
	if source == "" {
		return false
	}
	for _, r := range source {
		if r != '*' && r != '.' && r != '#' && r != '-' && r != '_' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r < 128 {
			return false
		}
	}
	return true
}

func nativeValue(source string) bool {
	lower := strings.ToLower(source)
	if strings.ContainsAny(lower, "\\;") || strings.Contains(lower, "!important") {
		return false
	}
	l := css.NewLexer(parse.NewInputString(lower))
	for {
		kind, data := l.Next()
		if kind == css.ErrorToken {
			return l.Err() == io.EOF
		}
		switch kind {
		case css.FunctionToken:
			switch string(data) {
			case "rgb(", "rgba(", "hsl(", "hsla(":
			default:
				return false
			}
		case css.URLToken:
			value := strings.TrimSpace(string(data)[4 : len(data)-1])
			value = strings.Trim(value, "\"'")
			if !strings.HasPrefix(value, "#") {
				return false
			}
		case css.IdentToken:
			switch string(data) {
			case "initial", "unset", "revert", "revert-layer", "context-fill", "context-stroke":
				return false
			}
		}
	}
}
