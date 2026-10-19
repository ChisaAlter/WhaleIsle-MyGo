package composer

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestRequiresWorkspaceFollowsSelectedSessionOwnership(t *testing.T) {
	for _, tc := range []struct {
		name       string
		snap       session.ViewState
		hero, want bool
	}{
		{"no session", session.ViewState{}, true, true},
		{"unowned empty", session.ViewState{SelectedID: "s", WorkspaceReady: true, SelectedCwd: "C:/removed"}, true, true},
		{"loading cwd bridge", session.ViewState{SelectedID: "s", SelectedCwd: "C:/work"}, true, false},
		{"loading no cwd", session.ViewState{SelectedID: "s"}, true, true},
		{"member", session.ViewState{SelectedID: "s", WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{SessionIDs: []string{"s"}}}}, true, false},
		{"foreign membership", session.ViewState{SelectedID: "s", WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{SessionIDs: []string{"other"}}}}, true, true},
		{"scratch", session.ViewState{SelectedID: "s", WorkspaceReady: true, ScratchCwd: "C:/scratch", SelectedCwd: "C:/scratch"}, true, false},
		{"active transcript", session.ViewState{SelectedID: "s", WorkspaceReady: true}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RequiresWorkspace(tc.snap, tc.hero); got != tc.want {
				t.Fatalf("inert=%v want %v", got, tc.want)
			}
		})
	}
}

func TestInertComposerCardOpensWorkspaceWithoutEditingOrSubmitting(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint("native=", native), func(t *testing.T) {
			draft := "retained draft"
			client := NewDraftClient(&draft)
			inert, picked, sent := true, 0, 0
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Draft: &draft, TextInput: client, UseTextInputClient: native, Hero: true,
					WorkspaceRoot: "C:/global-default", Inert: inert, OnRequestWorkspace: func() { picked++ }, OnSubmit: func() { sent++ }})
			}, 700, 500)
			if tt.Focused("Composer draft") {
				t.Fatal("inert input received focus")
			}
			if !tt.HasText(i18n.T(i18n.KeyChatHeroChooseWorkspace)) {
				t.Fatal("global default impersonated a selected workspace")
			}
			if err := tt.Click("Composer draft"); err != nil {
				t.Fatal(err)
			}
			if picked != 1 {
				t.Fatalf("card picker calls=%d", picked)
			}
			tt.Type("ignored")
			tt.Key(0, ui.KeyEnter)
			if picked != 2 || draft != "retained draft" || sent != 0 {
				t.Fatalf("picked=%d draft=%q sent=%d", picked, draft, sent)
			}
			tt.Key(0, ui.KeySpace)
			if picked != 3 {
				t.Fatalf("Space did not open picker: %d", picked)
			}
			card, _ := tt.Find("Composer card")
			tt.ClickAt(card.X+card.W-20, card.Y+card.H-20)
			if picked != 4 || sent != 0 {
				t.Fatalf("toolbar escaped workspace target: picked=%d sent=%d", picked, sent)
			}
			if len(composerToolItems(Model{Inert: true, OnPickImage: func() { sent++ }}, "")) != 0 {
				t.Fatal("inert tools remained available")
			}
			inert = false
			tt.Frame()
			if err := tt.Click("Composer draft"); err != nil {
				t.Fatal(err)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("chosen workspace draft")
			if draft != "chosen workspace draft" {
				t.Fatalf("draft not editable after choosing workspace: %q", draft)
			}
			inert = true
			tt.Frame()
			tt.Type("stale native input")
			if draft != "chosen workspace draft" || tt.Focused("Composer draft") {
				t.Fatal("removed workspace retained input ownership")
			}
		})
	}
}
