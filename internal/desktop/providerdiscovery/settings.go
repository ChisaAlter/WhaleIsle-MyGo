package providerdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

type Caller interface {
	Call(context.Context, string, any, any) error
}
type request func(string, any, any) error
type namespace struct {
	NS       string          `json:"ns"`
	Revision float64         `json:"revision"`
	Schema   json.RawMessage `json:"schema"`
	Value    json.RawMessage `json:"value"`
	Secrets  []struct {
		Path []string `json:"path"`
		Set  bool     `json:"set"`
	} `json:"secrets"`
}
type settingsView struct {
	Writable   bool        `json:"writable"`
	Namespaces []namespace `json:"namespaces"`
}
type provider struct {
	Provider  string   `json:"provider"`
	Name      string   `json:"displayName"`
	Namespace string   `json:"settingsNs"`
	Path      []string `json:"settingsPath"`
}
type Row struct {
	ID, Provider, Name, Namespace, Endpoint, Protocol, Ref              string
	Path                                                                []string
	Revision                                                            float64
	Writable, CredentialKnown, CredentialConfigured, ProtocolConfigured bool
	Blocked                                                             string
	Models                                                              []map[string]any
}

func profileAt(raw json.RawMessage, path []string) map[string]any {
	var node any
	if json.Unmarshal(raw, &node) != nil {
		return nil
	}
	for _, part := range path {
		object, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = object[part]
	}
	m, _ := node.(map[string]any)
	return m
}
func modelsAt(profile map[string]any) []map[string]any {
	rows, _ := profile["models"].([]any)
	out := []map[string]any{}
	for _, raw := range rows {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
func cloneModels(in []map[string]any) []map[string]any {
	b, _ := json.Marshal(in)
	var out []map[string]any
	_ = json.Unmarshal(b, &out)
	if out == nil {
		out = []map[string]any{}
	}
	return out
}
func schemaKind(raw json.RawMessage, path []string, want string) bool {
	var root any
	if json.Unmarshal(raw, &root) != nil {
		return false
	}
	refs := map[string]any{}
	if env, ok := root.(map[string]any); ok {
		if r, ok := env["refs"].(map[string]any); ok {
			refs = r
			root = env["uid"]
		}
	}
	var walk func(any, []string, int) bool
	walk = func(node any, remaining []string, depth int) bool {
		if depth > 64 {
			return false
		}
		m, ok := node.(map[string]any)
		if !ok {
			b, _ := json.Marshal(node)
			key := strings.Trim(string(b), `"`)
			return walk(refs[key], remaining, depth+1)
		}
		kind := textOf(m["type"])
		if len(remaining) == 0 && kind == want {
			return true
		}
		switch kind {
		case "object":
			if len(remaining) == 0 {
				return false
			}
			dict, _ := m["dict"].(map[string]any)
			return walk(dict[remaining[0]], remaining[1:], depth+1)
		case "dict":
			if len(remaining) == 0 {
				return false
			}
			return walk(m["inner"], remaining[1:], depth+1)
		case "union", "intersect":
			list, _ := m["list"].([]any)
			for _, sub := range list {
				if walk(sub, remaining, depth+1) {
					return true
				}
			}
		case "const":
			_, isString := m["value"].(string)
			return len(remaining) == 0 && want == "string" && isString
		}
		return false
	}
	return walk(root, path, 0)
}
func fieldPath(r Row, field string) []string { return append(append([]string(nil), r.Path...), field) }
func load(call request) ([]Row, error) {
	var view settingsView
	if call("settings/describe", nil, &view) != nil {
		return nil, errors.New("Unable to read provider settings. Reconnect and refresh.")
	}
	var directory []provider
	if call("llm/listConfigurableProviders", nil, &directory) != nil {
		return nil, errors.New("Unable to read the Host provider directory.")
	}
	rows := []Row{}
	seen := map[string]bool{}
	for _, p := range directory {
		// SP23 restores the pi-ai model-list editor, not account/DeepSeek forms.
		if p.Namespace != "llm-pi-ai" || p.Provider == "" {
			continue
		}
		for _, ns := range view.Namespaces {
			if ns.NS != p.Namespace {
				continue
			}
			identity, _ := json.Marshal([]any{p.Namespace, p.Provider, p.Path})
			id := string(identity)
			if seen[id] {
				continue
			}
			seen[id] = true
			profile := profileAt(ns.Value, p.Path)
			r := Row{ID: id, Name: p.Name, Provider: p.Provider, Namespace: p.Namespace, Path: append([]string(nil), p.Path...), Revision: ns.Revision, Writable: view.Writable, Endpoint: textOf(profile["baseURL"]), Protocol: textOf(profile["api"]), Ref: textOf(profile["apiKeyEnv"]), Models: modelsAt(profile)}
			if r.Name == "" {
				r.Name = r.Provider
			}
			r.ProtocolConfigured = r.Protocol != ""
			if !schemaKind(ns.Schema, fieldPath(r, "models"), "array") || !schemaKind(ns.Schema, fieldPath(r, "baseURL"), "string") || !schemaKind(ns.Schema, fieldPath(r, "api"), "string") {
				r.Blocked = "The Host schema does not expose the supported provider model fields."
			}
			if overrides, ok := profile["modelOverrides"].(map[string]any); ok && len(overrides) > 0 {
				r.Blocked = "This provider uses modelOverrides. Edit those overrides in provider settings before replacing its model list."
			}
			for _, secret := range ns.Secrets {
				if !secret.Set {
					continue
				}
				for _, field := range []string{"models", "baseURL", "api"} {
					path := fieldPath(r, field)
					n := len(path)
					if len(secret.Path) < n {
						n = len(secret.Path)
					}
					if reflect.DeepEqual(path[:n], secret.Path[:n]) {
						r.Blocked = "This provider has hidden values inside the edited fields. Use the owning settings editor to preserve them."
					}
				}
			}
			rows = append(rows, r)
		}
	}
	// Metadata only; never request or resolve the stored credential literal.
	refs := []string{}
	for _, row := range rows {
		if row.Ref != "" && !seen[row.Ref] {
			refs = append(refs, row.Ref)
			seen[row.Ref] = true
		}
	}
	states := map[string]struct {
		Configured bool `json:"configured"`
	}{}
	for start := 0; start < len(refs); start += 64 {
		end := start + 64
		if end > len(refs) {
			end = len(refs)
		}
		var batch map[string]struct {
			Configured bool `json:"configured"`
		}
		if call("credentials/describe", map[string]any{"refs": refs[start:end]}, &batch) == nil {
			for ref, info := range batch {
				states[ref] = info
			}
		}
	}
	for i := range rows {
		info, ok := states[rows[i].Ref]
		rows[i].CredentialKnown = ok
		rows[i].CredentialConfigured = info.Configured
	}
	return rows, nil
}

func findRow(rows []Row, id string) (Row, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return Row{}, false
}
func modelID(m map[string]any) string { return textOf(m["id"]) }
func jsonModels(models []map[string]any) (string, error) {
	b, err := json.Marshal(models)
	return string(b), err
}
func mergeCandidates(models []map[string]any, candidates []Candidate) []map[string]any {
	out := cloneModels(models)
	seen := map[string]bool{}
	for _, model := range out {
		seen[modelID(model)] = true
	}
	for _, candidate := range candidates {
		if !candidate.Picked || seen[candidate.ID] {
			continue
		}
		seen[candidate.ID] = true
		m := map[string]any{"id": candidate.ID, "name": candidate.Name}
		if candidate.ContextWindow > 0 {
			m["contextWindow"] = candidate.ContextWindow
		}
		if candidate.MaxTokens > 0 {
			m["maxTokens"] = candidate.MaxTokens
		}
		if candidate.InputModalities != nil {
			m["input"] = append([]string{}, candidate.InputModalities...)
		}
		out = append(out, m)
	}
	return out
}
