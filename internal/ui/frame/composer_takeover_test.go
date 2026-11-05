package frame

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestComposerInteractionSelectsPlanBeforeQuestionBeforeApproval(t *testing.T) {
	snapshot := interaction.ViewState{AutoAllow: true, Banner: "auto", QuestionStatus: "continued", Pending: []interaction.Pending{
		{EventID: "continued", Event: interaction.EventUserQuestions, Continued: true},
		{EventID: "question", Event: interaction.EventUserQuestions},
		{EventID: "approval", Event: interaction.EventApproval},
		{EventID: "plan", Event: interaction.EventUserQuestions},
	}, QuestionCards: map[string]interaction.QuestionCardView{"plan": {PlanReview: &interaction.PlanReview{}}}}
	got, takeover := composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "plan" || got.AutoAllow || got.Banner != "" || got.QuestionStatus != "" {
		t.Fatalf("takeover=%v snapshot=%+v", takeover, got)
	}
	snapshot.QuestionCards["plan"] = interaction.QuestionCardView{PlanReviewHidden: true}
	got, takeover = composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "question" {
		t.Fatalf("question not selected: %+v", got)
	}
	snapshot.QuestionCards["question"] = interaction.QuestionCardView{Hidden: true}
	got, takeover = composerInteraction(snapshot)
	if !takeover || len(got.Pending) != 1 || got.Pending[0].EventID != "approval" {
		t.Fatalf("approval not selected after hidden questions: %+v", got)
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
	for _, width := range []float32{1200, 400, 340} {
		for _, event := range []string{interaction.EventApproval, interaction.EventUserQuestions} {
			t.Run(fmt.Sprintf("%s/%g", event, width), func(t *testing.T) {
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
					sizes.ResolveColumns(width, false)
					colors := theme.Resolve("deepseek.light").Light
					ui.Column(c).Fill().Children(func() {
						mainRow(c, Model{Store: session.New(), Interactions: store, Sizes: &sizes, Draft: &draft, UseComposerTextInput: true, ComposerTextInput: client}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
					})
				}, int(width), 800)
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
				saveConversationEvidence(t, tt, fmt.Sprintf("composer-takeover-%s-%g", strings.ReplaceAll(event, "/", "-"), width))
				if event == interaction.EventUserQuestions {
					// Exercise the real conversation/composer seat, including its
					// own width and padding, rather than adding padding to a body fixture.
					for _, label := range []string{i18n.T(i18n.KeyChatQuestionPrev), i18n.T(i18n.KeyChatQuestionNext), i18n.T(i18n.KeyChatQuestionSkip), i18n.T(i18n.KeyChatQuestionSubmit)} {
						r, ok := tt.Find(label)
						if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.X+r.W > width+.5 || r.Y < 0 || r.Y+r.H > 800 {
							t.Fatalf("question footer %q escaped the real %g-wide seat: %+v found=%v", label, width, r, ok)
						}
					}
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
					if err := tt.Click(i18n.T(i18n.KeyChatQuestionReopen)); err != nil {
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
}
