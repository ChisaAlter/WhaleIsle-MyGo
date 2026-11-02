package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Entering the settings panel used to call openSettingsDocument before describe.
// Capture the real navigation callback's RPCs so a mocked Host catches that
// external-editor side effect without opening anything on the test machine.
func TestSettingsNavigationLoadsWithoutOpeningDocument(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	var mu sync.Mutex
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		requests = append(requests, r.URL.Path)
		mu.Unlock()
		value := any(map[string]any{})
		if r.URL.Path == "/api/settings/describe" {
			value = map[string]any{"writable": true, "namespaces": []any{}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": envelope.RPCID,
			"result": map[string]any{"ok": true, "value": value},
		})
	}))
	defer srv.Close()

	store := settingsstore.New()
	settled := make(chan struct{}, 8)
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if !store.Snapshot().Loading {
			settled <- struct{}{}
		}
	})
	sessions := session.New()
	s := &mainState{settings: store, sessions: sessions}
	sizes := frame.DefaultLayout()
	search, renameID, renameDraft := "", "", ""
	tt := ui.NewTester(func(c *ui.Context) {
		s.withMainNavigation(c, true, frame.Model{
			Store: sessions, Sizes: &sizes, ThemeID: theme.IDDeepseekLight,
			SessionSearch: &search, RenameSessionID: &renameID, RenameDraft: &renameDraft,
		}, func() { ui.Text(c, "Preserved conversation") })
	}, 1200, 800)
	for range 3 {
		tt.Frame()
	}
	mu.Lock()
	idleCalls := len(requests)
	mu.Unlock()
	if idleCalls != 0 {
		t.Fatalf("idle navigation made %d Host requests", idleCalls)
	}
	for visit := 0; visit < 3; visit++ {
		s.settingsOpen = false
		if err := tt.Click(i18n.T(i18n.KeySettingsAccountSeat)); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(i18n.T(i18n.KeySettingsAccountTitle)); err != nil {
			t.Fatalf("%v; menu=%q, texts=%q", err, tt.Menu(), tt.Texts())
		}
		select {
		case <-settled:
		case <-time.After(2 * time.Second):
			t.Fatal("settings navigation did not finish loading")
		}
		if !s.settingsOpen || !s.settingsAccountActive || s.settingsDesktopActive || store.Snapshot().Err != "" {
			t.Fatal("account navigation did not open the account page and load its data")
		}
		mu.Lock()
		got := append([]string(nil), requests...)
		mu.Unlock()
		if len(got) != visit+1 {
			t.Fatalf("visit %d made unexpected Host requests: %v", visit, got)
		}
		for _, method := range got {
			if method != "/api/settings/describe" {
				t.Fatalf("opening the panel launched an action instead of reading settings: %s", method)
			}
		}
	}
}
