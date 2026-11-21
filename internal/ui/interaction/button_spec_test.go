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

// The card's decisions must be the harness buttons: 36 DIP medium faces
// matching the Original's 36px card buttons; the confirming one uses the
// primary token.
func TestApprovalCardUsesHarnessButtonRecipes(t *testing.T) {
	tt, colors := approvalGallery(t)
	allow, ok := tt.Find(i18n.T(i18n.KeyInteractionAllow))
	if !ok {
		t.Fatalf("allow button missing: %q", tt.Texts())
	}
	if allow.H != 36 {
		t.Fatalf("allow height = %v, want the 36 DIP medium button", allow.H)
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
	if deny.H != 36 {
		t.Fatalf("deny height = %v, want the 36 DIP medium button", deny.H)
	}
	if deny.Y != allow.Y {
		t.Fatalf("deny/allow baseline drifted: %v vs %v", deny.Y, allow.Y)
	}
}

// The question card's submit is primary, skip is a quiet ghost face, and the
// header carries the Original's 24 DIP .iconButton affordances: minimize and
// one dismiss seat (nav.close while the Host named the call).
func TestQuestionCardButtonRecipes(t *testing.T) {
	store := galleryStore(t, dsh.EventsWaterfall{
		AgentID: "A", EventID: "questions-spec", Event: interaction.EventUserQuestions,
		Request: json.RawMessage(`{"callId":"call-q","questions":[{"id":"q","question":"Pick one","options":[{"label":"A"}]},{"id":"q2","question":"Next"}]}`),
	})
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, Model{Store: store, Snapshot: store.SnapshotForAgent("A"), ScopeAgentID: "A", ThemeID: "deepseek.light"})
		})
	}, 800, 600)
	if skip, ok := tt.Find(i18n.T(i18n.KeyChatQuestionSkip)); !ok {
		t.Fatalf("skip missing: %q", tt.Texts())
	} else if skip.H != 36 {
		t.Fatalf("skip height = %v, want 36 DIP", skip.H)
	}
	for _, label := range []string{
		questionStatusText("nav.minimize"),
		questionStatusText("nav.close"),
	} {
		button, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%q missing: %q", label, tt.Texts())
		}
		if button.W != 24 || button.H != 24 {
			t.Fatalf("%q seat = %vx%v, want the 24 DIP icon button", label, button.W, button.H)
		}
	}
	// The primary submit sits on the same baseline row as skip.
	// The pager's icon button shares the 下一题 name with the primary action;
	// pick the 36 DIP match (the pager seat is 24 DIP).
	submits := tt.FindAll(i18n.T(i18n.KeyChatQuestionStep))
	found := false
	for _, r := range submits {
		if r.H == 36 {
			found = true
		}
	}
	if !found {
		t.Fatalf("36 DIP submit missing among %v", submits)
	}
}

var _ = time.Second
