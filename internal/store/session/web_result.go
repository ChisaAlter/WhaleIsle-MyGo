package session

import (
	"encoding/json"
	"strings"
)

// WebSource is the lossless source projection used by the web tool card.
// It mirrors the source fields in the harness WebSearchResultView contract.
type WebSource struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

// WebResult is the typed presentation metadata carried by tool/result.meta.
// Kind is "search" for WebSearchResultView and "fetch" for
// WebFetchResultView. The raw metadata remains on Message.Raw for unknown or
// future presentation contracts.
type WebResult struct {
	Kind       string
	Sources    []WebSource
	Answer     string
	Truncated  bool
	URL        string
	StatusCode int
}

// parseToolWebResult recognizes only the protocol-defined web presentation
// metadata. Unknown metadata is deliberately left to the raw JSON fallback.
func parseToolWebResult(data json.RawMessage) *WebResult {
	var envelope struct {
		Meta json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}
	meta := strings.TrimSpace(string(envelope.Meta))
	if meta == "" || meta == "null" {
		return nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Meta, &fields); err != nil || fields == nil {
		return nil
	}

	if _, ok := fields["sources"]; ok {
		var search struct {
			Sources   []WebSource `json:"sources"`
			Truncated *bool       `json:"truncated"`
			Answer    *string     `json:"answer"`
		}
		if err := json.Unmarshal(envelope.Meta, &search); err != nil || search.Truncated == nil {
			return nil
		}
		result := &WebResult{
			Kind:      "search",
			Sources:   append([]WebSource(nil), search.Sources...),
			Truncated: *search.Truncated,
		}
		if search.Answer != nil {
			result.Answer = *search.Answer
		}
		return result
	}

	var fetch struct {
		URL        string `json:"url"`
		StatusCode *int   `json:"statusCode"`
		Truncated  *bool  `json:"truncated"`
	}
	if err := json.Unmarshal(envelope.Meta, &fetch); err != nil || fetch.URL == "" || fetch.StatusCode == nil || fetch.Truncated == nil {
		return nil
	}
	return &WebResult{
		Kind:       "fetch",
		URL:        fetch.URL,
		StatusCode: *fetch.StatusCode,
		Truncated:  *fetch.Truncated,
	}
}
