package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func TestACPAgentModelKeyboardAndMissingCurrentBack(t *testing.T) {
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "retired", ModelLabels: []string{"DS", "ACP Fast"}, ModelChoices: map[string]api.ModelCatalogDefault{"DS": {Provider: "deepseek-official", Model: "ds"}, "ACP Fast": {Provider: "acp:local", Model: "fast"}}, ModelCatalogChoices: []session.ModelCatalogChoice{{Label: "DS", Name: "DS", Provider: "deepseek-official"}, {Label: "ACP Fast", Name: "Fast", Provider: "acp:local", ProviderName: "Local ACP"}}}
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(v string) { chosen = v }}, testColors(), 24)
	}, 800, 600)
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	// Root autofocus begins on Model, Up reaches the new Agent row.
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyRight)
	if _, ok := tt.Find("Choose agent Local ACP"); !ok {
		t.Fatal("Agent row missing from keyboard navigation")
	}
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("Agent  DeepSeek Harness  ›"); err != nil {
		t.Fatal("escape failed to return from missing-current Agent pane", err)
	}
	if err := tt.Click("Choose agent Local ACP"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Choose model DS"); ok {
		t.Fatal("Harness model leaked into ACP selection")
	}
	if err := tt.Click("Choose model ACP Fast"); err != nil {
		t.Fatal(err)
	}
	if chosen != "ACP Fast" || snap.SelectedModel != "retired" || !tt.Focused("Model catalog") {
		t.Fatal(chosen)
	}
}
