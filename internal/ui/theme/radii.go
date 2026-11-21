package theme

import (
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Radii is the corner-radius token set the reference theme declares as
// --dsw-radius-* in base.css:26-40. Families override the tokens in their
// seeds' Overrides; defaults keep the original light values pixel for pixel.
type Radii struct {
	XS, SM, MD, LG, XL, Panel, Dialog, Compact, Inset, Badge, ButtonSM, Action, Button, Composer, ComposerFrame float32
}

// DefaultRadii is base.css:26-40 — the radius every element gets under a theme
// that does not override the token.
var DefaultRadii = Radii{XS: 4, SM: 8, MD: 12, LG: 16, XL: 20, Panel: 28, Dialog: 24, Compact: 6, Inset: 10, Badge: 13, ButtonSM: 14, Action: 15, Button: 18, Composer: 22, ComposerFrame: 26}

// ResolvedRadii is the radius table for the resolved theme and mode, like
// ResolvedSkinLayout resolves metrics: defaults first, then the selected
// family's seed Overrides for that mode, keeping only <number>px values.
func ResolvedRadii(id string, systemDark bool) Radii {
	dark := ResolveWithSystem(id, systemDark).Scheme == SchemeDark
	return resolvedRadii(selectedSeed(dark), dark)
}

// ContextRadii is the radius table of the context's mode, like ContextSkin is
// its palette: the selected family for the frame's theme mode.
func ContextRadii(c *ui.Context) Radii {
	return resolvedRadii(selectedSeed(c.Theme().Dark), c.Theme().Dark)
}

// radiiMemo holds the resolved table per mode. The key is the seed pointer:
// SetPreferences replaces the stored families wholesale, so a stale key cannot
// survive a preference write, and the memoized frame does not allocate.
var radiiMemo = struct {
	sync.Mutex
	seed [2]*Seed
	out  [2]Radii
}{}

func resolvedRadii(seed *Seed, dark bool) Radii {
	i := 0
	if dark {
		i = 1
	}
	radiiMemo.Lock()
	defer radiiMemo.Unlock()
	if radiiMemo.seed[i] == seed {
		return radiiMemo.out[i]
	}
	r := DefaultRadii
	if seed != nil {
		r.XS = pxRadius(seed.Overrides["--dsw-radius-xs"], r.XS)
		r.SM = pxRadius(seed.Overrides["--dsw-radius-sm"], r.SM)
		r.MD = pxRadius(seed.Overrides["--dsw-radius-md"], r.MD)
		r.LG = pxRadius(seed.Overrides["--dsw-radius-lg"], r.LG)
		r.XL = pxRadius(seed.Overrides["--dsw-radius-xl"], r.XL)
		r.Panel = pxRadius(seed.Overrides["--dsw-radius-panel"], r.Panel)
		r.Dialog = pxRadius(seed.Overrides["--dsw-radius-dialog"], r.Dialog)
		r.Compact = pxRadius(seed.Overrides["--dsw-radius-compact"], r.Compact)
		r.Inset = pxRadius(seed.Overrides["--dsw-radius-inset"], r.Inset)
		r.Badge = pxRadius(seed.Overrides["--dsw-radius-badge"], r.Badge)
		r.ButtonSM = pxRadius(seed.Overrides["--dsw-radius-button-sm"], r.ButtonSM)
		r.Action = pxRadius(seed.Overrides["--dsw-radius-action"], r.Action)
		r.Button = pxRadius(seed.Overrides["--dsw-radius-button"], r.Button)
		r.Composer = pxRadius(seed.Overrides["--dsw-radius-composer"], r.Composer)
		r.ComposerFrame = pxRadius(seed.Overrides["--dsw-radius-composer-frame"], r.ComposerFrame)
	}
	radiiMemo.seed[i] = seed
	radiiMemo.out[i] = r
	return r
}

// selectedSeed is the seed of the mode's selected family, as Preferences.Family
// resolves it for ResolvedSkinLayout: library first, then custom, then the
// library default when the id is gone. It reads the stored preferences without
// cloning, so the per-frame call does not allocate.
func selectedSeed(dark bool) *Seed {
	preferenceState.RLock()
	defer preferenceState.RUnlock()
	id := preferenceState.value.Light
	if dark {
		id = preferenceState.value.Dark
	}
	for i := range library {
		if library[i].ID == id {
			return familySeed(&library[i], dark)
		}
	}
	for i := range preferenceState.value.Custom {
		if preferenceState.value.Custom[i].ID == id {
			return familySeed(&preferenceState.value.Custom[i], dark)
		}
	}
	return familySeed(&library[0], dark)
}

func familySeed(f *Family, dark bool) *Seed {
	if dark {
		return &f.Dark
	}
	return &f.Light
}

// pxRadius keeps a token override that is exactly <number>px (spaces trimmed)
// and returns the fallback for anything else — percentages, calc(), inherit.
func pxRadius(raw string, fallback float32) float32 {
	v := strings.TrimSpace(raw)
	n, ok := strings.CutSuffix(v, "px")
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(n), 32)
	if err != nil {
		return fallback
	}
	return float32(f)
}
