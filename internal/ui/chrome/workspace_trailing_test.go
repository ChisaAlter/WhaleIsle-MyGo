package chrome

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceTrailingSourceActionCases(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   git.RepositoryStatus
		open     bool
		action   string
		disabled bool
	}{
		{"local dirty", git.RepositoryStatus{IsRepo: true, Branch: "main", Dirty: true}, false, "commit", false},
		{"default dirty", git.RepositoryStatus{IsRepo: true, Branch: "main", Dirty: true, HasOrigin: true, IsDefault: true}, false, "commit_push", false},
		{"feature dirty", git.RepositoryStatus{IsRepo: true, Branch: "feature", Dirty: true, HasOrigin: true}, false, "commit_push_pr", false},
		{"clean default ahead", git.RepositoryStatus{IsRepo: true, Branch: "main", HasOrigin: true, HasUpstream: true, IsDefault: true, Ahead: 1}, false, "commit_push", false},
		{"behind", git.RepositoryStatus{IsRepo: true, Branch: "feature", HasUpstream: true, Behind: 1}, false, "pull", false},
		{"diverged", git.RepositoryStatus{IsRepo: true, Branch: "feature", HasUpstream: true, Ahead: 1, Behind: 1}, false, "", true},
		{"no origin", git.RepositoryStatus{IsRepo: true, Branch: "main"}, false, "publish", false},
		{"existing PR", git.RepositoryStatus{IsRepo: true, Branch: "feature", HasUpstream: true}, true, "open_pr", false},
		{"detached", git.RepositoryStatus{IsRepo: true, Dirty: true}, false, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := resolveCaptionQuick(WorkspaceTrailingModel{Status: &tc.status, ChangeRequestOpen: tc.open})
			if a.id != tc.action || a.disabled != tc.disabled {
				t.Fatalf("action=%+v", a)
			}
		})
	}
}

func TestWorkspaceTrailingDensityAndIndependentPanelGates(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, density := range []string{"full", "cozy", "compact"} {
		t.Run(density, func(t *testing.T) {
			terminal, surfaces := 0, 0
			m := WorkspaceTrailingModel{Status: &git.RepositoryStatus{IsRepo: true, Branch: "feature/long-name", Dirty: true}, WorkspaceRoot: "fixture", GitLoaded: true, Density: density, OnGitAction: func(string) {}, ShowTerminal: true, ShowSurfaces: true, TerminalAvailable: false, OnToggleTerminal: func() { terminal++ }, OnToggleSurfaces: func() { surfaces++ }}
			tt := ui.NewTester(func(c *ui.Context) { WorkspaceTrailing(c, m) }, 800, 100)
			if (tt.HasText("feature/long-name")) != (density != "compact") {
				t.Fatal("branch density did not follow source", tt.Texts())
			}
			commitLabels := 0
			for _, label := range tt.Texts() {
				if label == "Commit" {
					commitLabels++
				}
			}
			wantLabels := 1
			if density == "full" {
				wantLabels = 2
			}
			if commitLabels != wantLabels {
				t.Fatal("primary visible text density did not follow source", tt.Texts())
			}
			if err := tt.Click("Toggle terminal drawer"); err != nil {
				t.Fatal(err)
			}
			if terminal != 0 {
				t.Fatal("terminal opened without a workspace")
			}
			if err := tt.Click("Toggle right panel"); err != nil {
				t.Fatal(err)
			}
			if surfaces != 1 {
				t.Fatal("blank/global surface toggle was disabled")
			}
			m.HideGit = true
			tt.Frame()
			if _, ok := tt.Find("Git action options"); ok {
				t.Fatal("explicit hide preference left Git controls")
			}
			if _, ok := tt.Find("Toggle right panel"); !ok {
				t.Fatal("Git preference hid panel toggles")
			}
			m.HideGit = false
			m.ManagedSession = true
			tt.Frame()
			if _, ok := tt.Find("Git action options"); ok {
				t.Fatal("managed session retained Git")
			}
			if _, ok := tt.Find("Toggle terminal drawer"); !ok {
				t.Fatal("managed session hid independent panels")
			}
		})
	}
}

func TestWorkspaceTrailingDisabledReasonsRemainVisibleOnHover(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, menu := range []bool{false, true} {
		name := "primary"
		if menu {
			name = "menu"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			status := &git.RepositoryStatus{IsRepo: true, Branch: "main", HasOrigin: true, HasUpstream: menu}
			m := WorkspaceTrailingModel{Status: status, WorkspaceRoot: "fixture", GitLoaded: true, Density: "full", OnGitAction: func(string) { calls++ }}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(16).Children(func() { WorkspaceTrailing(c, m) })
			}, 800, 400)
			now := time.Now()
			tt.SetTime(now)
			reason := "No local commits to push"
			if menu {
				if err := tt.Click("Git action options"); err != nil {
					t.Fatal(err)
				}
				reason = "Requires local commits on a branch that is not behind upstream"
			}
			button, ok := tt.Find("Push")
			if !ok {
				t.Fatal("missing disabled Push action", tt.Texts())
			}
			tt.Move(button.X+button.W/2, button.Y+button.H/2)
			tt.SetTime(now.Add(time.Second))
			if !tt.HasText(reason) {
				t.Fatalf("disabled %s action hid its reason: %q", name, tt.Texts())
			}
			if err := tt.Click("Push"); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("disabled action dispatched Git")
			}
		})
	}
}

func TestWorkspaceNonrepoInitDoesNotRequireBranchBadge(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var actions []string
	tt := ui.NewTester(func(c *ui.Context) {
		WorkspaceTrailing(c, WorkspaceTrailingModel{Status: &git.RepositoryStatus{}, GitLoaded: true, WorkspaceRoot: "fixture", OnGitAction: func(action string) { actions = append(actions, action) }})
	}, 500, 100)
	if len(actions) != 0 {
		t.Fatal("paint executed Initialize")
	}
	if err := tt.Click("Initialize Git"); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0] != "init" {
		t.Fatalf("wrong initialization gesture: %v", actions)
	}
}
