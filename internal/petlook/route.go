// Package petlook defines the pet's explicit image-model destination, separate
// from global vision fallback and the assistant's chat model.
package petlook

import (
	"strings"
	"unicode/utf16"
)

type Route struct{ Provider, Model, Label string }
type Catalog struct {
	Connected, Loading bool
	Err                string
	Routes             []Route
}

// NormalizeID follows pet-settings.js: trim and cap at 128 UTF-16 code units.
// Do not retain half a surrogate pair at the boundary.
func NormalizeID(value string) string {
	units := utf16.Encode([]rune(strings.TrimSpace(value)))
	if len(units) > 128 {
		units = units[:128]
		if units[127] >= 0xd800 && units[127] <= 0xdbff {
			units = units[:127]
		}
	}
	return string(utf16.Decode(units))
}

// Resolve requires an exact, image-capable catalog pair. A model-only legacy
// value is retained for display but cannot select a provider by implication.
func Resolve(provider, model string, catalog Catalog) bool {
	if provider == "" || model == "" || provider != NormalizeID(provider) || model != NormalizeID(model) || !catalog.Connected || catalog.Loading || catalog.Err != "" {
		return false
	}
	for _, route := range catalog.Routes {
		if route.Provider == provider && route.Model == model {
			return true
		}
	}
	return false
}
