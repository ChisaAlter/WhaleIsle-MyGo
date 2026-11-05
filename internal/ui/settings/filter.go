package settings

import (
	"strings"

	store "whaleisle/internal/store/settings"
)

// FilterNamespaces keeps namespace ids that match query (case-insensitive).
// Empty/whitespace query returns names unchanged. A match is a substring of
// the namespace id, its plan section id, or the localized section title.
func FilterNamespaces(names []string, query string) []string {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return names
	}
	out := make([]string, 0, len(names))
	for _, ns := range names {
		if namespaceMatchesQuery(ns, q) {
			out = append(out, ns)
		}
	}
	return out
}

func namespaceMatchesQuery(ns, q string) bool {
	if strings.Contains(strings.ToLower(ns), q) {
		return true
	}
	sec := SectionForNamespace(ns)
	if strings.Contains(strings.ToLower(string(sec)), q) {
		return true
	}
	return strings.Contains(strings.ToLower(sectionLabel(sec)), q)
}

// SecretLikeKey reports password-style field names (apiKey / token / secret).
func secretField(field store.Field) bool {
	return field.Role != "credential-ref" && field.Key != "apiKeyEnv" && (SecretLikeKey(field.Key) || SecretLikeKey(field.Label))
}

func SecretLikeKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	compact := strings.NewReplacer("-", "", "_", "", ".", "").Replace(k)
	if strings.Contains(compact, "apikey") ||
		strings.Contains(compact, "token") ||
		strings.Contains(compact, "secret") ||
		strings.Contains(compact, "password") ||
		strings.Contains(compact, "passwd") {
		return true
	}
	seg := k
	if i := strings.LastIndex(k, "."); i >= 0 {
		seg = k[i+1:]
	}
	switch strings.NewReplacer("-", "", "_", "").Replace(seg) {
	case "apikey", "token", "secret", "password", "passwd":
		return true
	}
	return false
}

// MultilineStringField is the generic-form heuristic for description-heavy strings.
func MultilineStringField(f store.Field) bool {
	if f.Kind != store.FieldString {
		return false
	}
	if len([]rune(f.Description)) >= 80 {
		return true
	}
	hay := strings.ToLower(f.Key + " " + f.Label)
	tokens := []string{
		"description", "prompt", "instruction", "instructions",
		"template", "systemprompt", "markdown", "message", "comment", "note", "body",
	}
	for _, tok := range tokens {
		if strings.Contains(hay, tok) {
			return true
		}
	}
	return false
}
