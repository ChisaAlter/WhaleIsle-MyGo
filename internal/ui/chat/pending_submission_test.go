package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestPendingSubmissionHonestStatusAndQueueSummary(t *testing.T) {
	p := session.PendingSubmission{RequestID: "r", Text: "awaiting", Accepted: true, Placement: "queued", Attachments: []session.ComposerAttachment{{Name: "notes.txt"}}}
	tt := ui.NewTester(func(c *ui.Context) { paintPendingSubmission(c, p, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 500, 160)
	if !tt.HasText("awaiting") || !tt.HasText("notes.txt") || !tt.HasText(i18n.T(i18n.KeyChatPendingAccepted)) {
		t.Fatalf("pending status missing: %v", tt.Texts())
	}
	board := session.PlanBoard{Pending: []session.PendingSubmission{p}}
	if board.Empty() || planBoardSummary(board) == "" {
		t.Fatal("pending queue disappears when collapsed")
	}
}
