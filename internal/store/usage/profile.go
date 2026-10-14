package usage

import (
	"encoding/json"
	"strings"
)

// ParseProfileSummary best-effort extracts email / name / plan from account/getProfile JSON.
func ParseProfileSummary(raw json.RawMessage) (email, name, plan string) {
	if len(raw) == 0 {
		return "", "", ""
	}
	var top any
	if err := json.Unmarshal(raw, &top); err != nil {
		return "", "", ""
	}
	return walkProfile(top)
}

func walkProfile(v any) (email, name, plan string) {
	switch t := v.(type) {
	case map[string]any:
		email = firstString(t, "email", "Email", "mail")
		name = firstString(t, "name", "displayName", "DisplayName", "userName", "username", "fullName")
		plan = firstString(t, "plan", "planName", "Plan", "subscription", "tier")
		if email != "" && name != "" && plan != "" {
			return email, name, plan
		}
		for _, key := range []string{"user", "profile", "account", "data", "result"} {
			if nested, ok := t[key]; ok {
				e2, n2, p2 := walkProfile(nested)
				if email == "" {
					email = e2
				}
				if name == "" {
					name = n2
				}
				if plan == "" {
					plan = p2
				}
			}
		}
		return email, name, plan
	default:
		return "", "", ""
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch s := v.(type) {
			case string:
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}
