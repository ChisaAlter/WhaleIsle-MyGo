package settings

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ServedDraftError mirrors the dedicated cards' validation before any write.
func ServedDraftError(s ViewState) string {
	if s.SelectedNS == "subagent-model-selection-settings" && len(s.Namespaces) > 0 {
		ns := s.Namespaces[0]
		if s.Dirty && ns.HasDraftRevision && ns.DraftRevision != ns.Revision {
			return "Subagent model selection changed on the Host; discard this draft before editing again"
		}
	}
	dirty := map[string]bool{}
	for key := range s.Drafts {
		dirty[key] = true
	}
	if err := validateServedDraft(s.SelectedNS, s.Drafts, dirty, s.Resets); err != nil {
		return err.Error()
	}
	return ""
}

func validateServedDraft(namespace string, drafts map[string]string, dirty, resets map[string]bool) error {
	minimums := map[string]float64{}
	switch namespace {
	case "subagent":
		minimums = map[string]float64{"maxDepth": 0, "maxActiveSubagents": 1}
	case "web-deepseek-api":
		minimums = map[string]float64{"count": 1}
	}
	for key, minimum := range minimums {
		text := strings.TrimSpace(drafts[key])
		if !dirty[key] || resets[key] || text == "" {
			continue
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || n < minimum || math.Signbit(n) && n == 0 || n > 9007199254740991 || namespace == "web-deepseek-api" && n > 50 {
			return fmt.Errorf("%s: enter a whole number in the supported range", key)
		}
	}
	if namespace == "subagent-model-selection-settings" && drafts["enabled"] == "true" {
		var routes []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		}
		if json.Unmarshal([]byte(drafts["allowedModels"]), &routes) != nil || len(routes) == 0 {
			return fmt.Errorf("Select at least one subagent model before saving")
		}
	}
	return nil
}
