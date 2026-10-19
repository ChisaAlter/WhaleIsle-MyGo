//go:build !windows || (!amd64 && !arm64)

package previewprocess

func (s *Session) PresentImageSurface(parent uintptr, x, y, w, h, cx, cy, cw, ch int) error {
	return s.PresentSurface(parent, x, y, w, h, cx, cy, cw, ch)
}

func (s *Session) PresentImageSurfaceClipped(parent uintptr, x, y, w, h, cx, cy, cw, ch int, _ ImageSurfaceClip) error {
	return s.PresentSurface(parent, x, y, w, h, cx, cy, cw, ch)
}
