package session

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/api"
)

// titleForItem picks the Host projection title, or "" when the Session has none
// yet. A blank or unnamed Session therefore carries no label: internal ids stay
// out of every surface, and each surface shows its own localized placeholder
// (the sidebar's row and the conversation title band own theirs).
func titleForItem(it api.SessionListValueItemsItem) string {
	if it.Projections != nil && len(it.Projections.Values) > 0 {
		if t := titleFromProjections(it.Projections.Values); t != "" {
			return t
		}
	}
	return ""
}

func titleFromProjections(raw json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, key := range []string{"title", "name", "label"} {
		if v, ok := m[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}
