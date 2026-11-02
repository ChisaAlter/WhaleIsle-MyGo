package chrome

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func captionGitLabelFixture(t *testing.T, density string) (*ui.Tester, *WorkspaceTrailingModel, *[]string, *int) {
	t.Helper()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	actions := []string{}
	opens := 0
	m := &WorkspaceTrailingModel{Status: &git.RepositoryStatus{IsRepo: true, Branch: "feature/label", Dirty: true}, WorkspaceRoot: "fixture", GitLoaded: true, Density: density, OnGitAction: func(id string) { actions = append(actions, id) }, OnGitMenuOpen: func() { opens++ }}
	m.RenderBranches = func(c *ui.Context, anchor ui.Element, open *bool) {
		ui.PopoverBase(c.Key("branch-fixture-menu"), anchor, open, func(panel ui.Element) {
			panel.Width(180).Height(70).Background(c.Theme().Background).Children(func() {
				if ui.Button(c, "Choose branch").Clicked() {
					*open = false
				}
			})
		})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() { WorkspaceTrailing(c, *m); ui.Button(c, "After Git") })
	}, 800, 400)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	return tt, m, &actions, &opens
}
func captionGitFind(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("missing %q: %q", label, tt.Texts())
	}
	return r
}
func captionGitLabelShown(tt *ui.Tester, part string) bool {
	r, ok := tt.Find("Git " + part + " label")
	return ok && r.W > 0
}
func captionGitNear(t *testing.T, what string, got, want float32) {
	t.Helper()
	if d := got - want; d < -.1 || d > .1 {
		t.Fatalf("%s=%v want%v", what, got, want)
	}
}

func TestCaptionGitLabelsStartHiddenInEveryDensityAndRevealIndependently(t *testing.T) {
	for _, density := range []string{"full", "cozy", "compact"} {
		t.Run(density, func(t *testing.T) {
			tt, _, _, _ := captionGitLabelFixture(t, density)
			if captionGitLabelShown(tt, "branch") || captionGitLabelShown(tt, "action") {
				t.Fatal("labels start revealed")
			}
			branch := captionGitFind(t, tt, "Switch branch")
			quick := captionGitFind(t, tt, "Commit")
			menu := captionGitFind(t, tt, "Git action options")
			wantBranch, wantQuick := float32(52), float32(34)
			if density == "compact" {
				wantBranch, wantQuick = 48, 30
			}
			captionGitNear(t, "collapsed branch", branch.W, wantBranch)
			captionGitNear(t, "collapsed quick", quick.W, wantQuick)
			captionGitNear(t, "menu", menu.W, 24)
			tt.Move(branch.X+3, branch.Y+branch.H/2)
			if !captionGitLabelShown(tt, "branch") || captionGitLabelShown(tt, "action") {
				t.Fatal("branch hover did not reveal only branch", tt.Texts())
			}
			quick = captionGitFind(t, tt, "Commit")
			tt.Move(quick.X+3, quick.Y+quick.H/2)
			if !captionGitLabelShown(tt, "branch") || !captionGitLabelShown(tt, "action") {
				t.Fatal("moving inside toolbar lost the latched branch", tt.Texts())
			}
			tt.Move(750, 300)
			if captionGitLabelShown(tt, "branch") || captionGitLabelShown(tt, "action") {
				t.Fatal("whole toolbar leave kept a label")
			}
			// Moving over the dropdown is action hover too, without revealing branch.
			menu = captionGitFind(t, tt, "Git action options")
			tt.Move(menu.X+menu.W/2, menu.Y+menu.H/2)
			if captionGitLabelShown(tt, "branch") || !captionGitLabelShown(tt, "action") {
				t.Fatal("dropdown hover did not reveal only action")
			}
		})
	}
}
func TestCaptionGitOpenMenusKeepOnlyTheirOwnLabelOnPointerLeave(t *testing.T) {
	for _, part := range []string{"branch", "action"} {
		t.Run(part, func(t *testing.T) {
			tt, _, _, opens := captionGitLabelFixture(t, "compact")
			label := "Switch branch"
			if part == "action" {
				label = "Git action options"
			}
			// A physical pointer enters before it clicks. Reveal may grow the
			// preceding quick label, so click the dropdown's new settled seat.
			seat := captionGitFind(t, tt, label)
			tt.Move(seat.X+seat.W/2, seat.Y+seat.H/2)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			tt.Move(750, 300)
			if !captionGitLabelShown(tt, part) {
				t.Fatal("open menu lost its label on toolbar leave")
			}
			other := "action"
			if part == "action" {
				other = "branch"
				if *opens != 1 {
					t.Fatalf("menu refresh callbacks=%d", *opens)
				}
			}
			if captionGitLabelShown(tt, other) {
				t.Fatal("one menu revealed the other part")
			}
			tt.Key(0, ui.KeyEscape)
			if captionGitLabelShown(tt, part) {
				t.Fatal("closed pointer-opened menu kept its label")
			}
		})
	}
}
func TestCaptionGitKeyboardRevealAndCallbacks(t *testing.T) {
	tt, _, actions, opens := captionGitLabelFixture(t, "full")
	// Initial automatic focus is not keyboard-visible. Tab to the branch and
	// continue with the toolkit's real key path rather than setting a test flag.
	for n := 0; n < 5 && !tt.Focused("Switch branch"); n++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Switch branch") {
		t.Fatal("branch is absent from keyboard order")
	}
	// If it already had automatic focus, cycling brings it back via the keyboard.
	if !captionGitLabelShown(tt, "branch") {
		for n := 0; n < 4; n++ {
			tt.Key(0, ui.KeyTab)
		}
	}
	if !captionGitLabelShown(tt, "branch") || captionGitLabelShown(tt, "action") {
		t.Fatal("keyboard branch focus reveal wrong", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Commit") || !captionGitLabelShown(tt, "action") || captionGitLabelShown(tt, "branch") {
		t.Fatal("keyboard action focus failed", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(*actions) != 1 || (*actions)[0] != "commit" {
		t.Fatalf("keyboard quick action=%v", *actions)
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Git action options") {
		t.Fatal("menu not in keyboard order")
	}
	tt.Key(0, ui.KeySpace)
	if *opens != 1 {
		t.Fatalf("keyboard menu-open callback=%d", *opens)
	}
	// Publish exists only in the popup, so this cannot accidentally click the
	// still-present quick Commit control underneath the nonmodal menu.
	if err := tt.Click("Publish repository"); err != nil {
		t.Fatal(err)
	}
	if len(*actions) != 2 || (*actions)[1] != "publish" {
		t.Fatalf("menu click action=%v", *actions)
	}
}
func TestCaptionGitLabelMotionUsesTheNativeClock(t *testing.T) {
	tt, _, _, _ := captionGitLabelFixture(t, "full")
	tt.SetPreferences(ui.Preferences{})
	now := time.Now()
	tt.SetTime(now)
	branch := captionGitFind(t, tt, "Switch branch")
	tt.Move(branch.X+2, branch.Y+branch.H/2)
	tt.SetTime(now.Add(100 * time.Millisecond))
	halfway := captionGitFind(t, tt, "Git branch label").W
	tt.SetTime(now.Add(200 * time.Millisecond))
	full := captionGitFind(t, tt, "Git branch label").W
	if !(halfway > 0 && halfway < full) {
		t.Fatalf("native label transition does not progress: half%v full%v", halfway, full)
	}
	tt.Move(750, 300)
	tt.SetTime(now.Add(400 * time.Millisecond))
	if captionGitLabelShown(tt, "branch") {
		t.Fatal("native close transition never settled")
	}
}
func TestCaptionGitActionLabelsStayEnglishInBothLocales(t *testing.T) {
	for _, locale := range []string{"zh", "en"} {
		i18n.SetLocale(locale)
		for _, tc := range []struct {
			s    git.RepositoryStatus
			want string
		}{{git.RepositoryStatus{Branch: "main", Dirty: true}, "Commit"}, {git.RepositoryStatus{Branch: "main", Dirty: true, IsDefault: true, HasOrigin: true}, "Commit & push"}, {git.RepositoryStatus{Branch: "topic", Dirty: true, HasOrigin: true}, "Commit, push & PR"}, {git.RepositoryStatus{Branch: "topic", HasOrigin: true, Ahead: 1}, "Push & create PR"}, {git.RepositoryStatus{Branch: "topic", HasUpstream: true, Behind: 1}, "Pull"}, {git.RepositoryStatus{Branch: "topic", HasUpstream: true, Behind: 1, Ahead: 1}, "Sync branch"}} {
			if got := resolveCaptionQuick(WorkspaceTrailingModel{Status: &tc.s}).label; got != tc.want {
				t.Fatalf("locale%s label%q want%q", locale, got, tc.want)
			}
		}
	}
	i18n.SetLocale(i18n.DefaultLocale)
}
