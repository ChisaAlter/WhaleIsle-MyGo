package settings

import (
	"encoding/json"
	"reflect"

	"whaleisle/internal/dsh/api"
)

// storedReceiptLayer is the layer a settings write lands in. The Host reports
// its effective value with every schema default filled in, so a provider
// profile comes back carrying context windows, compat flags and token budgets
// nobody submitted; only the user layer holds exactly what was stored. A Host
// that reports no user layer at all leaves the effective value as the only
// receipt of what it kept.
func storedReceiptLayer(out api.SettingsNamespaceView) json.RawMessage {
	if len(out.User) > 0 {
		return out.User
	}
	return out.Value
}

// namespaceWriteLanded reports whether a Host receipt holds the values a write
// submitted. A receipt that reports them only in its defaulted effective value
// is a landed write whose defaults the caller never sent, not a refusal.
func namespaceWriteLanded(out api.SettingsNamespaceView, patch map[string]json.RawMessage, ops []json.RawMessage) bool {
	layer := storedReceiptLayer(out)
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
		var wanted any
		if json.Unmarshal(raw, &wanted) != nil {
			return false
		}
		if len(out.User) == 0 {
			got, ok := read(out.Value, path)
			return ok && reflect.DeepEqual(got, wanted)
		}
		if got, ok := read(out.User, path); ok {
			return reflect.DeepEqual(got, wanted)
		}
		// Host config-editor prunes overrides that equal the inherited base layer,
		// leaving the key absent from out.User while both out.Base and out.Value
		// hold the requested value. If out.Base differs from wanted (for example a
		// refused provider creation where base is {} and user is {}), this stays false.
		baseVal, baseOK := read(out.Base, path)
		effVal, effOK := read(out.Value, path)
		return baseOK && effOK && reflect.DeepEqual(baseVal, wanted) && reflect.DeepEqual(effVal, wanted)
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
			if _, exists := read(layer, op.Path); exists {
				return false
			}
			continue
		}
		if op.Op != "set" || !same(op.Path, op.Value) {
			return false
		}
	}
	return true
}
