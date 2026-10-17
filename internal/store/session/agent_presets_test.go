package session

import (
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestFlattenAgentPresetRoster(t *testing.T) {
	hidden := true
	name := "Coder"
	broken := "missing file"
	roster := api.AgentPresetRoster{
		Presets: []api.AgentPresetRosterPresetsItem{
			{Id: "default", IsDefault: true, Name: &name},
			{Id: "hidden-one", Hidden: &hidden},
			{Id: "default", Name: &name},
			{Id: "broken", Broken: &broken},
		},
	}
	opts := flattenAgentPresetRoster(roster)
	if len(opts) != 2 {
		t.Fatalf("len = %d want 2", len(opts))
	}
	if opts[0].ID != "default" || opts[0].Label != "Coder" {
		t.Fatalf("first = %+v", opts[0])
	}
	if opts[1].ID != "broken" || opts[1].Label != "broken ⚠" {
		t.Fatalf("second = %+v", opts[1])
	}
	if got := defaultAgentPresetLabel(opts, roster); got != "Coder" {
		t.Fatalf("default label = %q want Coder", got)
	}
	if got := labelForAgentPresetID("default", opts); got != "Coder" {
		t.Fatalf("lookup = %q", got)
	}
}
