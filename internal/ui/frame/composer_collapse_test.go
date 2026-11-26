package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func seedLongConversationStore(t *testing.T, count int) *session.Store {
	t.Helper()
	var events strings.Builder
	events.WriteString(`{"type":"snapshot","records":[`)
	for i := 1; i <= count; i++ {
		if i > 1 {
			events.WriteByte(',')
		}
		if i%2 == 1 {
			fmt.Fprintf(&events, `{"type":"event","event":{"seq":%d,"time":%d,"type":"user/message","data":{"turn":%d,"content":[{"type":"text","text":"User turn %d message body with enough text to occupy vertical space in the transcript."}]}}}`, i, 1700000000000+i, (i+1)/2, i)
		} else {
			fmt.Fprintf(&events, `{"type":"event","event":{"seq":%d,"time":%d,"type":"assistant/message","data":{"turn":%d,"step":1,"message":{"id":"msg-%d","content":[{"type":"text","text":"Assistant response %d with multiple sentences so the conversation viewport overflows and requires scrolling to read earlier turns."}]}}}}`, i, 1700000000000+i, i/2, i, i)
		}
	}
	events.WriteString(`]}`)
	sessionBaseline := json.RawMessage(events.String())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.CloseNow()
			for {
				var frame dsh.ClientFrame
				if wsjson.Read(r.Context(), conn, &frame) != nil {
					return
				}
				if frame.Type != dsh.FrameOpen {
					continue
				}
				if frame.Endpoint == "session/follow" {
					_ = wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: sessionBaseline})
				}
			}
		}
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/session/list" {
			_, _ = w.Write([]byte(`{"type":"server-response","rpcId":"` + req.RPCID + `","result":{"ok":true,"value":{"items":[{"sessionId":"long-sess","cwd":"/host/a","title":"Long Conversation","blank":false,"running":false,"updatedAt":200}]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"type":"server-response","rpcId":"` + req.RPCID + `","result":{"ok":true,"value":{}}}`))
	}))
	store := session.New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		store.Detach()
		_ = mux.Close()
		server.Close()
	})
	store.Select("long-sess")
	store.Attach(client, nil)
	store.AttachMux(mux)
	store.RefreshList(ctx)
	store.FollowSelected(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.Published().Value.Messages) >= count {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if got := len(store.Published().Value.Messages); got < count {
		t.Fatalf("expected >=%d messages loaded, got %d", count, got)
	}
	return store
}

func TestComposerScrollCollapse(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)

	store := seedLongConversationStore(t, 40)
	layout := DefaultLayout()
	draft := ""
	chatList := &ui.ListState{FollowEnd: true}
	m := Model{
		Store:    store,
		Sizes:    &layout,
		ChatList: chatList,
		Draft:    &draft,
		ThemeID:  "deepseek.light",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 1100, 720)
	tt.Frame()

	// Initially at bottom: composer must be expanded.
	expandedCard, ok := tt.Find("Composer card")
	if !ok {
		t.Fatalf("Composer card not found at bottom: %v", tt.Texts())
	}

	// Scroll up well past 64 DIP in the transcript area.
	body, ok := tt.Find("Conversation body")
	if !ok {
		t.Fatal("conversation body not found")
	}
	cx, cy := body.X+body.W/2, body.Y+body.H/3
	for i := 0; i < 6; i++ {
		tt.Scroll(cx, cy, 0, -120)
	}
	tt.Frame()

	collapsedCard, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("Composer card missing after scroll up")
	}
	if collapsedCard.H >= expandedCard.H-10 {
		t.Fatalf("scrolling up >64 DIP did not collapse composer: expandedH=%.1f collapsedH=%.1f atEnd=%v", expandedCard.H, collapsedCard.H, chatList.AtEnd())
	}

	// Clicking the collapsed capsule expands the composer again.
	if err := tt.Click("Composer card"); err != nil {
		t.Fatalf("clicking collapsed composer capsule failed: %v", err)
	}
	tt.Frame()
	reexpandedCard, ok := tt.Find("Composer card")
	if !ok || reexpandedCard.H <= collapsedCard.H+10 {
		t.Fatalf("composer stayed collapsed after clicking capsule: collapsedH=%.1f afterClickH=%.1f", collapsedCard.H, reexpandedCard.H)
	}

	// Scroll back to bottom, then up again, then scroll back to bottom to verify auto-expand within 24 DIP.
	chatList.ScrollToEnd()
	tt.Frame()
	for i := 0; i < 6; i++ {
		tt.Scroll(cx, cy, 0, -120)
	}
	tt.Frame()
	collapsedAgain, _ := tt.Find("Composer card")
	if collapsedAgain.H >= expandedCard.H-10 {
		t.Fatalf("second scroll up did not re-collapse composer: expandedH=%.1f collapsedH=%.1f", expandedCard.H, collapsedAgain.H)
	}
	for i := 0; i < 12; i++ {
		tt.Scroll(cx, cy, 0, 180)
	}
	tt.Frame()
	finalCard, _ := tt.Find("Composer card")
	if finalCard.H <= collapsedAgain.H+10 {
		t.Fatalf("scrolling back to bottom did not expand composer: collapsedH=%.1f finalH=%.1f", collapsedAgain.H, finalCard.H)
	}
}
