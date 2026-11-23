package imagepreview

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"path/filepath"
	"strings"
)

// BuildSVGImage supplies one already-authorized SVG to the existing isolated
// image document, as a data-URL <img>, never as document markup. Unlike the
// static resvg path this preserves browser secure-animated-image semantics:
// SMIL/CSS animations, foreignObject and embedded data resources may render;
// scripts, interactivity and external loads stay disabled by the image context
// and the caller's existing opaque DocumentShell/CSP/resource policy.
// This entry is deliberately separate from ResourceDataURL's OpenUI gate.
func BuildSVGImage(path string, data []byte) (string, error) {
	if strings.ToLower(filepath.Ext(path)) != ".svg" {
		return "", errors.New("SVG image document requires an SVG file")
	}
	if err := validateBrowserSVG(data); err != nil {
		return "", err
	}
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
	return imageViewer(html.EscapeString(filepath.Base(path)), source), nil
}

func validateBrowserSVG(data []byte) error {
	if len(data) == 0 || len(data) > MaxSVGBytes {
		return errors.New("SVG exceeds the 1 MiB byte budget")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth, nodes, tokens := 0, 0, 0
	seen, closed := false, false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.New("invalid SVG XML")
		}
		tokens++
		if tokens > 40000 {
			return errors.New("SVG token budget exceeded")
		}
		switch t := token.(type) {
		case xml.StartElement:
			if closed || depth >= 64 || nodes >= 10000 {
				return errors.New("SVG structure exceeds the preview budget")
			}
			if !seen {
				if t.Name.Local != "svg" || t.Name.Space != "" && t.Name.Space != svgNS {
					return errors.New("SVG root element is missing")
				}
				if err := svgBrowserDimensions(t.Attr); err != nil {
					return err
				}
				seen = true
			}
			depth++
			nodes++
			for _, attr := range t.Attr {
				if len(attr.Value) > 64<<10 {
					return errors.New("SVG attribute budget exceeded")
				}
				if svgNumericAttribute(strings.ToLower(attr.Name.Local)) {
					if err := svgNumbersBounded(attr.Value); err != nil {
						return err
					}
				}
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return errors.New("invalid SVG nesting")
			}
			if depth == 0 {
				closed = true
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(t)) != "" {
				return errors.New("content outside SVG root")
			}
		case xml.Directive:
			return errors.New("SVG document types and entities are not allowed")
		case xml.ProcInst:
			if t.Target != "xml" || seen {
				return errors.New("SVG processing instructions are not allowed")
			}
		}
	}
	if !seen || !closed || depth != 0 {
		return errors.New("incomplete SVG")
	}
	return nil
}
