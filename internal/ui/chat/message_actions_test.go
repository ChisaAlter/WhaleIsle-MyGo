package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestAssistantTailOneCompactRowKeepsLifecycleTextAndBoundary(t *testing.T) {
	messages := []session.Message{
		{ID: "u", Role: "user", Kind: session.MessageKindText, Seq: 1, Body: "Question"},
		{ID: "earlier", Role: "assistant", Kind: session.MessageKindText, Seq: 2, Body: "Earlier tool step"},
		{ID: "final", SourceMessageID: "final", Role: "assistant", Kind: session.MessageKindText, Seq: 3, Body: "First block"},
		{ID: "final/text/2", SourceMessageID: "final", Role: "assistant", Kind: session.MessageKindText, Seq: 3, Body: "Second block"},
		{ID: "end", Role: "status", EventType: "turn/end", Kind: session.MessageKindText, Seq: 4},
	}
	quoted := ""
	forked := float64(0)
	m := Model{Snapshot: session.ViewState{Messages: messages, ClientReady: true}, OnQuoteMessage: func(value string) { quoted = value }, OnForkTurn: func(seq float64) { forked = seq }, FeedbackStore: session.New()}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	targets := assistantActionTargets(messages)
	if len(targets) != 1 || targets[4].message.Body != "First blockSecond block" {
		t.Fatalf("targets=%+v", targets)
	}
	actions, ok := tt.Find("Assistant message actions")
	if !ok || actions.H != 28 {
		t.Fatalf("tail geometry=%+v", actions)
	}
	for _, label := range []string{FeedbackButtonLabel(session.MessageFeedbackUp, ""), FeedbackButtonLabel(session.MessageFeedbackDown, ""), branchActionLabel()} {
		r, found := tt.Find(label)
		if !found || r.W != 28 || r.H != 28 || r.Y != actions.Y {
			t.Fatalf("%q=%+v on actions %+v", label, r, actions)
		}
	}
	// Click coordinates avoid the user row's identical copy label.
	tt.ClickAt(actions.X+14, actions.Y+14)
	if tt.Clipboard() != "First blockSecond block" {
		t.Fatalf("copy=%q", tt.Clipboard())
	}
	tt.Key(ui.Shift, ui.KeyF10)
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyChatQuote)); err != nil {
		t.Fatal(err)
	}
	if quoted != "First blockSecond block" {
		t.Fatalf("quote=%q", quoted)
	}
	if err := tt.Click(branchActionLabel()); err != nil {
		t.Fatal(err)
	}
	if forked != 4 {
		t.Fatalf("fork boundary=%v", forked)
	}
	if tt.HasText("读取反馈 / Load feedback") {
		t.Fatal("diagnostic feedback button remains")
	}
}

func TestAssistantForkDistinguishesMixedLifecycleFromLaterImage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		text          bool
		sameLifecycle bool
		wantActions   bool
		wantFork      bool
	}{
		{"text and image in one lifecycle", true, true, true, true},
		{"earlier text then separate image lifecycle", true, false, true, false},
		{"standalone image lifecycle", false, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messages := []session.Message{}
			if tc.text {
				id, seq := "earlier", float64(2)
				if tc.sameLifecycle {
					id, seq = "mixed", 3
				}
				messages = append(messages, session.Message{ID: id, SourceMessageID: id, Role: "assistant", Kind: session.MessageKindText, EventType: "assistant/message", Seq: seq, Body: "Final prose"})
			}
			imageID := "image-only"
			if tc.sameLifecycle {
				imageID = "mixed"
			}
			messages = append(messages, session.Message{ID: imageID + "/image/1", SourceMessageID: imageID, Role: "assistant", Kind: session.MessageKindImage, EventType: "assistant/message", Seq: 3, ImageRef: "attachment:fixture"}, session.Message{ID: "end", Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Seq: 4})
			targets := assistantActionTargets(messages)
			if (len(targets) > 0) != tc.wantActions {
				t.Fatalf("targets=%+v", targets)
			}
			if tc.sameLifecycle && session.CanForkTurn(messages, 4) {
				t.Fatal("fixture did not reproduce projected trailing-image mismatch")
			}
			forked := float64(0)
			m := Model{Snapshot: session.ViewState{Messages: messages, ClientReady: true}, OnForkTurn: func(seq float64) { forked = seq }, FeedbackStore: session.New()}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
			if _, found := tt.Find(branchActionLabel()); found != tc.wantActions {
				t.Fatalf("branch present=%v want%v", found, tc.wantActions)
			}
			if tc.wantActions {
				target := targets[len(messages)-1]
				if canForkAssistantTarget(messages, target) != tc.wantFork {
					t.Fatalf("fork guard mismatch target=%+v", target)
				}
				if err := tt.Click(branchActionLabel()); err != nil {
					t.Fatal(err)
				}
				wantSeq := float64(0)
				if tc.wantFork {
					wantSeq = 4
				}
				if forked != wantSeq {
					t.Fatalf("fork seq=%v want%v", forked, wantSeq)
				}
				if target.message.FeedbackMessageID() != messages[0].FeedbackMessageID() {
					t.Fatal("feedback target changed to image-only lifecycle")
				}
			}
		})
	}
}

func TestEarlierUserActionsRevealOnHoverAndKeyboardWithoutReflow(t *testing.T) {
	msg := presentationMessage("Earlier message")
	m := Model{Snapshot: session.ViewState{Messages: []session.Message{msg, {ID: "later", Role: "user", Kind: session.MessageKindText, Seq: 8, Body: "Later"}}}, OnQuoteMessage: func(string) {}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { userMessageView(c, m, msg, theme.Default().Active(false), nil, msg.Body, nil) })
	}, 800, 180)
	actions, _ := tt.Find("User message actions")
	before := tt.Image()
	tt.Move(actions.X+14, actions.Y+14)
	tt.Frame()
	if got, _ := tt.Find("User message actions"); got != actions {
		t.Fatal("hover changed row geometry")
	}
	after := tt.Image()
	changed := false
	for y := int(actions.Y); y < int(actions.Y+actions.H); y++ {
		for x := int(actions.X); x < int(actions.X+actions.W); x++ {
			if before.RGBAAt(x, y) != after.RGBAAt(x, y) {
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("hover did not reveal controls")
	}
	tt.Move(0, 0)
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !tt.Focused(i18n.T(i18n.KeyChatCopy)) {
		t.Fatal("hidden historical action left keyboard order")
	}
	tt.Key(0, ui.KeyEnter)
	if tt.Clipboard() != msg.Body {
		t.Fatal("keyboard copy lost source")
	}
}

// A message that can be copied must offer copy from its context menu even when
// no quote handler is wired; previously the menu was not created at all.
func TestMessageContextMenuCopiesWithoutQuoteHandler(t *testing.T) {
	messages := []session.Message{
		{ID: "q", Role: "user", Kind: session.MessageKindText, Seq: 1, Body: "Question"},
		{ID: "a", SourceMessageID: "a", Role: "assistant", Kind: session.MessageKindText, Seq: 2, Body: "Copy me", EventType: "assistant/message"},
	}
	m := Model{Snapshot: session.ViewState{Messages: messages, ClientReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	if err := tt.RightClick("Assistant message actions"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tt.Menu(), i18n.T(i18n.KeyChatCopy)) {
		t.Fatalf("context menu has no Copy: %v", tt.Menu())
	}
	if slices.Contains(tt.Menu(), i18n.T(i18n.KeyChatQuote)) {
		t.Fatalf("quote appeared without a handler: %v", tt.Menu())
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyChatCopy)); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "Copy me" {
		t.Fatalf("menu copy=%q", tt.Clipboard())
	}
}

func TestFeedbackColdHoverLoadsOnceAndKeepsRetryAndEditor(t *testing.T) {
	var reads atomic.Int32
	var failed atomic.Bool
	failed.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ RPCID string }
		json.NewDecoder(r.Body).Decode(&request)
		if r.URL.Path != "/api/messageFeedback/list" {
			t.Errorf("unexpected call %s", r.URL.Path)
		}
		reads.Add(1)
		if failed.Load() {
			http.Error(w, "fixture unavailable", http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"ok": true, "value": map[string]any{"items": []any{}}}}})
	}))
	defer srv.Close()
	st := session.New()
	st.Select("fixture") // Select before Attach keeps all test traffic scoped to feedback.
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	defer st.Detach()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			authoritativeFeedbackActions(c, st, "message", ui.RGB(90, 90, 90))
			authoritativeFeedbackStatus(c, st, ui.RGB(90, 90, 90))
			messageFeedbackDialog(c, Model{FeedbackStore: st, Snapshot: session.ViewState{SelectedID: "fixture"}})
		})
	}, 600, 400)
	if reads.Load() != 0 {
		t.Fatal("feedback loaded on mount")
	}
	r, _ := tt.Find(FeedbackButtonLabel(session.MessageFeedbackUp, ""))
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := st.Feedback()
			if reads.Load() > 0 && !v.Loading {
				tt.Frame()
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("feedback fixture timed out")
	}
	wait()
	if st.Feedback().Error == "" {
		t.Fatal("load failure hidden")
	}
	tt.Frame()
	if reads.Load() != 1 {
		t.Fatal("error caused automatic retry loop")
	}
	failed.Store(false)
	if err := tt.Click(i18n.T(i18n.KeyCommandCatalogRetry)); err != nil {
		t.Fatal(err)
	}
	wait()
	if !st.Feedback().Ready || reads.Load() != 2 {
		t.Fatalf("retry=%+v reads=%d", st.Feedback(), reads.Load())
	}
	if err := tt.Click(FeedbackButtonLabel(session.MessageFeedbackDown, "")); err != nil {
		t.Fatal(err)
	}
	if v := st.Feedback(); v.EditorID != "message" || v.Rating != session.MessageFeedbackDown {
		t.Fatalf("editor=%+v", v)
	}
	if err := tt.Click(feedbackText("close")); err != nil {
		t.Fatal(err)
	}
	if st.Feedback().EditorID != "" {
		t.Fatal("cancel failed")
	}
}

func TestZeroSequenceDoesNotEnterEditingState(t *testing.T) {
	msg := session.Message{ID: "legacy", Role: "user", Kind: session.MessageKindText, Body: "legacy fixture"}
	tt := messageTester(msg, 800, func(m *Model) { m.OnCancelEdit = func() { t.Fatal("cancel without edit") } })
	if tt.HasText(i18n.T(i18n.KeyChatEditHint)) {
		t.Fatal("zero sentinel masquerades as editing")
	}
}

func TestFullColumnTranscriptCentersContentAndKeepsScrollbarSpace(t *testing.T) {
	for _, width := range []int{400, 1200} {
		msg := presentationMessage("A user message")
		tt := messageTester(msg, width, func(m *Model) { m.FullColumnViewport = true; m.ContentWidth = 680 })
		bubble, ok := tt.Find("User message bubble")
		if !ok {
			t.Fatal("bubble missing")
		}
		bodyWidth := min(float32(680), float32(width)-69)
		// The full-column wrapper already clears the 32 DIP side gutter, so the
		// message spends no second 16 DIP inset: its right edge is the column's.
		wantRight := (float32(width) - 5 + bodyWidth) / 2
		rectNear(t, bubble.X+bubble.W, wantRight, "full-column content axis")
	}
}

func TestFeedbackSelectedLabelKeepsSemanticKeyboardFocus(t *testing.T) {
	selected := ""
	tt := ui.NewTester(func(c *ui.Context) {
		icon := messageIconLikeOutline
		if selected != "" {
			icon = messageIconLikeFill
		}
		messageAction(c, icon, FeedbackButtonLabel(session.MessageFeedbackUp, selected), true, ui.RGB(90, 90, 90), "feedback-positive")
	}, 100, 80)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(FeedbackButtonLabel(session.MessageFeedbackUp, "")) {
		t.Fatal("feedback focus unavailable")
	}
	selected = session.MessageFeedbackUp
	tt.Frame()
	if !tt.Focused(FeedbackButtonLabel(session.MessageFeedbackUp, selected)) {
		t.Fatal("selected glyph/label changed keyboard owner")
	}
}

func TestMessageEditStatusAndCopyParity(t *testing.T) {
	prev := i18n.Locale()
	defer i18n.SetLocale(prev)

	textMsg := session.Message{ID: "u1", Role: "user", Kind: session.MessageKindText, Seq: 2, Body: "hello"}
	imgMsg := session.Message{ID: "u2", Role: "user", Kind: session.MessageKindText, Seq: 4, Body: "[image] C:\\tmp\\a.png"}

	i18n.SetLocale(i18n.LocaleEN)
	if got := messageEditStatus(session.ViewState{PromptBusy: true, Messages: []session.Message{textMsg}}, textMsg, 0); got != "Wait for the current response to finish before editing" {
		t.Fatalf("running status en = %q", got)
	}
	if got := messageEditStatus(session.ViewState{MessageEditBusy: true, Messages: []session.Message{textMsg}}, textMsg, 0); got != "The input box is busy; editing is unavailable right now" {
		t.Fatalf("busy status en = %q", got)
	}
	if got := messageEditStatus(session.ViewState{Messages: []session.Message{imgMsg}}, imgMsg, 0); got != "Messages with non-text content cannot be edited yet" {
		t.Fatalf("unsupported status en = %q", got)
	}
	if got := messageEditStatus(session.ViewState{Messages: []session.Message{textMsg, {ID: "u3", Role: "user", Kind: session.MessageKindText, Seq: 6, Body: "later"}}}, textMsg, 2); got != "Newer messages arrived, so this one can no longer be resent" {
		t.Fatalf("stale status en = %q", got)
	}

	i18n.SetLocale(i18n.LocaleZH)
	if got := messageEditStatus(session.ViewState{PromptBusy: true, Messages: []session.Message{textMsg}}, textMsg, 0); got != "等待当前回复结束后编辑" {
		t.Fatalf("running status zh = %q", got)
	}
	if got := messageEditStatus(session.ViewState{MessageEditBusy: true, Messages: []session.Message{textMsg}}, textMsg, 0); got != "输入框正忙，暂时无法编辑" {
		t.Fatalf("busy status zh = %q", got)
	}
	if got := messageEditStatus(session.ViewState{Messages: []session.Message{imgMsg}}, imgMsg, 0); got != "包含非文本内容，暂不支持编辑" {
		t.Fatalf("unsupported status zh = %q", got)
	}
	if got := messageEditStatus(session.ViewState{Messages: []session.Message{textMsg, {ID: "u3", Role: "user", Kind: session.MessageKindText, Seq: 6, Body: "later"}}}, textMsg, 2); got != "会话已有更新的消息，此条不能再重新发送" {
		t.Fatalf("stale status zh = %q", got)
	}
}

