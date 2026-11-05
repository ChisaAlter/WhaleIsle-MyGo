package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestOrdinaryPreferenceRefusalSurvivesDescribeAndRetriesInPlace(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	var refuse, accepted atomic.Bool
	refuse.Store(true)
	accepted.Store(true)
	var writes atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	schema := json.RawMessage(`{"type":"object","dict":{"viewTabs":{"type":"boolean"}}}`)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://preference.fixture.invalid"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var request struct {
			Method  string `json:"method"`
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args map[string]any `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		var result any
		if request.Method == "settings/mutate" {
			if writes.Add(1) == 1 {
				close(started)
				<-release
			}
			if refuse.Load() {
				return nil, errors.New("synthetic ordinary switch refusal")
			}
			ops := request.Payload.Args["ops"].([]any)
			accepted.Store(ops[0].(map[string]any)["value"].(bool))
		}
		namespace := map[string]any{"ns": "ui-conversation", "revision": 7, "schema": schema, "value": map[string]any{"viewTabs": accepted.Load()}}
		switch request.Method {
		case "settings/describe":
			result = map[string]any{"writable": true, "namespaces": []any{namespace}}
		case "settings/mutate":
			result = namespace
		default:
			return nil, errors.New("unexpected fixture RPC: " + request.Method)
		}
		w := httptest.NewRecorder()
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": result}})
		return w.Result(), nil
	})}}
	s := store.New()
	s.Attach(client, nil)
	s.RefreshDescribe(context.Background())
	wait := func(done func() bool) {
		t.Helper()
		for end := time.Now().Add(2 * time.Second); time.Now().Before(end); {
			if done() {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("preference fixture did not settle", s.Snapshot())
	}
	wait(func() bool { return s.Snapshot().Described && !s.Snapshot().Loading })
	tt := ui.NewTester(func(c *ui.Context) {
		snapshot, owner, patch := s.BindDescribedPreferences()
		host := &BodySettingsAccess{Owner: owner, Snapshot: snapshot, OnChange: func(ns, key, value string) { patch(context.Background(), ns, map[string]string{key: value}) }}
		View(c, Model{Store: s, InterfaceActive: true, Interface: &InterfaceAccess{Host: host}, Conversation: &ConversationAccess{Host: host}})
	}, 1000, 900)
	const label = "Conversation and trajectory tabs"
	toggle := func() {
		t.Helper()
		for range 30 {
			if tt.Focused(label) {
				break
			}
			tt.Key(0, ui.KeyTab)
		}
		if !tt.Focused(label) {
			t.Fatal("ordinary Host switch was not focusable")
		}
		tt.Key(0, ui.KeySpace)
	}
	before, _ := tt.Find(label)
	toggle()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("ordinary write did not start")
	}
	tt.Frame()
	if snap := s.Snapshot(); !snap.PreferencePending["ui-conversation.viewTabs"] || snap.Saving {
		t.Fatal("ordinary write lost field-local pending", snap)
	}
	tt.Key(0, ui.KeySpace)
	if writes.Load() != 1 {
		t.Fatal("pending switch admitted another mutation")
	}
	releaseOnce.Do(func() { close(release) })
	wait(func() bool {
		snap := s.Snapshot()
		return !snap.Loading && snap.Described && snap.Err == "" && len(snap.PreferencePending) == 0 && snap.PreferenceErrors["ui-conversation.viewTabs"] != ""
	})
	_ = tt.Announcements()
	tt.Frame()
	if !tt.HasText("synthetic ordinary switch refusal") || !accepted.Load() {
		t.Fatal("describe swallowed refusal or failed value replaced accepted state", tt.Texts())
	}
	tt.SetTime(time.Now().Add(time.Second))
	saveBodySnapshot(t, tt, "ordinary-preference-refused.png")
	after, _ := tt.Find(label)
	if before != after {
		t.Fatalf("failure changed row layout: before=%+v after=%+v", before, after)
	}
	announcements := strings.Join(tt.Announcements(), "\n")
	if strings.Count(announcements, "synthetic ordinary switch refusal") != 1 {
		t.Fatal("refusal did not announce exactly once", announcements)
	}
	for range 3 {
		tt.Frame()
	}
	if strings.Contains(strings.Join(tt.Announcements(), "\n"), "synthetic ordinary switch refusal") {
		t.Fatal("refusal was reissued every frame")
	}
	toggle() // An explicit retry may return the identical refusal before a paint.
	wait(func() bool {
		snap := s.Snapshot()
		return writes.Load() == 2 && !snap.Loading && snap.Described && snap.Err == "" && len(snap.PreferencePending) == 0 && snap.PreferenceErrors["ui-conversation.viewTabs"] != ""
	})
	tt.Frame()
	if !tt.HasText("synthetic ordinary switch refusal") || strings.Count(strings.Join(tt.Announcements(), "\n"), "synthetic ordinary switch refusal") != 1 {
		t.Fatal("an explicit retry with the same refusal was silently deduplicated")
	}
	refuse.Store(false)
	toggle()
	wait(func() bool {
		snap := s.Snapshot()
		return len(snap.PreferencePending) == 0 && snap.PreferenceErrors["ui-conversation.viewTabs"] == "" && writes.Load() == 3
	})
	tt.Frame()
	if accepted.Load() || tt.HasText("synthetic ordinary switch refusal") || writes.Load() != 3 {
		t.Fatal("explicit retry did not accept only the current field and clear its refusal")
	}
}

func TestOrdinaryPreferenceErrorsBelongToVisibleFieldsAndHost(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	h := bodyFixtureHost()
	h.Snapshot.PreferenceErrors = map[string]string{"ui-conversation.viewTabs": "owned interface refusal", "ui-chat.transcriptView": "other page refusal"}
	h.Snapshot.Err = "other page refusal" // The global slot can briefly retain another queue's error.
	page, open := SectionInterface, true
	tt := ui.NewTester(func(c *ui.Context) {
		if !open {
			ClearPreferenceErrorNotices(c)
			ui.Text(c, "Workspace")
			return
		}
		ModalView(c, Model{Store: store.New(), InterfaceActive: page == SectionInterface, ConversationActive: page == SectionConversation, Interface: &InterfaceAccess{Host: h}, Conversation: &ConversationAccess{Host: h}, OnClose: func() { open = false }})
	}, 1200, 900)
	if !tt.HasText("owned interface refusal") || tt.HasText("other page refusal") {
		t.Fatal("interface displayed another page's field error", tt.Texts())
	}
	page = SectionConversation
	tt.Frame()
	if tt.HasText("owned interface refusal") || !tt.HasText("other page refusal") {
		t.Fatal("page switch retained an unmounted field's toast", tt.Texts())
	}
	h.Owner++
	h.Snapshot.PreferenceErrors = nil
	h.Snapshot.Err = ""
	tt.Frame()
	if tt.HasText("other page refusal") {
		t.Fatal("old Host field toast survived replacement")
	}
	h.Snapshot.PreferenceErrors = map[string]string{"ui-chat.transcriptView": "new Host refusal"}
	tt.Frame()
	if !tt.HasText("new Host refusal") {
		t.Fatal("replacement Host could not present its own refusal")
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsBack)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Workspace") || tt.HasText("new Host refusal") {
		t.Fatal("closing settings left its field refusal in the workspace")
	}
	open = true
	tt.Frame()
	if !tt.HasText("new Host refusal") {
		t.Fatal("reopened consumer did not read its retained failure")
	}
	open = false // The shell shortcut/deeplink boundary bypasses OnClose.
	tt.Frame()
	if tt.HasText("new Host refusal") {
		t.Fatal("outer shell cleanup left an old field refusal")
	}
}

func TestUsagePreferenceRefusalsDoNotReadOtherFields(t *testing.T) {
	h := bodyFixtureHost()
	h.Snapshot.PreferenceErrors = map[string]string{"ui-conversation.sessionCost": "usage field refusal", "ui-conversation.viewTabs": "unrelated interface refusal"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { UsagePreferences(c, h, c.Theme().Text, c.Theme().TextMuted) })
	}, 800, 600)
	if !tt.HasText("usage field refusal") || tt.HasText("unrelated interface refusal") {
		t.Fatal("usage preferences aggregated unrelated field failures")
	}
	h.Owner++
	h.Snapshot.PreferenceErrors = nil
	tt.Frame()
	if tt.HasText("usage field refusal") {
		t.Fatal("usage retained the old Host's refusal")
	}
}

func TestSpecialPreferenceRowsOwnTheirOrdinaryFieldErrors(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	for _, field := range []string{"permission.defaultPreset", "ui-conversation.typingFx"} {
		t.Run(field, func(t *testing.T) {
			h := bodyFixtureHost()
			h.Snapshot.Err = "unrelated interface refusal"
			h.Snapshot.PreferenceErrors = map[string]string{field: "owned special-row refusal", "ui-conversation.viewTabs": h.Snapshot.Err}
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: store.New()}
				if field == "permission.defaultPreset" {
					m.ConversationActive = true
					m.Conversation = &ConversationAccess{Host: h}
				} else {
					m.AppearanceActive = true
					m.Appearance = &AppearanceAccess{Host: h}
				}
				View(c, m)
			}, 1000, 1000)
			if !tt.HasText("owned special-row refusal") || tt.HasText("unrelated interface refusal") {
				t.Fatal("special row failed to own its ordinary field refusal", tt.Texts())
			}
		})
	}
}
