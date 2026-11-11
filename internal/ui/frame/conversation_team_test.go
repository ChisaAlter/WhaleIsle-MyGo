package frame

import (
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

func TestOrdinaryConversationTeamPopoverAlignsWithReference(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(oldLocale)

	state := &projects.OrdinaryTeamViewState{FloatingOpen: true, Pinned: true}
	team := &projectflow.TeamProjection{
		StatusKnown: true,
		Members: []projectflow.TeamMember{
			{
				ID:         "lead-session",
				Name:       "lead",
				Role:       "lead",
				Phase:      "active",
				Running:    false,
				ModelLabel: "claude-sonnet-4-6",
				ModelError: "dsh session/projections: decode value: simulated error",
			},
		},
	}
	snap := session.ViewState{SelectedID: "lead-session"}
	m := Model{
		ThemeID:    theme.IDDeepseekDark,
		SystemDark: true,
		OrdinaryTeam: func() projects.OrdinaryTeamModel {
			return projects.OrdinaryTeamModel{
				SessionID: "lead-session",
				Team:      team,
				State:     state,
			}
		},
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Padding(12, 80).Children(func() {
			conversationTeamAction(c, m, snap, false, ui.RGB(207, 211, 214))
		})
	}, 520, 300)
	tt.SetDark(true)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
	tt.Frame()
	_ = colors

	texts := strings.Join(tt.Texts(), "\n")
	if tt.HasText("持续观察") || tt.HasText("Keep observing") {
		t.Fatalf("ordinary session Team popover must not render Project surfaceToolbar: %q", texts)
	}
	if strings.Contains(texts, "session/projections") || strings.Contains(texts, "活动暂未确认") {
		t.Fatalf("ordinary session Team popover rendered raw projection error or unconfirmed text: %q", texts)
	}
	for _, want := range []string{"成员", "lead", "当前会话", "未运行", "暂无共享任务，可以通过对话创建"} {
		if !tt.HasText(want) {
			t.Fatalf("missing expected text %q in %q", want, texts)
		}
	}
	if strings.Contains(texts, "claude-sonnet-4-6") {
		t.Fatalf("model label should be hidden until member card is hovered/focused: %q", texts)
	}
	memberRect, ok := tt.Find("lead")
	if !ok {
		t.Fatalf("member card 'lead' not found")
	}
	tt.Move(memberRect.X+10, memberRect.Y+10)
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "模型: claude-sonnet-4-6") {
		t.Fatalf("hovering member card did not reveal model label: %q", tt.Texts())
	}
	tt.Move(5, 5)
	tt.Frame()

	if out := os.Getenv("TEAM_POPOVER_SNAPSHOT"); out != "" {
		f, err := os.Create(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOrdinaryConversationTeamPopoverWithMembersAndTasks(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(oldLocale)

	state := &projects.OrdinaryTeamViewState{FloatingOpen: true, Pinned: true}
	team := &projectflow.TeamProjection{
		StatusKnown: true,
		Members: []projectflow.TeamMember{
			{
				ID:         "lead-session",
				Name:       "lead",
				Role:       "lead",
				Phase:      "active",
				Running:    true,
				ModelLabel: "claude-sonnet-4-6",
			},
			{
				ID:         "worker-1",
				Name:       "researcher",
				Role:       "worker",
				Phase:      "active",
				Running:    false,
				ModelLabel: "deepseek-reasoner",
			},
		},
		Tasks: []projectflow.TeamTask{
			{
				ID:          "#1",
				Subject:     "对齐团队模式浮层与侧栏样式",
				Description: "逐行对照 TeamAction.js 与 TeamAction.module.css，确保成员卡片、状态指示器、当前会话标签与共享任务卡片完全一致。",
				Status:      "in_progress",
				OwnerName:   "researcher",
				Ready:       true,
				WriteScopes: []string{"internal/ui/projects", "internal/ui/frame"},
			},
		},
	}
	snap := session.ViewState{SelectedID: "lead-session"}
	m := Model{
		ThemeID:    theme.IDDeepseekDark,
		SystemDark: true,
		OrdinaryTeam: func() projects.OrdinaryTeamModel {
			return projects.OrdinaryTeamModel{
				SessionID:    "lead-session",
				Team:         team,
				State:        state,
				OnOpenMember: func(string) {},
			}
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Padding(12, 40).Children(func() {
			conversationTeamAction(c, m, snap, false, ui.RGB(207, 211, 214))
		})
	}, 600, 420)
	tt.SetDark(true)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
	tt.Frame()

	for _, want := range []string{"成员", "2", "lead", "当前会话", "运行中", "researcher", "未运行", "共享任务", "1", "对齐团队模式浮层与侧栏样式", "进行中", "#1", "Owner: researcher", "写入范围: internal/ui/projects, internal/ui/frame"} {
		if !tt.HasText(want) {
			t.Fatalf("missing expected text %q in %q", want, tt.Texts())
		}
	}

	if out := os.Getenv("TEAM_POPOVER_FULL_SNAPSHOT"); out != "" {
		f, err := os.Create(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOrdinaryConversationTeamHoverThenClickPinsPopoverOpen(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(oldLocale)

	state := &projects.OrdinaryTeamViewState{}
	team := &projectflow.TeamProjection{
		StatusKnown: true,
		Members: []projectflow.TeamMember{
			{ID: "lead-session", Name: "lead", Role: "lead", Phase: "active"},
		},
	}
	snap := session.ViewState{SelectedID: "lead-session"}
	m := Model{
		ThemeID:    theme.IDDeepseekDark,
		SystemDark: true,
		OrdinaryTeam: func() projects.OrdinaryTeamModel {
			return projects.OrdinaryTeamModel{
				SessionID: "lead-session",
				Team:      team,
				State:     state,
			}
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Padding(12, 40).Children(func() {
			conversationTeamAction(c, m, snap, false, ui.RGB(207, 211, 214))
		})
	}, 600, 420)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.Frame()

	btnRect, ok := tt.Find("团队")
	if !ok {
		t.Fatal("missing 团队 button")
	}
	// Simulate hovering over the button for >150ms (which opens the popover unpinned),
	// followed by clicking the button. Clicking must pin and keep the popover open!
	tt.Move(btnRect.X+btnRect.W/2, btnRect.Y+btnRect.H/2)
	tt.Frame()
	tt.SetTime(start.Add(200 * time.Millisecond))
	tt.Frame()
	if !state.FloatingOpen || state.Pinned {
		t.Fatalf("after 200ms hover, expected FloatingOpen=true Pinned=false, got FloatingOpen=%v Pinned=%v", state.FloatingOpen, state.Pinned)
	}
	if err := tt.Click("团队"); err != nil {
		t.Fatalf("click 团队 failed: %v", err)
	}
	tt.Frame()
	if !state.FloatingOpen || !state.Pinned {
		t.Fatalf("clicking after hover closed popover instead of pinning: FloatingOpen=%v Pinned=%v", state.FloatingOpen, state.Pinned)
	}
	if !tt.HasText("成员") || !tt.HasText("lead") {
		t.Fatalf("popover content missing after click: %v", tt.Texts())
	}
}
