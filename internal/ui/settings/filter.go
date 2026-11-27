package settings

import (
	"bytes"
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

var (
	secretTokAPIKey   = []byte("apikey")
	secretTokToken    = []byte("token")
	secretTokSecret   = []byte("secret")
	secretTokPassword = []byte("password")
	secretTokPasswd   = []byte("passwd")
)

func SecretLikeKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return false
	}
	var stack [64]byte
	compact := stack[:0]
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		if c == '-' || c == '_' || c == '.' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		compact = append(compact, c)
	}
	return bytes.Contains(compact, secretTokAPIKey) ||
		bytes.Contains(compact, secretTokToken) ||
		bytes.Contains(compact, secretTokSecret) ||
		bytes.Contains(compact, secretTokPassword) ||
		bytes.Contains(compact, secretTokPasswd)
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
