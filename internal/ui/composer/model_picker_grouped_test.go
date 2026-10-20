package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func TestGroupedModelPickerSearchAndSelection(t *testing.T) {
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "custom — alpha", ModelLabels: []string{"custom — alpha", "account — beta", "official — gamma", "custom — delta", "official — deepseek-chat"}, ModelChoices: map[string]api.ModelCatalogDefault{
		"custom — alpha": {Provider: "custom", Model: "alpha"}, "account — beta": {Provider: "deepseek-account", Model: "beta"}, "official — gamma": {Provider: "deepseek-official", Model: "gamma"}, "custom — delta": {Provider: "custom", Model: "delta"}, "official — deepseek-chat": {Provider: "deepseek-official", Model: "deepseek-chat"},
	}}
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(s string) { chosen = s }}, testColors(), 24)
	}, 800, 600)
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal("grouped catalog trigger absent:", err)
	}
	if err := tt.Click(composerText("模型", "Model") + "  " + snap.SelectedModel + "  ›"); err != nil {
		t.Fatal("model submenu absent:", err)
	}
	if !tt.HasText("deepseek-account") || !tt.HasText("deepseek-official") {
		t.Fatal("provider groups absent")
	}
	if err := tt.Click("Search models"); err != nil {
		t.Fatal(err)
	}
	tt.Type("dsc")
	if _, ok := tt.Find("Choose model custom — alpha"); ok {
		t.Fatal("ordered subsequence search lost result")
	}
	if err := tt.Click("Choose model official — deepseek-chat"); err != nil {
		t.Fatal(err)
	}
	if chosen != "official — deepseek-chat" || snap.SelectedModel != "custom — alpha" {
		t.Fatal("callback changed or optimistic selection invented")
	}
	if !tt.Focused("Model catalog") {
		t.Fatal("mouse selection did not return focus to the trigger")
	}
}

func TestGroupedModelPickerOrderRankingAndSmallCatalog(t *testing.T) {
	labels := []string{"z", "official", "account", "a", "b", "c"}
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, ModelLabels: labels, ModelChoices: map[string]api.ModelCatalogDefault{
		"z": {Provider: "custom"}, "official": {Provider: "deepseek-official"}, "account": {Provider: "deepseek-account"}, "a": {Provider: "custom"}, "b": {Provider: "custom"}, "c": {Provider: "custom"},
	}, ModelCatalogChoices: []session.ModelCatalogChoice{
		{Label: "z", Name: "deepseek-chat", Provider: "custom"}, {Label: "a", Name: "dsc-prefixed", Provider: "custom"}, {Label: "b", Name: "deepseek-chat", Provider: "custom"}, {Label: "c", Name: "unrelated", Provider: "custom"},
	}}
	groups := groupedModelChoices(snap, "")
	if len(groups) != 3 || groups[0].Provider != "deepseek-account" || groups[1].Provider != "deepseek-official" {
		t.Fatal("provider order changed")
	}
	groups = groupedModelChoices(snap, "DsC")
	if len(groups) != 1 || len(groups[0].Choices) != 3 || groups[0].Choices[0].Label != "a" || groups[0].Choices[1].Label != "z" || groups[0].Choices[2].Label != "b" {
		t.Fatalf("prefix/subsequence/stable ranking: %+v", groups)
	}
	snap.ModelLabels = labels[:3]
	if len(groupedModelChoices(snap, "not-found")) != 3 {
		t.Fatal("small catalog retained hidden query")
	}
}

func TestGroupedModelPickerBusyAndSessionIsolation(t *testing.T) {
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "one", ModelLabels: []string{"one", "two"}}
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(s string) { chosen = s }}, testColors(), 24)
	}, 800, 600)
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(composerText("模型", "Model") + "  " + snap.SelectedModel + "  ›"); err != nil {
		t.Fatal(err)
	}
	snap.SelectedID = "other"
	tt.Frame()
	if _, ok := tt.Find("Choose model one"); ok {
		t.Fatal("menu crossed session")
	}
	snap.ModelSelectionBusy = true
	tt.Frame()
	if _, ok := tt.Find("Model catalog"); !ok {
		t.Fatal("busy selection removed its visible trigger")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(composerText("模型", "Model") + "  " + snap.SelectedModel + "  ›"); err != nil {
		t.Fatal("busy catalog cannot be inspected:", err)
	}
	if err := tt.Click("Choose model two"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if chosen != "" {
		t.Fatal("navigation submitted a choice")
	}
}

func TestGroupedModelPickerWithoutCatalogSelectionOpensModelPaneDirectly(t *testing.T) {
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "retired-model", ModelLabels: []string{"one", "two"}}
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(label string) { chosen = label }}, testColors(), 24)
	}, 800, 600)
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Choose model two"); err != nil {
		t.Fatal("missing selection did not directly open model catalog:", err)
	}
	if chosen != "two" || snap.SelectedModel != "retired-model" {
		t.Fatal("direct catalog selection lost Host authority")
	}
}

func TestGroupedModelPickerKeyboardSearchKeepsHostAuthority(t *testing.T) {
	for _, key := range []struct {
		name  string
		value ui.Key
	}{{"Enter", ui.KeyEnter}, {"Tab", ui.KeyTab}} {
		t.Run(key.name, func(t *testing.T) {
			snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "a", ModelLabels: []string{"a", "b", "c", "d", "e"}}
			chosen := ""
			tt := ui.NewTester(func(c *ui.Context) {
				paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(s string) { chosen = s }}, testColors(), 24)
			}, 800, 600)
			if err := tt.Click("Model catalog"); err != nil {
				t.Fatal(err)
			}
			// The root model row owns keyboard focus and drills into the searchable
			// model pane; the first click does not directly open the catalog.
			tt.Key(0, ui.KeyRight)
			if err := tt.Click("Search models"); err != nil {
				t.Fatal(err)
			}
			tt.Type("e")
			tt.Key(0, key.value)
			if chosen != "e" || snap.SelectedModel != "a" {
				t.Fatalf("keyboard selection %q", chosen)
			}
			if _, ok := tt.Find("Search models"); ok {
				t.Fatal("menu not closed after selection")
			}
			if !tt.Focused("Model catalog") {
				t.Fatal("keyboard selection did not return focus to the trigger")
			}
		})
	}
}

func TestGroupedModelPickerEscapeReturnsFocusWithoutSelecting(t *testing.T) {
	snap := session.ViewState{NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true}, SelectedID: "s", SelectedModel: "a", ModelLabels: []string{"a", "b", "c", "d", "e"}}
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(label string) { chosen = label }}, testColors(), 24)
	}, 800, 600)
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if !tt.Focused("Model catalog") || chosen != "" {
		t.Fatal("root Escape changed selection or failed to restore trigger focus")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	if _, ok := tt.Find("Search models"); !ok {
		t.Fatal("model search pane missing")
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.Find("Search models"); ok {
		t.Fatal("first Escape did not return from search to root")
	}
	tt.Key(0, ui.KeyEscape)
	if !tt.Focused("Model catalog") || chosen != "" {
		t.Fatal("search Escape changed selection or failed to restore trigger focus")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	if _, ok := tt.Find("Search models"); !ok {
		t.Fatal("model search pane missing before Shift+Tab")
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if _, ok := tt.Find("Search models"); ok {
		t.Fatal("Shift+Tab did not return from search to root")
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if !tt.Focused("Model catalog") || chosen != "" {
		t.Fatal("Shift+Tab changed selection or failed to restore trigger focus")
	}
}
