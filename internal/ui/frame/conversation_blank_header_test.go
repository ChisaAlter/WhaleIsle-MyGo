package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestConversationBlankHeaderUsesLifecycleInsteadOfHeroGeometry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*session.ViewState)
		active bool
		want   bool
	}{
		{"blank", nil, false, true},
		{"unbound", func(s *session.ViewState) { s.SelectedID = "" }, false, false},
		{"history not authoritative", func(s *session.ViewState) { s.SelectedSummaryBlank = false }, false, false},
		{"running blank hero", func(s *session.ViewState) { s.Items[0].Running = true }, false, false},
		{"first prompt pending", func(s *session.ViewState) { s.PromptBusy = true }, false, false},
		{"failed first prompt remains engaging", func(s *session.ViewState) { s.PromptAttempted = true }, false, false},
		{"active target", nil, true, false},
		{"messages", func(s *session.ViewState) { s.Messages = []session.Message{{Body: "first"}} }, false, false},
		{"presentation with ordinary composer", func(s *session.ViewState) { s.Items[0].Presentation = &session.SessionPresentation{} }, false, false},
		{"managed presentation", func(s *session.ViewState) {
			s.Items[0].Presentation = &session.SessionPresentation{Composer: "managed"}
		}, false, false},
		{"other presentation is irrelevant", func(s *session.ViewState) {
			s.Items = append(s.Items, session.Item{SessionID: "other", Presentation: &session.SessionPresentation{}, Running: true})
		}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := session.ViewState{SelectedID: "selected", SelectedSummaryBlank: true, Items: []session.Item{{SessionID: "selected", Blank: true}}}
			if tc.change != nil {
				tc.change(&snap)
			}
			if got := conversationHeaderHidden(snap, tc.active); got != tc.want {
				t.Fatalf("hidden=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestBlankCaptionPreservesOriginalLeadingAndCornerOnly(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 1000
	snap := session.ViewState{SelectedID: "s", SelectedSummaryBlank: true, Items: []session.Item{{SessionID: "s", Blank: true}}}
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceSharedCaption(c, Model{Sizes: &sizes, Store: session.New(), OnOpenSessionPlugins: func() { t.Fatal("implicit plugin activation") }, ConversationContributions: func(c *ui.Context, _ session.ViewState, slot string) { ui.Button(c, slot).Height(28) }, WorkspaceTrailing: func(c *ui.Context, _ string) { ui.Button(c, "native corner controls").Height(28) }}, snap, theme.Resolve("deepseek.light").Light, true)
	}, 1000, 100)
	for _, slot := range []string{"conversation.session.header.leading", "conversation.session.header.corner", "native corner controls", "Blank conversation caption"} {
		if _, ok := tt.Find(slot); !ok {
			t.Fatalf("missing retained %s", slot)
		}
	}
	for _, label := range []string{"conversation.session.header.actions", "conversation.session.header.utilities", i18n.T(i18n.KeyLauncherOpen), "Conversation session title"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("blank caption retained %s", label)
		}
	}
	// Running remains a header lifecycle fact even though the native empty-body
	// layout still uses the hero, and the existing plugin action stays reachable.
	snap.Items[0].Running = true
	tt.Frame()
	if _, ok := tt.Find("更多操作"); !ok && !tt.HasText("More actions") {
		t.Fatal("running hero lost its explicit plugin menu")
	}
}

func TestSelectedSessionPluginActionRejectsStaleRowSelection(t *testing.T) {
	store := session.New()
	store.Select("a")
	calls := 0
	action := selectedSessionPluginAction(Model{Store: store}, "a", func() { calls++ })
	if action == nil {
		t.Fatal("selected callback unavailable")
	}
	action("b")
	if calls != 0 {
		t.Fatal("foreign row opened selected session")
	}
	action("a")
	if calls != 1 {
		t.Fatal("current row action did not run")
	}
	store.Select("b")
	action("a")
	if calls != 1 {
		t.Fatal("stale menu opened new selection")
	}
	if selectedSessionPluginAction(Model{}, "a", func() {}) != nil || selectedSessionPluginAction(Model{Store: store}, "", func() {}) != nil {
		t.Fatal("unbound menu became actionable")
	}
}
