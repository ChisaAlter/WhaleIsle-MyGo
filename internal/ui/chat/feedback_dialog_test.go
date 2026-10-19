package chat

import (
	"context"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestFeedbackModalSourceGeometryAndWorkflow(t *testing.T) {
	var mode atomic.Int32
	var writes atomic.Int32
	started, release := make(chan struct{}, 1), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct {
				Args struct{ Request map[string]any }
			}
		}
		json.NewDecoder(r.Body).Decode(&q)
		value := any(map[string]any{"ok": true, "value": map[string]any{"items": []any{}}})
		switch r.URL.Path {
		case "/api/messageFeedback/list":
		case "/api/messageFeedback/put":
			writes.Add(1)
			req := q.Payload.Args.Request
			if req["sessionId"] != "fixture" || req["messageId"] != "answer" {
				t.Error("wrong modal target", req)
			}
			if mode.Load() == 2 {
				started <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
			}
			item := map[string]any{"messageId": "answer", "rating": req["rating"], "version": "version2"}
			if mode.Load() == 0 {
				value = map[string]any{"ok": false, "error": map[string]any{"code": "version-conflict", "current": item}}
			} else {
				value = map[string]any{"ok": true, "value": item}
			}
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	st := session.New()
	st.Select("fixture")
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	defer st.Detach()
	st.RefreshFeedback(context.Background())
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := st.Feedback()
			if !v.Loading && v.BusyID == "" {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("feedback fixture timed out")
	}
	wait()
	msg := session.Message{ID: "answer", Role: "assistant", Kind: session.MessageKindText, Seq: 2, Body: "A completed answer"}
	m := Model{FeedbackStore: st, Snapshot: session.ViewState{SelectedID: "fixture", Messages: []session.Message{msg}, ClientReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 700)
	up := FeedbackButtonLabel(session.MessageFeedbackUp, "")
	if err := tt.Click(up); err != nil {
		t.Fatal(err)
	}
	card, ok := tt.Find(feedbackText("title"))
	if !ok {
		t.Fatal("feedback modal missing")
	}
	detail, _ := tt.Find(feedbackText("detail"))
	submit, _ := tt.Find(feedbackText("submit"))
	closeButton, _ := tt.Find(feedbackText("close"))
	rectNear(t, card.W, 488, "source feedback panel width")
	rectNear(t, card.X, 156, "modal horizontal center")
	rectNear(t, card.Y, (700-card.H)/2, "modal vertical center")
	rectNear(t, detail.W, 440, "detail width")
	rectNear(t, detail.H, 116, "detail min-height")
	rectNear(t, detail.X-card.X, 24, "body side inset")
	rectNear(t, submit.H, 44, "source submit height")
	rectNear(t, submit.W, 440, "source full-width submit")
	rectNear(t, submit.Y-(detail.Y+detail.H), 38, "source content/footer gap")
	rectNear(t, card.Y+card.H-(submit.Y+submit.H), 24, "footer bottom inset")
	rectNear(t, closeButton.X+closeButton.W, card.X+card.W-14, "header trailing inset")
	if !tt.Focused(feedbackText("close")) {
		t.Fatal("Modal's first focus must be its close control")
	}
	tt.SetSize(400, 700)
	narrow, _ := tt.Find(feedbackText("title"))
	rectNear(t, narrow.W, 352, "narrow modal width")
	rectNear(t, narrow.X, 24, "narrow viewport clearance")
	tt.SetSize(800, 700)
	st.EditFeedback(strings.Repeat("A long detail line\n", 25), "")
	tt.Frame()
	longField, _ := tt.Find(feedbackText("detail"))
	rectNear(t, longField.H, 280, "detail max-height")
	st.EditFeedback("", "")
	tt.Frame()
	if dir := os.Getenv("FEEDBACK_DIALOG_SNAPSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "feedback-dialog.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	category := feedbackText("task-result")
	if err := tt.Click(category); err != nil {
		t.Fatal(err)
	}
	if st.Feedback().Category != "task-result" {
		t.Fatal("category not selected")
	}
	if err := tt.Click(category); err != nil {
		t.Fatal(err)
	}
	if st.Feedback().Category != "" {
		t.Fatal("same chip did not clear category")
	}
	if err := tt.Click(feedbackText("product-interaction")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(feedbackText("detail")); err != nil {
		t.Fatal(err)
	}
	tt.Type("Please keep this draft after a conflict")
	if st.Feedback().Note != "Please keep this draft after a conflict" {
		t.Fatal("detail callback lost text")
	}
	if err := tt.Click(feedbackText("submit")); err != nil {
		t.Fatal(err)
	}
	wait()
	tt.Frame()
	if v := st.Feedback(); v.EditorID != "answer" || v.Note == "" || v.Category != "product-interaction" || v.Error == "" {
		t.Fatalf("failure lost draft %+v", v)
	}
	if !tt.HasText(feedbackText("conflict")) {
		t.Fatal("source conflict failure notice unavailable")
	}
	tt.Key(0, ui.KeyEscape)
	if st.Feedback().EditorID != "" {
		t.Fatal("Escape did not dismiss")
	}
	if !tt.Focused(FeedbackButtonLabel(session.MessageFeedbackUp, session.MessageFeedbackUp)) {
		t.Fatal("Escape did not return focus to invoking rating")
	}
	mode.Store(1)
	if err := tt.Click(FeedbackButtonLabel(session.MessageFeedbackDown, "")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(feedbackText("submit")); err != nil {
		t.Fatal(err)
	}
	wait()
	tt.Frame()
	if st.Feedback().EditorID != "" || st.Feedback().Acknowledgement != 1 || !tt.HasText(feedbackText("recorded")) {
		t.Fatalf("success did not close/acknowledge %+v", st.Feedback())
	}
	activeDown := FeedbackButtonLabel(session.MessageFeedbackDown, session.MessageFeedbackDown)
	if err := tt.RightClick(activeDown); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("编辑反馈 / Edit feedback"); err != nil {
		t.Fatal(err)
	}
	if st.Feedback().EditorID != "answer" {
		t.Fatal("edit callback inaccessible")
	}
	if err := tt.Click(feedbackText("close")); err != nil {
		t.Fatal(err)
	}
	if st.Feedback().EditorID != "" {
		t.Fatal("close did not discard editor")
	}
	if err := tt.Click(up); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(10, 10)
	if st.Feedback().EditorID != "" {
		t.Fatal("mask did not dismiss")
	}
	if err := tt.Click(up); err != nil {
		t.Fatal(err)
	}
	mode.Store(2)
	if err := tt.Click(feedbackText("submit")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("submission did not begin")
	}
	tt.Frame()
	if !tt.HasText(feedbackText("submitting")) {
		t.Fatal("busy submit state unavailable")
	}
	beforeNote := st.Feedback().Note
	if err := tt.Click(feedbackText("detail")); err != nil {
		t.Fatal(err)
	}
	tt.Type("must not edit while submitting")
	if st.Feedback().Note != beforeNote {
		t.Fatal("submitting field accepted edits")
	}
	tt.Key(0, ui.KeyEscape)
	if v := st.Feedback(); v.EditorID != "" || v.BusyID == "" {
		t.Fatalf("busy dismiss canceled transport or kept draft %+v", v)
	}
	unblock()
	wait()
	tt.Frame()
	if st.Feedback().EditorID != "" || st.Feedback().Acknowledgement != 2 {
		t.Fatal("late success reopened dismissed draft or lost acknowledgement")
	}
	if writes.Load() != 3 {
		t.Fatalf("writes=%d", writes.Load())
	}
}

func TestZeroEditReturnSentinelCannotStealDraftFocus(t *testing.T) {
	msg := session.Message{ID: "legacy", Role: "user", Kind: session.MessageKindText, Body: "zero-seq fixture"}
	returnSeq := float64(0)
	draft := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).Grow(1).MinHeight(0).Children(func() {
				View(c, Model{Snapshot: session.ViewState{Messages: []session.Message{msg}}, OnEditMessage: func(float64, string) {}, EditReturnSeq: &returnSeq})
			})
			ui.TextInput(c.Key("draft"), &draft).Label("draft").AutoFocus()
		})
	}, 800, 500)
	if !tt.Focused("draft") {
		t.Fatal("zero return sentinel stole draft focus")
	}
	tt.Frame()
	if !tt.Focused("draft") {
		t.Fatal("zero return sentinel stole focus on rebuild")
	}
	if tt.HasText(i18n.T(i18n.KeyChatEditHint)) {
		t.Fatal("zero no-edit sentinel rendered editor chrome")
	}
}
