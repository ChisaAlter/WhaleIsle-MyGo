package avatarimage

import "image"

// ApplyJPEGOrientation reuses the avatar importer's checked EXIF transform for
// an already-decoded JPEG. The returned view is lazy: callers can downsample
// upright pixels without allocating a second full-size orientation canvas.
// The caller remains responsible for byte/pixel validation before decoding.
func ApplyJPEGOrientation(decoded image.Image, data []byte) image.Image {
	if decoded != nil {
		if o := orientation(data); o > 1 {
			return oriented{Image: decoded, orientation: o}
		}
	}
	return decoded
}
