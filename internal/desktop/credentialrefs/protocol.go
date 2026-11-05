package credentialrefs

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"whaleisle/internal/dsh/api"
)

// Caller is the existing authenticated Host RPC transport, never a shell bridge.
type Caller interface {
	Call(context.Context, string, any, any) error
}

type namespace struct {
	NS       string          `json:"ns"`
	Revision float64         `json:"revision"`
	Schema   json.RawMessage `json:"schema"`
	Value    json.RawMessage `json:"value"`
}
type settingsView struct {
	Writable   bool        `json:"writable"`
	Namespaces []namespace `json:"namespaces"`
}
type provider struct {
	Provider    string                                             `json:"provider"`
	DisplayName string                                             `json:"displayName"`
	Namespace   string                                             `json:"settingsNs"`
	Path        []string                                           `json:"settingsPath"`
	BaseURLEnv  []string                                           `json:"baseURLEnv,omitempty"`
	NativeAuth  *api.ListConfigurableProvidersResultItemNativeAuth `json:"nativeAuth,omitempty"`
}
type credentialInfo struct {
	Configured bool   `json:"configured"`
	Writable   bool   `json:"writable"`
	Source     string `json:"source"`
}
type request func(string, any, any) error

// Row addresses a provider-owned apiKeyEnv field; it never contains a stored key.
type Row struct {
	ID, Name, Provider, Namespace, Ref                           string
	Path                                                         []string
	Revision                                                     float64
	BindReference, SettingsWritable, Known, Configured, Writable bool
	NamedReference                                               bool
	BaseURLEnv                                                   []string
	NativeAuth                                                   *api.ListConfigurableProvidersResultItemNativeAuth
	NativeChecks                                                 []CredentialCheck
}

// CredentialCheck is public presence metadata, never a credential literal.
type CredentialCheck struct {
	Ref, Source       string
	Known, Configured bool
}

func (r Row) KeylessNativeAuth() bool {
	return !r.NamedReference && r.NativeAuth != nil && len(r.NativeAuth.Env) == 0 && r.NativeAuth.Oauth
}

func (r Row) nativeRefs() []string {
	if r.NamedReference || r.NativeAuth == nil {
		return nil
	}
	refs := []string{}
	for _, ref := range r.NativeAuth.Env {
		endpoint := false
		for _, env := range r.BaseURLEnv {
			endpoint = endpoint || ref == env
		}
		if !endpoint {
			refs = append(refs, ref)
		}
	}
	return refs
}

var referencePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var routeSeparators = regexp.MustCompile(`[^A-Z0-9]+`)

func deriveRef(route string) string {
	return routeSeparators.ReplaceAllString(strings.ToUpper(route), "_") + "_API_KEY"
}
func profileRef(value json.RawMessage, path []string) string {
	var node any
	if json.Unmarshal(value, &node) != nil {
		return ""
	}
	for _, key := range append(append([]string(nil), path...), "apiKeyEnv") {
		m, ok := node.(map[string]any)
		if !ok {
			return ""
		}
		node = m[key]
	}
	ref, _ := node.(string)
	return ref
}
func rowsFrom(view settingsView, directory []provider) []Row {
	rows := []Row{}
	seen := map[string]bool{}
	add := func(ns namespace, p provider, fallback string) {
		path := append(append([]string(nil), p.Path...), "apiKeyEnv")
		if !schemaStringAt(ns.Schema, path) {
			return
		}
		ref := profileRef(ns.Value, p.Path)
		named := ref != ""
		bind := ref == "" && ns.NS == "llm-pi-ai"
		if ref == "" {
			ref = fallback
		}
		if !referencePattern.MatchString(ref) {
			return
		}
		identity, _ := json.Marshal([]any{ns.NS, p.Provider, p.Path})
		id := string(identity)
		if seen[id] {
			return
		}
		seen[id] = true
		name := p.DisplayName
		if name == "" {
			name = p.Provider
		}
		r := Row{ID: id, Name: name, Provider: p.Provider, Namespace: ns.NS, Ref: ref, Path: append([]string(nil), p.Path...), Revision: ns.Revision, BindReference: bind, SettingsWritable: view.Writable, NamedReference: named, BaseURLEnv: append([]string(nil), p.BaseURLEnv...), NativeAuth: p.NativeAuth}
		rows = append(rows, r)
	}
	for _, ns := range view.Namespaces {
		if ns.NS == "web-search-deepseek" || ns.NS == "web-deepseek-api" {
			add(ns, provider{Provider: ns.NS, DisplayName: "Web Search", Namespace: ns.NS}, "DEEPSEEK_API_KEY")
			continue
		}
		if ns.NS != "llm-deepseek" && ns.NS != "llm-deepseek-api-key" && ns.NS != "llm-pi-ai" {
			continue
		}
		for _, p := range directory {
			if p.Namespace == ns.NS && p.Provider != "deepseek-account" {
				fallback := deriveRef(p.Provider)
				if ns.NS == "llm-deepseek" || ns.NS == "llm-deepseek-api-key" {
					fallback = "DEEPSEEK_API_KEY"
				}
				add(ns, p, fallback)
			}
		}
	}
	return rows
}
func load(call request) ([]Row, error) {
	var view settingsView
	if err := call("settings/describe", nil, &view); err != nil {
		return nil, errors.New("Unable to read Host settings. Refresh after reconnecting.")
	}
	var directory []provider
	if err := call("llm/listConfigurableProviders", nil, &directory); err != nil {
		return nil, errors.New("Unable to read the Host provider directory. Refresh after reconnecting.")
	}
	rows := rowsFrom(view, directory)
	refs := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		requested := row.nativeRefs()
		if !row.KeylessNativeAuth() {
			requested = append(requested, row.Ref)
		}
		for _, ref := range requested {
			if !seen[ref] {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
	}
	sort.Strings(refs)
	states := map[string]credentialInfo{}
	// Host credentials/describe accepts at most 64 references per request.
	for start := 0; start < len(refs); start += 64 {
		end := start + 64
		if end > len(refs) {
			end = len(refs)
		}
		var batch map[string]credentialInfo
		if err := call("credentials/describe", map[string]any{"refs": refs[start:end]}, &batch); err != nil {
			return nil, errors.New("Unable to verify credential writability. Refresh before editing.")
		}
		for ref, info := range batch {
			states[ref] = info
		}
	}
	for i := range rows {
		info, ok := states[rows[i].Ref]
		rows[i].Known = ok
		rows[i].Configured = info.Configured
		rows[i].Writable = ok && info.Writable && rows[i].SettingsWritable && !rows[i].KeylessNativeAuth()
		for _, ref := range rows[i].nativeRefs() {
			info, known := states[ref]
			rows[i].NativeChecks = append(rows[i].NativeChecks, CredentialCheck{Ref: ref, Known: known, Configured: info.Configured, Source: info.Source})
		}
	}
	return rows, nil
}

// schemaStringAt resolves inline and serialized schemastery nodes. Only an
// adapter-declared string credential-reference field can become a write target.
func schemaStringAt(raw json.RawMessage, path []string) bool {
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
		if _, ok := node.(map[string]any); !ok {
			key := ""
			switch x := node.(type) {
			case string:
				key = x
			case float64:
				b, _ := json.Marshal(x)
				key = string(b)
			default:
				return false
			}
			return walk(refs[key], remaining, depth+1)
		}
		m := node.(map[string]any)
		kind, _ := m["type"].(string)
		if len(remaining) == 0 {
			return kind == "string"
		}
		switch kind {
		case "object":
			dict, _ := m["dict"].(map[string]any)
			return walk(dict[remaining[0]], remaining[1:], depth+1)
		case "dict":
			return walk(m["inner"], remaining[1:], depth+1)
		case "intersect", "union":
			list, _ := m["list"].([]any)
			for _, part := range list {
				if walk(part, remaining, depth+1) {
					return true
				}
			}
		}
		return false
	}
	return walk(root, path, 0)
}
