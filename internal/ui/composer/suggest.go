// Package composer is the multiline prompt input for M4–M6.
package composer

import (
	"strings"
	"unicode"
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

// MentionToken finds an active mention at the end of the draft.
func MentionToken(draft string) (query string, tok TokenRange, ok bool) {
	return MentionTokenAt(draft, len(draft))
}

// MentionTokenAt follows the Host grammar at the current byte caret. Open
// quoted paths may contain spaces; complete canonical session mentions are not
// completion queries. Only the prefix before the caret is replaced.
func MentionTokenAt(draft string, caret int) (query string, tok TokenRange, ok bool) {
	if caret < 0 || caret > len(draft) || !utf8.ValidString(draft[:caret]) {
		return "", TokenRange{}, false
	}
	before := draft[:caret]
	for at := 0; at < len(before); at++ {
		if before[at] != '@' || !mentionStartOK(before, at) {
			continue
		}
		tail := before[at+1:]
		if strings.HasPrefix(tail, `"`) && !strings.Contains(tail[1:], `"`) {
			return tail[1:], TokenRange{at, caret}, true
		}
	}
	at := strings.LastIndexByte(before, '@')
	if at < 0 || !mentionStartOK(before, at) {
		return "", TokenRange{}, false
	}
	tail := before[at+1:]
	if strings.ContainsAny(tail, `"[]`) || strings.IndexFunc(tail, unicode.IsSpace) >= 0 {
		return "", TokenRange{}, false
	}
	return tail, TokenRange{at, caret}, true
}

// FormatFileMention uses the Host's quoted path grammar and rejects characters
// that cannot be represented safely. Directories leave the quote open to drill.
func FormatFileMention(path string, directory, preserveQuote bool) (string, bool) {
	if directory {
		path = strings.TrimSuffix(path, "/") + "/"
	}
	if path == "" {
		return "", false
	}
	for _, r := range path {
		if r == '"' || r < 0x20 || r >= 0x7f && r <= 0x9f {
			return "", false
		}
	}
	if preserveQuote || strings.IndexFunc(path, unicode.IsSpace) >= 0 {
		if directory {
			return `@"` + path, true
		}
		return `@"` + path + `"`, true
	}
	return "@" + path, true
}

func mentionStartOK(draft string, at int) bool {
	if at == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(draft[:at])
	return unicode.IsSpace(prev)
}

// ActiveSuggest picks slash over mention when both could apply (draft starts with /).
func ActiveSuggest(draft string) (kind SuggestKind, query string, tok TokenRange, ok bool) {
	return ActiveSuggestAt(draft, len(draft))
}

func ActiveSuggestAt(draft string, caret int) (kind SuggestKind, query string, tok TokenRange, ok bool) {
	if caret < 0 || caret > len(draft) {
		return SuggestNone, "", TokenRange{}, false
	}
	if q, tr, okSlash := SlashToken(draft[:caret]); okSlash {
		return SuggestSlash, q, tr, true
	}
	if q, tr, okAt := MentionTokenAt(draft, caret); okAt {
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
		{Line: "/model", Description: "Choose model and reasoning effort", LocalAction: "model"},
		{Line: "/permission", Description: "Choose permission preset", LocalAction: "permission"},
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
