package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTodoPanelViewHiddenWhenEmpty(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		TodoPanelView(c, TodoPanelModel{ThemeID: "system"})
	}, 400, 200)
	if tt.HasText("任务") {
		t.Fatal("empty todos should not paint")
	}
}

func TestTodoPanelViewShowsProgress(t *testing.T) {
	collapsed := false
	tt := ui.NewTester(func(c *ui.Context) {
		TodoPanelView(c, TodoPanelModel{
			Todos: []session.PlanTodo{
				{Content: "a", Status: "completed"},
				{Content: "b", Status: "in_progress"},
				{Content: "c", Status: "pending"},
			},
			ThemeID:   "system",
			Collapsed: &collapsed,
		})
	}, 480, 320)
	joined := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"任务", "1/3 已完成", "已完成", "进行中", "待处理", "a", "b", "c"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if got := todoCompletedFractionCount(2, 5); got != "2/5 已完成" {
		t.Fatalf("fraction = %q", got)
	}
}

func TestQueueDockInlineEditUnsupportedAndNotices(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)
	i18n.SetLocale(i18n.LocaleEN)

	savedID, savedText := "", ""
	m := PlanBoardModel{
		ThemeID: "deepseek.light",
		Board: session.PlanBoard{
			Queue: []session.PlanQueueItem{
				{ID: "q-text", Preview: "editable prompt"},
				{ID: "q-img", Preview: "[image] C:\\tmp\\pic.png"},
			},
			QueueActions: session.QueueActionState{Mutable: true, Running: false},
		},
		OnQueueEdit: func(id, text string) {
			savedID, savedText = id, text
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { QueueDockView(c, m) }, 640, 360)
	if !tt.HasText("2 queued messages") {
		t.Fatalf("missing count label: %q", tt.Texts())
	}
	if err := tt.Click("Expand or collapse queue"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Contains non-text content; editing is not supported yet") {
		t.Fatalf("missing unsupported edit notice: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyPlanBoardQueueEdit)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find(queueSaveEditCopy()); !ok {
		t.Fatalf("inline save button missing: %q", tt.Texts())
	}
	if _, ok := tt.Find(queueCancelEditCopy()); !ok {
		t.Fatalf("inline cancel button missing: %q", tt.Texts())
	}
	if err := tt.Click(queueSaveEditCopy()); err != nil {
		t.Fatal(err)
	}
	if savedID != "q-text" || savedText != "editable prompt" {
		t.Fatalf("saved = %q:%q, want q-text:editable prompt", savedID, savedText)
	}
	if _, ok := tt.Find(queueSaveEditCopy()); ok {
		t.Fatal("inline editor remained open after save")
	}

	i18n.SetLocale(i18n.LocaleZH)
	if got := normalizeConversationNotice("session_in_use"); got != "当前会话已被占用，可能是其他正在运行的 DSH 导致的（如其他 dsh web、桌面端），请退出其他正在运行的 DSH 后重试。" {
		t.Fatalf("sessionInUseNotice zh = %q", got)
	}
	if got := normalizeConversationNotice("parent_offline"); got != "父会话已离线，无法继续发送；仍可停止当前运行" {
		t.Fatalf("parentOfflineNotice zh = %q", got)
	}
	if got := normalizeConversationNotice("read_only"); got != "只读子会话，无法发送消息" {
		t.Fatalf("readOnlyChildNotice zh = %q", got)
	}
}
