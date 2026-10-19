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

// NeedsBrowserSVG reports SVGs outside the bounded static native renderer's
// actual support. The caller retains the original snapshot/owner when choosing
// its already-existing isolated image-document surface.
func NeedsBrowserSVG(data []byte) bool {
	if validateBrowserSVG(data) != nil {
		return false
	}
	if validateNativeSVG(data) != nil {
		return true
	}
	if _, _, exact, err := svgIntrinsicHint(data); err == nil && !exact {
		return true
	}
	// resvg's static image renderer cannot play an embedded raster timeline.
	// Preserve GIF resources in their original image context, and route WebP
	// sequences when their actual bounded data carries animation metadata.
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := d.Token()
		if err != nil {
			return false
		}
		element, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		// A stylesheet that passed validateNativeSVG contains only the
		// static renderer's supported properties, never width/height/layout.
		// Unknown geometry or font-relative root dimensions already selected
		// the browser above. Do not displace supported static CSS artwork.
		for _, a := range element.Attr {
			if a.Name.Local != "href" {
				continue
			}
			source := strings.TrimSpace(a.Value)
			if strings.HasPrefix(source, "data:image/gif;") {
				return true
			}
			if strings.HasPrefix(source, "data:image/webp;base64,") {
				payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(source, "data:image/webp;base64,"))
				if err == nil && IsAnimatedWebP(payload) {
					return true
				}
			}
		}
	}
}

// SVGImageSize returns an absolute-unit layout hint without starting a helper.
// The live image's browser-reported natural dimensions replace this hint after
// decode, including default object sizing, percentages and font/CSS lengths.
func SVGImageSize(data []byte) (float32, float32, error) {
	if err := validateBrowserSVG(data); err != nil {
		return 0, 0, err
	}
	w, h, _, err := svgIntrinsicHint(data)
	return w, h, err
}

// SVGSurfaceHTML is fixed image-only markup for the existing isolated inline
// helper. Supplied bytes occur only inside a base64 image source. The child's
// additional CSP disables document scripts even in that helper's general shell;
// its pre-existing trusted readiness bootstrap is installed before this markup.
func SVGSurfaceHTML(data []byte) (string, error) {
	if err := validateBrowserSVG(data); err != nil {
		return "", err
	}
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
	return `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"><style>html,body{margin:0!important;width:100%;height:100%;overflow:hidden;background:transparent!important}img{display:block;width:100vw;height:100vh;object-fit:contain;pointer-events:none;user-select:none;-webkit-user-drag:none}</style><img alt="" draggable="false" referrerpolicy="no-referrer" src="` + source + `">`, nil
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
