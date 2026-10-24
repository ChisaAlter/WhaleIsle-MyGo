package chat

import (
	"encoding/json"
	"math"

	"whaleisle/internal/store/session"
)

// toolWebModel is the installed settled web card's display data. URLs retain
// their recorded spelling and confer no navigation or host-opening authority.
// StatusCode retains signed JavaScript Number.isInteger semantics, including
// integral IEEE-754 values outside the native int and safe-integer ranges.
type toolWebModel struct {
	Kind, Answer, URL string
	StatusCode        float64
	Truncated         bool
	Sources           []toolWebSource
}

type toolWebSource struct {
	URL, Title, Snippet, PublishedAt string
}

// toolWebModelFor derives only a successful, settled root web call with a
// retained call head and valid root receipt. Metadata owns all card data; source
// calls and events stay intact, and each returned source slice is independent.
func toolWebModelFor(call *session.ToolCallView) *toolWebModel {
	if call == nil || call.Name != "web_search" && call.Name != "web_fetch" || call.Call == nil || call.Result == nil || call.State != "ok" || call.ParentCallID != "" || call.RootCallID != "" && call.RootCallID != call.CallID {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	if call.Name == "web_search" {
		queries, valid := contextRawArray(args["queries"])
		if !valid || len(queries) == 0 {
			return nil
		}
		for _, raw := range queries {
			query, valid := terminalModelString(raw)
			if !valid || terminalModelBlank(query) {
				return nil
			}
		}
	} else {
		url, valid := terminalModelString(args["url"])
		if !valid || terminalModelBlank(url) {
			return nil
		}
	}
	// A projected successful root cannot turn a nested or malformed wire receipt
	// into a root card. In particular, valid metadata needs data.message object.
	envelope, ok := terminalModelObject(call.Result.Raw)
	if !ok {
		return nil
	}
	typ, ok := terminalModelString(envelope["type"])
	if !ok || typ != "tool/result" || call.Result.EventType != "" && call.Result.EventType != typ {
		return nil
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return nil
	}
	if _, ok := terminalModelObject(data["message"]); !ok {
		return nil
	}
	result := toolSummaryReadResult(call.Result)
	if result.nested || result.failed || result.stopped {
		return nil
	}
	meta, ok := terminalModelObject(result.meta)
	if !ok {
		return nil
	}
	truncated, ok := terminalModelBool(meta["truncated"])
	if !ok {
		return nil
	}
	model := &toolWebModel{Truncated: truncated}
	if call.Name == "web_search" {
		model.Kind = "search"
		model.Answer, ok = toolWebModelOptionalString(meta, "answer")
		if !ok {
			return nil
		}
		sources, valid := contextRawArray(meta["sources"])
		if !valid {
			return nil
		}
		model.Sources = make([]toolWebSource, 0, len(sources))
		for _, raw := range sources {
			source, valid := terminalModelObject(raw)
			if !valid {
				return nil
			}
			url, validURL := terminalModelString(source["url"])
			title, validTitle := toolWebModelOptionalString(source, "title")
			snippet, validSnippet := toolWebModelOptionalString(source, "snippet")
			publishedAt, validPublishedAt := toolWebModelOptionalString(source, "publishedAt")
			if !validURL || !validTitle || !validSnippet || !validPublishedAt {
				return nil
			}
			model.Sources = append(model.Sources, toolWebSource{URL: url, Title: title, Snippet: snippet, PublishedAt: publishedAt})
		}
	} else {
		model.Kind = "fetch"
		model.URL, ok = terminalModelString(meta["url"])
		if !ok {
			return nil
		}
		model.StatusCode, ok = toolReadModelInteger(meta["statusCode"], math.Inf(-1))
		if !ok {
			return nil
		}
	}
	return model
}

// The card renders absent and empty optional strings equivalently, but an
// explicitly recorded null or other non-string must still decline the model.
func toolWebModelOptionalString(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, exists := fields[key]
	if !exists {
		return "", true
	}
	return terminalModelString(raw)
}
