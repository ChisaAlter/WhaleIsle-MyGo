package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestPlanBoardViewEmptyAndFilled(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		PlanBoardView(c, PlanBoardModel{
			Board:      session.PlanBoard{},
			ThemeID:    "system",
			SystemDark: false,
			Open:       &open,
		})
	}, 400, 200)
	for _, label := range []string{
		i18n.T(i18n.KeyPlanBoardTitle),
		i18n.T(i18n.KeyPlanBoardEmpty),
		i18n.T(i18n.KeyPlanBoardCollapse),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}

	board := session.PlanBoard{
		PlanMode: true,
		Goal:     &session.PlanGoal{Objective: "Ship panel", Phase: "active"},
		Todos: []session.PlanTodo{
			{Content: "parse events", Status: "completed"},
			{Content: "wire UI", Status: "in_progress"},
		},
		Queue: []session.PlanQueueItem{{ID: "1", Preview: "next prompt", Source: "team"}},
	}
	tt2 := ui.NewTester(func(c *ui.Context) {
		PlanBoardView(c, PlanBoardModel{
			Board:      board,
			ThemeID:    "system",
			SystemDark: false,
			Open:       &open,
		})
	}, 480, 320)
	for _, label := range []string{
		i18n.T(i18n.KeyPlanBoardPlanOn),
		i18n.T(i18n.KeyPlanBoardGoal),
		"[active] Ship panel",
		i18n.T(i18n.KeyPlanBoardQueue),
		"team: next prompt",
		i18n.T(i18n.KeyPlanBoardTodos),
		"parse events",
		"wire UI",
		"completed",
		"in_progress",
	} {
		if !tt2.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt2.Texts())
		}
	}
}

func TestPlanBoardGoalCreateControls(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	open := true
	draft := ""
	var created string
	tt := ui.NewTester(func(c *ui.Context) {
		PlanBoardView(c, PlanBoardModel{
			Board:      session.PlanBoard{},
			ThemeID:    "system",
			SystemDark: false,
			Open:       &open,
			GoalDraft:  &draft,
			OnCreateGoal: func(objective string) {
				created = objective
			},
		})
	}, 400, 240)
	for _, label := range []string{
		i18n.T(i18n.KeyPlanBoardEmpty),
		i18n.T(i18n.KeyPlanBoardGoal),
		i18n.T(i18n.KeyPlanBoardNoGoal),
		i18n.T(i18n.KeyPlanBoardGoalCreate),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
	_ = created
}

func TestPlanBoardGoalActionButtons(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	open := true
	draft := "rewrite"
	tt := ui.NewTester(func(c *ui.Context) {
		PlanBoardView(c, PlanBoardModel{
			Board: session.PlanBoard{
				Goal: &session.PlanGoal{ID: "g1", Objective: "Ship", Phase: "active", Activation: "armed"},
			},
			ThemeID:        "system",
			SystemDark:     false,
			Open:           &open,
			GoalDraft:      &draft,
			OnEditGoal:     func(string) {},
			OnPauseGoal:    func() {},
			OnCompleteGoal: func() {},
			OnClearGoal:    func() {},
		})
	}, 480, 280)
	for _, label := range []string{
		"[active / armed] Ship",
		i18n.T(i18n.KeyPlanBoardGoalEdit),
		i18n.T(i18n.KeyPlanBoardGoalPause),
		i18n.T(i18n.KeyPlanBoardGoalComplete),
		i18n.T(i18n.KeyPlanBoardGoalClear),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
}

func TestPlanBoardCollapsedShowsExpand(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	open := false
	tt := ui.NewTester(func(c *ui.Context) {
		PlanBoardView(c, PlanBoardModel{
			Board: session.PlanBoard{
				Todos: []session.PlanTodo{{Content: "a", Status: "pending"}},
			},
			ThemeID:    "system",
			SystemDark: false,
			Open:       &open,
		})
	}, 400, 120)
	if !tt.HasText(i18n.T(i18n.KeyPlanBoardExpand)) {
		t.Fatalf("missing expand; texts=%q", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyPlanBoardCollapse)) {
		t.Fatal("collapsed strip should not show collapse")
	}
}

func TestTodoStatusGlyph(t *testing.T) {
	if todoStatusGlyph("completed") != "✓" {
		t.Fatal(todoStatusGlyph("completed"))
	}
	if todoStatusGlyph("in_progress") != "●" {
		t.Fatal(todoStatusGlyph("in_progress"))
	}
	if todoStatusGlyph("pending") != "○" {
		t.Fatal(todoStatusGlyph("pending"))
	}
}
