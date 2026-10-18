//go:build !windows || (!amd64 && !arm64)

package previewprocess

import "errors"

func (s *Session) PresentSurface(parent uintptr, x, y, w, h, cx, cy, cw, ch int) error {
	return errors.New("inline isolated preview requires Windows amd64 or arm64")
}
