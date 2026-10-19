package theme

import "strings"

// TextFamily mirrors base.css's custom text family followed by its built-in
// fallback stack. It changes drawing only; saved font preferences stay raw.
func TextFamily(family string) string { return appendFontFallback(family, DefaultTextFamily) }

// CodeFamily uses the separate built-in code stack, including its CJK tail.
// Terminal typography has its own contract and does not use this stack.
func CodeFamily(family string) string { return appendFontFallback(family, DefaultCodeFamily) }

func appendFontFallback(family, fallback string) string {
	family = strings.TrimSpace(family)
	if family == "" || family == fallback {
		return fallback
	}
	if strings.HasSuffix(family, ", "+fallback) {
		return family
	}
	return family + ", " + fallback
}

func renderPreferences(p Preferences) Preferences {
	p = p.Clone()
	p.Interface.Family = TextFamily(p.Interface.Family)
	p.Chat.Family = TextFamily(p.Chat.Family)
	p.Composer.Family = TextFamily(p.Composer.Family)
	p.Code.Family = CodeFamily(p.Code.Family)
	return p
}

// RenderPreferences returns the drawing snapshot prepared by SetPreferences.
// Family strings are composed once per preference change, not per text shape.
// CurrentPreferences remains the editable and persistable raw snapshot.
func RenderPreferences() Preferences {
	preferenceState.RLock()
	defer preferenceState.RUnlock()
	return preferenceState.rendered.Clone()
}
