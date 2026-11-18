package svgimage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/ui/i18n"
)

// Owner is supplied by the native window controller. ID binds window, Host and
// session generations; Valid must be safe on workers and never read UI locals.
type Owner struct {
	ID         string
	Window     *mygo.Window
	Context    context.Context
	HelperPath string
	Valid      func() bool
}

type Exclusion = previewprocess.ImageSurfaceShape

// Seat belongs to one immutable decoded image/thumbnail/lightbox, not an image
// URL cache. Its lease retires work after that particular UI seat disappears.
type Seat struct {
	width, height float32
	data          []byte
	mu            sync.Mutex
	ownerID       string
	generation    uint64
	lastPaint     time.Time
	pending       bool
	attempted     bool
	hidden        bool
	hiddenSince   time.Time
	openFn        func(ctx context.Context, owner *Owner, width, height float32, appearance string, valid func() bool) (*Surface, error)
	notify        func()
	surface       *Surface
	cancel        context.CancelFunc
	ctx           context.Context
	err           error
}

func NewSeat(data []byte) (*Seat, error) {
	w, h, err := imagepreview.SVGImageSize(data)
	if err != nil {
		return nil, err
	}
	return &Seat{width: w, height: h, data: bytes.Clone(data)}, nil
}
func (s *Seat) Size() (float32, float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.width, s.height
}

// Prepare only starts or observes work. Neither file reads nor helper startup
// runs on the UI thread. Paint, not a built-but-clipped element, renews authority.
func (s *Seat) Prepare(c *ui.Context, owner *Owner) {
	s.mu.Lock()
	s.notify = c.Services().Invalidate
	if owner == nil || owner.ID == "" || owner.Context == nil || owner.Window == nil || owner.Valid == nil {
		s.err = errors.New("isolated SVG image owner is unavailable")
		s.mu.Unlock()
		return
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		s.pending = false
		s.surface = nil
		s.err = nil
	}
	if s.ownerID != owner.ID {
		if s.cancel != nil {
			s.cancel()
		}
		if s.surface != nil {
			go s.surface.Close()
		}
		s.generation++
		s.ownerID = owner.ID
		s.surface = nil
		s.pending = false
		s.err = nil
		s.attempted = false
	}
	// Seat owns its live surface lease via explicit visibility transitions and
	// owner validity, keeping visible settled images at 0 Hz.
	if s.surface != nil {
		if s.surface.Reclaimed() {
			s.surface = nil
			s.pending = false
			s.err = nil
			s.attempted = false
		} else {
			select {
			case <-s.surface.Done():
				if s.surface.Reclaimed() {
					s.err = nil
					s.attempted = false
				} else {
					s.err = s.surface.Error()
					if s.err == nil {
						s.err = errors.New("isolated SVG image closed")
					}
				}
				s.surface = nil
				s.pending = false
			default:
			}
		}
	}
	if !owner.Valid() {
		if s.cancel != nil {
			s.cancel()
		}
		if surface := s.surface; surface != nil {
			s.surface = nil
			go surface.Close()
		}
		s.pending = false
		s.err = errors.New("SVG image owner is no longer active")
		s.mu.Unlock()
		return
	}
	if s.surface != nil || s.pending || s.err != nil {
		s.mu.Unlock()
		return
	}
	// A hidden or offscreen seat must not relaunch on unrelated view builds.
	// Its first visible paint invalidates once to resume with this same snapshot.
	if s.attempted && s.hidden {
		s.mu.Unlock()
		return
	}
	s.attempted = true
	ctx, cancel := context.WithCancel(owner.Context)
	s.ctx = ctx
	s.cancel = cancel
	s.pending = true
	s.hidden = false
	s.hiddenSince = time.Time{}
	s.generation++
	generation := s.generation
	// The first build grants startup visibility; offscreen reclaim clears this.
	s.lastPaint = time.Now()
	width, height := s.width, s.height
	s.mu.Unlock()
	appearance := "light"
	if c.Theme().Dark {
		appearance = "dark"
	}
	notify := c.Services().Invalidate
	valid := func() bool {
		if !owner.Valid() {
			cancel()
			return false
		}
		s.mu.Lock()
		hiddenExpired := s.hidden && !s.hiddenSince.IsZero() && time.Since(s.hiddenSince) >= defaultSVGOffscreenReclaimTTL
		current := s.generation == generation && s.ownerID == owner.ID && !hiddenExpired
		s.mu.Unlock()
		if !current {
			cancel()
		}
		return current
	}
	openingDone := make(chan struct{})
	go func() {
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-openingDone:
				return
			case <-tick.C:
				if !valid() {
					return
				}
			}
		}
	}()
	openSurface := s.openFn
	if openSurface == nil {
		openSurface = func(ctx context.Context, owner *Owner, width, height float32, appearance string, valid func() bool) (*Surface, error) {
			return Open(ctx, owner.Window, owner.HelperPath, s.data, int(width), int(height), appearance, valid)
		}
	}
	go func() {
		defer close(openingDone)
		surface, err := openSurface(ctx, owner, width, height, appearance, valid)
		s.mu.Lock()
		current := s.generation == generation && s.ownerID == owner.ID && ctx.Err() == nil
		if current {
			if surface != nil && err == nil {
				s.width, s.height = surface.Size()
			}
			s.pending = false
			s.surface = surface
			s.err = err
		}
		s.mu.Unlock()
		if !current && surface != nil {
			_ = surface.Close()
		}
		notify()
	}()
}

func (s *Seat) Error() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.surface != nil && s.surface.Error() != nil {
		return s.surface.Error()
	}
	return s.err
}

// LoadStatus distinguishes provisional geometry from a completed helper load.
// It contains no image bytes, paths or owner identifiers.
type LoadStatus struct {
	Pending, Loaded, Retired bool
	Generation               uint64
	PaintAgeMS               int64
}

func (s *Seat) LoadStatus() LoadStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := LoadStatus{Pending: s.pending, Loaded: s.surface != nil, Retired: s.ctx != nil && s.ctx.Err() != nil, Generation: s.generation}
	if !s.lastPaint.IsZero() {
		status.PaintAgeMS = time.Since(s.lastPaint).Milliseconds()
	}
	return status
}

// Paint supports the native image's existing Contain/Cover/zoom rectangle.
// The child is pointer-inert. Clipped/hidden seats retire rather than owning a
// stale process or accepting a late decode from another tab/attachment owner.
func (s *Seat) Paint(p *ui.Painter, r ui.Rect, fit ui.Fit) {
	s.PaintClipped(p, r, fit, 0, nil)
}

func (s *Seat) PaintClipped(p *ui.Painter, r ui.Rect, fit ui.Fit, radius float32, holes []Exclusion, backgrounds ...ui.Color) {
	clip := p.NativeSurfaceClip()
	visible := min(r.X+r.W, clip.X+clip.W) > max(r.X, clip.X) && min(r.Y+r.H, clip.Y+clip.H) > max(r.Y, clip.Y)
	s.mu.Lock()
	if visible {
		s.lastPaint = time.Now()
		s.hidden = false
		s.hiddenSince = time.Time{}
	} else if !s.hidden {
		s.hidden = true
		s.hiddenSince = time.Now()
	}
	if s.surface != nil && s.surface.Reclaimed() {
		s.surface = nil
		s.err = nil
		s.attempted = false
	}
	surface := s.surface
	width, height := s.width, s.height
	err := s.err
	restart := visible && surface == nil && !s.pending && err == nil && s.notify != nil
	notify := s.notify
	s.mu.Unlock()
	if !visible {
		if surface != nil {
			surface.Hide()
		}
		return
	}
	if surface == nil {
		if restart {
			notify()
		}
		message := "SVG…"
		if err != nil {
			message = i18n.T(i18n.KeySvgImageUnavailable)
		}
		p.Clip(r, 0, func() { p.Text(r.X+4, r.Y+4, message, 12, p.Theme().TextMuted) })
		return
	}
	factor := min(r.W/width, r.H/height)
	if fit == ui.Cover {
		factor = max(r.W/width, r.H/height)
	}
	if fit == ui.FillBox {
		factor = 0
	}
	imageRect := r
	if factor > 0 {
		imageRect = ui.Rect{X: r.X + (r.W-width*factor)/2, Y: r.Y + (r.H-height*factor)/2, W: width * factor, H: height * factor}
	}
	appearance := "light"
	if p.Theme().Dark {
		appearance = "dark"
	}
	bg := p.Theme().Background
	if len(backgrounds) > 0 {
		bg = backgrounds[0]
	}
	background := fmt.Sprintf("#%02x%02x%02x", bg.R, bg.G, bg.B)
	p.Clip(r, radius, func() {
		if e := surface.PaintClipped(p, imageRect, appearance, background, previewprocess.ImageSurfaceClip{Bounds: r, Radius: radius, Exclude: holes}); e != nil {
			s.mu.Lock()
			s.err = e
			s.mu.Unlock()
			// Build once so Prepare retires the surface and shows the error.
			if notify != nil {
				notify()
			}
		}
	})
}

func (s *Seat) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.generation++
	if s.cancel != nil {
		s.cancel()
	}
	surface := s.surface
	s.surface = nil
	s.pending = false
	s.mu.Unlock()
	if surface != nil {
		go surface.Close()
	}
}
