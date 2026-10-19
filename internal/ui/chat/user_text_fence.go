package chat

import "strings"

// User attachment extraction and reference recognition share the same literal
// fence boundary. Closing fences must use the opening marker, be at least as
// long, and contain no trailing content. An unclosed fence stays literal.
type userTextFence struct {
	marker byte
	length int
}

func (f *userTextFence) consumeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	marker, count := byte(0), 0
	if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
		marker = trimmed[0]
		for count < len(trimmed) && trimmed[count] == marker {
			count++
		}
	}
	if f.marker != 0 {
		if marker == f.marker && count >= f.length && strings.TrimSpace(trimmed[count:]) == "" {
			f.marker = 0
		}
		return true
	}
	if count >= 3 {
		f.marker, f.length = marker, count
		return true
	}
	return false
}
