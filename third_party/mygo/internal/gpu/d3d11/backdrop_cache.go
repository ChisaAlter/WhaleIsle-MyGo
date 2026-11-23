//go:build windows

package d3d11

import (
	"os"
	"unsafe"

	"github.com/egoist/mygo/internal/scene"
)

// backdropCacheOff turns the cache of backdrops off, for comparing frames
// with and without it: MYGO_BACKDROP_CACHE=off.
var backdropCacheOff = os.Getenv("MYGO_BACKDROP_CACHE") == "off"

// cachedBackdrop is a backdrop computed in an earlier frame, which the
// renderer draws again while what is under its area stays the same.
type cachedBackdrop struct {
	key   uint64
	bk    scene.Backdrop
	tex   texture
	valid bool
}

// backdropSalt is what the keys of backdrops (gpu.Builder.BackdropKeys) do
// not cover: the frame's size and clear color, and the contents of the
// atlases and images the instances sample.
func (r *Renderer) backdropSalt(s *scene.Scene) uint64 {
	h := uint64(s.Width)<<32 | uint64(uint32(s.Height))
	c := s.Clear
	h = h*0x9E3779B97F4A7C15 ^ (uint64(c.R) | uint64(c.G)<<8 | uint64(c.B)<<16 | uint64(c.A)<<24)
	h = h*0x9E3779B97F4A7C15 ^ r.mask.gen
	h = h*0x9E3779B97F4A7C15 ^ r.color.gen<<20
	h = h*0x9E3779B97F4A7C15 ^ r.imageEpoch
	if r.composed {
		h = ^h
	}
	return h
}

// backdropView returns the view of backdrop j of the frame: the one an
// earlier frame computed, when what is under its area did not change since,
// else one computed now from the back buffer (readBackdrop), which it keeps
// for the next frames.
func (r *Renderer) backdropView(j int, salt uint64) uintptr {
	bk := r.b.Backdrops[j]
	key := r.bdKeys[j] ^ salt
	for len(r.bdCache) <= j {
		r.bdCache = append(r.bdCache, cachedBackdrop{})
	}
	e := &r.bdCache[j]
	w, h := bk.Size()
	if !backdropCacheOff && e.valid && e.key == key && e.bk == bk && e.tex.w >= w && e.tex.h >= h {
		r.bdHits++
		return e.tex.srv
	}
	r.bdMisses++
	r.readBackdrop(bk)
	if backdropCacheOff {
		return r.backdrop[0].srv
	}
	e.valid = false
	if e.tex.w < w || e.tex.h < h {
		if err := r.newTarget(&e.tex, max(w+w/4, e.tex.w), max(h+h/4, e.tex.h), true); err != nil {
			return r.backdrop[0].srv
		}
	}
	from := box{Right: uint32(w), Bottom: uint32(h), Back: 1}
	call(r.ctx, ctxCopySubresourceRegion, e.tex.tex, 0, 0, 0, 0, r.backdrop[0].tex, 0, uintptr(unsafe.Pointer(&from)))
	e.key, e.bk, e.valid = key, bk, true
	return e.tex.srv
}

// trimBackdrops lets go of the backdrops kept beyond the n of the frame.
func (r *Renderer) trimBackdrops(n int) {
	for i := n; i < len(r.bdCache); i++ {
		e := &r.bdCache[i]
		free(&e.tex.rtv)
		free(&e.tex.srv)
		free(&e.tex.tex)
	}
	if n < len(r.bdCache) {
		clear(r.bdCache[n:])
		r.bdCache = r.bdCache[:n]
	}
}
