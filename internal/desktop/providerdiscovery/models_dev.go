package providerdiscovery

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type metadataModel struct {
	Name             string                            `json:"name"`
	Limit            struct{ Context, Output float64 } `json:"limit"`
	Reasoning        *bool                             `json:"reasoning"`
	ReasoningOptions []struct {
		Type   string   `json:"type"`
		Values []string `json:"values"`
	} `json:"reasoning_options"`
}
type metadataProvider struct {
	ID     string                   `json:"id"`
	Models map[string]metadataModel `json:"models"`
}

var modelsMetadata struct {
	sync.Mutex
	catalog map[string]metadataProvider
}
var officialMetadataRules = []struct{ provider, pattern string }{
	{"deepseek", `^deepseek(?:[-/.]|$)`}, {"openai", `^(?:gpt(?:[-/.]|$)|o[134](?:[-/.]|$)|codex(?:[-/.]|$))`}, {"xai", `^(?:grok|x-ai/grok|xai/grok)(?:[-/.]|$)`}, {"anthropic", `^claude(?:[-/.]|$)`}, {"google", `^gemini(?:[-/.]|$)`}, {"mistral", `^mistral(?:[-/.]|$)`}, {"cohere", `^command(?:[-/.]|$)`}, {"nvidia", `^nemotron(?:[-/.]|$)`}, {"meta", `^llama(?:[-/.]|$)`}, {"xiaomi", `^mimo(?:[-/.]|$)`}, {"alibaba", `^qwen(?:[-/.]|$)`},
}

// Discovery already succeeded. A metadata error leaves those actual results
// intact; it never turns an unavailable catalog into invented capabilities.
func enrichDiscoveredCandidates(ctx context.Context, rows []Candidate) []Candidate {
	modelsMetadata.Lock()
	catalog := modelsMetadata.catalog
	modelsMetadata.Unlock()
	if catalog == nil {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://models.dev/api.json", nil)
		if err != nil {
			return rows
		}
		response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
		if err != nil {
			return rows
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return rows
		}
		if json.NewDecoder(response.Body).Decode(&catalog) != nil {
			return rows
		}
		modelsMetadata.Lock()
		modelsMetadata.catalog = catalog
		modelsMetadata.Unlock()
	}
	out := append([]Candidate(nil), rows...)
	for i := range out {
		candidate := &out[i]
		record, ok := selectMetadataRecord(catalog, candidate.ID)
		if !ok {
			continue
		}
		if candidate.Name == "" || candidate.Name == candidate.ID {
			if record.Name != "" {
				candidate.Name = record.Name
			}
		}
		if candidate.ContextWindow == 0 {
			candidate.ContextWindow = capacity(&record.Limit.Context)
		}
		if candidate.MaxTokens == 0 {
			candidate.MaxTokens = capacity(&record.Limit.Output)
		}
		if record.Reasoning != nil && !*record.Reasoning {
			candidate.ReasoningEfforts = false
			continue
		}
		for _, option := range record.ReasoningOptions {
			if option.Type != "effort" {
				continue
			}
			efforts := map[string]any{}
			nonOff := false
			for _, level := range option.Values {
				if level == "none" || level == "off" {
					efforts["off"] = nil
				} else if strings.Contains("|minimal|low|medium|high|xhigh|max|", "|"+level+"|") {
					efforts[level] = level
					nonOff = true
				}
			}
			if nonOff {
				candidate.ReasoningEfforts = efforts
			}
			break
		}
	}
	return out
}
func selectMetadataRecord(catalog map[string]metadataProvider, id string) (metadataModel, bool) {
	normalized := strings.ToLower(id)
	bare := normalized
	if slash := strings.LastIndex(bare, "/"); slash >= 0 {
		bare = bare[slash+1:]
	}
	official := ""
	for _, rule := range officialMetadataRules {
		r := regexp.MustCompile(rule.pattern)
		if r.MatchString(normalized) || r.MatchString(bare) {
			official = rule.provider
			break
		}
	}
	type match struct {
		provider string
		model    metadataModel
	}
	matches := []match{}
	for key, provider := range catalog {
		pid := provider.ID
		if pid == "" {
			pid = key
		}
		if model, ok := provider.Models[id]; ok {
			matches = append(matches, match{pid, model})
			if pid == official {
				return model, true
			}
		}
	}
	if official != "" {
		for key, p := range catalog {
			pid := p.ID
			if pid == "" {
				pid = key
			}
			if pid != official {
				continue
			}
			for modelID, model := range p.Models {
				if strings.EqualFold(modelID, id) || strings.EqualFold(modelID, bare) {
					return model, true
				}
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		for _, limits := range [][2]float64{{a.model.Limit.Context, b.model.Limit.Context}, {a.model.Limit.Output, b.model.Limit.Output}} {
			x, y := limits[0], limits[1]
			if x == y {
				continue
			}
			if x <= 0 {
				return false
			}
			if y <= 0 {
				return true
			}
			return x < y
		}
		return a.provider < b.provider
	})
	if len(matches) > 0 {
		return matches[0].model, true
	}
	return metadataModel{}, false
}
