package frame

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/theme"
)

func TestComposerInteractionSelectsApprovalBeforeQuestion(t *testing.T) {
	snapshot := interaction.ViewState{AutoAllow: true, Banner: "auto", QuestionStatus: "continued", Pending: []interaction.Pending{
		{EventID: "continued", Event: interaction.EventUserQuestions, Continued: true},
		{EventID: "question", Event: interaction.EventUserQuestions},
		{EventID: "approval", Event: interaction.EventApproval},
	}}
	got, takeover := composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "approval" || got.AutoAllow || got.Banner != "" || got.QuestionStatus != "" {
		t.Fatalf("takeover=%v snapshot=%+v", takeover, got)
	}
	snapshot.Pending = snapshot.Pending[:2]
	got, takeover = composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "question" {
		t.Fatalf("question not selected: %+v", got)
	}
	snapshot.Pending = snapshot.Pending[:1]
	got, takeover = composerInteraction(snapshot)
	if takeover || len(got.Pending) != 1 || !got.AutoAllow {
		t.Fatal("continued history replaced the composer")
	}
	snapshot.Pending = []interaction.Pending{{EventID: "earlier", Event: interaction.EventUserQuestions}, {EventID: "latest", Event: interaction.EventUserQuestions}}
	snapshot.QuestionCards = map[string]interaction.QuestionCardView{"latest": {Hidden: true}}
	got, takeover = composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "earlier" {
		t.Fatal("minimizing the latest question did not reveal the earlier composer contribution")
	}
	snapshot.QuestionCards["earlier"] = interaction.QuestionCardView{Hidden: true}
	if _, takeover = composerInteraction(snapshot); takeover {
		t.Fatal("hidden questions kept the ordinary composer covered")
	}
}

func TestConversationApprovalAndQuestionTakeComposerSeatThenRestoreDraft(t *testing.T) {
	for _, event := range []string{interaction.EventApproval, interaction.EventUserQuestions} {
		t.Run(event, func(t *testing.T) {
			store := interaction.New()
			defer store.Detach()
			ready := make(chan struct{}, 1)
			store.Attach(func() {
				select {
				case ready <- struct{}{}:
				default:
				}
			})
			sizes := DefaultLayout()
			sizes.Bottom = 0
			draft := "keep my draft"
			client := composer.NewDraftClient(&draft)
			snap := session.ViewState{SelectedID: "A", HistoryReady: true, SelectedCwd: "C:/work", Messages: []session.Message{{ID: "m", Role: "user", Body: "transcript"}}}
			tt := ui.NewTester(func(c *ui.Context) {
				originalScrollbarTheme(c)
				sizes.ResolveColumns(1200, false)
				colors := theme.Resolve("deepseek.light").Light
				ui.Column(c).Fill().Children(func() {
					mainRow(c, Model{Store: session.New(), Interactions: store, Sizes: &sizes, Draft: &draft, UseComposerTextInput: true, ComposerTextInput: client}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, 1200, 800)
			if err := tt.Click("Composer draft"); err != nil {
				t.Fatal(err)
			}
			client.SetSelectionUTF16(2, 6)
			before, _ := tt.Find("Composer card")
			request := json.RawMessage(`{"toolName":"exec_command","reason":"needs approval"}`)
			if event == interaction.EventUserQuestions {
				request = json.RawMessage(`{"questions":[{"id":"q","question":"Choose an option","options":[{"label":"One"},{"label":"Two"}]}]}`)
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				_, _ = store.HandleWaterfall(dsh.EventsWaterfall{AgentID: "A", EventID: "pending", Event: event, Request: request})
			}()
			select {
			case <-ready:
			case <-time.After(time.Second):
				t.Fatal("interaction did not open")
			}
			tt.Frame()
			if tt.Focused("Composer draft") {
				t.Fatal("ordinary composer retained keyboard ownership during takeover")
			}
			tt.Type("must not edit draft")
			if draft != "keep my draft" {
				t.Fatalf("takeover edited retained draft: %q", draft)
			}
			if _, ok := tt.Find("Composer card"); ok {
				t.Fatal("ordinary composer remained visible during takeover")
			}
			saveConversationEvidence(t, tt, "composer-takeover-"+strings.ReplaceAll(event, "/", "-"))
			if event == interaction.EventUserQuestions {
				store.HideQuestion("pending")
				tt.Frame()
				if _, ok := tt.Find("Composer card"); !ok {
					t.Fatal("minimized question kept the ordinary draft hidden")
				}
				if err := tt.Click("Composer draft"); err != nil {
					t.Fatal(err)
				}
				if !tt.Focused("Composer draft") {
					t.Fatal("minimized question left draft inaccessible")
				}
				if err := tt.Click("重新打开问题 / Reopen questions"); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
				if _, ok := tt.Find("Composer card"); ok {
					t.Fatal("reopened question did not retake the composer seat")
				}
				if draft != "keep my draft" {
					t.Fatal("minimize/reopen discarded the draft")
				}
				client.SetSelectionUTF16(2, 6)
			}
			store.OnCancel(dsh.EventsCancel{EventID: "pending"})
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("cancelled interaction stuck")
			}
			tt.Frame()
			after, ok := tt.Find("Composer card")
			if !ok || after.X != before.X || after.W != before.W || draft != "keep my draft" {
				t.Fatalf("draft/card not restored: before=%+v after=%+v draft=%q", before, after, draft)
			}
			if client.Selection().Range != (ui.TextInputRange{Start: 2, End: 6}) {
				t.Fatalf("draft selection lost: %+v", client.Selection())
			}
		})
	}
}
