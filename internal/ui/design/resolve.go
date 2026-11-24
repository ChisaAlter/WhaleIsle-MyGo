package design

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// Scheme is the light or dark half of the tokens.
type Scheme uint8

const (
	Light Scheme = iota
	Dark
)

// Tokens are the resolved design tokens of one window: the coss palette of
// its scheme with the Host accent standing in for coss's neutral primary.
// Treat a *Tokens from For as read-only.
type Tokens struct {
	Palette
	Scheme Scheme
	// Glass reports whether the optional frosted appearance was on when the
	// tokens resolved; surfaces are opaque coss tokens when it is off.
	Glass bool
}

// IsDark reports whether these are the dark tokens.
func (t *Tokens) IsDark() bool { return t.Scheme == Dark }

// InvalidRingAlpha is destructive/16 in light and /24 in dark.
func (t *Tokens) InvalidRingAlpha() float32 {
	if t.Scheme == Dark {
		return InvalidRingAlphaDark
	}
	return InvalidRingAlphaLight
}

// Base returns the unmodified coss palette of scheme.
func Base(s Scheme) Palette {
	if s == Dark {
		return cossDark
	}
	return cossLight
}

// Resolve returns the tokens of scheme with accent as primary, ring and the
// sidebar's primary/ring. onAccent is the Host theme's on-accent foreground
// (--dsw-alias-label-primary-foreground): upstream DSH decides what reads on
// its accent, so it wins when present; a computed ReadableOn pick is the
// fallback for an empty foreground. A transparent accent keeps coss's own
// primary.
func Resolve(s Scheme, accent, onAccent ui.Color) Tokens {
	t := Tokens{Palette: Base(s), Scheme: s}
	if accent.A > 0 {
		accent.A = 255
		fg := onAccent
		if fg.A == 0 {
			fg = ReadableOn(accent)
		}
		t.Primary, t.PrimaryForeground, t.Ring = accent, fg, accent
		t.SidebarPrimary, t.SidebarPrimaryForeground, t.SidebarRing = accent, fg, accent
	}
	if g := glass.Load(); g != nil && g.enabled && g.filter != nil {
		t.Glass = true
		g.filter(s, &t.Palette)
	}
	return t
}

// ReadableOn returns a text color for the opaque fill bg with a contrast of
// at least 4.5:1: coss's own primary foregrounds when one qualifies, the
// higher-contrast of white and black otherwise (one of them always reaches
// 4.58:1).
func ReadableOn(bg ui.Color) ui.Color {
	light, dark := cossLight.PrimaryForeground, cossLight.Foreground
	cl, cd := Contrast(light, bg), Contrast(dark, bg)
	switch {
	case cl >= 4.5 && cl >= cd:
		return light
	case cd >= 4.5:
		return dark
	case cl >= 4.5:
		return light
	}
	white, black := ui.Color{R: 255, G: 255, B: 255, A: 255}, ui.Color{A: 255}
	if Contrast(white, bg) >= Contrast(black, bg) {
		return white
	}
	return black
}

// Contrast is the WCAG 2 contrast ratio of two opaque colors.
func Contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(c ui.Color) float64 {
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// SurfaceFilter rewrites surface tokens for the optional frosted appearance
// (translucent card, popover and background over the wallpaper).
type SurfaceFilter func(s Scheme, p *Palette)

type glassHook struct {
	enabled bool
	filter  SurfaceFilter
}

var glass atomicHook

// SetGlass is the single switch of the frosted-glass appearance theme. It
// is off by default: surfaces then use coss's opaque tokens. The glass
// theme registers its filter here; turning it on or off re-resolves every
// window's tokens on its next frame.
func SetGlass(enabled bool, filter SurfaceFilter) {
	glass.Store(&glassHook{enabled: enabled, filter: filter})
	cache.reset()
	tokensRev.Add(1)
}

var tokensRev atomic.Uint64

// Revision counts the changes to what For resolves: the glass hook and any
// window's bound scheme, accent or on-accent foreground. A memo key folding
// resolved tokens mixes it in, so a token change rebuilds the region.
func Revision() uint64 { return tokensRev.Load() }

// GlassEnabled reports whether the frosted appearance is on.
func GlassEnabled() bool {
	g := glass.Load()
	return g != nil && g.enabled && g.filter != nil
}

type atomicHook struct {
	mu sync.Mutex
	h  *glassHook
}

func (a *atomicHook) Load() *glassHook {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.h
}

func (a *atomicHook) Store(h *glassHook) {
	a.mu.Lock()
	a.h = h
	a.mu.Unlock()
}

// binding is what Bind pinned for a window.
type binding struct {
	scheme   Scheme
	accent   ui.Color
	onAccent ui.Color
}

type tokenCache struct {
	mu      sync.Mutex
	bound   map[*ui.Context]binding
	entries [4]cacheEntry
	next    int
}

type cacheEntry struct {
	key    binding
	tokens *Tokens
}

var cache tokenCache

func (c *tokenCache) reset() {
	c.mu.Lock()
	c.entries = [4]cacheEntry{}
	c.mu.Unlock()
}

// Bind pins the window's scheme, Host accent and on-accent foreground, which
// the app's appearance pass does every frame (cmd/whaleisle
// applyAppearanceTheme). Windows never bound resolve from their ui.Theme's
// Dark, Accent and AccentText instead.
func Bind(c *ui.Context, s Scheme, accent, onAccent ui.Color) {
	if c == nil {
		return
	}
	b := binding{s, accent, onAccent}
	cache.mu.Lock()
	if cache.bound == nil {
		cache.bound = map[*ui.Context]binding{}
	}
	if prev, ok := cache.bound[c]; !ok || prev != b {
		tokensRev.Add(1)
	}
	cache.bound[c] = b
	cache.mu.Unlock()
}

// Unbind forgets a closed window's binding.
func Unbind(c *ui.Context) {
	cache.mu.Lock()
	delete(cache.bound, c)
	cache.mu.Unlock()
}

// For returns the tokens of the window c builds: its bound scheme and
// accent, or its theme's. Results are cached; resolving costs nothing per
// frame once the scheme and accent settle.
func For(c *ui.Context) *Tokens {
	key := binding{}
	cache.mu.Lock()
	b, ok := cache.bound[c]
	cache.mu.Unlock()
	if ok {
		key = b
	} else if c != nil {
		if th := c.Theme(); th != nil {
			if th.Dark {
				key.scheme = Dark
			}
			key.accent = th.Accent
			key.onAccent = th.AccentText
		}
	}
	return Lookup(key.scheme, key.accent, key.onAccent)
}

// Lookup returns cached tokens for scheme, accent and on-accent foreground.
// The pointer stays valid after eviction; it is never written again.
func Lookup(s Scheme, accent, onAccent ui.Color) *Tokens {
	key := binding{s, accent, onAccent}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for i := range cache.entries {
		if e := &cache.entries[i]; e.tokens != nil && e.key == key {
			return e.tokens
		}
	}
	t := Resolve(s, accent, onAccent)
	cache.entries[cache.next] = cacheEntry{key: key, tokens: &t}
	cache.next = (cache.next + 1) % len(cache.entries)
	return &t
}
