package pagekit

import (
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/ui/i18n"
)

// Memo versions. A memo region's version must change with everything its
// build reads (ui.Element.Memo): the store revisions, the locale and, for
// regions that draw pictures, the media handles they asked for. Mix folds
// those into one uint64 without allocating.

const (
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

// Version accumulates the inputs of a memo region.
type Version uint64

// NewVersion starts a version from the active locale, which every page's
// copy depends on.
func NewVersion() Version { return Version(fnvOffset).String(i18n.Locale()) }

// Uint mixes n in.
func (v Version) Uint(n uint64) Version {
	h := uint64(v)
	for i := 0; i < 8; i++ {
		h ^= n & 0xff
		h *= fnvPrime
		n >>= 8
	}
	return Version(h)
}

// Bool mixes b in.
func (v Version) Bool(b bool) Version {
	if b {
		return v.Uint(1)
	}
	return v.Uint(2)
}

// String mixes s in.
func (v Version) String(s string) Version {
	h := uint64(v)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime
	}
	h ^= 0xff
	h *= fnvPrime
	return Version(h)
}

// Float mixes a size or position in, to the 1/64 DIP.
func (v Version) Float(f float32) Version { return v.Uint(uint64(int64(f * 64))) }

// Key is the version as Memo takes it.
func (v Version) Key() uint64 { return uint64(v) }

// MediaWatch notices pictures a memo region asked for while they were
// still loading: a kept region does not build again, so it would not see
// the picture arrive. The region Tracks each pending handle it draws; the
// page mixes Epoch into the region's version every frame, which moves when
// any tracked handle settled (the media owner's notify woke the frame).
// Keep one per page with ui.Local, outside the regions.
type MediaWatch struct {
	pending []*media.Handle
	epoch   uint64
}

// Track notes h if it is still pending.
func (w *MediaWatch) Track(h *media.Handle) {
	if w == nil || h == nil {
		return
	}
	if h.State() != media.Pending {
		return
	}
	for _, p := range w.pending {
		if p == h {
			return
		}
	}
	w.pending = append(w.pending, h)
}

// Epoch returns a counter that moves each time a tracked handle settled
// since the last call, forgetting the settled ones.
func (w *MediaWatch) Epoch() uint64 {
	if w == nil {
		return 0
	}
	kept := w.pending[:0]
	for _, h := range w.pending {
		if h.State() == media.Pending {
			kept = append(kept, h)
		} else {
			w.epoch++
		}
	}
	clear(w.pending[len(kept):])
	w.pending = kept
	return w.epoch
}
