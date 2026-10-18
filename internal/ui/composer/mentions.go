package composer

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// DraftMentionKind distinguishes @path vs /skill tokens in the draft.
type DraftMentionKind int

const (
	DraftMentionAt DraftMentionKind = iota
	DraftMentionSkill
	DraftMentionSession
)

// DraftMention is one display token parsed from the composer draft.
type DraftMention struct {
	Kind  DraftMentionKind
	Text  string // full token including leading @ or /
	Start int    // byte offset in draft
	End   int    // exclusive byte offset
}

// ChipEdit is optional popup state for rewriting one draft mention.
type ChipEdit struct {
	Open  bool
	Start int    // byte offset of the mention when opened
	Orig  string // original token text
	Draft string // TextInput buffer
}

// ParseDraftMentions finds completed @path and /skill tokens at whitespace boundaries.
// Bare "@" / "/" are skipped. Tokens run until whitespace (space, tab, newline).
func ParseDraftMentions(draft string) []DraftMention {
	if draft == "" {
		return nil
	}
	out := make([]DraftMention, 0, 4)
	i := 0
	for i < len(draft) {
		if !tokenBoundary(draft, i) {
			i = skipToken(draft, i)
			continue
		}
		r, size := utf8.DecodeRuneInString(draft[i:])
		switch r {
		case '@', '/':
			end := i + size
			for end < len(draft) {
				nr, nsz := utf8.DecodeRuneInString(draft[end:])
				if nr == ' ' || nr == '\t' || nr == '\n' {
					break
				}
				end += nsz
			}
			if r == '@' {
				if match := sessionMentionPattern.FindString(draft[i:]); match != "" {
					end = i + len(match)
				} else if i+1 < len(draft) && draft[i+1] == '"' {
					if close := strings.IndexByte(draft[i+2:], '"'); close >= 0 {
						end = i + 3 + close
					} else {
						i = skipToken(draft, i)
						continue
					}
				}
			}
			if end > i+size {
				kind := DraftMentionAt
				if r == '@' && strings.HasPrefix(draft[i:end], "@[") {
					kind = DraftMentionSession
				}
				if r == '/' {
					kind = DraftMentionSkill
				}
				out = append(out, DraftMention{
					Kind:  kind,
					Text:  draft[i:end],
					Start: i,
					End:   end,
				})
			}
			i = end
		default:
			i = skipToken(draft, i)
		}
	}
	return out
}

// RemoveDraftMention deletes one mention from draft and tidies surrounding spaces.
func RemoveDraftMention(draft string, m DraftMention) string {
	m, ok := resolveMention(draft, m)
	if !ok {
		return draft
	}
	left := draft[:m.Start]
	right := draft[m.End:]
	leftTrim := strings.TrimRight(left, " \t")
	rightTrim := strings.TrimLeft(right, " \t")
	if leftTrim == "" {
		return rightTrim
	}
	if rightTrim == "" {
		return leftTrim
	}
	if strings.HasSuffix(leftTrim, "\n") || strings.HasPrefix(rightTrim, "\n") {
		return leftTrim + rightTrim
	}
	return leftTrim + " " + rightTrim
}

// ReplaceDraftMention swaps one mention token for newText (trimmed). Empty newText removes it.
func ReplaceDraftMention(draft string, m DraftMention, newText string) string {
	newText = strings.TrimSpace(newText)
	if newText == "" {
		return RemoveDraftMention(draft, m)
	}
	m, ok := resolveMention(draft, m)
	if !ok {
		return draft
	}
	return draft[:m.Start] + newText + draft[m.End:]
}

// ReorderDraftMentions moves the mention at fromIndex so it occupies toIndex
// in the resulting mention list (0-based; toIndex may equal len for append).
// Token strings are swapped across existing mention spans; surrounding prose stays put.
func ReorderDraftMentions(draft string, fromIndex, toIndex int) string {
	tags := ParseDraftMentions(draft)
	n := len(tags)
	if fromIndex < 0 || fromIndex >= n || toIndex < 0 || toIndex > n {
		return draft
	}
	if toIndex > fromIndex {
		toIndex--
	}
	if fromIndex == toIndex {
		return draft
	}
	texts := make([]string, n)
	for i, t := range tags {
		texts[i] = t.Text
	}
	item := texts[fromIndex]
	texts = append(texts[:fromIndex], texts[fromIndex+1:]...)
	texts = append(texts[:toIndex], append([]string{item}, texts[toIndex:]...)...)

	var b strings.Builder
	b.Grow(len(draft))
	prev := 0
	for i, t := range tags {
		b.WriteString(draft[prev:t.Start])
		b.WriteString(texts[i])
		prev = t.End
	}
	b.WriteString(draft[prev:])
	return b.String()
}

// MentionUTF16Range returns half-open UTF-16 offsets of m in draft for DraftClient
// selection. ok is false when the mention cannot be resolved.
func MentionUTF16Range(draft string, m DraftMention) (start, end int, ok bool) {
	m, ok = resolveMention(draft, m)
	if !ok {
		return 0, 0, false
	}
	return ui.UTF16Len(draft[:m.Start]), ui.UTF16Len(draft[:m.End]), true
}

// MentionRuneRange returns inclusive-start / exclusive-end rune offsets of m in draft
// for TextArea.SetTextSelection. ok is false when the mention cannot be resolved.
func MentionRuneRange(draft string, m DraftMention) (start, end int, ok bool) {
	m, ok = resolveMention(draft, m)
	if !ok {
		return 0, 0, false
	}
	start = utf8.RuneCountInString(draft[:m.Start])
	end = start + utf8.RuneCountInString(draft[m.Start:m.End])
	return start, end, true
}

var sessionMentionPattern = regexp.MustCompile(`^@\[((?:\\.|[^\\\]])*)\]\((dsh-session:[A-Za-z0-9_-]+)\)`)

// ChipLabel returns a short display label with a kind prefix for chip buttons.
func ChipLabel(m DraftMention) string {
	switch m.Kind {
	case DraftMentionSession:
		match := sessionMentionPattern.FindStringSubmatch(m.Text)
		if len(match) > 1 {
			return "session @" + strings.NewReplacer(`\]`, "]", `\\`, `\`).Replace(match[1])
		}
		return "session " + m.Text
	case DraftMentionSkill:
		if strings.HasPrefix(m.Text, "/") {
			return "skill " + m.Text
		}
		return "skill /" + m.Text
	default:
		if strings.HasPrefix(m.Text, "@") {
			return "file " + m.Text
		}
		return "file @" + m.Text
	}
}

func resolveMention(draft string, m DraftMention) (DraftMention, bool) {
	if m.Start < 0 || m.End > len(draft) || m.Start >= m.End {
		if m.Text == "" {
			return m, false
		}
		if j := strings.Index(draft, m.Text); j >= 0 {
			m.Start, m.End = j, j+len(m.Text)
			return m, true
		}
		return m, false
	}
	if draft[m.Start:m.End] != m.Text && m.Text != "" {
		if j := strings.Index(draft, m.Text); j >= 0 {
			m.Start, m.End = j, j+len(m.Text)
			return m, true
		}
		return m, false
	}
	return m, true
}

func tokenBoundary(draft string, i int) bool {
	if i == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(draft[:i])
	return prev == ' ' || prev == '\t' || prev == '\n'
}

func skipToken(draft string, i int) int {
	for i < len(draft) {
		r, size := utf8.DecodeRuneInString(draft[i:])
		if r == ' ' || r == '\t' || r == '\n' {
			return i + size
		}
		i += size
	}
	return i
}
