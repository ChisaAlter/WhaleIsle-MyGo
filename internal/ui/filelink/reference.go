// Package filelink carries in-app workspace references, never OS file uploads.
package filelink

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Reference is an internal drag value bound to its originating Host/session.
// Roots are opaque identities; they are never interpreted on the local machine.
type Reference struct{ BindingID, SessionID, WorkspaceRoot, Path string }

func (r Reference) Matches(binding, session, root string) bool {
	return binding != "" && session != "" && root != "" && r.BindingID == binding && r.SessionID == session && r.WorkspaceRoot == root
}

// Serialize follows ui-files/composerMention.ts serializeComposerFileLink.
// The tree only produces workspace-relative file paths. Reject ambiguous roots,
// traversal and control characters without reading or resolving any local path.
func Serialize(path string) (string, bool) {
	if path == "" || !utf8.ValidString(path) || strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\") {
		return "", false
	}
	parts := strings.Split(path, "/")
	if strings.Contains(parts[0], ":") {
		return "", false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return "", false
		}
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	name := parts[len(parts)-1]
	name = strings.NewReplacer("\\", `\\`, "[", `\[`, "]", `\]`).Replace(name)
	const hex = "0123456789ABCDEF"
	var dest strings.Builder
	for i := 0; i < len(path); i++ {
		b := path[i]
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune(";/,:@&=+$-_.!~*'", rune(b)) {
			dest.WriteByte(b)
		} else {
			dest.WriteByte('%')
			dest.WriteByte(hex[b>>4])
			dest.WriteByte(hex[b&15])
		}
	}
	return "[" + name + "](" + dest.String() + ")", true
}

// Insertion adds boundary spaces only where needed. Offsets are validated UTF-8
// byte boundaries; native callers convert their UTF-16 selection before calling.
func Insertion(text string, start, end int, path string) (replacement string, ok bool) {
	ref, ok := Serialize(path)
	if !ok || start < 0 || end < start || end > len(text) || !utf8.ValidString(text[:start]) || !utf8.ValidString(text[:end]) {
		return "", false
	}
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		if !unicode.IsSpace(r) {
			ref = " " + ref
		}
	}
	if end == len(text) {
		ref += " "
	} else {
		r, _ := utf8.DecodeRuneInString(text[end:])
		if !unicode.IsSpace(r) {
			ref += " "
		}
	}
	return ref, true
}
