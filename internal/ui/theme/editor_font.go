package theme

import (
	"fmt"
	"strings"
	"unicode"
)

// SaveFontSize is the General settings stepper's single-field write. Its
// text role is conversation content, independent of the interface scale.
func (e *Editor) SaveFontSize(role string, size float32) bool {
	lo, hi := float32(10), float32(20)
	if role == "text" {
		hi = 22
	}
	if size < lo || size > hi || size != float32(int(size)) {
		e.Status = fmt.Sprintf("Font size must be an integer from %.0f to %.0f", lo, hi)
		return false
	}
	next, draft := e.Saved.Clone(), e.Draft.Clone()
	switch role {
	case "text":
		next.Chat.Size, draft.Chat.Size = size, size
	case "code":
		next.Code.Size, draft.Code.Size = size, size
	case "terminal":
		next.Terminal.Size, draft.Terminal.Size = size, size
	default:
		return false
	}
	return e.saveFontField(next, draft)
}

// SaveFontFamily normalizes only an explicit General font edit, as the
// original FontFamilyRow does. Reading settings never rewrites raw families.
func (e *Editor) SaveFontFamily(role, family string) bool {
	family = normalizeFontFamily(family)
	next, draft := e.Saved.Clone(), e.Draft.Clone()
	switch role {
	case "text":
		next.Interface.Family, next.Chat.Family = family, family
		draft.Interface.Family, draft.Chat.Family = family, family
	case "code":
		next.Code.Family, draft.Code.Family = family, family
	case "terminal":
		next.Terminal.Family, draft.Terminal.Family = family, family
	default:
		return false
	}
	return e.saveFontField(next, draft)
}

func (e *Editor) saveFontField(next, draft Preferences) bool {
	if err := next.Validate(); err != nil {
		e.Status = err.Error()
		return false
	}
	if e.OnSave == nil {
		return false
	}
	if err := e.OnSave(next.Clone()); err != nil {
		e.Status = err.Error()
		return false
	}
	e.Saved, e.Draft, e.Preview = next, draft, false
	SetPreferences(next)
	RefreshWindows()
	e.Status = "已保存 / Saved"
	return true
}

// Mirrors ui-theme/theme-settings.ts normalizeFontFamily.
func normalizeFontFamily(value string) string {
	var families []string
	for _, name := range strings.Split(value, ",") {
		name = strings.Map(func(r rune) rune {
			if r == '"' || r == '\'' || r == '\\' || r == '<' || r == '>' || r < 0x20 || r == 0x7f {
				return -1
			}
			return r
		}, name)
		name = strings.Join(strings.FieldsFunc(name, unicode.IsSpace), " ")
		if name == "" {
			continue
		}
		switch strings.ToLower(name) {
		case "serif", "sans-serif", "monospace", "cursive", "fantasy", "system-ui", "ui-serif", "ui-sans-serif", "ui-monospace", "ui-rounded", "emoji", "math", "fangsong", "-apple-system", "blinkmacsystemfont":
		default:
			name = `"` + name + `"`
		}
		families = append(families, name)
	}
	return strings.Join(families, ", ")
}
