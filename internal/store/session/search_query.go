package session

import "strings"

// SanitizeSessionSearchInput follows WorkspaceBrowser's controlled input: strip
// NUL and retain at most 500 UTF-16 units, without splitting a supplementary
// rune. Spaces remain editable so a user can type a multi-word query.
func SanitizeSessionSearchInput(query string) string {
	var sanitized strings.Builder
	units := 0
	for _, r := range query {
		if r == 0 {
			continue
		}
		n := 1
		if r > 0xffff {
			n = 2
		}
		if units+n > 500 {
			break
		}
		sanitized.WriteRune(r)
		units += n
	}
	return sanitized.String()
}

// NormalizeSessionSearchQuery is the one canonical query used by local matches,
// content-search requests and the batch-selection scope.
func NormalizeSessionSearchQuery(query string) string {
	return strings.TrimSpace(SanitizeSessionSearchInput(query))
}
