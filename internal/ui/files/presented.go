package files

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// OpenPresentedPath handles an explicit conversation-file click through the same
// tab lifecycle as the Files tree. WorkspaceRoot must be the initiating session's
// actual cwd, never a configured fallback or the application's working directory.
// An unhandled path is left unchanged for the existing Host opener.
func OpenPresentedPath(m Model, path string) (bool, error) {
	path = strings.TrimSpace(path)
	base, line, hasLine := presentedLineSuffix(path)
	// Prefer an existing literal filename such as "notes:12" or "notes#L12".
	handled, err := openPresentedPathExact(m, path, hasLine)
	if handled || !hasLine {
		return handled, err
	}
	handled, err = openPresentedPathExact(m, base, false)
	if handled {
		RequestLine(m, line)
	}
	return handled, err
}

var presentedLinePattern = regexp.MustCompile(`(?i)^(.*?)(?::([1-9][0-9]*)(?::[1-9][0-9]*)?|#L([1-9][0-9]*)(?:-L?[1-9][0-9]*)?)$`)

func presentedLineSuffix(path string) (string, int, bool) {
	if strings.Contains(path, "://") || strings.ContainsRune(path, 0) {
		return "", 0, false
	}
	match := presentedLinePattern.FindStringSubmatch(path)
	if len(match) != 4 || strings.TrimSpace(match[1]) == "" {
		return "", 0, false
	}
	value := match[2]
	if value == "" {
		value = match[3]
	}
	line, err := strconv.Atoi(value)
	if err != nil {
		return "", 0, false
	}
	return match[1], line, true
}

func openPresentedPathExact(m Model, path string, numericSuffix bool) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" || strings.ContainsRune(path, 0) {
		return false, errors.New("invalid file path")
	}
	if u, err := url.Parse(path); strings.Contains(path, "://") || (!numericSuffix && err == nil && u.Scheme != "" && len(u.Scheme) != 1) {
		return false, errors.New("file cards cannot open URLs")
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if !filepath.IsAbs(root) {
		return false, nil
	}
	// Foreign-platform paths must reach Host unchanged, not become local filenames.
	if !filepath.IsAbs(path) && (strings.HasPrefix(path, `\`) || (len(path) > 1 && path[1] == ':')) {
		return false, nil
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || outsideRoot(rel) {
		return false, nil
	}
	// Only claim files this local Files surface can resolve. This also prevents a
	// symlink within cwd from exposing a file outside the workspace authority.
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false, nil
	}
	realFile, err := filepath.EvalSymlinks(full)
	if err != nil {
		return false, nil
	}
	realRel, err := filepath.Rel(realRoot, realFile)
	if err != nil || outsideRoot(realRel) {
		return false, nil
	}
	info, err := os.Stat(full)
	if err != nil || (!info.Mode().IsRegular() && !(info.IsDir() && rel == ".")) {
		return false, nil
	}
	prepareTabs(m)
	if rel == "." {
		capture(m)
		if w := tabsFor(m); w != nil {
			w.Active = ""
		}
		restore(m, &FileTab{})
		return true, nil
	}
	selectFile(m, filepath.ToSlash(rel))
	return true, nil
}

func outsideRoot(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel)
}
