package chat

import (
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func TestToolRunningDetailExtractsQuestionText(t *testing.T) {
	args := `{"questions":[{"id":"q1","header":"Confirm","question":"Pick a color?","options":[{"label":"Red"}]}]}`
	if got := toolRunningDetail("ask_user_question", args); got != "Pick a color?" {
		t.Fatalf("question detail = %q, want %q", got, "Pick a color?")
	}
}

func TestToolRunningDetailExtractsCommandAndNormalizesWhitespace(t *testing.T) {
	if got := toolRunningDetail("bash", `{"command":"echo  hi\nthere"}`); got != "echo hi there" {
		t.Fatalf("command detail = %q", got)
	}
}

func TestToolRunningDetailFallsBackToToolName(t *testing.T) {
	if got := toolRunningDetail("grep", `{"other":1}`); got != "grep" {
		t.Fatalf("fallback detail = %q, want tool name", got)
	}
	if got := toolRunningDetail("bash", `not json`); got != "" {
		t.Fatalf("unparseable detail = %q, want empty", got)
	}
}

func TestToolRunningDetailTruncates(t *testing.T) {
	long := strings.Repeat("a", 200)
	got := toolRunningDetail("read", `{"path":"`+long+`"}`)
	if len([]rune(got)) != 161 || !strings.HasSuffix(got, "…") {
		t.Fatalf("detail not truncated at 160 chars: %d runes", len([]rune(got)))
	}
}

func TestProcessGroupLabelShowsSemanticDetailNotRawJSON(t *testing.T) {
	group := []session.Message{{
		Kind: session.MessageKindTool,
		Tool: &session.ToolCallView{
			CallID:    "call-1",
			Name:      "ask_user_question",
			State:     "running",
			Arguments: `{"questions":[{"id":"q1","question":"Pick a color?","options":[{"label":"Red"}]}]}`,
		},
	}}
	label := processGroupLabel(group, true, true)
	if strings.Contains(label, "{") || strings.Contains(label, "questions") {
		t.Fatalf("fold label leaks raw JSON arguments: %q", label)
	}
	if !strings.Contains(label, "Pick a color?") {
		t.Fatalf("fold label missing semantic question detail: %q", label)
	}
}
