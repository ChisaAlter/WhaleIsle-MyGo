package composer

import (
	"sort"
	"strings"

	"whaleisle/internal/store/session"
)

// SkillInsert returns the literal slash invocation for a skill name (ui-skill onPick).
func SkillInsert(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "/" + name + " "
}

// AppendSkillInvocation appends a /name token to the draft with separating space when needed.
func AppendSkillInvocation(draft *string, name string) {
	if draft == nil {
		return
	}
	insert := SkillInsert(name)
	if insert == "" {
		return
	}
	cur := strings.TrimSpace(*draft)
	if cur == "" {
		*draft = insert
		return
	}
	if !strings.HasSuffix(cur, " ") && !strings.HasSuffix(cur, "\n") {
		*draft = cur + " " + insert
		return
	}
	*draft = cur + insert
}

// FilterSkills uses the original rankByName semantics: ordered subsequence,
// prefix first, alignment score second, stable catalog order for ties.
func FilterSkills(all []session.SkillOption, query string, limit int) []session.SkillOption {
	if limit <= 0 {
		limit = 12
	}
	q := strings.ToLower(strings.TrimSpace(query))
	type match struct {
		skill  session.SkillOption
		score  int
		prefix bool
	}
	matches := make([]match, 0, len(all))
	for _, sk := range all {
		if score, ok := modelNameScore(sk.Name, q); ok {
			matches = append(matches, match{sk, score, strings.HasPrefix(strings.ToLower(sk.Name), q)})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].prefix != matches[j].prefix {
			return matches[i].prefix
		}
		return matches[i].score > matches[j].score
	})
	out := make([]session.SkillOption, 0, min(limit, len(matches)))
	for _, m := range matches {
		if len(out) >= limit {
			break
		}
		out = append(out, m.skill)
	}
	return out
}
