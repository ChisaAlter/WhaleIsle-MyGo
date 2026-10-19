//go:build windows && (amd64 || arm64)

package d3d11

import (
	"fmt"

	"github.com/egoist/mygo/internal/gpu/d3d11/roundclip"
	"github.com/egoist/mygo/internal/scene"
)

var (
	procFrameGetAncestor = user32.NewProc("GetAncestor")
	procFrameGetStyle    = user32.NewProc("GetWindowLongPtrW")
	procFrameDPI         = user32.NewProc("GetDpiForWindow")
)

type roundedFrame struct {
	clip   roundclip.Clip
	radius int
}

// NewComposedRounded uses the ordinary native swap chain and clips its visual
// to the window's alpha silhouette, without a new renderer or shader.
func NewComposedRounded(hwnd uintptr, radius int) (*Renderer, error) {
	r, err := NewComposed(hwnd)
	if err != nil {
		return nil, err
	}
	r.rounded = &roundedFrame{radius: radius}
	return r, nil
}

func (f *roundedFrame) apply(r *Renderer, s *scene.Scene) error {
	root, _, _ := procFrameGetAncestor.Call(r.hwnd, 2)        // GA_ROOT
	style, _, _ := procFrameGetStyle.Call(root, ^uintptr(15)) // GWL_STYLE=-16
	dpi, _, _ := procFrameDPI.Call(root)
	if dpi == 0 {
		dpi = 96
	}
	radius := float32(f.radius) * float32(dpi) / 96
	// Full screen removes WS_CAPTION; maximize keeps native WS_MAXIMIZE.
	if style&0x01000000 != 0 || style&0x00C00000 == 0 {
		radius = 0
	}
	changed, err := f.clip.Apply(r.dcomp, r.visual, float32(s.Width), float32(s.Height), radius)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if hr := call(r.dcomp, dcompCommit); failed(hr) {
		return fmt.Errorf("d3d11: committing rounded frame clip: HRESULT 0x%08X", uint32(hr))
	}
	return nil
}
