package theme

import (
	"fmt"
	"strings"
	"unicode"
)

// GeneralFontFamily restores an uncommitted General input after navigation,
// including a failed save, without projecting Appearance's independent draft.
func (e *Editor) GeneralFontFamily(role string) string {
	if value, ok := e.generalFontEdits[role]; ok {
		return value
	}
	switch role {
	case "text":
		return e.Saved.Interface.Family
	case "code":
		return e.Saved.Code.Family
	case "terminal":
		return e.Saved.Terminal.Family
	}
	return ""
}

// QueueGeneralFontEdit records typing only. It never persists per keystroke.
func (e *Editor) QueueGeneralFontEdit(role, value string) {
	if current, ok := e.generalFontEdits[role]; ok && current == value {
		return
	}
	if e.generalFontEdits == nil {
		e.generalFontEdits = make(map[string]string)
	}
	e.generalFontEdits[role] = value
	e.generalFontRevision++
}

// CommitGeneralFontEdit consumes one input on its explicit blur or Enter.
// Failure retains the pending value and prevents automatic retry on paints.
func (e *Editor) CommitGeneralFontEdit(role string) bool {
	value, ok := e.generalFontEdits[role]
	if !ok {
		return true
	}
	if !e.SaveFontFamily(role, value) {
		e.generalFontAttempted = e.generalFontRevision
		return false
	}
	if e.Pending == nil || !e.Pending() {
		delete(e.generalFontEdits, role)
	}
	return true
}

// FlushGeneralFontEdits is the General-leaving seam. An unchanged failed
// revision is attempted once automatically; explicit collapse/close may force
// one retry. It never saves any other Appearance or font drafts.
func (e *Editor) FlushGeneralFontEdits(force bool) bool {
	if len(e.generalFontEdits) == 0 {
		return true
	}
	if !force && e.generalFontAttempted == e.generalFontRevision {
		return false
	}
	e.generalFontAttempted = e.generalFontRevision
	for _, role := range []string{"text", "code", "terminal"} {
		if !e.CommitGeneralFontEdit(role) {
			return false
		}
	}
	return true
}

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
	next, draft := e.writeBaseline(), e.Draft.Clone()
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
	if !e.saveFontField(next, draft) {
		return false
	}
	if e.FlushWrites != nil {
		e.FlushWrites()
	}
	return true
}

// SaveFontFamily normalizes only an explicit General font edit, as the
// original FontFamilyRow does. Reading settings never rewrites raw families.
func (e *Editor) SaveFontFamily(role, family string) bool {
	family = normalizeFontFamily(family)
	next, draft := e.writeBaseline(), e.Draft.Clone()
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
	if !e.saveFontField(next, draft) {
		return false
	}
	if e.FlushWrites != nil {
		e.FlushWrites()
	}
	return true
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
	pending := e.Pending != nil && e.Pending()
	if pending {
		submitted := next.Clone()
		e.pendingSave = &submitted
	} else {
		e.Saved = next
	}
	e.Draft, e.Preview = draft, false
	SetPreferences(next)
	RefreshWindows()
	e.Status = "已保存 / Saved"
	if pending {
		e.Status = "正在保存… / Saving…"
	}
	return true
}

// Appearance's typography controls commit their own source-defined role only;
// a custom-theme draft stays with its editor until its separate Save.
func (e *Editor) SaveAppearanceFont(role string, value FontPreference) bool {
	next, draft := e.writeBaseline(), e.Draft.Clone()
	switch role {
	case "interface":
		next.Interface = value
		draft.Interface = value
		next.Chat.Family = value.Family
		draft.Chat.Family = value.Family
	case "code":
		next.Code = value
		draft.Code = value
	case "composer":
		next.Composer.Family = value.Family
		draft.Composer.Family = value.Family
	case "terminal":
		next.Terminal.Family = value.Family
		draft.Terminal.Family = value.Family
	default:
		return false
	}
	return e.saveFontField(next, draft)
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
