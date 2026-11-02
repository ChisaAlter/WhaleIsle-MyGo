package markdown

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var fileLineFragment = regexp.MustCompile(`^L([1-9][0-9]*)(?:-L([1-9][0-9]*))?$`)
var fileColonLine = regexp.MustCompile(`^(.*):([1-9][0-9]*)$`)
var destinationScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// ParseFileLink accepts authored local Markdown destinations, optionally with a
// #L24, #L24-L30 or :24 location. URLs, UNC paths, queries and malformed ranges
// stay inert. Percent escapes decode once before the existing file opener runs.
func ParseFileLink(value string) (path string, line int, ok bool) {
	destination, fragment, hasFragment := strings.Cut(value, "#")
	if strings.Contains(destination, "?") {
		return "", 0, false
	}
	path, err := url.PathUnescape(destination)
	if err != nil || path == "" || strings.HasPrefix(path, "//") || strings.HasPrefix(path, `\\`) {
		return "", 0, false
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return "", 0, false
		}
	}
	if hasFragment {
		match := fileLineFragment.FindStringSubmatch(fragment)
		if match == nil {
			return "", 0, false
		}
		line, err = strconv.Atoi(match[1])
		if err != nil || line <= 0 || int64(line) > 9007199254740991 {
			return "", 0, false
		}
		if match[2] != "" {
			end, err := strconv.Atoi(match[2])
			if err != nil || end < line || int64(end) > 9007199254740991 {
				return "", 0, false
			}
		}
	} else if match := fileColonLine.FindStringSubmatch(path); match != nil {
		path = match[1]
		line, err = strconv.Atoi(match[2])
		if err != nil || path == "" || int64(line) > 9007199254740991 {
			return "", 0, false
		}
	}
	windowsPath := len(path) >= 3 && path[1] == ':' && (path[2] == '/' || path[2] == '\\')
	if destinationScheme.MatchString(path) && !windowsPath {
		return "", 0, false
	}
	return path, line, true
}

func isDeliverableDocumentPath(value string) bool {
	path, _, ok := ParseFileLink(value)
	if !ok {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc", ".pdf", ".xlsx", ".xls", ".pptx", ".ppt", ".md", ".txt", ".csv", ".html", ".htm":
		return true
	default:
		return false
	}
}

func collectFileTargets(st *Style, targets map[int]string) {
	if st.Streaming {
		return
	}
	if st.OnOpenFile != nil && st.ResolveFile != nil {
		st.codeSpanAt = func(index int, value string) {
			if path := st.ResolveFile(value); path != "" {
				targets[index] = path
			}
		}
	}
	if st.OnOpenFile == nil && st.OnOpenURL != nil {
		return
	}
	st.fileSpanAt = func(index int, value string) {
		path, line, ok := ParseFileLink(value)
		if !ok {
			return
		}
		if st.OnOpenFile == nil && !isDeliverableDocumentPath(value) {
			return
		}
		if line > 0 {
			path += "#L" + strconv.Itoa(line)
		}
		targets[index] = path
	}
}
