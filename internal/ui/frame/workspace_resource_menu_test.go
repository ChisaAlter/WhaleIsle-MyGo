package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceResourceMenuRetainsRoutesAndBrowserCallback(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	selected := "files"
	browserCalls := 0
	marketCalls := 0
	var draftRegion string
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Button(c, "Resources").Menu(func(menu *ui.Menu) {
			workspaceResourceMenu(menu, Model{
				RightPaneTab:         &selected,
				OnOpenBrowserPreview: func() { browserCalls++ },
				OnOpenMarket:         func() { marketCalls++ },
				OnCreatePluginDraft:  func(region string) { draftRegion = region },
			})
		})
	}, 400, 100)
	for _, item := range []struct{ kind, label string }{
		{"team", "Team"},
		{"diff", i18n.T(i18n.KeyRightPaneDiff)}, {"agents", i18n.T(i18n.KeyRightPaneAgents)}, {"files", i18n.T(i18n.KeyRightPaneFiles)},
	} {
		if err := tt.Click("Resources"); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(item.label); err != nil {
			t.Fatal(err)
		}
		if selected != item.kind {
			t.Fatalf("menu route=%q want %q", selected, item.kind)
		}
	}
	if err := tt.Click("Resources"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyBrowserPreviewTitle)); err != nil {
		t.Fatal(err)
	}
	if browserCalls != 1 || selected != "files" {
		t.Fatalf("browser callback=%d route=%q", browserCalls, selected)
	}
	if err := tt.Click("Resources"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Browse other plugins"); err != nil {
		t.Fatal(err)
	}
	if marketCalls != 1 {
		t.Fatalf("marketCalls=%d want 1", marketCalls)
	}
	if err := tt.Click("Resources"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Create your plugin"); err != nil {
		t.Fatal(err)
	}
	if draftRegion != "rightbar" {
		t.Fatalf("draftRegion=%q want rightbar", draftRegion)
	}
}

func TestWorkspaceResourceGuideMoreToolsCardAndMenu(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, locale := range []struct {
		code, moreLabel, browseLabel, createLabel string
	}{
		{"zh", "更多工具", "浏览更多插件", "创建你的插件"},
		{"en", "More tools", "Browse other plugins", "Create your plugin"},
	} {
		t.Run(locale.code, func(t *testing.T) {
			i18n.SetLocale(locale.code)
			selected := "guide"
			marketCalls := 0
			var draftRegion string
			tt := ui.NewTester(func(c *ui.Context) {
				workspaceResourceGuide(c, Model{
					RightPaneTab:        &selected,
					OnOpenMarket:        func() { marketCalls++ },
					OnCreatePluginDraft: func(region string) { draftRegion = region },
				})
			}, 400, 600)
			if !tt.HasText(locale.moreLabel) {
				t.Fatalf("missing guide more label %q in texts %v", locale.moreLabel, tt.Texts())
			}
			for _, it := range tt.Interactives() {
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("interactive %q (%q) cursor=%v want CursorPointer", it.Role, it.Label, it.Cursor)
				}
			}
			if err := tt.Click(locale.moreLabel); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(locale.browseLabel); err != nil {
				t.Fatal(err)
			}
			if marketCalls != 1 {
				t.Fatalf("marketCalls=%d want 1", marketCalls)
			}
			if err := tt.Click(locale.moreLabel); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(locale.createLabel); err != nil {
				t.Fatal(err)
			}
			if draftRegion != "rightbar" {
				t.Fatalf("draftRegion=%q want rightbar", draftRegion)
			}
		})
	}
}
