package files

import (
	"errors"
	"net/url"
	"os"
	pathpkg "path"
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
	if path == "" || strings.ContainsRune(path, 0) {
		return false, errors.New("invalid file path")
	}
	if u, err := url.Parse(path); strings.Contains(path, "://") || (!hasLine && err == nil && u.Scheme != "" && len(u.Scheme) != 1) {
		return false, errors.New("file cards cannot open URLs")
	}
	if m.ReadRemoteContext != nil || m.ReadTextPageContext != nil {
		// A Host workspace need not exist on the desktop. The captured reader,
		// not local metadata, enforces filesystem and symlink authority.
		if hasLine {
			path = base
		}
		if rel, ok := hostPresentedPath(m.WorkspaceRoot, path); ok {
			prepareTabs(m)
			if rel == "." {
				capture(m)
				if w := tabsFor(m); w != nil {
					w.Active = ""
				}
				restore(m, &FileTab{})
			} else {
				selectFile(m, rel)
				if hasLine {
					RequestLine(m, line)
				}
			}
			return true, nil
		}
		return false, nil
	}
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

// PresentedPath exposes the existing line-address syntax to resource navigation.
// It does not resolve a path or grant file access; OpenPresentedPath retains its
// literal-local-filename preference and captured Host reader contract.
func PresentedPath(path string) (base string, line int, hasLine bool) {
	return presentedLineSuffix(path)
}

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

// hostPresentedPath is lexical and platform-neutral: a POSIX Host can be viewed
// from Windows and vice versa. It never confers access outside the selected root.
func hostPresentedPath(root, value string) (string, bool) {
	root = strings.ReplaceAll(strings.TrimSpace(root), `\`, "/")
	value = strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	if root == "" || value == "" || strings.ContainsRune(value, 0) || strings.Contains(value, "://") {
		return "", false
	}
	windows := len(root) > 2 && root[1] == ':' || strings.HasPrefix(root, "//")
	if !strings.HasPrefix(root, "/") && !(len(root) > 2 && root[1] == ':' && root[2] == '/') {
		return "", false
	}
	root = pathpkg.Clean(root)
	absolute := strings.HasPrefix(value, "/") || len(value) > 1 && value[1] == ':'
	if absolute {
		full := pathpkg.Clean(value)
		prefix := strings.TrimRight(root, "/") + "/"
		compare, base := full, prefix
		if windows {
			compare, base = strings.ToLower(compare), strings.ToLower(base)
		}
		equal := full == root
		if windows {
			equal = strings.EqualFold(full, root)
		}
		if equal {
			return ".", true
		}
		if !strings.HasPrefix(compare, base) {
			return "", false
		}
		value = full[len(prefix):]
	}
	value = pathpkg.Clean(value)
	if value == ".." || strings.HasPrefix(value, "../") || strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", false
	}
	return value, true
}

// WorkspaceRelativePath resolves a descriptor lexically across Host platforms.
// Actual read/write authority remains with the captured resource owner.
func WorkspaceRelativePath(root, value string) (string, bool) { return hostPresentedPath(root, value) }
