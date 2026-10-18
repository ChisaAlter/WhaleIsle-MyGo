package imagepreview

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const svgNS = "http://www.w3.org/2000/svg"
const xlinkNS = "http://www.w3.org/1999/xlink"

var svgNumber = regexp.MustCompile(`[-+]?(?:[0-9]*\.)?[0-9]+(?:[eE][-+]?[0-9]+)?`)
var svgFunction = regexp.MustCompile(`[A-Za-z][A-Za-z0-9-]*[ \t\r\n]*\(`)
var svgID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:-]{0,127}$`)

type svgNode struct {
	children   []int
	references []string
}

// validateSVG permits bounded static vector/text artwork, gradients, local
// clips/masks and acyclic local references. It rejects scripts, animation,
// filters, nested images, foreign content, external resources and CSS escapes.
func validateSVG(data []byte) error {
	if len(data) > MaxSVGBytes {
		return errors.New("SVG exceeds the 1 MiB byte budget")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var nodes []svgNode
	var stack []int
	ids := map[string]int{}
	rootSeen, rootClosed := false, false
	styleText := map[int]string{}
	tokens := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.New("invalid SVG XML")
		}
		tokens++
		if tokens > 40_000 {
			return errors.New("SVG token budget exceeded")
		}
		switch t := token.(type) {
		case xml.StartElement:
			if len(stack) > 0 {
				if _, inStyle := styleText[stack[len(stack)-1]]; inStyle {
					return errors.New("SVG stylesheets cannot contain child elements")
				}
			}
			if rootClosed || len(nodes) >= 10_000 || len(stack) >= 64 {
				return errors.New("SVG structure exceeds the preview budget")
			}
			if t.Name.Space != "" && t.Name.Space != svgNS {
				return errors.New("SVG foreign namespaces are not supported")
			}
			name := strings.ToLower(t.Name.Local)
			switch name {
			case "svg", "g", "defs", "title", "desc", "path", "rect", "circle",
				"ellipse", "line", "polyline", "polygon", "text", "tspan", "textpath",
				"lineargradient", "radialgradient", "stop", "clippath", "mask", "use", "symbol", "style":
			default:
				return errors.New("SVG contains unsupported active or resource-heavy elements")
			}
			if !rootSeen {
				if name != "svg" {
					return errors.New("SVG root element is missing")
				}
				if err := svgDimensions(t.Attr); err != nil {
					return err
				}
				rootSeen = true
			} else if name == "svg" {
				return errors.New("nested SVG viewports are not supported")
			}
			index := len(nodes)
			nodes = append(nodes, svgNode{})
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				nodes[parent].children = append(nodes[parent].children, index)
			}
			stack = append(stack, index)
			for _, attr := range t.Attr {
				key := strings.ToLower(attr.Name.Local)
				value := strings.TrimSpace(attr.Value)
				if len(value) > 64<<10 {
					return errors.New("SVG attribute budget exceeded")
				}
				if attr.Name.Space == "xmlns" || key == "xmlns" {
					if value != svgNS && value != xlinkNS {
						return errors.New("SVG namespace is unsupported")
					}
					continue
				}
				if attr.Name.Space != "" && attr.Name.Space != xlinkNS && attr.Name.Space != "http://www.w3.org/XML/1998/namespace" {
					return errors.New("SVG attribute namespace is unsupported")
				}
				if strings.HasPrefix(key, "on") || key == "base" || key == "filter" ||
					key == "src" || strings.Contains(value, "\\") || strings.ContainsRune(value, 0) {
					return errors.New("SVG active or unbounded resources are not allowed")
				}
				if key == "id" {
					if !svgID.MatchString(value) {
						return errors.New("SVG id is unsupported")
					}
					if _, exists := ids[value]; exists {
						return errors.New("SVG duplicate ids are ambiguous")
					}
					ids[value] = index
				}
				if key == "href" {
					if !strings.HasPrefix(value, "#") || !svgID.MatchString(strings.TrimPrefix(value, "#")) {
						return errors.New("SVG references must stay inside this image")
					}
					nodes[index].references = append(nodes[index].references, value[1:])
				}
				if key == "style" {
					refs, err := svgDeclarations(value)
					if err != nil {
						return err
					}
					nodes[index].references = append(nodes[index].references, refs...)
				} else {
					refs, err := svgResourceValues(value)
					if err != nil {
						return err
					}
					nodes[index].references = append(nodes[index].references, refs...)
				}
				if svgNumericAttribute(key) {
					if err := svgNumbersBounded(value); err != nil {
						return err
					}
				}
			}
			if name == "style" {
				styleText[index] = ""
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return errors.New("invalid SVG nesting")
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				rootClosed = true
			}
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(t)) != "" {
					return errors.New("content outside SVG root")
				}
			} else {
				index := stack[len(stack)-1]
				if value, isStyle := styleText[index]; isStyle {
					value += string(t)
					if len(value) > 64<<10 {
						return errors.New("SVG stylesheet budget exceeded")
					}
					styleText[index] = value
				}
			}
		case xml.Directive:
			return errors.New("SVG document types and entities are not allowed")
		case xml.ProcInst:
			if t.Target != "xml" || rootSeen {
				return errors.New("SVG processing instructions are not allowed")
			}
		}
	}
	if !rootSeen || !rootClosed || len(stack) != 0 {
		return errors.New("incomplete SVG")
	}
	for _, css := range styleText {
		refs, err := svgStylesheet(css)
		if err != nil {
			return err
		}
		if len(refs) != 0 {
			return errors.New("SVG stylesheet resource references must be explicit element attributes")
		}
	}
	// Count the expanded local-reference graph, not merely the XML byte count.
	// Memoization is safe because each parent's expansion still adds each use.
	state, costs := make([]uint8, len(nodes)), make([]int, len(nodes))
	var cost func(int, int) (int, error)
	cost = func(index, depth int) (int, error) {
		if depth > 64 || state[index] == 1 {
			return 0, errors.New("SVG references are cyclic or too deep")
		}
		if state[index] == 2 {
			return costs[index], nil
		}
		state[index] = 1
		sum := 1
		targets := append([]int(nil), nodes[index].children...)
		for _, ref := range nodes[index].references {
			target, exists := ids[ref]
			if !exists {
				return 0, errors.New("SVG local reference is unresolved")
			}
			targets = append(targets, target)
		}
		for _, target := range targets {
			n, err := cost(target, depth+1)
			if err != nil {
				return 0, err
			}
			sum += n
			if sum > 50_000 {
				return 0, errors.New("SVG expanded resource budget exceeded")
			}
		}
		state[index], costs[index] = 2, sum
		return sum, nil
	}
	_, err := cost(0, 0)
	return err
}

func svgDimensions(attrs []xml.Attr) error {
	values := map[string]string{}
	for _, a := range attrs {
		values[strings.ToLower(a.Name.Local)] = strings.TrimSpace(a.Value)
	}
	var viewW, viewH float64
	if view := values["viewbox"]; view != "" {
		fields := strings.Fields(strings.ReplaceAll(view, ",", " "))
		if len(fields) != 4 {
			return errors.New("SVG viewBox is invalid")
		}
		numbers := make([]float64, 4)
		for i, field := range fields {
			n, err := strconv.ParseFloat(field, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1_000_000 {
				return errors.New("SVG viewBox is unbounded")
			}
			numbers[i] = n
		}
		viewW, viewH = numbers[2], numbers[3]
		if err := svgArea(viewW, viewH); err != nil {
			return err
		}
	}
	width, height := viewW, viewH
	if width == 0 {
		width, height = 300, 150
	}
	for i, key := range []string{"width", "height"} {
		value := values[key]
		if value == "" {
			continue
		}
		if strings.HasSuffix(value, "%") {
			if viewW <= 0 {
				return errors.New("SVG percentage dimensions require a bounded viewBox")
			}
			n, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
			if err != nil || n <= 0 || n > 100 {
				return errors.New("SVG percentage dimensions are unsupported")
			}
			continue
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(value, "px"), 64)
		if err != nil {
			return errors.New("SVG dimensions require pixels or a viewBox")
		}
		if i == 0 {
			width = n
		} else {
			height = n
		}
	}
	return svgArea(width, height)
}

func svgArea(width, height float64) error {
	if math.IsNaN(width) || math.IsNaN(height) || math.IsInf(width, 0) || math.IsInf(height, 0) ||
		width <= 0 || height <= 0 || width > 32768 || height > 32768 || width*height > float64(MaxPixels) {
		return errors.New("SVG dimensions exceed the pixel budget")
	}
	return nil
}

func svgNumericAttribute(key string) bool {
	switch key {
	case "d", "points", "transform", "gradienttransform", "patterntransform",
		"x", "y", "x1", "x2", "y1", "y2", "cx", "cy", "r", "rx", "ry", "dx", "dy",
		"width", "height", "font-size", "stroke-width", "stroke-dasharray", "stroke-dashoffset":
		return true
	}
	return false
}

func svgNumbersBounded(value string) error {
	for _, token := range svgNumber.FindAllString(value, -1) {
		n, err := strconv.ParseFloat(token, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 32768 {
			return errors.New("SVG geometry exceeds the numeric budget")
		}
	}
	return nil
}

func svgResourceValues(value string) ([]string, error) {
	if strings.ContainsAny(value, "\\\x00") || strings.Contains(value, "/*") || strings.Contains(value, "@") {
		return nil, errors.New("SVG escaped or imported resources are not supported")
	}
	for _, function := range svgFunction.FindAllString(value, -1) {
		name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(function, "(")))
		switch name {
		case "url", "rgb", "rgba", "hsl", "hsla", "matrix", "translate", "scale", "rotate", "skewx", "skewy":
		default:
			return nil, errors.New("SVG resource-producing and computed functions are unsupported")
		}
	}
	if strings.Contains(strings.ToLower(value), "data:") || strings.Contains(value, "://") {
		return nil, errors.New("SVG external and nested data resources are not allowed")
	}
	lower := strings.ToLower(value)
	var refs []string
	for {
		index := strings.Index(lower, "url(")
		if index < 0 {
			break
		}
		end := strings.Index(lower[index+4:], ")")
		if end < 0 {
			return nil, errors.New("invalid SVG resource reference")
		}
		end += index + 4
		ref := strings.TrimSpace(value[index+4 : end])
		if len(ref) > 1 && ((ref[0] == '\'' && ref[len(ref)-1] == '\'') || (ref[0] == '"' && ref[len(ref)-1] == '"')) {
			ref = ref[1 : len(ref)-1]
		}
		if !strings.HasPrefix(ref, "#") || !svgID.MatchString(ref[1:]) {
			return nil, errors.New("SVG external and nested data resources are not allowed")
		}
		refs = append(refs, ref[1:])
		value, lower = value[end+1:], lower[end+1:]
	}
	return refs, nil
}

func svgDeclarations(css string) ([]string, error) {
	if strings.ContainsAny(css, "{}\\\x00@") || strings.Contains(css, "/*") {
		return nil, errors.New("SVG CSS escapes, imports and nested rules are not supported")
	}
	var refs []string
	for _, declaration := range strings.Split(css, ";") {
		if strings.TrimSpace(declaration) == "" {
			continue
		}
		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid SVG CSS declaration")
		}
		key, value := strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1])
		switch key {
		case "fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-opacity",
			"stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-dasharray", "stroke-dashoffset",
			"opacity", "color", "font-size", "font-family", "font-style", "font-weight", "text-anchor",
			"dominant-baseline", "stop-color", "stop-opacity", "display", "visibility", "vector-effect",
			"paint-order", "clip-path", "clip-rule", "mask", "letter-spacing", "word-spacing", "text-decoration":
		default:
			return nil, errors.New("SVG stylesheet contains unsupported layout or active properties")
		}
		if len(value) > 2048 {
			return nil, errors.New("SVG CSS value exceeds the budget")
		}
		links, err := svgResourceValues(value)
		if err != nil {
			return nil, err
		}
		if err := svgNumbersBounded(value); err != nil {
			return nil, err
		}
		refs = append(refs, links...)
	}
	return refs, nil
}

func svgStylesheet(css string) ([]string, error) {
	var refs []string
	for strings.TrimSpace(css) != "" {
		open := strings.Index(css, "{")
		if open < 1 {
			return nil, errors.New("invalid SVG stylesheet")
		}
		selector := css[:open]
		if len(selector) > 2048 || strings.ContainsAny(selector, "\\@{}") || strings.Contains(selector, "/*") {
			return nil, errors.New("SVG selector exceeds the supported subset")
		}
		close := strings.Index(css[open+1:], "}")
		if close < 0 {
			return nil, errors.New("invalid SVG stylesheet")
		}
		close += open + 1
		links, err := svgDeclarations(css[open+1 : close])
		if err != nil {
			return nil, err
		}
		refs = append(refs, links...)
		css = css[close+1:]
	}
	return refs, nil
}
