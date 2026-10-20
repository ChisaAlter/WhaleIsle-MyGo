package interaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// approvalGallery mounts one approval and returns its tester plus the palette the
// card paints with.
func approvalGallery(t *testing.T) (*ui.Tester, theme.Colors) {
	t.Helper()
	store := galleryStore(t, dsh.EventsWaterfall{
		AgentID: "A", EventID: "approval-spec", Event: interaction.EventApproval,
		Request: json.RawMessage(`{"toolName":"bash","reason":"writes files","args":{"command":"go test ./..."}}`),
	})
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("A"), ScopeAgentID: "A", ThemeID: "deepseek.light"})
		})
	}, 800, 400)
	return tt, theme.Resolve("deepseek.light").Light
}

// The card's decisions must be the harness buttons: 28 DIP small faces, the
// confirming one filled with the primary token.
func TestApprovalCardUsesHarnessButtonRecipes(t *testing.T) {
	tt, colors := approvalGallery(t)
	allow, ok := tt.Find(i18n.T(i18n.KeyInteractionAllow))
	if !ok {
		t.Fatalf("allow button missing: %q", tt.Texts())
	}
	if allow.H != 28 {
		t.Fatalf("allow height = %v, want the 28 DIP small button", allow.H)
	}
	img := tt.Image()
	// Sample inside the rounded face, clear of the 8 DIP corner radius.
	face := img.RGBAAt(int(allow.X)+6, int(allow.Y+allow.H/2))
	want := theme.Color(colors.BrandPrimary)
	if face.R != want.R || face.G != want.G || face.B != want.B {
		t.Fatalf("allow face = %v, want the primary token %v", face, want)
	}
	deny, ok := tt.Find(i18n.T(i18n.KeyInteractionDeny))
	if !ok {
		t.Fatalf("deny button missing: %q", tt.Texts())
	}
	if deny.H != 28 {
		t.Fatalf("deny height = %v, want the 28 DIP small button", deny.H)
	}
	if deny.Y != allow.Y {
		t.Fatalf("deny/allow baseline drifted: %v vs %v", deny.Y, allow.Y)
	}
}

// The question card's submit is primary, skip/hide are quiet ghost faces.
func TestQuestionCardButtonRecipes(t *testing.T) {
	store := galleryStore(t, dsh.EventsWaterfall{
		AgentID: "A", EventID: "questions-spec", Event: interaction.EventUserQuestions,
		Request: json.RawMessage(`{"questions":[{"id":"q","question":"Pick one","options":[{"label":"A"}]},{"id":"q2","question":"Next"}]}`),
	})
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("A"), ScopeAgentID: "A", ThemeID: "deepseek.light"})
		})
	}, 800, 600)
	for _, label := range []string{
		i18n.T(i18n.KeyChatQuestionSkip),
		i18n.T(i18n.KeyChatQuestionHide),
		i18n.T(i18n.KeyInteractionDeny),
	} {
		button, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%q missing: %q", label, tt.Texts())
		}
		if button.H != 28 {
			t.Fatalf("%q height = %v, want 28 DIP", label, button.H)
		}
	}
	// The primary submit sits on the same baseline row as skip.
	submit, ok := tt.Find(i18n.T(i18n.KeyChatQuestionStep))
	if !ok {
		t.Fatalf("submit missing: %q", tt.Texts())
	}
	if submit.H != 28 {
		t.Fatalf("submit height = %v, want 28 DIP", submit.H)
	}
}

var _ = time.Second
