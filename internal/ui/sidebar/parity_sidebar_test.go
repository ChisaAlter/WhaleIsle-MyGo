package sidebar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParitySidebarSessionsExpandAndCollapse(t *testing.T) {
	items := make([]session.Item, 0, 8)
	ids := make([]string, 0, 8)
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("sess-%d", i)
		ids = append(ids, id)
		items = append(items, session.Item{
			SessionID: id,
			Title:     fmt.Sprintf("会话 %d", i),
			Cwd:       "/workspace/alpha",
		})
	}
	snapshot := session.ViewState{
		WorkspaceReady: true,
		SelectedID:     "sess-1",
		Items:          items,
		Workspaces: []session.WorkspaceRow{
			{
				WorkspaceID: "ws-alpha",
				Title:       "Alpha",
				Path:        "/workspace/alpha",
				SessionIDs:  ids,
			},
		},
	}

	for _, tc := range []struct {
		locale            string
		wantExpand        string
		wantExpandLabel   string
		wantCollapse      string
		wantCollapseLabel string
	}{
		{
			locale:            "zh",
			wantExpand:        "展开其余 3 个会话",
			wantExpandLabel:   "显示更多 ws-alpha",
			wantCollapse:      "收起",
			wantCollapseLabel: "收起会话 ws-alpha",
		},
		{
			locale:            "en",
			wantExpand:        "Show 3 more sessions",
			wantExpandLabel:   "Show more ws-alpha",
			wantCollapse:      "Show less",
			wantCollapseLabel: "Show less ws-alpha",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			defer i18n.SetLocale(i18n.DefaultLocale)

			m := Model{
				ThemeID:   "deepseek.light",
				ViewportW: 280,
				ViewportH: 864,
				Snapshot:  snapshot,
				OnSelect:  func(string) {},
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 864)
			tt.Frame()

			if !tt.HasText(tc.wantExpand) {
				t.Fatalf("expected expand text %q in %v", tc.wantExpand, tt.Texts())
			}
			if err := tt.Click(tc.wantExpandLabel); err != nil {
				t.Fatalf("click expand button %q failed: %v", tc.wantExpandLabel, err)
			}
			if !tt.HasText(tc.wantCollapse) {
				t.Fatalf("expected collapse text %q after clicking expand, got %v", tc.wantCollapse, tt.Texts())
			}
			if err := tt.Click(tc.wantCollapseLabel); err != nil {
				t.Fatalf("click collapse button %q failed: %v", tc.wantCollapseLabel, err)
			}
		})
	}
}

func TestParitySidebarWorkspaceRevealAndRevealFailed(t *testing.T) {
	for _, tc := range []struct {
		locale     string
		wantReveal string
		wantFailed string
	}{
		{
			locale:     "zh",
			wantReveal: "在 Finder 中打开工作区",
			wantFailed: "无法打开工作区文件夹，请确认目录仍存在后重试",
		},
		{
			locale:     "en",
			wantReveal: "Open workspace in Finder",
			wantFailed: "Could not open the workspace folder. Check that it still exists and try again.",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			defer i18n.SetLocale(i18n.DefaultLocale)

			if got := i18n.T(i18n.KeyWSWorkspaceReveal); got != tc.wantReveal {
				t.Fatalf("KeyWSWorkspaceReveal = %q, want %q", got, tc.wantReveal)
			}
			if got := i18n.T(i18n.KeyWSWorkspaceRevealFailed); got != tc.wantFailed {
				t.Fatalf("KeyWSWorkspaceRevealFailed = %q, want %q", got, tc.wantFailed)
			}

			dismissed := false
			m := Model{
				ThemeID:   "deepseek.light",
				ViewportW: 280,
				ViewportH: 864,
				Snapshot: session.ViewState{
					WorkspaceReady: true,
					Items:          []session.Item{{SessionID: "s1", Title: "Demo", Cwd: "/demo"}},
					Workspaces:     []session.WorkspaceRow{{WorkspaceID: "ws1", Title: "Demo", Path: "/demo", SessionIDs: []string{"s1"}}},
				},
				OnRevealWorkspacePath: func(string, string) {},
				ActionToast: &RowActionToastState{
					Kind:      "reveal-failed",
					OnDismiss: func() { dismissed = true },
				},
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 864)
			tt.Frame()
			if !tt.HasText(tc.wantFailed) {
				t.Fatalf("expected reveal-failed toast %q in %v", tc.wantFailed, tt.Texts())
			}
			if err := tt.Click("×"); err != nil {
				t.Fatalf("click dismiss button failed: %v", err)
			}
			if !dismissed {
				t.Fatalf("expected toast dismiss callback to fire")
			}
		})
	}
}

func TestParitySidebarArchiveConfirmDialog(t *testing.T) {
	for _, tc := range []struct {
		locale     string
		wantTitle  string
		wantDesc   string
		wantHeader string
		wantLines  []string
		wantAction string
		wantCancel string
	}{
		{
			locale:     "zh",
			wantTitle:  "停止并归档此会话？",
			wantDesc:   "“测试会话”仍有正在进行的工作。归档会先停止这些工作；之后可在侧栏筛选“全部对话（显示已归档）”中恢复会话，被停止的工作不会自动继续。",
			wantHeader: "将被停止的工作",
			wantLines: []string{
				"进行中的回合",
				"2 个运行中的子智能体：sub1、sub2",
				"1 个后台任务：job1",
				"1 条定时提醒：sched1",
				"1 项其他工作（indexing）",
			},
			wantAction: "停止并归档",
			wantCancel: "取消",
		},
		{
			locale:     "en",
			wantTitle:  "Stop and archive this session?",
			wantDesc:   "“测试会话” still has work in progress. Archiving stops it first; you can restore the session later from the “All conversations (show archived)” filter in the sidebar, and the stopped work will not resume on its own.",
			wantHeader: "Work that will be stopped",
			wantLines: []string{
				"The turn in progress",
				"2 running subagents: sub1, sub2",
				"1 background job: job1",
				"1 scheduled reminder: sched1",
				"1 other item of work (indexing)",
			},
			wantAction: "Stop and archive",
			wantCancel: "Cancel",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			defer i18n.SetLocale(i18n.DefaultLocale)

			var confirmedID string
			cancelled := false
			state := &ArchiveConfirmState{
				SessionID:   "s1",
				Title:       "测试会话",
				TurnRunning: true,
				Subagents:   []string{"sub1", "sub2"},
				Jobs:        []string{"job1"},
				Schedules:   []string{"sched1"},
				Others:      []ArchiveConfirmOtherWork{{Kind: "indexing", Count: 1}},
				OnCancel:    func() { cancelled = true },
				OnConfirm:   func(id string) { confirmedID = id },
			}

			m := Model{
				ThemeID:        "deepseek.light",
				ViewportW:      480,
				ViewportH:      640,
				ArchiveConfirm: state,
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 480, 640)
			tt.Frame()

			expectedAll := append([]string{tc.wantTitle, tc.wantDesc, tc.wantHeader, tc.wantAction}, tc.wantLines...)
			for _, expected := range expectedAll {
				if !tt.HasText(expected) {
					t.Fatalf("missing expected archive dialog text %q; got %v", expected, tt.Texts())
				}
			}

			if err := tt.Click(tc.wantAction); err != nil {
				t.Fatalf("click confirm button %q failed: %v", tc.wantAction, err)
			}
			if confirmedID != "s1" {
				t.Fatalf("expected OnConfirm(\"s1\"), got %q", confirmedID)
			}

			if err := tt.Click(tc.wantCancel); err != nil {
				t.Fatalf("click cancel button %q failed: %v", tc.wantCancel, err)
			}
			if !cancelled {
				t.Fatalf("expected OnCancel to be called")
			}
		})
	}
}

func TestParitySidebarDeleteSessionDialogAndErrorCodes(t *testing.T) {
	errorCodes := []struct {
		code   string
		wantZH string
		wantEN string
	}{
		{
			code:   "workspace/session-running",
			wantZH: "会话仍在运行，请先停止后再删除。",
			wantEN: "The session is still running. Stop it before deleting.",
		},
		{
			code:   "workspace/session-not-archived",
			wantZH: "只能删除已归档的会话。",
			wantEN: "Only archived sessions can be deleted.",
		},
		{
			code:   "workspace/session-not-found",
			wantZH: "找不到该会话。",
			wantEN: "That session could not be found.",
		},
		{
			code:   "workspace/session-live-unowned",
			wantZH: "会话仍在运行且无法安全结束，请重试或重启后再删。",
			wantEN: "The session is still live and could not be shut down safely. Retry or restart, then delete.",
		},
		{
			code:   "workspace/session-delete-partial",
			wantZH: "部分子会话已删除，但根会话仍在。请重试删除。",
			wantEN: "Some child sessions were deleted, but the root remains. Retry the delete.",
		},
		{
			code:   "workspace/session-delete-incomplete",
			wantZH: "会话日志已删除，但归档状态未能完全更新。请刷新后检查。",
			wantEN: "The conversation log was deleted, but archive state did not fully update. Refresh and check.",
		},
		{
			code:   "workspace/session-delete-in-progress",
			wantZH: "该会话正在删除中，请稍后再试。",
			wantEN: "That session is already being deleted. Try again in a moment.",
		},
		{
			code:   "workspace/session-active",
			wantZH: "会话仍有正在进行或计划中的工作，未完成此次操作。",
			wantEN: "This session still has running or scheduled work. The operation was not completed.",
		},
	}

	for _, ec := range errorCodes {
		i18n.SetLocale("zh")
		if got := FormatSessionDeleteError(ec.code); got != ec.wantZH {
			t.Fatalf("zh FormatSessionDeleteError(%q) = %q, want %q", ec.code, got, ec.wantZH)
		}
		i18n.SetLocale("en")
		if got := FormatSessionDeleteError(ec.code); got != ec.wantEN {
			t.Fatalf("en FormatSessionDeleteError(%q) = %q, want %q", ec.code, got, ec.wantEN)
		}
	}
	i18n.SetLocale(i18n.DefaultLocale)

	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var confirmedID string
	m := Model{
		ThemeID:   "deepseek.light",
		ViewportW: 420,
		ViewportH: 640,
		DeleteConfirm: &DeleteConfirmState{
			SessionID: "arch-1",
			Title:     "旧会话",
			ErrorCode: "workspace/session-not-archived",
			OnCancel:  func() {},
			OnConfirm: func(id string) { confirmedID = id },
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 420, 640)
	tt.Frame()
	if !tt.HasText("删除会话") {
		t.Fatalf("missing delete confirm title in %v", tt.Texts())
	}
	if !tt.HasText("只能删除已归档的会话。") {
		t.Fatalf("missing delete error message in %v", tt.Texts())
	}
	if err := tt.Click("确认删除会话"); err != nil {
		t.Fatalf("click delete confirm button failed: %v", err)
	}
	if confirmedID != "arch-1" {
		t.Fatalf("expected delete confirm callback with arch-1, got %q", confirmedID)
	}
}

func TestParitySidebarRowActionToasts(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	kinds := []struct {
		kind    string
		message string
		want    string
	}{
		{kind: "archived", want: "会话已归档，可"},
		{kind: "stopped-archived", want: "已停止并归档，可"},
		{kind: "fork-requires-open", want: "请先打开原会话完成迁移，再创建分支"},
		{kind: "session-id-copied", want: "Session ID 已复制"},
		{kind: "session-id-copy-failed", want: "复制失败，请重试"},
		{kind: "pin-failed", want: "置顶失败，请稍后重试"},
		{kind: "unpin-failed", want: "取消置顶失败，请稍后重试"},
		{kind: "create-failed", message: "磁盘已满", want: "新建会话失败：磁盘已满"},
		{kind: "archived-not-openable", want: "已归档对话暂时无法查看，请取消归档后查看"},
	}

	for _, tc := range kinds {
		m := Model{
			ThemeID:   "deepseek.light",
			ViewportW: 420,
			ViewportH: 640,
			ActionToast: &RowActionToastState{
				Kind:         tc.kind,
				Message:      tc.message,
				OnUndo:       func() {},
				OnViewAll:    func() {},
				OnOpenSource: func() {},
				OnDismiss:    func() {},
			},
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 420, 640)
		tt.Frame()
		if !tt.HasText(tc.want) {
			t.Fatalf("toast kind %q missing %q in %v", tc.kind, tc.want, tt.Texts())
		}
	}
}

func TestParitySidebarCatalogKeysNonEmpty(t *testing.T) {
	keys := []string{
		i18n.KeySidebarRightTabGuideMore,
		i18n.KeyWSSessionsExpand,
		i18n.KeyWSSessionsCollapse,
		i18n.KeyWSWorkspaceReveal,
		i18n.KeyWSWorkspaceRevealFailed,
		i18n.KeyWSSearchPending,
		i18n.KeyWSSearchNoMatches,
		i18n.KeyWSSearchHasMore,
		i18n.KeyWSArchiveConfirmTitle,
		i18n.KeyWSArchiveConfirmDesc,
		i18n.KeyWSArchiveConfirmActivity,
		i18n.KeyWSArchiveConfirmTurn,
		i18n.KeyWSArchiveConfirmSubagentsOne,
		i18n.KeyWSArchiveConfirmSubagentsOther,
		i18n.KeyWSArchiveConfirmJobsOne,
		i18n.KeyWSArchiveConfirmJobsOther,
		i18n.KeyWSArchiveConfirmSchedulesOne,
		i18n.KeyWSArchiveConfirmSchedulesOther,
		i18n.KeyWSArchiveConfirmOtherOne,
		i18n.KeyWSArchiveConfirmOtherOther,
		i18n.KeyWSArchiveConfirmAction,
		i18n.KeyWSArchiveConfirmPending,
		i18n.KeyWSDeleteSessionTitle,
		i18n.KeyWSDeleteSessionDesc,
		i18n.KeyWSDeleteSessionRunning,
		i18n.KeyWSDeleteSessionNotArchived,
		i18n.KeyWSDeleteSessionNotFound,
		i18n.KeyWSDeleteSessionLiveUnowned,
		i18n.KeyWSDeleteSessionPartial,
		i18n.KeyWSDeleteSessionIncomplete,
		i18n.KeyWSDeleteSessionInProgress,
		i18n.KeyWSSelectionErrorActive,
	}

	for _, locale := range []string{"zh", "en"} {
		i18n.SetLocale(locale)
		for _, k := range keys {
			val := i18n.T(k)
			if strings.TrimSpace(val) == "" || val == k {
				t.Fatalf("locale %s key %q resolved to %q", locale, k, val)
			}
		}
	}
	i18n.SetLocale(i18n.DefaultLocale)
}
