package settings

import "strings"

func filterSkillRows(rows []SkillRow, query, source *string) []SkillRow {
	q, src := "", ""
	if query != nil {
		q = strings.ToLower(strings.TrimSpace(*query))
	}
	if source != nil {
		src = *source
	}
	out := make([]SkillRow, 0, len(rows))
	for _, row := range rows {
		if src != "" && row.Source != src {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(row.Name+" "+row.Description+" "+row.Source), q) {
			continue
		}
		out = append(out, row)
	}
	return out
}
