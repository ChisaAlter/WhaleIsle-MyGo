package chat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTrajectorySearchFullContent(t *testing.T) {
	tail := strings.Repeat("prefix ", 90) + "UNIQUE_TAIL"
	for _, msg := range []session.Message{
		{ID: "user", Kind: session.MessageKindText, Role: "user", Body: tail},
		{ID: "assistant", Kind: session.MessageKindText, Role: "assistant", Body: tail},
		{ID: "thinking", Kind: session.MessageKindThinking, Body: tail},
		{ID: "call", Kind: session.MessageKindTool, EventType: "tool/call", Args: tail},
		{ID: "result", Kind: session.MessageKindTool, EventType: "tool/result", Args: tail},
		{ID: "raw", Raw: json.RawMessage(`{"extra":"` + tail + `"}`)},
		{ID: "files", Kind: session.MessageKindDeliverables, Files: []string{tail}},
	} {
		t.Run(msg.ID, func(t *testing.T) {
			if got := buildTrajectoryRows([]session.Message{msg}, "unique_tail", false, false); len(got) != 1 {
				t.Fatal("search lost text beyond preview")
			}
		})
	}
}

func TestTrajectoryDetailAndCopyFullContent(t *testing.T) {
	body := strings.Repeat("完整内容", 30) + "FINAL"
	for _, tc := range []struct {
		name, tab string
		msg       session.Message
	}{
		{"input", "overview", session.Message{Kind: session.MessageKindText, Role: "user", Body: body}},
		{"output", "result", session.Message{Kind: session.MessageKindText, Role: "assistant", Body: body}},
		{"thinking", "overview", session.Message{Kind: session.MessageKindThinking, Body: body}},
		{"tool-result", "result", session.Message{Kind: session.MessageKindTool, EventType: "tool/result", Args: body}},
		{"raw", "args", session.Message{Raw: json.RawMessage(`{"payload":"` + strings.Repeat("x", 500) + body + `"}`)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.msg.ID = "selected"
			selected, tab, query := "selected", tc.tab, ""
			copied := ""
			tt := ui.NewTester(func(c *ui.Context) {
				TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{Messages: []session.Message{tc.msg}}, SelectedID: &selected, DetailTab: &tab, SearchQuery: &query, OnCopyDetail: func(s string) { copied = s }})
			}, 1000, 1400)
			if !strings.Contains(strings.Join(tt.Texts(), "\n"), body) {
				t.Fatal("inspector lost full content")
			}
			if err := tt.Click(i18n.T(i18n.KeyTrajCopy)); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(copied, body) {
				t.Fatal("copy lost full content")
			}
			rows := buildTrajectoryRows([]session.Message{tc.msg}, "", false, false)
			if len([]rune(rows[0].Input)) > 81 || len([]rune(rows[0].Output)) > 81 || len([]rune(rows[0].Think)) > 81 {
				t.Fatal("ledger preview no longer bounded")
			}
		})
	}
}
