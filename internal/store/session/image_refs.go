package session

import (
	"strings"

	"whaleisle/internal/markdownsyntax"
)

// ImageRefs returns local/remote image paths referenced by a message body
// (placeholder lines `[image] path` and markdown `![](path)`).
func ImageRefs(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, ref := range markdownsyntax.ImageReferences(body) {
		add(ref)
	}
	return out
}

// BodyWithoutImagePlaceholders strips `[image] …` lines for markdown rendering
// when images are shown inline separately.
func BodyWithoutImagePlaceholders(body string) string {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[image] ") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
