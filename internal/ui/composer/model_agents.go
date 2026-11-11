package composer

import (
	"strings"
	"whaleisle/internal/store/session"
)

func hasExternalAgents(s session.ViewState) bool {
	for _, g := range groupedModelChoices(s, "") {
		if strings.HasPrefix(g.Provider, "acp:") {
			return true
		}
	}
	return false
}
func selectedModelAgent(s session.ViewState) string {
	for _, g := range groupedModelChoices(s, "") {
		for _, choice := range g.Choices {
			if choice.Label == s.SelectedModel && strings.HasPrefix(g.Provider, "acp:") {
				return g.Provider
			}
		}
	}
	return "harness"
}
func modelGroupsForAgent(groups []modelCatalogGroup, agent string) []modelCatalogGroup {
	var out []modelCatalogGroup
	for _, g := range groups {
		if agent == g.Provider || (agent == "" || agent == "harness") && !strings.HasPrefix(g.Provider, "acp:") {
			out = append(out, g)
		}
	}
	return out
}
