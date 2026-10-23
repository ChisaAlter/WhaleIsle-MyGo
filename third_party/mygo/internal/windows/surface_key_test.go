//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

type surfaceKeyHandler struct {
	platform.WindowHandler
	taken bool
	last  platform.SurfaceEvent
}

func (h *surfaceKeyHandler) SurfaceEvent(ev platform.SurfaceEvent) bool {
	h.last = ev
	return h.taken
}

func TestSurfaceConsumedSystemKeySuppressesDefaultWithActiveTextInput(t *testing.T) {
	for _, active := range []bool{false, true} {
		for _, consumed := range []bool{false, true} {
			h := &surfaceKeyHandler{taken: consumed}
			s := &surface{w: &window{opts: &platform.WindowOptions{}, h: h}, input: platform.TextInputState{Active: active}}
			_, handled := s.message(0, wmSysKeyDown, 0x1B, 1<<29|1<<16|1)
			if handled != consumed || h.last.Kind != platform.KeyPressed || h.last.Key != platform.KeyEscape {
				t.Fatalf("active=%v consumed=%v: handled=%v event=%+v", active, consumed, handled, h.last)
			}
			for _, key := range []uintptr{0x73, 0x20, 0x79} { // F4, Space, F10
				h.taken = false
				if _, handled := s.message(0, wmSysKeyDown, key, 1); handled {
					t.Fatalf("active=%v: unhandled system key %#x lost its native action", active, key)
				}
			}
		}
	}
}
