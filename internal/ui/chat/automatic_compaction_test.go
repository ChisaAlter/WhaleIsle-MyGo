package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestAutomaticCompactionCardFullSummaryAndMissingEvidence(t *testing.T) {
	count, tokens := 2, 42
	full := strings.Repeat("完整摘要", 100) + "TAIL"
	summary := "## Summary heading\n\n" + full + "\n\n```go\nfmt.Println(\"checkpoint\")\n```"
	msg := session.Message{Kind: session.MessageKindCompaction, AutomaticCompaction: &session.ManualCompactionView{Summary: summary, ShadowedItemCount: &count, ShadowedTokenCount: &tokens}}
	tt := ui.NewTester(func(c *ui.Context) { CompactionView(c, msg, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.Transparent) }, 800, 1400)
	if tt.HasText(full) || tt.HasText("Summary heading") {
		t.Fatal("summary disclosed before explicit expansion")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatCompaction)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(full) || !tt.HasText("Summary heading") || !tt.HasText(`fmt.Println("checkpoint")`) {
		t.Fatal("expanded Markdown lost summary or code")
	}
	if tt.HasText("## Summary heading") || tt.HasText("```go") {
		t.Fatal("summary rendered as raw Markdown")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatCompaction)); err != nil {
		t.Fatal(err)
	}
	if tt.HasText(full) {
		t.Fatal("summary did not collapse")
	}
	msg.AutomaticCompaction = &session.ManualCompactionView{}
	tt.Frame()
	if !tt.HasText(i18n.T(i18n.KeyChatCompactionSummaryMissing)) {
		t.Fatal("missing summary not explicit")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatCompaction)); err != nil {
		t.Fatal(err)
	}
	if tt.HasText(full) {
		t.Fatal("missing evidence reopened the old summary")
	}
	msg.AutomaticCompaction = &session.ManualCompactionView{Summary: summary}
	tt.Frame()
	if tt.HasText(full) {
		t.Fatal("disabled marker toggled a hidden disclosure")
	}
}

func TestAutomaticCompactionNilMetadataHidesRawBody(t *testing.T) {
	msg := session.Message{
		Role: "system",
		Kind: session.MessageKindCompaction,
		Body: "compaction:e5a04416-42ef-461b-8c80-49f17545e662",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		CompactionView(c, msg, ui.RGB(238, 242, 246), ui.RGB(149, 160, 175), ui.RGB(26, 30, 36))
	}, 800, 200)
	if !tt.HasText(i18n.T(i18n.KeyChatCompaction)) {
		t.Fatal("compaction title missing")
	}
	if !tt.HasText(i18n.T(i18n.KeyChatCompactionSummaryMissing)) {
		t.Fatal("missing summary caption not shown when AutomaticCompaction is nil")
	}
	if tt.HasText("compaction:e5a04416") || tt.HasText(msg.Body) {
		t.Fatal("raw compaction:<uuid> body leaked into UI")
	}
}
