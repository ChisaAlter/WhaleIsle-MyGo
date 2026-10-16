package session

import (
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestDeveloperPresetProjectionPreservesCustomOverrides(t *testing.T) {
	name := "Named minimal"
	rows := flattenAgentPresetRoster(api.AgentPresetRoster{Presets: []api.AgentPresetRosterPresetsItem{{Id: "ptc"}, {Id: "minimal", Name: &name}, {Id: "standard"}}})
	s := New()
	s.agentPresetOptions = rows
	hidden := s.AgentPresetLabelsForDeveloperTools(false)
	if len(hidden) != 2 || hidden[0] != name || hidden[1] != "standard" {
		t.Fatalf("wrong filtered roster: %v", hidden)
	}
	if all := s.AgentPresetLabelsForDeveloperTools(true); len(all) != 3 {
		t.Fatal("projection mutated roster")
	}
}
