package interaction

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	storeinteraction "whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
)

func TestParityApprovalEscalationAndQuestionCardCursors(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		store := storeinteraction.New()
		pendingApproval := storeinteraction.Pending{
			EventID: "ev-approval-1",
			AgentID: "agent-1",
			Event:   storeinteraction.EventApproval,
			Title:   "bash",
			Args:    `{"command":"rm -rf /tmp/test"}`,
		}
		snap := store.Snapshot()
		snap.Pending = []storeinteraction.Pending{pendingApproval}

		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				View(c, Model{
					Store:    store,
					Snapshot: snap,
					ThemeID:  "deepseek.light",
				})
			})
		}, 720, 420)

		wantEscalation := i18n.ChatParityf(i18n.KeyParityApprovalEscalation, "toolName", "bash")
		if !tt.HasText(wantEscalation) {
			t.Fatalf("[%s] missing approval escalation %q in %v", locale, wantEscalation, tt.Texts())
		}
		if !tt.HasText(i18n.T(i18n.KeyParityApprovalReject)) {
			t.Fatalf("[%s] missing approval reject %q in %v", locale, i18n.T(i18n.KeyParityApprovalReject), tt.Texts())
		}

		for _, b := range tt.Interactives() {
			if slices.Contains(b.Kinds, "click") && !slices.Contains(b.Kinds, "text") && b.Cursor != ui.CursorPointer {
				t.Fatalf("[%s] clickable interaction element missing CursorPointer: %s", locale, b.String())
			}
		}
	}
}
