package frame

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestConversationPhaseSeparatesHistoryReadinessFromEmpty(t *testing.T) {
	for _, tc := range []struct {
		name         string
		snap         session.ViewState
		interactions bool
		want         conversationLayoutPhase
	}{
		{"sessionless", session.ViewState{}, false, conversationHero},
		{"unknown cold", session.ViewState{SelectedID: "old"}, false, conversationSettling},
		{"unknown loading", session.ViewState{SelectedID: "old", HistoryLoading: true}, false, conversationSettling},
		{"known blank loading", session.ViewState{SelectedID: "new", HistoryLoading: true, SelectedSummaryBlank: true}, false, conversationHero},
		{"known blank failed", session.ViewState{SelectedID: "new", HistoryError: "offline", SelectedSummaryBlank: true}, false, conversationHero},
		{"unknown failed", session.ViewState{SelectedID: "old", HistoryError: "offline"}, false, conversationActive},
		{"ready empty", session.ViewState{SelectedID: "old", HistoryReady: true}, false, conversationHero},
		{"retained history reconnect", session.ViewState{SelectedID: "old", HistoryLoading: true, Messages: []session.Message{{Body: "retained"}}}, false, conversationActive},
		// A running session with no messages yet keeps the Hero: the Original
		// blank-draft surface stays until the first message arrives, and the
		// composer owns the running/stop affordance.
		{"running blank", session.ViewState{SelectedID: "new", SelectedSummaryBlank: true, Items: []session.Item{{SessionID: "new", Running: true}}}, false, conversationHero},
		{"prompt pending", session.ViewState{SelectedID: "new", PromptBusy: true, SelectedSummaryBlank: true}, false, conversationActive},
		{"first prompt failed", session.ViewState{SelectedID: "new", PromptAttempted: true, SelectedSummaryBlank: true}, false, conversationActive},
		{"interaction", session.ViewState{SelectedID: "new", SelectedSummaryBlank: true}, true, conversationActive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := conversationPhase(tc.snap, tc.interactions); got != tc.want {
				t.Fatalf("phase=%v want %v", got, tc.want)
			}
		})
	}
}

func TestConversationHeroFirstMessageKeepsCardAndDraftIdentity(t *testing.T) {
	for _, column := range []int{400, 900, 1500} {
		for _, savedWidth := range []float32{0, 460} {
			t.Run(fmt.Sprintf("column%d/resize%v", column, savedWidth), func(t *testing.T) {
				sizes := DefaultLayout()
				sizes.Bottom = 0
				sizes.Right = 518
				right := float32(518)
				list := ui.ListState{}
				var viewport ui.Rect
				draft := "ABCDE"
				client := composer.NewDraftClient(&draft)
				state := composer.DraftState{}
				snap := session.ViewState{SelectedID: "current", SelectedCwd: "C:/work", HistoryReady: true}
				st := session.New()
				tt := ui.NewTester(func(c *ui.Context) {
					originalScrollbarTheme(c)
					cols := sizes.ResolveColumns(float32(column+800), true)
					right = cols.Details
					colors := theme.Resolve("deepseek.light").Light
					m := Model{Store: st, Sizes: &sizes, RightPaneOpen: true, FilePaneWidth: &right, Draft: &draft, ComposerState: &state, ChatList: &list,
						UseComposerTextInput: true, ComposerTextInput: client,
						ConversationPreferences: settings.ConversationPreferences{ComposerResize: savedWidth > 0}, ComposerResize: &composer.ResizeState{Width: savedWidth}}
					ui.Column(c).Fill().Children(func() { mainRow(c, m, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted)) })
					if len(snap.Messages) > 0 {
						c.Resolve(list.Handle).DrawOver(func(_ *ui.Painter, r ui.Rect) { viewport = r })
					}
				}, column+800, 800)
				before, ok := tt.Find("Composer card")
				if !ok {
					t.Fatal("hero card missing")
				}
				saveConversationEvidence(t, tt, fmt.Sprintf("conversation-hero-%d-resize%v", column, savedWidth))
				if err := tt.Click("Composer draft"); err != nil {
					t.Fatal(err)
				}
				client.SetSelectionUTF16(1, 4)
				snap.Messages = []session.Message{{ID: "first", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: "first message"}}
				tt.Frame()
				after, ok := tt.Find("Composer card")
				if !ok || before.X != after.X || before.W != after.W {
					t.Fatalf("hero=%+v active=%+v", before, after)
				}
				if viewport.X != 281 || viewport.W != float32(column)-conversationScrollMargin {
					t.Fatalf("transcript viewport=%+v, want full column minus source2DIP margin", viewport)
				}
				if !tt.Focused("Composer draft") || client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
					t.Fatalf("first message focus=%v selection=%+v", tt.Focused("Composer draft"), client.Selection())
				}
				saveConversationEvidence(t, tt, fmt.Sprintf("conversation-first-message-%d-resize%v", column, savedWidth))
				tt.Type("Z")
				if draft != "AZE" {
					t.Fatalf("draft=%q", draft)
				}
			})
		}
	}
}

// An empty but ready session is the Hero: the blank-draft surface, never the
// 13px "select a session" hint or an empty transcript column.
func TestEmptyReadySessionRendersHero(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	snap := session.ViewState{SelectedID: "old", HistoryReady: true}
	st := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		originalScrollbarTheme(c)
		sizes.ResolveColumns(1200, false)
		colors := theme.Resolve("deepseek.light").Light
		ui.Column(c).Fill().Children(func() {
			mainRow(c, Model{Store: st, Sizes: &sizes}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1200, 800)
	if !tt.HasText(i18n.T(i18n.KeyChatHeroSlogan)) || !tt.HasText(i18n.T(i18n.KeyChatHeroPreview)) {
		t.Fatalf("empty session lost the hero: %q", tt.Texts())
	}
	if _, ok := tt.Find("Composer card"); !ok {
		t.Fatal("hero lost its centered composer")
	}
	if tt.HasText(i18n.T(i18n.KeyChatEmpty)) {
		t.Fatal("empty session fell back to the 13px hint")
	}
}

// Unknown history settles on the Hero shell: skeleton placeholders instead of a
// blank box, and no real composer to focus or submit through.
func TestSettlingPaintsSkeletonWithComposerGeometry(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	snap := session.ViewState{SelectedID: "old", HistoryLoading: true}
	st := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		originalScrollbarTheme(c)
		sizes.ResolveColumns(1200, false)
		colors := theme.Resolve("deepseek.light").Light
		ui.Column(c).Fill().Children(func() {
			mainRow(c, Model{Store: st, Sizes: &sizes}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1200, 800)
	skeleton, ok := tt.Find("Loading conversation")
	if !ok {
		t.Fatalf("settling painted no skeleton: %q", tt.Texts())
	}
	if _, ok := tt.Find("Composer card"); ok {
		t.Fatal("settling mounted the real composer")
	}
	if skeleton.W <= 0 || skeleton.H <= 0 {
		t.Fatalf("skeleton collapsed: %+v", skeleton)
	}
}

func TestConversationSettlingAndFailureUseSharedStatus(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	snap := session.ViewState{SelectedID: "old", HistoryLoading: true}
	st := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		originalScrollbarTheme(c)
		sizes.ResolveColumns(1200, false)
		colors := theme.Resolve("deepseek.light").Light
		ui.Column(c).Fill().Children(func() {
			mainRow(c, Model{Store: st, Sizes: &sizes, ConnectionBanner: "connection unavailable"}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1200, 800)
	if tt.HasText(i18n.T(i18n.KeyChatHeroSlogan)) || tt.Focused("Composer draft") {
		t.Fatal("settling flashed hero or focused hidden input")
	}
	if !tt.HasText("connection unavailable") {
		t.Fatal("connection status hidden while settling")
	}
	saveConversationEvidence(t, tt, "conversation-settling")
	snap.HistoryLoading = false
	snap.HistoryError = "follow failed"
	tt.Frame()
	if tt.HasText(i18n.T(i18n.KeyChatHeroSlogan)) || !tt.HasText("follow failed") {
		t.Fatal("unknown failed history appeared blank or lost its error")
	}
	if _, ok := tt.Find("Retry conversation history"); !ok {
		t.Fatal("history retry unavailable")
	}
	saveConversationEvidence(t, tt, "conversation-follow-error")
	snap.SelectedSummaryBlank = true
	tt.Frame()
	if !tt.HasText(i18n.T(i18n.KeyChatHeroSlogan)) || !tt.HasText("follow failed") {
		t.Fatal("authoritative blank exception lost hero or common error")
	}
}

func TestConversationAncestrySharesTitleBandWithUtilities(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	snap := session.ViewState{SelectedID: "current", HistoryReady: true,
		Items:    []session.Item{{SessionID: "current", Title: "Current conversation"}},
		Messages: []session.Message{{ID: "first", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: "history"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		originalScrollbarTheme(c)
		sizes.ResolveColumns(1800, false)
		colors := theme.Resolve("deepseek.light").Light
		ui.Column(c).Fill().Children(func() {
			mainRow(c, Model{Store: session.New(), Sizes: &sizes, OnOpenAgents: func() {}}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1800, 800)
	title, ok := tt.Find("Conversation session title")
	if !ok {
		t.Fatal("session ancestry missing")
	}
	utility, ok := tt.Find(i18n.T(i18n.KeyFrameCaptionAgents))
	if !ok {
		t.Fatal("header utility missing")
	}
	if title.Y >= utility.Y+utility.H || utility.Y >= title.Y+title.H {
		t.Fatalf("ancestry=%+v utility=%+v occupy separate rows", title, utility)
	}
}

func saveConversationEvidence(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SHELL_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestActiveConversationDockRendersOnlyActualBoardContent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		board    session.PlanBoard
		editor   session.QueueEditorState
		wantText string
	}{
		{"empty", session.PlanBoard{}, session.QueueEditorState{}, ""},
		{"plan mode", session.PlanBoard{PlanMode: true}, session.QueueEditorState{}, i18n.T(i18n.KeyPlanBoardPlanOn)},
		{"goal", session.PlanBoard{Goal: &session.PlanGoal{ID: "goal", Objective: "keep actual goal", Phase: "active"}}, session.QueueEditorState{}, "keep actual goal"},
		{"todo", session.PlanBoard{Todos: []session.PlanTodo{{Content: "keep actual todo", Status: "in_progress"}}}, session.QueueEditorState{}, "keep actual todo"},
		{"queue", session.PlanBoard{Queue: []session.PlanQueueItem{{ID: "queue", Preview: "keep queued prompt"}}}, session.QueueEditorState{}, "keep queued prompt"},
		{"pending", session.PlanBoard{Pending: []session.PendingSubmission{{RequestID: "pending", Text: "keep admitted prompt"}}}, session.QueueEditorState{}, "keep admitted prompt"},
		{"queue editor", session.PlanBoard{}, session.QueueEditorState{ItemID: "editing", Ready: true, Text: "keep editor draft"}, i18n.T(i18n.KeyPlanBoardQueueEdit)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom = 0
			st := session.New()
			snap := session.ViewState{SelectedID: "current", HistoryReady: true, PlanBoard: tc.board, QueueEditor: tc.editor,
				Messages: []session.Message{{ID: "first", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: "actual conversation"}}}
			tt := ui.NewTester(func(c *ui.Context) {
				originalScrollbarTheme(c)
				sizes.ResolveColumns(1200, false)
				colors := theme.Resolve("deepseek.light").Light
				ui.Column(c).Fill().Children(func() {
					mainRow(c, Model{Store: st, Sizes: &sizes}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, 1200, 800)
			if tc.wantText == "" {
				if tt.HasText(i18n.T(i18n.KeyPlanBoardTitle)) || tt.HasText(i18n.T(i18n.KeyPlanBoardEmpty)) || tt.HasText(i18n.T(i18n.KeyPlanBoardNoGoal)) {
					t.Fatal("empty diagnostics occupy conversation dock")
				}
			} else if !tt.HasText(tc.wantText) {
				t.Fatalf("actual dock state hidden: %q", tc.wantText)
			}
			if _, ok := tt.Find("Composer card"); !ok {
				t.Fatal("dock lost composer")
			}
		})
	}
}
