package terminal

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SelectionTarget is inert terminal output, never a shell command. Path lines
// are one-based; columns are deliberately ignored like the original opener.
type SelectionTarget struct {
	Text     string
	URL      bool
	Loopback bool
	Line     int
}

const maxTargetSelectionBytes = 64 << 10

var terminalURLPattern = regexp.MustCompile("(?i)https?://[^\\s\"'`<>]+")
var terminalSchemePattern = regexp.MustCompile("(?:^|[\\s\"'`<>(\\[{])[A-Za-z][A-Za-z0-9+.-]+:[^\\s\"'`<>]*")
var terminalPathPattern = regexp.MustCompile("(?:~/|\\.{1,2}/|/|[A-Za-z]:[\\\\/]|\\\\\\\\)[^\\s\"'`<>]+|[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)+(?::[0-9]+){0,2}")
var terminalPositionPattern = regexp.MustCompile(`^(.*?):([0-9]+)(?::[0-9]+)?$`)
var terminalWindowsPathPattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
var terminalHomePattern = regexp.MustCompile(`^(/(?:Users|home)/[^/]+|[A-Za-z]:\\Users\\[^\\]+)`)

func trimTargetPunctuation(value string) string {
	value = strings.TrimRight(value, ".,;!?")
	for _, pair := range [][2]string{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
		for strings.HasSuffix(value, pair[1]) && strings.Count(value, pair[1]) > strings.Count(value, pair[0]) {
			value = strings.TrimSuffix(value, pair[1])
		}
	}
	return value
}

// FirstSelectionTarget mirrors the original selection toolbar: first match in
// reading order, URL spans winning overlaps, punctuation outside links removed.
// Other URI schemes are not reinterpreted as local path fragments.
func FirstSelectionTarget(selection string) (SelectionTarget, bool) {
	if len(selection) > maxTargetSelectionBytes || strings.ContainsRune(selection, 0) {
		return SelectionTarget{}, false
	}
	bestStart := len(selection) + 1
	var best SelectionTarget
	for _, span := range terminalURLPattern.FindAllStringIndex(selection, -1) {
		if span[0] < bestStart {
			raw := trimTargetPunctuation(selection[span[0]:span[1]])
			best, bestStart = SelectionTarget{Text: raw, URL: true, Loopback: isTerminalLoopback(raw)}, span[0]
		}
	}
	// A scheme-like token (including an incomplete URI) blocks path fallback.
	// Require two scheme letters to preserve Windows drive paths. The leading
	// token boundary prevents src/file.go:12 from being mistaken for a scheme.
	schemes := terminalSchemePattern.FindAllStringIndex(selection, -1)
	schemeIndex := 0
	for _, span := range terminalPathPattern.FindAllStringIndex(selection, -1) {
		for schemeIndex < len(schemes) && schemes[schemeIndex][1] <= span[0] {
			schemeIndex++
		}
		overlaps := schemeIndex < len(schemes) && schemes[schemeIndex][0] < span[1]
		if overlaps || span[0] >= bestStart {
			continue
		}
		raw := trimTargetPunctuation(selection[span[0]:span[1]])
		if raw != "" {
			best, bestStart = SelectionTarget{Text: raw}, span[0]
		}
	}
	return best, bestStart <= len(selection)
}

func terminalHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (strings.ToLower(u.Scheme) != "http" && strings.ToLower(u.Scheme) != "https") {
		return nil, errors.New("Only HTTP(S) links without embedded credentials can be opened")
	}
	return u, nil
}

func isTerminalLoopback(raw string) bool {
	u, err := terminalHTTPURL(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	}
	return false
}

func resolveTerminalPath(target SelectionTarget, cwd string) (SelectionTarget, error) {
	if strings.TrimSpace(cwd) == "" {
		return SelectionTarget{}, errors.New("Terminal workspace directory is unavailable")
	}
	path := target.Text
	if match := terminalPositionPattern.FindStringSubmatch(path); len(match) == 3 {
		line, err := strconv.Atoi(match[2])
		if err != nil {
			return SelectionTarget{}, errors.New("Terminal file line is out of range")
		}
		target.Line, path = max(1, line), match[1]
	}
	join := func(base, next string) string {
		base = strings.TrimRight(base, `/\`)
		if terminalWindowsPathPattern.MatchString(base) || strings.HasPrefix(base, `\\`) {
			return base + `\` + strings.ReplaceAll(next, "/", `\`)
		}
		return base + "/" + strings.TrimLeft(next, "/")
	}
	if strings.HasPrefix(path, "~/") {
		if home := terminalHomePattern.FindString(cwd); home != "" {
			path = join(home, path[2:])
		}
	}
	if !strings.HasPrefix(path, "/") && !terminalWindowsPathPattern.MatchString(path) && !strings.HasPrefix(path, `\\`) {
		path = join(cwd, path)
	}
	target.Text = path
	return target, nil
}

// SelectionTargetActions are explicit UI actions. Valid must still own the
// initiating terminal/workspace/Host; a missing gate or handler fails closed.
type SelectionTargetActions struct {
	Valid        func() bool
	OpenBrowser  func(string) error
	OpenExternal func(string) error
	OpenPath     func(string, int) error
}

func OpenSelectionTarget(selection, cwd string, actions SelectionTargetActions) error {
	stale := errors.New("Terminal or workspace changed; select the target again")
	if actions.Valid == nil || !actions.Valid() {
		return stale
	}
	target, ok := FirstSelectionTarget(selection)
	if !ok {
		return errors.New("No supported URL or workspace path in this selection")
	}
	var open func() error
	if target.URL {
		if _, err := terminalHTTPURL(target.Text); err != nil {
			return err
		}
		handler := actions.OpenExternal
		if target.Loopback {
			handler = actions.OpenBrowser
		}
		if handler != nil {
			open = func() error { return handler(target.Text) }
		}
	} else {
		var err error
		target, err = resolveTerminalPath(target, cwd)
		if err != nil {
			return err
		}
		if actions.OpenPath != nil {
			open = func() error { return actions.OpenPath(target.Text, target.Line) }
		}
	}
	if open == nil {
		return errors.New("Terminal target opener is unavailable")
	}
	if !actions.Valid() {
		return stale
	}
	return open()
}
