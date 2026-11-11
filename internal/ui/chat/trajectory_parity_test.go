package chat

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTrajectoryParityLocalesAndViews(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	start, first, end := 1000.0, 1200.0, 2000.0
	turn1, step1 := 1.0, 1.0
	resSeq := 2.0
	usageJSON := json.RawMessage(`{"inputTokens":120,"outputTokens":80,"reasoningTokens":45,"totalTokens":245}`)

	snap := session.ViewState{
		SelectedID: "s-traj",
		Messages: []session.Message{
			{
				ID:               "u1",
				Seq:              1,
				Role:             "user",
				Kind:             session.MessageKindText,
				Body:             "Analyze image",
				ConversationTurn: &turn1,
				ImageRef:         "attachment:img-1",
			},
			{
				ID:               "a1",
				Seq:              2,
				Role:             "assistant",
				Kind:             session.MessageKindText,
				Body:             "Done analyzing",
				ConversationTurn: &turn1,
				ConversationStep: &step1,
				TrajectoryTime: &session.TrajectoryTiming{
					OperationID:  "assistant:a1",
					StartedAt:    &start,
					FirstTokenAt: &first,
					CompletedAt:  &end,
					Closed:       true,
				},
			},
			{
				ID:               "err1",
				Seq:              3,
				Role:             "assistant",
				Kind:             session.MessageKindTool,
				EventType:        "tool/error",
				Body:             "Tool execution failed: timeout",
				ConversationTurn: &turn1,
				ConversationStep: &step1,
				Tool: &session.ToolCallView{
					CallID: "c1",
					Name:   "bash",
					State:  "failed",
				},
			},
		},
		TrajectoryRequests: []session.TrajectoryRequest{
			{
				ID:          "req-1",
				StartSeq:    2,
				ResultSeq:   &resSeq,
				Purpose:     "chat",
				Status:      "completed",
				StartedAt:   &start,
				CompletedAt: &end,
				Usage:       usageJSON,
			},
		},
	}

	for _, tc := range []struct {
		locale         string
		wantTimeline   string
		wantTotalDur   string
		wantGeneration string
		wantTTFTDec    string
		wantOptionsTab string
		wantOptionsMsg string
		wantUsageTab   string
		wantReasoning  string
		wantError      string
		wantAttach     string
		wantImageIdx   string
	}{
		{
			locale:         i18n.LocaleZH,
			wantTimeline:   "总计 1000 ms",
			wantTotalDur:   "总时长",
			wantGeneration: "生成",
			wantTTFTDec:    "首 token 200 ms · 解码 800 ms",
			wantOptionsTab: "选项",
			wantOptionsMsg: "未记录选项",
			wantUsageTab:   "用量",
			wantReasoning:  "推理: 45 tok",
			wantError:      "错误",
			wantAttach:     "附件",
			wantImageIdx:   "图片 1",
		},
		{
			locale:         i18n.LocaleEN,
			wantTimeline:   "Total 1000 ms",
			wantTotalDur:   "Total Duration",
			wantGeneration: "Generation",
			wantTTFTDec:    "TTFT 200 ms · Decoding 800 ms",
			wantOptionsTab: "Options",
			wantOptionsMsg: "Options not recorded",
			wantUsageTab:   "Usage",
			wantReasoning:  "Reasoning: 45 tok",
			wantError:      "Error",
			wantAttach:     "Attachments",
			wantImageIdx:   "Image 1",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			selected := "a1"
			tab := "timing"
			var list ui.ListState

			tt := ui.NewTester(func(c *ui.Context) {
				TrajectoryView(c, TrajectoryModel{
					Snapshot:   snap,
					PaneWidth:  980,
					SelectedID: &selected,
					DetailTab:  &tab,
					List:       &list,
				})
			}, 980, 640)

			texts := strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantTimeline, tc.wantTotalDur, tc.wantGeneration, tc.wantTTFTDec, tc.wantOptionsTab} {
				if !strings.Contains(texts, want) {
					t.Fatalf("locale %s timing view missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			// Switch to Options tab.
			tab = "options"
			tt.Frame()
			texts = strings.Join(tt.Texts(), "\n")
			if !strings.Contains(texts, tc.wantOptionsMsg) {
				t.Fatalf("locale %s options tab missing %q in:\n%s", tc.locale, tc.wantOptionsMsg, texts)
			}

			// Select request row and switch to Usage tab.
			selected = "request:req-1"
			tab = "usage"
			tt.Frame()
			texts = strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantUsageTab, tc.wantReasoning} {
				if !strings.Contains(texts, want) {
					t.Fatalf("locale %s usage tab missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			// Select error tool row in overview tab.
			selected = "err1"
			tab = "overview"
			tt.Frame()
			texts = strings.Join(tt.Texts(), "\n")
			if !strings.Contains(texts, tc.wantError) {
				t.Fatalf("locale %s error overview missing %q in:\n%s", tc.locale, tc.wantError, texts)
			}

			// Select user message with image attachment in overview tab.
			selected = "u1"
			tab = "overview"
			tt.Frame()
			texts = strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantAttach, tc.wantImageIdx} {
				if !strings.Contains(texts, want) {
					t.Fatalf("locale %s attachment overview missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			// Verify buttons have CursorPointer.
			for _, it := range tt.Interactives() {
				if !slices.Contains(it.Kinds, "click") || slices.Contains(it.Kinds, "text") || it.Role == ui.RoleSplitter {
					continue
				}
				if it.Label == "" {
					continue
				}
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("locale %s button %q cursor = %v, want CursorPointer", tc.locale, it.Label, it.Cursor)
				}
			}
		})
	}
}
