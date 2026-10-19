package market

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"

	"whaleisle/internal/desktop/imagepreview"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// README artwork has the same bounded static capability as Files/composer.
// Active, foreign and external-resource documents retain their authored alt.
func supportedReadmeSVG(data []byte) bool {
	return len(data) <= imagepreview.MaxSVGBytes && (fastReadmeSVG(data) || imagepreview.ValidateNativeSVG(data) == nil)
}

// The inexpensive vector path is retained only for artwork the native parser
// actually renders. Other accepted static SVGs use the shared WASM rasterizer.
func fastReadmeSVG(data []byte) bool {
	d := xml.NewDecoder(bytes.NewReader(data))
	var style strings.Builder
	var inStyle, root bool
	ignored := 0
	for {
		token, err := d.Token()
		if err == io.EOF {
			return root
		}
		if err != nil {
			return false
		}
		switch token := token.(type) {
		case xml.Directive:
			return false // DTD/entity and external stylesheet interpretation is absent.
		case xml.ProcInst:
			if token.Target != "xml" {
				return false
			}
		case xml.StartElement:
			if ignored > 0 {
				ignored++
				continue
			}
			if token.Name.Space != "" && token.Name.Space != "http://www.w3.org/2000/svg" {
				return false
			}
			if !root {
				if token.Name.Local != "svg" {
					return false
				}
				root = true
			}
			switch token.Name.Local {
			case "title", "desc", "metadata":
				ignored = 1 // Nonpainting author/editor metadata may use other namespaces.
				continue
			case "svg", "g", "a", "switch", "defs", "symbol", "use", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "linearGradient", "radialGradient", "stop", "clipPath", "mask":
			case "style":
				inStyle = true
				style.Reset()
			default:
				return false // Includes text/image/filter/pattern/marker/foreignObject/SMIL.
			}
			for _, attr := range token.Attr {
				if !readmeSVGAttribute(token.Name.Local, attr) {
					return false
				}
			}
		case xml.EndElement:
			if ignored > 0 {
				ignored--
			} else if token.Name.Local == "style" {
				inStyle = false
				if !supportedReadmeSVGCSS(style.String(), false) {
					return false
				}
			}
		case xml.CharData:
			if inStyle && ignored == 0 {
				style.Write(token)
			}
		}
	}
}

func readmeSVGAttribute(element string, attr xml.Attr) bool {
	if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
		return true
	}
	if attr.Name.Space == "http://www.w3.org/XML/1998/namespace" {
		return attr.Name.Local == "space" || attr.Name.Local == "lang"
	}
	if attr.Name.Space != "" && attr.Name.Space != "http://www.w3.org/1999/xlink" {
		return true // Editor metadata is ignored by both renderers.
	}
	name := attr.Name.Local
	if name == "style" {
		return supportedReadmeSVGCSS(attr.Value, true)
	}
	if readmeSVGProperty(name) {
		return readmeSVGValue(attr.Value)
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

func readmeSVGProperty(name string) bool {
	// Mirrors third_party/mygo/internal/svg/style.go's implemented properties.
	switch name {
	case "fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-opacity", "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-dasharray", "stroke-dashoffset", "opacity", "color", "display", "visibility", "paint-order", "clip-path", "clip-rule", "mask", "mask-type", "stop-color", "stop-opacity":
		return true
	}
	return false
}

func supportedReadmeSVGCSS(source string, inline bool) bool {
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
				if !readmeSVGSimpleSelector(strings.TrimSpace(part)) {
					return false
				}
			}
		case css.DeclarationGrammar:
			if !readmeSVGProperty(strings.ToLower(string(data))) {
				return false
			}
			var value strings.Builder
			for _, token := range p.Values() {
				value.Write(token.Data)
			}
			if !readmeSVGValue(value.String()) {
				return false
			}
		default:
			return false // At-rules, animation, variables and nested CSS are not rendered.
		}
	}
}

func readmeSVGSimpleSelector(source string) bool {
	if source == "" {
		return false
	}
	// MyGo only matches one compound type/id/class selector. Reject escapes,
	// combinators, attribute selectors and pseudo-classes instead of dropping them.
	for _, r := range source {
		if r != '*' && r != '.' && r != '#' && r != '-' && r != '_' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r < 128 {
			return false
		}
	}
	return true
}

func readmeSVGValue(source string) bool {
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
