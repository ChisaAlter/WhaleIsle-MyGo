package settings

import (
	"encoding/json"
	"reflect"

	"whaleisle/internal/dsh/api"
)

func namespaceWriteLanded(out api.SettingsNamespaceView, patch map[string]json.RawMessage, ops []json.RawMessage) bool {
	read := func(raw json.RawMessage, path []string) (any, bool) {
		var value any
		if json.Unmarshal(raw, &value) != nil {
			return nil, false
		}
		for _, key := range path {
			object, ok := value.(map[string]any)
			if !ok {
				return nil, false
			}
			value, ok = object[key]
			if !ok {
				return nil, false
			}
		}
		return value, true
	}
	same := func(path []string, raw json.RawMessage) bool {
		got, ok := read(out.Value, path)
		var wanted any
		return ok && json.Unmarshal(raw, &wanted) == nil && reflect.DeepEqual(got, wanted)
	}
	for key, value := range patch {
		if !same([]string{key}, value) {
			return false
		}
	}
	for _, raw := range ops {
		var op struct {
			Op    string          `json:"op"`
			Path  []string        `json:"path"`
			Value json.RawMessage `json:"value"`
		}
		if json.Unmarshal(raw, &op) != nil {
			return false
		}
		if op.Op == "unset" {
			if _, exists := read(out.User, op.Path); exists {
				return false
			}
		} else if op.Op != "set" || !same(op.Path, op.Value) {
			return false
		}
	}
	return true
}
