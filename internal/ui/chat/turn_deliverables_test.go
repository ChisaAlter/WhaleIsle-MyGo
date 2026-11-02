package chat

import (
	"context"
	"encoding/json"
	"errors"
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

func TestPresentedCardsPreviewAndExplicitHostActions(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)

	first := session.TurnFileTarget{Kind: "presented", Turn: 4, Seq: 12, Index: 0}
	second := session.TurnFileTarget{Kind: "presented", Turn: 4, Seq: 14, Index: 1}
	delivery := session.TurnDelivery{Turn: 4, Files: []session.TurnPresentedFile{
		{Target: first, Path: `C:\one\report.txt`, Description: "First report"},
		{Target: second, Path: `C:\two\report.txt`, Description: "Second report"},
	}}
	snapshot := session.TurnFilesSnapshot{Scope: session.TurnFileScope{Binding: "private-binding", SessionID: "private-session", Generation: 3}, HostLabel: "private-host", Results: map[session.TurnFileRequest]session.TurnFileResult{
		{Target: first, Action: "host"}:          {Host: &session.TurnPresentedHost{Name: "desktop", Available: true}},
		{Target: first, Action: "applications"}:  {Applications: []session.TurnFileApplication{{ID: "app-1", Name: "Editor"}}},
		{Target: second, Action: "applications"}: {Applications: []session.TurnFileApplication{{ID: "app-2", Name: "Editor"}}},
	}}
	var previews []session.TurnFileTarget
	var actions []session.TurnFileRequest
	preview := func(target session.TurnFileTarget) error {
		previews = append(previews, target)
		if target == second {
			return errors.New("Cannot preview this file")
		}
		return nil
	}
	run := func(target session.TurnFileTarget, action string) {
		actions = append(actions, session.TurnFileRequest{Target: target, Action: action})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		addressedTurnDeliveryView(c, delivery, snapshot, true, run, preview, ToolCardStyle{})
	}, 620, 450)
	for _, text := range tt.Texts() {
		for _, private := range []string{"private-binding", "private-session", "private-host", "event 12", "index 0", "Check Host desktop"} {
			if strings.Contains(text, private) {
				t.Fatalf("internal state is visible in %q", text)
			}
		}
	}
	if !tt.HasText("First report") || !tt.HasText("Second report") || !tt.HasText("report.txt") {
		t.Fatalf("card metadata missing: %q", tt.Texts())
	}
	if len(previews) != 0 || len(actions) != 0 {
		t.Fatal("rendering a card performed an action")
	}
	if err := tt.Click("First report"); err != nil {
		t.Fatal(err)
	}
	if len(previews) != 1 || previews[0] != first || len(actions) != 0 {
		t.Fatalf("first card did not preview its exact target: previews=%v actions=%v", previews, actions)
	}
	if err := tt.Click("Second report"); err != nil {
		t.Fatal(err)
	}
	if len(previews) != 2 || previews[1] != second || len(actions) != 0 || !tt.HasText("Cannot preview this file") {
		t.Fatalf("preview failure was not contained to the second card: previews=%v actions=%v texts=%q", previews, actions, tt.Texts())
	}
	for _, item := range []struct {
		card   string
		choice []string
		action string
	}{
		{"First report", []string{"Open on Host"}, "open"},
		{"Second report", []string{"Reveal on Host"}, "reveal"},
		{"Second report", []string{"Open with…", "Editor"}, session.TurnFileApplicationAction("app-2")},
	} {
		// Menus share a visible basename, so address the menu beside its own card.
		r, ok := tt.Find(item.card)
		if !ok {
			t.Fatalf("missing %q", item.card)
		}
		label := "Preview report.txt"
		firstCard, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing %q", label)
		}
		tt.ClickAt(firstCard.X+firstCard.W+20, r.Y+r.H/2)
		if err := tt.ChooseMenuItem(item.choice...); err != nil {
			t.Fatal(err)
		}
		last := actions[len(actions)-1]
		want := second
		if item.action == "open" {
			want = first
		}
		if last.Target != want || last.Action != item.action || len(previews) != 2 {
			t.Fatalf("wrong explicit action: got %#v, previews %v", last, previews)
		}
	}
}

func TestPresentedCardUnavailableAndActionError(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	target := session.TurnFileTarget{Kind: "presented", Turn: 1, Seq: 1, Index: 0}
	delivery := session.TurnDelivery{Turn: 1, Files: []session.TurnPresentedFile{{Target: target, Path: "demo.exe"}}}
	snapshot := session.TurnFilesSnapshot{Results: map[session.TurnFileRequest]session.TurnFileResult{
		{Target: target, Action: "host"}:                                     {Host: &session.TurnPresentedHost{Available: true}},
		{Target: target, Action: "applications"}:                             {},
		(session.TurnFileRequest{Target: target, Action: "open"}).StateKey(): {Action: "open", Error: "Host refused the open"},
	}}
	run := func(session.TurnFileTarget, string) { t.Fatal("action ran without user gesture") }
	tt := ui.NewTester(func(c *ui.Context) {
		addressedTurnDeliveryView(c, delivery, snapshot, true, run, func(session.TurnFileTarget) error { return errors.New("This file type cannot be previewed") }, ToolCardStyle{})
	}, 400, 180)
	if !tt.HasText("Host refused the open") {
		t.Fatalf("explicit action error is missing: %q", tt.Texts())
	}
	if err := tt.Click("Preview demo.exe"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("This file type cannot be previewed") || !tt.HasText("Host refused the open") {
		t.Fatalf("preview error lost: %q", tt.Texts())
	}
}

// A loopback fake follow stream exercises the real store's scope and coordinate
// gate. Neither callback contacts an actual desktop or touches an executable.
func TestDeliveryCardUsesLiveStoreAuthorityForPreview(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
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
			if frame.Type == dsh.FrameOpen && frame.Endpoint == "session/follow" {
				value := json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"deliverables/presented","seq":12,"data":{"turn":4,"callId":"p","files":[{"path":"a/report.txt"},{"path":"b/report.txt"}]} }},{"type":"event","event":{"type":"assistant/message","seq":13,"data":{"turn":4,"message":{"id":"done","content":[{"type":"text","text":"done"}]}}}},{"type":"event","event":{"type":"turn/end","seq":14,"data":{"turn":4}}}],"hasMore":false}`)
				if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: value}) != nil {
					return
				}
			}
		}
	}))
	defer server.Close()
	store := session.New()
	store.Select("history")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	store.Attach(client, nil)
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	store.AttachMux(mux)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	store.FollowSelected(ctx)
	var delivery session.TurnDelivery
	for {
		delivery = session.TurnDeliveries(store.Snapshot().Messages)[4]
		if len(delivery.Files) == 2 && store.Snapshot().TurnFiles.Scope.Binding != "" {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("fake history did not load both delivery targets")
		case <-time.After(time.Millisecond):
		}
	}
	// A fabricated card's displayed path must not become authority.
	delivery.Files[1].Path = "fabricated/report.txt"
	calls := []string{}
	generalOpens := 0
	m := Model{HistoryStore: store, Snapshot: store.Snapshot(), OnOpenDeliverable: func(string) { generalOpens++ }, OnPreviewDeliverable: func(path string) error { calls = append(calls, path); return nil }}
	// Prevent background metadata reads during the widget test.
	m.Snapshot.TurnFiles.Results = map[session.TurnFileRequest]session.TurnFileResult{
		{Target: delivery.Files[0].Target, Action: "host"}: {},
	}
	tt := ui.NewTester(func(c *ui.Context) { TurnDeliveryView(c, delivery, m, ToolCardStyle{}) }, 500, 250)
	first, ok := tt.Find("Preview report.txt")
	if !ok {
		t.Fatalf("missing preview card: %q", tt.Texts())
	}
	tt.ClickAt(first.X+first.W/2, first.Y+first.H/2+68)
	if len(calls) != 1 || calls[0] != "b/report.txt" || generalOpens != 0 {
		t.Fatalf("preview used display data or general opener: calls=%v opens=%d", calls, generalOpens)
	}
	store.Select("different-session")
	tt.ClickAt(first.X+first.W/2, first.Y+first.H/2+68)
	if len(calls) != 1 || generalOpens != 0 || !tt.HasText("no longer available") {
		t.Fatalf("stale scope remained actionable: calls=%v opens=%d texts=%q", calls, generalOpens, tt.Texts())
	}
}
