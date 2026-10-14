package session

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/api"
)

// titleForItem picks a sidebar label from list projections or session id.
func titleForItem(it api.SessionListValueItemsItem) string {
	if it.Projections != nil && len(it.Projections.Values) > 0 {
		if t := titleFromProjections(it.Projections.Values); t != "" {
			return t
		}
	}
	id := it.SessionId
	if len(id) > 8 {
		id = id[:8]
	}
	if it.Blank {
		return "New chat · " + id
	}
	return id
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
