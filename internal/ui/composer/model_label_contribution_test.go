package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func modelLabelFixture() session.ViewState {
	return session.ViewState{
		SelectedID: "label-session", SelectedModel: "official opaque key",
		NativeContributions: dsh.NativeContributions{Known: true, ModelCommand: true, ModelLabel1017: true},
		ModelLabels:         []string{"official opaque key", "account opaque key", "custom opaque key", "small opaque key", "other opaque key"},
		ModelChoices: map[string]api.ModelCatalogDefault{
			"official opaque key": {Provider: "deepseek-official", Model: "deepseek-flash"},
			"account opaque key":  {Provider: "deepseek-account", Model: "deepseek-flash"},
			"custom opaque key":   {Provider: "custom", Model: "deepseek-flash"},
			"small opaque key":    {Provider: "custom", Model: "small"},
			"other opaque key":    {Provider: "custom", Model: "other"},
		},
		ModelCatalogChoices: []session.ModelCatalogChoice{
			{Label: "official opaque key", Name: "DeepSeek-V4.1-Flash", Provider: "deepseek-official", ProviderName: "Official"},
			{Label: "account opaque key", Name: "DeepSeek-V4.1-Flash", Provider: "deepseek-account", ProviderName: "Account"},
			{Label: "custom opaque key", Name: "DeepSeek-V4.1-Flash", Provider: "custom", ProviderName: "Custom"},
			{Label: "small opaque key", Name: "Small", Provider: "custom", ProviderName: "Custom"},
			{Label: "other opaque key", Name: "Other", Provider: "custom", ProviderName: "Custom"},
		},
	}
}

func modelBadgeCount(tt *ui.Tester) int {
	count := 0
	for _, text := range tt.Texts() {
		if text == "1017" {
			count++
		}
	}
	return count
}

func TestModelLabelExactIdentityAcrossPlacements(t *testing.T) {
	for _, placement := range []ModelLabelPlacement{ModelLabelTrigger, ModelLabelMenu, ModelLabelOption} {
		for _, selected := range []bool{false, true} {
			for _, provider := range []string{"deepseek-account", "deepseek-official", "custom"} {
				t.Run(string(placement)+"/"+provider+map[bool]string{false: "/unselected", true: "/selected"}[selected], func(t *testing.T) {
					snap := modelLabelFixture()
					route := snap.ModelChoices["official opaque key"]
					route.Provider = provider
					snap.ModelChoices["official opaque key"] = route
					snap.ModelCatalogChoices[0].Provider = provider
					tt := ui.NewTester(func(c *ui.Context) {
						ModelLabel(c, snap, "official opaque key", placement, selected, c.Theme().Text, 13)
					}, 300, 80)
					if !tt.HasText("DeepSeek-V4.1-Flash") {
						t.Fatal("catalog name disappeared")
					}
					want := 1
					if provider == "custom" {
						want = 0
					}
					if count := modelBadgeCount(tt); count != want {
						t.Fatalf("badge count=%d want %d", count, want)
					}
				})
			}
		}
	}
}

func TestModelLabelDoesNotInferMissingOrDifferentIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		text   string
		change func(*session.ViewState)
	}{
		{"unknown graph", "DeepSeek-V4.1-Flash", func(s *session.ViewState) { s.NativeContributions.Known = false }},
		{"withdrawn adapter", "DeepSeek-V4.1-Flash", func(s *session.ViewState) { s.NativeContributions.ModelLabel1017 = false }},
		{"missing catalog", "official opaque key", func(s *session.ViewState) { s.ModelCatalogChoices = nil }},
		{"missing route", "official opaque key", func(s *session.ViewState) { s.ModelChoices = nil }},
		{"different route", "DeepSeek-V4.1-Flash", func(s *session.ViewState) {
			route := s.ModelChoices["official opaque key"]
			route.Model = "other"
			s.ModelChoices["official opaque key"] = route
		}},
		{"mismatched provider metadata", "official opaque key", func(s *session.ViewState) { s.ModelCatalogChoices[0].Provider = "custom" }},
		{"custom catalog name", "My Flash", func(s *session.ViewState) { s.ModelCatalogChoices[0].Name = "My Flash" }},
		{"dated catalog name", "DeepSeek-V4.1-Flash-1017", func(s *session.ViewState) { s.ModelCatalogChoices[0].Name = "DeepSeek-V4.1-Flash-1017" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := modelLabelFixture()
			tc.change(&snap)
			tt := ui.NewTester(func(c *ui.Context) {
				ModelLabel(c, snap, "official opaque key", ModelLabelOption, false, c.Theme().Text, 13)
			}, 320, 80)
			if modelBadgeCount(tt) != 0 {
				t.Fatal("incomplete or nonmatching identity invented a release badge")
			}
			if _, ok := tt.Find(tc.text); !ok {
				t.Fatalf("missing or nonmatching contribution removed fallback %q", tc.text)
			}
		})
	}
}

func TestModelPickerUnknownCurrentHasNoInferredBadge(t *testing.T) {
	snap := modelLabelFixture()
	snap.SelectedModel = "deepseek-official / deepseek-flash"
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap}, c.Theme().TextMuted, 28)
	}, 600, 600)
	if modelBadgeCount(tt) != 0 {
		t.Fatal("unknown current choice borrowed another catalog entry's badge")
	}
	if _, ok := tt.Find(snap.SelectedModel); !ok {
		t.Fatal("unknown current route lost its owner's fallback")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Choose model account opaque key"); !ok {
		t.Fatal("unknown current route could not open the real model choices")
	}
	if modelBadgeCount(tt) != 2 {
		t.Fatal("known matching catalog options did not retain their own badges")
	}
}

func TestModelPickerColdUnknownDoesNotInventSeat(t *testing.T) {
	for _, contribution := range []dsh.NativeContributions{
		{ModelCommand: true, ModelLabel1017: true},
		{Known: true, ModelLabel1017: true},
	} {
		snap := modelLabelFixture()
		snap.NativeContributions = contribution
		tt := ui.NewTester(func(c *ui.Context) {
			paintModelPicker(c, Model{Snapshot: snap}, c.Theme().TextMuted, 28)
		}, 600, 100)
		if _, ok := tt.Find("Model catalog"); ok || modelBadgeCount(tt) != 0 || tt.HasText("DeepSeek-V4.1-Flash") {
			t.Fatal("unknown or unavailable model owner invented a local seat")
		}
	}
}

func TestModelPickerLabelAndOwnerWithdrawalsHaveDifferentLifetimes(t *testing.T) {
	snap := modelLabelFixture()
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintModelPicker(c, Model{Snapshot: snap, OnModelChange: func(label string) { chosen = label }}, c.Theme().TextMuted, 28)
	}, 600, 600)
	if modelBadgeCount(tt) != 1 {
		t.Fatal("actual trigger is missing its label contribution")
	}
	trigger, triggerOK := tt.Find("Model catalog")
	badge, badgeOK := tt.Find("1017")
	if !triggerOK || !badgeOK || badge.X < trigger.X || badge.X+badge.W > trigger.X+trigger.W+.1 {
		t.Fatal("trigger badge is outside its owning control")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	if modelBadgeCount(tt) != 2 {
		t.Fatal("actual root menu value did not render its own label")
	}
	if err := tt.Click(composerText("模型", "Model") + "  " + snap.SelectedModel + "  ›"); err != nil {
		t.Fatal(err)
	}
	if modelBadgeCount(tt) != 3 {
		t.Fatal("matching selected and unselected options must both be marked")
	}
	if err := tt.Click("Search models"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Flash")
	snap.NativeContributions.ModelLabel1017 = false
	tt.Frame()
	if modelBadgeCount(tt) != 0 {
		t.Fatal("withdrawn label adapter remained visible")
	}
	if _, ok := tt.Find("Search models"); !ok {
		t.Fatal("label-only withdrawal closed the owner's menu")
	}
	if _, ok := tt.Find("Choose model small opaque key"); ok {
		t.Fatal("label-only withdrawal discarded the active filter")
	}
	snap.NativeContributions.ModelLabel1017 = true
	tt.Frame()
	if modelBadgeCount(tt) != 3 {
		t.Fatal("label remount lost the retained menu or option contributions")
	}
	if err := tt.Click("Choose model account opaque key"); err != nil {
		t.Fatal(err)
	}
	if chosen != "account opaque key" || snap.SelectedModel != "official opaque key" {
		t.Fatal("decorated selection changed the opaque route or invented Host acknowledgement")
	}
	if err := tt.Click("Model catalog"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	snap.NativeContributions.ModelCommand = false
	tt.Frame()
	for _, label := range []string{"Model catalog", "Search models", "1017"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("owner withdrawal retained %q", label)
		}
	}
	snap.NativeContributions.ModelCommand = true
	tt.Frame()
	if _, ok := tt.Find("Model catalog"); !ok {
		t.Fatal("model owner remount did not restore its seat")
	}
	if _, ok := tt.Find("Search models"); ok {
		t.Fatal("withdrawn owner's local popup survived remount")
	}
	snap.NativeContributions.Known = false
	tt.Frame()
	if _, ok := tt.Find("Model catalog"); ok {
		t.Fatal("unknown contribution state retained the model seat")
	}
	if chosen != "account opaque key" || snap.ModelChoices[chosen].Model != "deepseek-flash" {
		t.Fatal("withdrawal modified the Host catalog or dispatched another selection")
	}
}
