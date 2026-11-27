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
	tt.SetPreferences(ui.Preferences{ReduceMotion: false})
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

func captionGitNear(t *testing.T, what string, got, want float32) {
	t.Helper()
	if d := got - want; d < -.1 || d > .1 {
		t.Fatalf("%s=%v want%v", what, got, want)
	}
}

func TestCaptionGitToolbarConstantWidthZeroShiftAcrossDensities(t *testing.T) {
	for _, density := range []string{"full", "cozy", "compact"} {
		t.Run(density, func(t *testing.T) {
			tt, _, _, _ := captionGitLabelFixture(t, density)
			now := time.Unix(2000, 0)
			tt.SetTime(now)

			toolbar0 := captionGitFind(t, tt, "Git toolbar")
			branch0 := captionGitFind(t, tt, "Switch branch")
			quick0 := captionGitFind(t, tt, "Commit")
			menu0 := captionGitFind(t, tt, "Git action options")
			after0 := captionGitFind(t, tt, "After Git")

			if density == "compact" {
				if tt.HasText("feature/label") {
					t.Fatalf("compact density must remain icon-only, got %q", tt.Texts())
				}
				captionGitNear(t, "compact branch", branch0.W, 48)
				captionGitNear(t, "compact quick", quick0.W, 30)
			} else {
				if !tt.HasText("feature/label") {
					t.Fatalf("%s density must render static truncated branch label, got %q", density, tt.Texts())
				}
				if branch0.W <= 52 {
					t.Fatalf("%s branch button unexpectedly collapsed: %+v", density, branch0)
				}
			}
			captionGitNear(t, "menu", menu0.W, 24)

			assertNoShift := func(step string) {
				t.Helper()
				if got := captionGitFind(t, tt, "Git toolbar"); got != toolbar0 {
					t.Fatalf("%s shifted toolbar: %+v -> %+v", step, toolbar0, got)
				}
				if got := captionGitFind(t, tt, "Switch branch"); got != branch0 {
					t.Fatalf("%s shifted branch: %+v -> %+v", step, branch0, got)
				}
				if got := captionGitFind(t, tt, "Commit"); got != quick0 {
					t.Fatalf("%s shifted quick action: %+v -> %+v", step, quick0, got)
				}
				if got := captionGitFind(t, tt, "Git action options"); got != menu0 {
					t.Fatalf("%s shifted action menu trigger: %+v -> %+v", step, menu0, got)
				}
				if got := captionGitFind(t, tt, "After Git"); got != after0 {
					t.Fatalf("%s shifted sibling control: %+v -> %+v", step, after0, got)
				}
			}

			// Hover each control and advance clock across old 200ms reveal window:
			// geometry must stay 100% invariant while hover paints a background tint.
			tt.Move(750, 300)
			tt.Frame()
			idlePx := tt.Image().RGBAAt(int(branch0.X+2), int(branch0.Y+branch0.H/2))

			tt.Move(branch0.X+3, branch0.Y+branch0.H/2)
			tt.SetTime(now.Add(100 * time.Millisecond))
			assertNoShift("branch hover 100ms")
			tt.SetTime(now.Add(250 * time.Millisecond))
			assertNoShift("branch hover 250ms")
			hoverPx := tt.Image().RGBAAt(int(branch0.X+2), int(branch0.Y+branch0.H/2))
			if hoverPx == idlePx {
				t.Fatalf("branch hover did not paint hover background tint: idle=%v hover=%v", idlePx, hoverPx)
			}

			tt.Move(quick0.X+3, quick0.Y+quick0.H/2)
			tt.SetTime(now.Add(350 * time.Millisecond))
			assertNoShift("quick hover")

			tt.Move(menu0.X+menu0.W/2, menu0.Y+menu0.H/2)
			tt.SetTime(now.Add(450 * time.Millisecond))
			assertNoShift("menu hover")

			tt.Move(750, 300)
			tt.SetTime(now.Add(650 * time.Millisecond))
			assertNoShift("pointer leave")
		})
	}
}

func TestCaptionGitOpenMenusStayStableOnPointerLeave(t *testing.T) {
	for _, part := range []string{"branch", "action"} {
		t.Run(part, func(t *testing.T) {
			tt, _, _, opens := captionGitLabelFixture(t, "compact")
			label := "Switch branch"
			if part == "action" {
				label = "Git action options"
			}
			toolbar0 := captionGitFind(t, tt, "Git toolbar")
			seat := captionGitFind(t, tt, label)
			tt.Move(seat.X+seat.W/2, seat.Y+seat.H/2)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			tt.Move(750, 300)
			tt.Frame()
			if got := captionGitFind(t, tt, "Git toolbar"); got != toolbar0 {
				t.Fatalf("open %s menu shifted toolbar: %+v -> %+v", part, toolbar0, got)
			}
			if part == "action" && *opens != 1 {
				t.Fatalf("menu refresh callbacks=%d", *opens)
			}
			if part == "branch" && !tt.HasText("Choose branch") {
				t.Fatalf("branch menu closed on pointer leave: %q", tt.Texts())
			}
			if part == "action" && !tt.HasText("Publish repository") {
				t.Fatalf("action menu closed on pointer leave: %q", tt.Texts())
			}
			tt.Key(0, ui.KeyEscape)
			if tt.HasText("Choose branch") || tt.HasText("Publish repository") {
				t.Fatalf("Escape did not close %s menu: %q", part, tt.Texts())
			}
		})
	}
}

func TestCaptionGitKeyboardNavigationAndCallbacksZeroShift(t *testing.T) {
	tt, _, actions, opens := captionGitLabelFixture(t, "full")
	toolbar0 := captionGitFind(t, tt, "Git toolbar")
	for n := 0; n < 5 && !tt.Focused("Switch branch"); n++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Switch branch") {
		t.Fatal("branch is absent from keyboard order")
	}
	if got := captionGitFind(t, tt, "Git toolbar"); got != toolbar0 {
		t.Fatalf("branch keyboard focus shifted toolbar: %+v -> %+v", toolbar0, got)
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Commit") {
		t.Fatal("keyboard action focus failed", tt.Texts())
	}
	if got := captionGitFind(t, tt, "Git toolbar"); got != toolbar0 {
		t.Fatalf("action keyboard focus shifted toolbar: %+v -> %+v", toolbar0, got)
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
	if err := tt.Click("Publish repository"); err != nil {
		t.Fatal(err)
	}
	if len(*actions) != 2 || (*actions)[1] != "publish" {
		t.Fatalf("menu click action=%v", *actions)
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
