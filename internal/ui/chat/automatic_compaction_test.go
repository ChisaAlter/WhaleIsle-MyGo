package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestAutomaticCompactionCardFullSummaryAndMissingEvidence(t *testing.T) {
	count, tokens := 2, 42
	summary := strings.Repeat("完整摘要", 100) + "TAIL"
	msg := session.Message{Kind: session.MessageKindCompaction, AutomaticCompaction: &session.ManualCompactionView{Summary: summary, ShadowedItemCount: &count, ShadowedTokenCount: &tokens}}
	tt := ui.NewTester(func(c *ui.Context) { CompactionView(c, msg, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.Transparent) }, 800, 1000)
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), summary) {
		t.Fatal("summary clipped")
	}
	msg.AutomaticCompaction = &session.ManualCompactionView{}
	tt.Frame()
	if !tt.HasText(i18n.T(i18n.KeyChatCompactionSummaryMissing)) {
		t.Fatal("missing summary not explicit")
	}
}
