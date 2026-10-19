package previewprocess

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/egoist/mygo"
)

// ImageSize is trusted startup metadata from this same contained image
// session, read after decode. It never reopens bytes or starts a second helper.
func (s *Session) ImageSize() (float64, float64) { return s.imageWidth, s.imageHeight }

func (s *Session) acceptImageDimensions(d Document, loaded packet) error {
	if !d.InertImage {
		if loaded.ImageWidth != 0 || loaded.ImageHeight != 0 {
			return errors.New("ordinary document returned image metadata")
		}
		return nil
	}
	if !boundedImageDimensions(loaded.ImageWidth, loaded.ImageHeight) {
		return errors.New("preview helper did not return bounded image intrinsic dimensions")
	}
	s.imageWidth, s.imageHeight = loaded.ImageWidth, loaded.ImageHeight
	return nil
}

func boundedImageDimensions(width, height float64) bool {
	return width > 0 && height > 0 && width <= 32768 && height <= 32768 && width*height <= 32_000_000
}

// The opaque child reports numbers through its existing trusted readiness
// bootstrap. Eval reads only the outer shell's state: it cannot inspect the
// opaque iframe DOM, execute SVG code or choose an arbitrary script.
func inertImageDimensions(ctx context.Context, window *mygo.Window, d Document) (float64, float64, error) {
	if !d.InertImage {
		return 0, 0, nil
	}
	value, err := window.Page().EvalContext(ctx, "({width:window.__previewState?.imageWidth||0,height:window.__previewState?.imageHeight||0})")
	if err != nil {
		return 0, 0, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return 0, 0, err
	}
	var size struct{ Width, Height float64 }
	if err := json.Unmarshal(raw, &size); err != nil {
		return 0, 0, err
	}
	if !boundedImageDimensions(size.Width, size.Height) {
		return 0, 0, errors.New("image decoded without bounded intrinsic dimensions")
	}
	return size.Width, size.Height, nil
}
