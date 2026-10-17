// Package composer is the multiline prompt input for M4–M6.
package composer

import (
	"strings"
	"unicode/utf8"
)

// SuggestKind is which completion popup is active.
type SuggestKind int

const (
	SuggestNone SuggestKind = iota
	SuggestSlash
	SuggestMention
)

// TokenRange is a byte range [Start, End) in the draft string.
type TokenRange struct {
	Start int
	End   int
}

// SlashToken returns the filter query and range when draft is a slash command line.
// The draft must start with '/' and the token runs until whitespace or end.
func SlashToken(draft string) (query string, tok TokenRange, ok bool) {
	if draft == "" || draft[0] != '/' {
		return "", TokenRange{}, false
	}
	// Only complete on the first line of the draft.
	if i := strings.IndexByte(draft, '\n'); i >= 0 {
		return "", TokenRange{}, false
	}
	end := len(draft)
	if j := strings.IndexFunc(draft, func(r rune) bool { return r == ' ' || r == '\t' }); j >= 0 {
		end = j
	}
	if end <= 1 {
		return draft[1:], TokenRange{0, end}, true
	}
	return draft[1:end], TokenRange{0, end}, true
}

// MentionToken finds the active @ mention at the end of the draft (caret at end).
func MentionToken(draft string) (query string, tok TokenRange, ok bool) {
	at := -1
	for i := 0; i < len(draft); {
		r, size := utf8.DecodeRuneInString(draft[i:])
		if r == '@' && mentionStartOK(draft, i) {
			at = i
		}
		i += size
	}
	if at < 0 {
		return "", TokenRange{}, false
	}
	// Ignore earlier mentions when the tail is already past this token.
	start := at + 1
	end := len(draft)
	for i := start; i < len(draft); {
		r, size := utf8.DecodeRuneInString(draft[i:])
		if r == ' ' || r == '\n' || r == '\t' {
			return "", TokenRange{}, false
		}
		i += size
	}
	return draft[start:end], TokenRange{at, end}, true
}

func mentionStartOK(draft string, at int) bool {
	if at == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(draft[:at])
	return prev == ' ' || prev == '\n' || prev == '\t'
}

// ActiveSuggest picks slash over mention when both could apply (draft starts with /).
func ActiveSuggest(draft string) (kind SuggestKind, query string, tok TokenRange, ok bool) {
	if q, tr, okSlash := SlashToken(draft); okSlash {
		return SuggestSlash, q, tr, true
	}
	if q, tr, okAt := MentionToken(draft); okAt {
		return SuggestMention, q, tr, true
	}
	return SuggestNone, "", TokenRange{}, false
}

// SlashCommand is one row in the / command popup.
type SlashCommand struct {
	Line        string // e.g. "/clear"
	Description string
	LocalAction string // "clear", "new", or "" for remote execute
}

// StubSlashCommands are used when commands/list is unavailable.
func StubSlashCommands() []SlashCommand {
	return []SlashCommand{
		{Line: "/clear", Description: "Clear draft", LocalAction: "clear"},
		{Line: "/new", Description: "New session", LocalAction: "new"},
	}
}

// MergeSlashCommands keeps stubs and appends remote names (deduped by line).
func MergeSlashCommands(stubs, remote []SlashCommand) []SlashCommand {
	seen := make(map[string]struct{}, len(stubs)+len(remote))
	out := make([]SlashCommand, 0, len(stubs)+len(remote))
	for _, c := range stubs {
		key := strings.ToLower(c.Line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	for _, c := range remote {
		key := strings.ToLower(c.Line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}

// FilterSlashCommands returns commands whose line (without leading /) matches query prefix.
func FilterSlashCommands(all []SlashCommand, query string, limit int) []SlashCommand {
	if limit <= 0 {
		limit = 8
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]SlashCommand, 0, limit)
	for _, c := range all {
		line := c.Line
		if !strings.HasPrefix(line, "/") {
			line = "/" + line
		}
		name := strings.TrimPrefix(strings.ToLower(line), "/")
		if q == "" || strings.HasPrefix(name, q) || strings.Contains(name, q) {
			copy := c
			copy.Line = line
			out = append(out, copy)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

// FilterPaths returns workspace-relative paths matching query by prefix (case-insensitive).
func FilterPaths(paths []string, query string, limit int) []string {
	if limit <= 0 {
		limit = 8
	}
	q := strings.ToLower(query)
	out := make([]string, 0, limit)
	for _, p := range paths {
		low := strings.ToLower(p)
		if q == "" || strings.HasPrefix(low, q) || strings.Contains(low, q) {
			out = append(out, p)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

// ReplaceToken replaces tok in draft with insert and returns the new draft.
func ReplaceToken(draft string, tok TokenRange, insert string) string {
	if tok.Start < 0 || tok.End > len(draft) || tok.Start > tok.End {
		return draft
	}
	return draft[:tok.Start] + insert + draft[tok.End:]
}

// NormalizeCommandLine ensures a leading slash.
func NormalizeCommandLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if !strings.HasPrefix(line, "/") {
		line = "/" + line
	}
	return line
}

// LocalSlashAction returns stub action name for an exact command line, or "".
func LocalSlashAction(line string, stubs []SlashCommand) string {
	line = NormalizeCommandLine(line)
	for _, c := range stubs {
		if strings.EqualFold(NormalizeCommandLine(c.Line), line) {
			return c.LocalAction
		}
	}
	return ""
}
