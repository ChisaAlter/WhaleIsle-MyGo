package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// skillsSurfaceFixture renders the skills page with two rows, one of them bundled,
// so the header actions, the group count and the row chrome all paint for real.
func skillsSurfaceFixture(t *testing.T, a *SkillsAccess) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(12).Children(func() {
			skillsPane(c, Model{Skills: a}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 780, 900)
}

func skillsSurfaceAccess() *SkillsAccess {
	query := ""
	source := ""
	toggled := map[string]bool{}
	a := &SkillsAccess{
		Skills: []SkillRow{
			{Name: "research", Description: "Read and summarize source material", Source: "bundled", Writable: true, ModelInvocable: true},
			{Name: "writer", Description: "Draft and revise prose", Source: "user-dsh", Writable: true},
		},
		Query:               &query,
		SourceFilter:        &source,
		OnSetModelInvocable: func(name string, value bool) { toggled[name] = value },
		OnRefresh:           func() {},
	}
	return a
}

// TestSkillsHeaderOffersThreeActionsAndFilters covers the page chrome: the three
// header actions, the two filter controls and the per-group count line.
func TestSkillsHeaderOffersThreeActionsAndFilters(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	a := skillsSurfaceAccess()
	tt := skillsSurfaceFixture(t, a)
	if !tt.HasText(i18n.T(i18n.KeySurfacesSkillsBrowseHub)) {
		t.Fatalf("browse action missing: %v", tt.Texts())
	}
	if _, ok := tt.Find("Skill source filter"); !ok {
		t.Fatalf("source dropdown missing: %v", tt.Texts())
	}
	if !tt.HasText("2 workspace and personal skills") {
		t.Fatalf("group count line missing: %v", tt.Texts())
	}
	if !tt.HasText("Bundled") {
		t.Fatalf("bundled badge missing: %v", tt.Texts())
	}
}

// TestSkillsRowSwitchIsTheHarnessSizeAndToggles keeps the row control honest: the
// 44×24 switch flips the model-invocation value of its own skill only.
func TestSkillsRowSwitchIsTheHarnessSizeAndToggles(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	toggled := map[string]bool{}
	a := skillsSurfaceAccess()
	a.OnSetModelInvocable = func(name string, value bool) { toggled[name] = value }
	tt := skillsSurfaceFixture(t, a)
	rect, ok := tt.Find("Model invocation · writer")
	if !ok {
		t.Fatalf("row switch missing: %v", tt.Texts())
	}
	if rect.W != toggleWidth || rect.H != toggleHeight {
		t.Fatalf("row switch is %v×%v, want %v×%v", rect.W, rect.H, toggleWidth, toggleHeight)
	}
	if err := tt.Click("Model invocation · writer"); err != nil {
		t.Fatalf("row switch is not clickable: %v", err)
	}
	if value, seen := toggled["writer"]; !seen || !value {
		t.Fatalf("row switch did not report its own row: %v", toggled)
	}
}
