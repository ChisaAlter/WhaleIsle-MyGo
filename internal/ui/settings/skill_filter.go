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
		if src != "" && src != "all" && skillSourceBucket(row.Source) != src {
			continue
		}
		if q != "" {
			matched := false
			values := append([]string{row.Name, row.Description, row.WhenToUse}, row.Groups...)
			for _, value := range values {
				if strings.Contains(strings.ToLower(value), q) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, row)
	}
	return out
}

func skillSourceBucket(source string) string {
	switch source {
	case "user-dsh", "user-agents":
		return "user"
	case "project-dsh", "project-agents":
		return "project"
	case "bundled":
		return "bundled"
	default:
		return "other"
	}
}

func skillSourceLabel(source string) string {
	switch skillSourceBucket(source) {
	case "user":
		return providerCopy("用户", "User")
	case "project":
		return providerCopy("项目", "Project")
	case "bundled":
		return providerCopy("内置", "Bundled")
	default:
		return providerCopy("其他", "Other")
	}
}
