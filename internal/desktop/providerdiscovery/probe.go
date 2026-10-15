// Package providerdiscovery implements explicitly requested, one-shot desktop
// model listing. It never resolves Host credentials or follows redirects.
package providerdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const maxResponseBytes = 4 * 1024 * 1024

var Protocols = []string{"openai-completions", "openai-responses", "anthropic-messages"}

type Candidate struct {
	ID, Name                 string
	ContextWindow, MaxTokens int64
	Picked                   bool
}

func ListingURL(endpoint, protocol string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("Enter a provider base URL without credentials, a query, or a fragment.")
	}
	local := strings.EqualFold(u.Hostname(), "localhost")
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		local = ip.IsLoopback()
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", errors.New("Use HTTPS, or HTTP on an explicit loopback address for an unauthenticated local provider.")
	}
	base := strings.TrimRight(u.Path, "/")
	switch protocol {
	case "openai-completions", "openai-responses":
		u.Path = base + "/models"
	case "anthropic-messages":
		u.Path = strings.TrimSuffix(base, "/v1") + "/v1/models"
		u.RawQuery = "limit=1000"
	default:
		return "", errors.New("This protocol has no supported model-listing endpoint. Use the provider settings editor for manual models.")
	}
	u.RawPath = ""
	return u.String(), nil
}

func probe(ctx context.Context, transport http.RoundTripper, endpoint, protocol, key string) ([]Candidate, error) {
	target, err := ListingURL(endpoint, protocol)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if strings.HasPrefix(target, "http:") && key != "" {
		return nil, errors.New("Loopback HTTP discovery is anonymous only. Use HTTPS to send a temporary key.")
	}
	if strings.ContainsAny(key, "•*●") {
		return nil, errors.New("Enter a temporary API key, not a masked credential placeholder.")
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return nil, errors.New("The temporary key contains unsupported HTTP header characters.")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, errors.New("The provider URL is invalid.")
	}
	req.Header.Set("Accept", "application/json")
	if protocol == "anthropic-messages" {
		req.Header.Set("anthropic-version", "2023-06-01")
		if key != "" {
			req.Header.Set("x-api-key", key)
		}
	} else if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	// Do not inherit Host cookies, credential resolvers, or persisted headers.
	client := &http.Client{Transport: transport, Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("The desktop could not reach the provider, or the request was cancelled. Check the endpoint and try again.")
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return nil, errors.New("The provider requested a redirect. No redirect was followed; enter the final trusted HTTPS endpoint explicitly.")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("The provider returned HTTP %d. Check the endpoint, protocol, and temporary key.", response.StatusCode)
	}
	if response.ContentLength > maxResponseBytes {
		return nil, errors.New("The model listing exceeds the 4 MiB limit.")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, errors.New("The provider response could not be read.")
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("The model listing exceeds the 4 MiB limit.")
	}
	return parseListing(body)
}

func textOf(v any) string { s, _ := v.(string); return strings.TrimSpace(s) }
func positive(v ...any) int64 {
	for _, value := range v {
		if f, ok := value.(float64); ok && f > 0 && f <= 9007199254740991 && f == math.Trunc(f) {
			return int64(f)
		}
	}
	return 0
}
func parseListing(body []byte) ([]Candidate, error) {
	var listing map[string]any
	if json.Unmarshal(body, &listing) != nil {
		return nil, errors.New("The provider did not return a JSON model listing.")
	}
	entries := []map[string]any{}
	if rows, ok := listing["data"].([]any); ok {
		for _, raw := range rows {
			if m, ok := raw.(map[string]any); ok {
				entries = append(entries, m)
			}
		}
	} else if rows, ok := listing["models"].(map[string]any); ok {
		keys := []string{}
		for key := range rows {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if m, ok := rows[key].(map[string]any); ok {
				if key != "" {
					m["id"] = key
				}
				entries = append(entries, m)
			}
		}
	} else {
		return nil, errors.New("The response has neither a data array nor a models object. Enter models manually in provider settings.")
	}
	out := []Candidate{}
	seen := map[string]bool{}
	for _, m := range entries {
		id := textOf(m["id"])
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		name := id
		for _, field := range []string{"name", "display_name", "displayName"} {
			if s := textOf(m[field]); s != "" {
				name = s
				break
			}
		}
		limit, _ := m["limit"].(map[string]any)
		top, _ := m["top_provider"].(map[string]any)
		out = append(out, Candidate{ID: id, Name: name, ContextWindow: positive(m["contextWindow"], m["context_window"], m["context_length"], m["max_input_tokens"], limit["context"]), MaxTokens: positive(m["maxOutputTokens"], m["max_output_tokens"], m["maxTokens"], m["max_tokens"], limit["output"], top["max_completion_tokens"])})
	}
	return out, nil
}
