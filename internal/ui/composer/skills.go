package composer

import (
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

// FilterSkills returns skills whose name matches query by prefix (case-insensitive).
func FilterSkills(all []session.SkillOption, query string, limit int) []session.SkillOption {
	if limit <= 0 {
		limit = 12
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]session.SkillOption, 0, limit)
	for _, sk := range all {
		name := strings.ToLower(sk.Name)
		if q == "" || strings.HasPrefix(name, q) || strings.Contains(name, q) {
			out = append(out, sk)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}
