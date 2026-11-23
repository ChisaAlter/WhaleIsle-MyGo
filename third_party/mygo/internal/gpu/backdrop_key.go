package gpu

import (
	"hash/maphash"
	"unsafe"
)

var backdropSeed = maphash.MakeSeed()

// BackdropKeys returns, for each backdrop of the scene built, a hash of
// what is drawn under its area before the effect reading it: the instances
// whose clipped rectangles reach the area, with their batches' images and
// effects, and the keys of the backdrops of effects among them. A renderer
// that keeps the backdrop it computed can draw it again while the key, its
// area and what the key does not cover (the frame's size and clear color,
// the atlases and the images' contents) stay the same. keys is reused.
func (b *Builder) BackdropKeys(keys []uint64) []uint64 {
	keys = append(keys[:0], make([]uint64, len(b.Backdrops))...)
	for bi, batch := range b.Batches {
		if batch.Backdrop == 0 {
			continue
		}
		j := batch.Backdrop - 1
		keys[j] = b.backdropKey(bi, keys)
	}
	return keys
}

// backdropKey hashes what the batches before batch bi draw under the area
// of its backdrop; keys holds those of the backdrops before.
func (b *Builder) backdropKey(bi int, keys []uint64) uint64 {
	bk := b.Backdrops[b.Batches[bi].Backdrop-1]
	// A pixel blends what covers it; shapes are antialiased within a pixel
	// of their rectangles.
	ax0, ay0 := float32(bk.Area.Min.X-1), float32(bk.Area.Min.Y-1)
	ax1, ay1 := float32(bk.Area.Max.X+1), float32(bk.Area.Max.Y+1)
	var h maphash.Hash
	h.SetSeed(backdropSeed)
	word := func(v uint64) {
		var buf [8]byte
		for i := range buf {
			buf[i] = byte(v >> (8 * i))
		}
		h.Write(buf[:])
	}
	word(uint64(bk.Area.Min.X)<<32 | uint64(uint32(bk.Area.Min.Y)))
	word(uint64(bk.Area.Max.X)<<32 | uint64(uint32(bk.Area.Max.Y)))
	for _, batch := range b.Batches[:bi] {
		sc := batch.Scissor
		if batch.Count == 0 || float32(sc.Right) <= ax0 || float32(sc.Left) >= ax1 || float32(sc.Bottom) <= ay0 || float32(sc.Top) >= ay1 {
			continue
		}
		reached := false
		for i := batch.Start; i < batch.Start+batch.Count; i++ {
			in := &b.Instances[i]
			x0, y0 := max(in.Rect[0], in.Clip[0], float32(sc.Left)), max(in.Rect[1], in.Clip[1], float32(sc.Top))
			x1 := min(in.Rect[0]+in.Rect[2], in.Clip[0]+in.Clip[2], float32(sc.Right))
			y1 := min(in.Rect[1]+in.Rect[3], in.Clip[1]+in.Clip[3], float32(sc.Bottom))
			if x1 <= ax0 || x0 >= ax1 || y1 <= ay0 || y0 >= ay1 || x1 <= x0 || y1 <= y0 {
				continue
			}
			reached = true
			h.Write(unsafe.Slice((*byte)(unsafe.Pointer(in)), unsafe.Sizeof(*in)))
		}
		if !reached {
			continue
		}
		word(uint64(sc.Left)<<32 | uint64(uint32(sc.Top)))
		word(uint64(sc.Right)<<32 | uint64(uint32(sc.Bottom)))
		word(uint64(batch.Image))
		word(uint64(uintptr(unsafe.Pointer(batch.Effect))))
		if batch.Backdrop != 0 {
			word(keys[batch.Backdrop-1])
		}
	}
	return h.Sum64()
}
