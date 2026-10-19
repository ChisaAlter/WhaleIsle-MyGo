package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type enterFrameTransport struct{ calls atomic.Int32 }

func (f *enterFrameTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	var envelope struct {
		Type, Method string
		RPCID        string `json:"rpcId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Type != "client-request" || envelope.Method != "session/list" || envelope.RPCID == "" {
		return nil, fmt.Errorf("first-send layout fixture rejected Host/model request %s", envelope.Method)
	}
	cwd := "C:/fixture-workspace"
	value := api.SessionListValue{Items: []api.SessionListValueItemsItem{{SessionId: "first-send", Cwd: &cwd, Blank: true, AgentAvailable: true}}}
	body, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func enterFrameStore(t *testing.T) (*session.Store, *enterFrameTransport) {
	t.Helper()
	store := session.New()
	store.Select("first-send") // Selected before Attach: no automatic follow.
	f := &enterFrameTransport{}
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://first-send.fixture.invalid"}, HTTP: &http.Client{Transport: f}}, nil)
	t.Cleanup(store.Detach)
	store.RefreshList(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snap := store.Snapshot()
		if !snap.ListLoading && snap.ListErr == "" && snap.SelectedSummaryBlank && snap.SelectedCwd != "" {
			return store, f
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("accepted blank Session roster did not settle: %+v", store.Snapshot())
	return nil, nil
}

func TestConversationFirstSubmissionPinsFirstFrameAndRetainsNativeDraft(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, reduced := range []bool{false, true} {
		t.Run(fmt.Sprintf("reduced-%t", reduced), func(t *testing.T) {
			store, transport := enterFrameStore(t)
			sizes := DefaultLayout()
			sizes.Bottom = 0
			list := ui.ListState{}
			draft := "ABCDE"
			draftState := composer.DraftState{}
			client := composer.NewDraftClient(&draft)
			var tt *ui.Tester
			var firstActive ui.Rect
			capturedFirst := false
			tt = ui.NewTester(func(c *ui.Context) {
				originalScrollbarTheme(c)
				sizes.ResolveColumns(1200, false)
				colors := theme.Resolve("deepseek.light").Light
				snap := store.Snapshot()
				root := ui.Column(c).Fill()
				// The headless tester may settle many animation frames. Capture
				// the first committed active frame from its paint, rather than
				// mistaking Frame's final sample for the first-send position.
				root.DrawOver(func(_ *ui.Painter, _ ui.Rect) {
					if tt != nil && snap.PromptAttempted && !capturedFirst {
						firstActive, capturedFirst = tt.Find("Composer card")
					}
				})
				root.Children(func() {
					mainRow(c, Model{Store: store, Sizes: &sizes, Draft: &draft, ComposerState: &draftState, ChatList: &list,
						UseComposerTextInput: true, ComposerTextInput: client}, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, 1200, 800)
			if reduced {
				tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			}
			before, ok := tt.Find("Composer card")
			if !ok || !tt.HasText(i18n.T(i18n.KeyChatHeroSlogan)) || store.Snapshot().PromptAttempted {
				t.Fatal("fixture did not begin at the real blank Session Hero")
			}
			if err := tt.Click("Composer draft"); err != nil {
				t.Fatal(err)
			}
			client.SetSelectionUTF16(1, 4)
			tt.Frame()
			retainedNative := tt.NativeTextInputClient()
			if retainedNative == nil {
				t.Fatal("Hero did not own a native input adapter")
			}
			target, err := store.CaptureComposerTarget("first-send")
			if err != nil {
				t.Fatal(err)
			}
			// This is the production UI's local first-admission/echo method.
			// Deliberately do not call Submit: no model/Host prompt is sent.
			if request := target.BeginSubmission("first fixture submission", nil); request == "" {
				t.Fatal("real Store rejected the first local admission")
			}
			if snap := store.Snapshot(); !snap.PromptAttempted || len(snap.Messages) == 0 {
				t.Fatal("first-admission fixture did not mark the real Store field")
			}
			tt.Frame()
			if !capturedFirst {
				t.Fatal("no committed first active card frame was observed")
			}
			if !reduced && (math.Abs(float64(firstActive.Y-before.Y)) > .5 || firstActive.X != before.X || firstActive.W != before.W) {
				t.Fatalf("first active paint jumped before the hold: Hero=%+v first=%+v", before, firstActive)
			}
			// At most 450ms hold + 300ms glide. Frames continue to use the
			// real source clock; bound the wait and require a settled sample.
			deadline := time.Now().Add(1200 * time.Millisecond)
			var final ui.Rect
			for time.Now().Before(deadline) {
				tt.Frame()
				final, ok = tt.Find("Composer card")
				if ok && final.Y > before.Y+32 && time.Now().After(deadline.Add(-400*time.Millisecond)) {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !ok || final.Y <= before.Y+32 || final.X != before.X || final.W != before.W {
				t.Fatalf("composer did not settle into its active dock: Hero=%+v final=%+v", before, final)
			}
			tt.Frame()
			settled, _ := tt.Find("Composer card")
			if math.Abs(float64(settled.Y-final.Y)) > .5 {
				t.Fatalf("finite glide kept moving after its deadline: %+v -> %+v", final, settled)
			}
			if reduced && math.Abs(float64(firstActive.Y-final.Y)) > .5 {
				t.Fatal("reduced motion held or animated the first active frame", firstActive, final)
			}
			if !tt.Focused("Composer draft") || client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
				t.Fatal("first-send replaced the retained draft focus/selection", client.Selection())
			}
			// Use the pre-transition native adapter, not a newly acquired one.
			// A disposed/replaced editable binding must reject this mutation.
			retainedNative.ReplaceText(nil, "Z")
			tt.Frame()
			if draft != "AZE" {
				t.Fatal("the original native draft binding did not survive first-send", draft)
			}
			if transport.calls.Load() != 1 {
				t.Fatalf("layout/first-admission performed extra Host/model requests: %d", transport.calls.Load())
			}
		})
	}
}
