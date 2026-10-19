package imagepreview

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"io"
	"regexp"
	"strings"

	_ "golang.org/x/image/webp"
)

var nativeSVGProperties = regexp.MustCompile(`(?i)(^|[;{])\s*(filter|marker-start|marker-mid|marker-end)\s*:`)

// ValidateNativeSVG exposes the same resource and geometry gate used by the
// isolated static rasterizer, without rendering or acquiring resources.
func ValidateNativeSVG(data []byte) error { return validateNativeSVG(data) }

// The native rasterizer has no resource authority and a bounded WASM heap.
// Admit its static filters/embedded raster images without changing the stricter
// browser-document validator. A resource-free projection reuses all existing
// structure/geometry/reference limits; rendering always uses the original SVG.
func validateNativeSVG(data []byte) error {
	if len(data) > MaxSVGBytes {
		return errors.New("SVG exceeds the 1 MiB byte budget")
	}
	var projected bytes.Buffer
	encoder := xml.NewEncoder(&projected)
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var names []string
	var pixels int64
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
		if tokens > 40000 {
			return errors.New("SVG token budget exceeded")
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space != "" && t.Name.Space != svgNS {
				return errors.New("SVG foreign namespaces are unsupported")
			}
			original := strings.ToLower(t.Name.Local)
			name := original
			switch original {
			case "filter", "fegaussianblur", "feoffset", "fecolormatrix", "feblend", "fecomposite", "femerge", "femergenode", "feflood", "fecomponenttransfer", "fefuncr", "fefuncg", "fefuncb", "fefunca", "femorphology", "feturbulence", "fedisplacementmap", "feconvolvematrix", "fediffuselighting", "fespecularlighting", "fedistantlight", "fepointlight", "fespotlight", "fedropshadow", "pattern", "marker":
				name = "g"
			case "image", "feimage":
				name = "rect"
			case "svg":
				if len(names) > 0 {
					name = "g"
				}
			}
			t.Name = xml.Name{Local: name}
			var attrs []xml.Attr
			for _, a := range t.Attr {
				key := strings.ToLower(a.Name.Local)
				if a.Name.Space == "xmlns" || key == "xmlns" {
					if a.Value != svgNS && a.Value != xlinkNS {
						return errors.New("SVG namespace is unsupported")
					}
					continue
				}
				if a.Name.Space != "" && a.Name.Space != xlinkNS && a.Name.Space != "http://www.w3.org/XML/1998/namespace" {
					return errors.New("SVG attribute namespace is unsupported")
				}
				value := strings.TrimSpace(a.Value)
				if key == "href" && (original == "image" || original == "feimage") && strings.HasPrefix(value, "data:") {
					count, err := nativeSVGEmbeddedPixels(value)
					if err != nil {
						return err
					}
					pixels += count
					if pixels > MaxPixels {
						return errors.New("embedded SVG raster pixels exceed the budget")
					}
					continue
				}
				if key == "filter" {
					key = "clip-path"
				}
				if key == "style" {
					a.Value = nativeSVGProperties.ReplaceAllString(a.Value, "${1}clip-path:")
				}
				// Numeric filter kernels and lighting parameters must remain finite/bounded.
				switch key {
				case "stddeviation", "radius", "scale", "basefrequency", "numoctaves", "kernelmatrix", "order", "divisor", "bias", "surfacescale", "diffuseconstant", "specularconstant", "specularexponent", "azimuth", "elevation":
					if err := svgNumbersBounded(value); err != nil {
						return err
					}
				}
				a.Name = xml.Name{Local: key}
				attrs = append(attrs, a)
			}
			t.Attr = attrs
			names = append(names, name)
			token = t
		case xml.EndElement:
			if len(names) == 0 {
				return errors.New("invalid SVG nesting")
			}
			t.Name = xml.Name{Local: names[len(names)-1]}
			names = names[:len(names)-1]
			token = t
		case xml.CharData:
			if len(names) > 0 && names[len(names)-1] == "style" {
				token = xml.CharData(nativeSVGProperties.ReplaceAllString(string(t), "${1}clip-path:"))
			}
		}
		if err := encoder.EncodeToken(token); err != nil {
			return err
		}
		if projected.Len() > 2*MaxSVGBytes {
			return errors.New("SVG projection exceeds the byte budget")
		}
	}
	if err := encoder.Flush(); err != nil {
		return err
	}
	return validateSVG(projected.Bytes())
}

func nativeSVGEmbeddedPixels(raw string) (int64, error) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return 0, errors.New("invalid embedded SVG image")
	}
	header := raw[:comma]
	expected := ""
	switch header {
	case "data:image/png;base64":
		expected = "png"
	case "data:image/jpeg;base64":
		expected = "jpeg"
	case "data:image/gif;base64":
		expected = "gif"
	case "data:image/webp;base64":
		expected = "webp"
	default:
		return 0, errors.New("SVG embedded image must be a base64 PNG, JPEG, GIF or WebP")
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(raw[comma+1:])
	if err != nil {
		return 0, errors.New("invalid embedded SVG image encoding")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil || format != expected {
		return 0, errors.New("SVG embedded image type does not match its data")
	}
	count, err := rasterPixels(uint64(config.Width), uint64(config.Height))
	if err != nil {
		return 0, err
	}
	if format == "gif" {
		if err := validateGIF(payload); err != nil {
			return 0, err
		}
	}
	return count, nil
}
