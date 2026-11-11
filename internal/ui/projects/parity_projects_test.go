package projects

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func TestOrdinaryAndProjectTeamParityAndCursors(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	teamProj := &projectflow.TeamProjection{
		Failure: "corrupt record v2",
		Members: []projectflow.TeamMember{
			{ID: "m1", Name: "Lead", Role: "lead", Phase: "active", Running: true, ModelLabel: "deepseek-reasoner"},
			{ID: "m2", Name: "Worker", Phase: "active", Running: false, ModelLabel: "deepseek-chat"},
		},
		Tasks: []projectflow.TeamTask{
			{
				ID:          "T-1",
				Subject:     "Setup schema",
				Description: "Create initial database schema and migrations for parity check.\nLine 2\nLine 3",
				Status:      "completed",
				OwnerName:   "Lead",
				Ready:       true,
			},
			{
				ID:          "T-2",
				Subject:     "Implement UI",
				Description: "Wire up UI components and tests.",
				Status:      "pending",
				OwnerName:   "Worker",
				Ready:       false,
				BlockedBy:   []string{"T-1"},
				WriteScopes: []string{"internal/ui/projects"},
			},
		},
	}

	for _, tc := range []struct {
		locale      string
		wantFailure string
		wantBlocked string
		wantScopes  string
		wantMembers string
		wantTasks   string
	}{
		{
			locale:      i18n.LocaleZH,
			wantFailure: "团队持久记录无效：corrupt record v2",
			wantBlocked: "依赖: ",
			wantScopes:  "写入范围: internal/ui/projects",
			wantMembers: "成员",
			wantTasks:   "共享任务",
		},
		{
			locale:      i18n.LocaleEN,
			wantFailure: "Invalid team persistence record: corrupt record v2",
			wantBlocked: "Blocked by: ",
			wantScopes:  "Write scopes: internal/ui/projects",
			wantMembers: "Members",
			wantTasks:   "Shared tasks",
		},
	} {
		t.Run("ordinary_"+tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			state := &OrdinaryTeamViewState{Expanded: map[string]bool{"T-1": true}}
			tt := ui.NewTester(func(c *ui.Context) {
				OrdinaryTeamView(c, OrdinaryTeamModel{
					SessionID:    "m1",
					ThemeID:      "system",
					Team:         teamProj,
					State:        state,
					Columns:      2,
					OnRefresh:    func() {},
					OnOpenMember: func(string) {},
				})
			}, 640, 560)

			texts := strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantFailure, tc.wantBlocked, tc.wantScopes, tc.wantMembers, tc.wantTasks} {
				if !strings.Contains(texts, want) {
					t.Fatalf("ordinary %s missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			for _, it := range tt.Interactives() {
				if !slices.Contains(it.Kinds, "click") {
					continue
				}
				// Current session member card is intentionally inert (CursorDefault).
				if it.Label == "Lead" {
					continue
				}
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("ordinary %s interactive %q cursor = %v, want CursorPointer", tc.locale, it.Label, it.Cursor)
				}
			}
		})

		t.Run("project_"+tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			state := &TeamViewState{ExpandedTaskIDs: map[string]bool{"T-2": true}}
			tt := ui.NewTester(func(c *ui.Context) {
				ProjectTeamBody(c, ProjectTeamModel{
					Project: projectflow.Project{
						ID:                   "proj-1",
						CoordinatorSessionID: "m1",
					},
					SessionID: "m1",
					ThemeID:   "system",
					State:     state,
					Overview:  &projectflow.Overview{Team: teamProj},
					Columns:   2,
					OnRefresh: func() {},
					OnOpenMember: func(string) error {
						return nil
					},
				})
			}, 640, 560)

			texts := strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantFailure, tc.wantBlocked, tc.wantScopes, tc.wantMembers} {
				if !strings.Contains(texts, want) {
					t.Fatalf("project %s missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			for _, it := range tt.Interactives() {
				if !slices.Contains(it.Kinds, "click") || it.Label == "协调者" || it.Label == "Coordinator" {
					continue
				}
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("project %s interactive %q cursor = %v, want CursorPointer", tc.locale, it.Label, it.Cursor)
				}
			}
		})
	}
}
