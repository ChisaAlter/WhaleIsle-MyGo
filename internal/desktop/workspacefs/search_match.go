package workspacefs

import (
	"strings"
	"unicode"
)

// Match the original project file picker: ignore a leading mention/path prefix
// and whitespace, then find the query characters in order within the path.
func normalizeFileSearch(query string) string {
	query = strings.TrimLeft(strings.TrimSpace(query), "@./")
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, query)
}

func matchesFileSearch(path, query string) bool {
	needle := []rune(query)
	if len(needle) == 0 {
		return true
	}
	at := 0
	for _, ch := range path {
		if unicode.ToLower(ch) == needle[at] {
			at++
			if at == len(needle) {
				return true
			}
		}
	}
	return false
}
