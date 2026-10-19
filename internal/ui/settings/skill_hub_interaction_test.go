package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func skillHubEnglish(t *testing.T) {
	t.Helper()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
}

func clickSkillControl(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestSkillsHubScopeAndInstallReviewInteraction(t *testing.T) {
	skillHubEnglish(t)
	query := "synthetic"
	hit := SkillHubHit{SkillName: "owned-skill", Repo: "owned/repository", Path: "skills/owned-skill", Stars: 7}
	hub := &SkillHubAccess{}
	a := &SkillsAccess{Create: &SkillCreateAccess{}, Hub: hub, ProjectCwd: "owned-workspace", TargetRoot: "user-dsh", HubQuery: &query, HubHits: []SkillHubHit{hit}}
	var roots []string
	searches, reviews, installs := 0, 0, 0
	a.OnSearchHub = func(string) { searches++ }
	a.OnSelectRoot = func(root string) { roots = append(roots, root); a.TargetRoot = root }
	a.OnInstallHub = func(repo, path, name string) {
		if repo != hit.Repo || path != hit.Path || name != hit.SkillName {
			t.Fatal("review changed the selected source")
		}
		reviews++
		hub.Confirming = &hit
	}
	hub.OnOpen = func() { hub.Open = true }
	hub.OnClose = func() { hub.Open, hub.Confirming = false, nil }
	hub.OnConfirmInstall = func() { installs++ }
	hub.OnCancelInstall = func() { hub.Confirming = nil }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(8).Children(func() { skillsPane(c, Model{Skills: a}, c.Theme().Text, c.Theme().TextMuted) })
	}, 1100, 900)
	clickSkillControl(t, tt, "Find skills")
	clickSkillControl(t, tt, "Project skills")
	clickSkillControl(t, tt, "User skills")
	if strings.Join(roots, ",") != "project-dsh,user-dsh" || !hub.Open || a.Create.Open {
		t.Fatalf("Hub roots depend on creation: roots=%v open=%v create=%v", roots, hub.Open, a.Create.Open)
	}
	clickSkillControl(t, tt, i18n.T(i18n.KeySettingsSkillsHubInstall))
	if reviews != 1 || installs != 0 || searches != 0 {
		t.Fatalf("selecting Install bypassed review: reviews=%d installs=%d searches=%d", reviews, installs, searches)
	}
	for _, want := range []string{"Confirm skill installation", hit.Repo, hit.Path, "User skills", "not be overwritten"} {
		if !tt.HasText(want) {
			t.Fatalf("review omitted %q: %v", want, tt.Texts())
		}
	}
	clickSkillControl(t, tt, "Cancel")
	if installs != 0 || hub.Confirming != nil {
		t.Fatal("cancelling review installed a skill")
	}
	clickSkillControl(t, tt, i18n.T(i18n.KeySettingsSkillsHubInstall))
	clickSkillControl(t, tt, "Confirm installation")
	if installs != 1 {
		t.Fatalf("explicit confirmation did not invoke installation once: %d", installs)
	}
}

func TestSkillsHubBusyControlsAndMissingProjectInteraction(t *testing.T) {
	skillHubEnglish(t)
	query := "synthetic"
	hit := SkillHubHit{SkillName: "owned-skill", Repo: "owned/repository", Path: "skills/owned-skill"}
	hub := &SkillHubAccess{Open: true}
	a := &SkillsAccess{Hub: hub, HubQuery: &query, HubHits: []SkillHubHit{hit}, TargetRoot: "user-dsh"}
	operations := 0
	a.OnSelectRoot = func(string) { operations++ }
	a.OnSearchHub = func(string) { operations++ }
	a.OnInstallHub = func(string, string, string) { operations++ }
	hub.OnClose = func() { operations++ }
	hub.OnCancelInstall = func() { operations++ }
	hub.OnConfirmInstall = func() { operations++ }
	tt := ui.NewTester(func(c *ui.Context) { skillHubDialog(c, a, c.Theme().Text, c.Theme().TextMuted) }, 1100, 900)
	clickSkillControl(t, tt, "Project skills")
	if operations != 0 {
		t.Fatal("project scope was selectable without a workspace")
	}
	hub.Searching = true
	tt.Frame()
	for _, label := range []string{"Close", "Searching…", "Project skills", i18n.T(i18n.KeySettingsSkillsHubInstall)} {
		clickSkillControl(t, tt, label)
	}
	hub.Searching, hub.Installing, hub.Confirming = false, true, &hit
	tt.Frame()
	for _, label := range []string{"Close", "Cancel", "Installing…"} {
		clickSkillControl(t, tt, label)
	}
	if operations != 0 || !hub.Open {
		t.Fatalf("busy UI admitted an action or closure: actions=%d open=%v", operations, hub.Open)
	}
}

func TestSkillCreateValidationSurvivesFramesAndAllowsCorrectedInput(t *testing.T) {
	skillHubEnglish(t)
	draft := SkillCreateDraft{Description: "Owned description", Content: "Owned body", Root: "user-dsh", ModelInvocable: true, UserInvocable: true}
	create := &SkillCreateAccess{Open: true, Draft: &draft}
	a := &SkillsAccess{Create: create, ProjectCwd: "owned-workspace"}
	var saved []SkillCreateDraft
	create.OnSave = func(d SkillCreateDraft) { saved = append(saved, d) }
	tt := ui.NewTester(func(c *ui.Context) { skillCreateDialog(c, a, c.Theme().Text, c.Theme().TextMuted) }, 1100, 900)
	tt.Scroll(650, 600, 0, 4000)
	clickSkillControl(t, tt, "Create skill")
	tt.Frame()
	if len(saved) != 0 || !tt.HasText("Use a lowercase kebab-case name") {
		t.Fatal("invalid create did not retain its rendered validation message")
	}
	tt.Scroll(650, 600, 0, -4000)
	label, ok := tt.Find("Name")
	if !ok {
		t.Fatal("name field label missing")
	}
	// The production form places its text input immediately below the name label.
	tt.ClickAt(label.X+50, label.Y+label.H+24)
	tt.Type("owned-skill")
	if draft.Name != "owned-skill" {
		t.Fatalf("native input did not update the creation draft: %q", draft.Name)
	}
	tt.Scroll(650, 600, 0, 4000)
	clickSkillControl(t, tt, "Project skills")
	clickSkillControl(t, tt, "Create skill")
	if len(saved) != 1 || saved[0].Name != "owned-skill" || saved[0].Root != "project-dsh" || saved[0].Content != draft.Content {
		t.Fatalf("corrected form was not submitted intact: %+v", saved)
	}
}
