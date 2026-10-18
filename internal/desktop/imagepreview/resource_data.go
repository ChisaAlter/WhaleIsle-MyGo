package imagepreview

import "net/http"

// ResourceDataURL validates immutable public image bytes before producing an
// inert image source. No server-provided MIME or filename selects a decoder.
// SVG is accepted only through the existing restrictive static-vector gate;
// this does not allow arbitrary data SVG supplied by an OpenUI program.
func ResourceDataURL(data []byte) (string, error) {
	ext := ".svg" // Non-bitmaps must pass the complete static SVG validator.
	switch http.DetectContentType(data) {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	case "image/bmp":
		ext = ".bmp"
	case "image/x-icon":
		ext = ".ico"
	default:
		if len(data) >= 12 && string(data[4:8]) == "ftyp" {
			ext = ".avif" // The existing validator checks brands and item layout.
		}
	}
	return imageDataURL("public-image"+ext, data)
}
