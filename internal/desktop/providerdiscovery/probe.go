// Package providerdiscovery offers one-shot Host adapter model discovery and
// explicit adoption of its candidates into the provider settings draft.
package providerdiscovery

import (
	"errors"
	"math"
	"strings"

	"whaleisle/internal/dsh/api"
)

// Common editable protocols. Discovery support and catalog lookup belong to the
// Host adapter; this list never gates a configured provider's discovery request.
var Protocols = []string{"openai-completions", "openai-responses", "anthropic-messages"}

type Candidate struct {
	ID, Name                 string
	ContextWindow, MaxTokens int64
	InputModalities          []string
	ReasoningEfforts         any
	Picked                   bool
}

func discover(call request, row Row, endpoint, protocol, key string) ([]Candidate, error) {
	key = strings.TrimSpace(key)
	if strings.ContainsAny(key, "•*●") {
		return nil, errors.New("Enter a temporary API key, not a masked credential placeholder.")
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return nil, errors.New("The temporary key contains unsupported HTTP header characters.")
		}
	}
	// Identifying the route lets the Host use its installed catalog without an
	// endpoint, or its saved headers and credential resolver for network discovery.
	// Omitted draft fields must stay omitted so adapter defaults remain effective.
	query := api.LlmModelDiscoveryRequest{}
	if row.Provider != "" {
		query.Provider = &row.Provider
	}
	if endpoint = strings.TrimSpace(endpoint); endpoint != "" {
		query.BaseURL = &endpoint
	}
	if protocol = strings.TrimSpace(protocol); protocol != "" {
		query.Api = &protocol
	}
	if key != "" {
		query.ApiKey = &key
	}
	var models api.DiscoverModelsResult
	if err := call("llm/discoverModels", map[string]any{"settingsNs": row.Namespace, "request": query}, &models); err != nil {
		// A provider error may include request details; keep credentials out of UI state.
		return nil, errors.New("Host model discovery failed. Check the provider, endpoint, protocol, and credential, or enter models manually.")
	}
	out := []Candidate{}
	seen := map[string]bool{}
	for _, model := range models {
		id := strings.TrimSpace(model.Id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		name := id
		if model.Name != nil && strings.TrimSpace(*model.Name) != "" {
			name = strings.TrimSpace(*model.Name)
		}
		candidate := Candidate{ID: id, Name: name, ContextWindow: capacity(model.ContextWindow), MaxTokens: capacity(model.MaxTokens)}
		if model.InputModalities != nil {
			candidate.InputModalities = append([]string{}, model.InputModalities...)
		}
		out = append(out, candidate)
	}
	return out, nil
}

// EditorProbe is the target currently shown by a provider editor. For a new
// custom route Provider is empty: the Host interrogates its unsaved endpoint.
type EditorProbe struct {
	Namespace, Provider, Endpoint, Protocol, Key string
}

// MergeEditorCandidates returns model rows only. It never writes settings or
// installs another controller-owned draft beside the provider editor's draft.
func MergeEditorCandidates(models []map[string]any, candidates []Candidate) []map[string]any {
	return mergeCandidates(models, candidates)
}

func textOf(v any) string { s, _ := v.(string); return strings.TrimSpace(s) }
func capacity(value *float64) int64 {
	if value != nil && *value > 0 && *value <= 9007199254740991 && *value == math.Trunc(*value) {
		return int64(*value)
	}
	return 0
}
