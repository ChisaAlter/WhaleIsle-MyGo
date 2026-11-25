package assistant

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/errcopy"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
)

func TestAssistantSurfacesSanitizeRawErrors(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	rawErr := "rpc error: code = Unavailable desc = dial tcp 127.0.0.1:4321: connection refused"
	want := errcopy.SentenceText(rawErr)

	t.Run("panel-last-err", func(t *testing.T) {
		draft := "hello"
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{
				ThemeID:      "deepseek",
				Mode:         assiststore.ModeWhale,
				HarnessReady: true,
				WhaleEnabled: true,
				Draft:        &draft,
				LastErr:      rawErr,
			})
		}, 900, 1200)
		joined := strings.Join(tt.Texts(), "\n")
		if strings.Contains(joined, "127.0.0.1:4321") || strings.Contains(joined, "connection refused") {
			t.Fatalf("raw LastErr leaked into panel:\n%s", joined)
		}
		if !strings.Contains(joined, want) {
			t.Fatalf("missing sanitized LastErr %q:\n%s", want, joined)
		}
	})

	t.Run("proactive-errors", func(t *testing.T) {
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{
				ThemeID: "deepseek",
				Proactive: ProactiveModel{
					Open:     true,
					Error:    rawErr,
					Selected: "t1",
					State: proactive.Snapshot{
						Tasks: []proactive.Task{{
							ID:    "t1",
							Title: "Check release",
							State: "failed",
							Execution: &proactive.Execution{
								SessionID: "s1",
								Error:     rawErr,
							},
						}},
						Outbox: []proactive.Approval{{
							ID:          "a1",
							TaskID:      "t1",
							Status:      "approved",
							Action:      "run",
							Effect:      "failed",
							EffectError: rawErr,
							ToolCall:    &proactive.ToolCall{Source: "harness:fs"},
						}},
					},
				},
			})
		}, 900, 1200)
		joined := strings.Join(tt.Texts(), "\n")
		if strings.Contains(joined, "127.0.0.1:4321") || strings.Contains(joined, "connection refused") {
			t.Fatalf("raw proactive error leaked:\n%s", joined)
		}
		if !strings.Contains(joined, want) {
			t.Fatalf("missing sanitized proactive error %q:\n%s", want, joined)
		}
	})

	t.Run("whale-task-errors", func(t *testing.T) {
		var task pluginrpc.WhaleTask
		task.ID = "wt1"
		task.Goal = "Organize notes"
		task.State = "failed"
		task.Error = rawErr
		task.Effects = []pluginrpc.WhaleTaskEffect{{ID: "e1", Target: "im", State: "failed", Error: rawErr}}
		tt := ui.NewTester(func(c *ui.Context) {
			whaleTaskRow(c, WhaleTasksModel{}, task, false, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.RGB(200, 200, 200), ui.RGB(37, 99, 235))
		}, 800, 600)
		joined := strings.Join(tt.Texts(), "\n")
		if strings.Contains(joined, "127.0.0.1:4321") || strings.Contains(joined, "connection refused") {
			t.Fatalf("raw whale task error leaked:\n%s", joined)
		}
		if !strings.Contains(joined, want) {
			t.Fatalf("missing sanitized whale task error %q:\n%s", want, joined)
		}
	})
}
