package usage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestRankingErrorKeepsRetryActionReachable(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "synthetic page refusal", http.StatusServiceUnavailable)
			return
		}
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"sessions": []any{}, "hasMore": false}}})
	}))
	defer server.Close()
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	wait := func() {
		t.Helper()
		for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
			if !s.SessionRanking().Loading {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("synthetic ranking request did not settle")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		rankingView(c, Model{Store: s}, store.StatsView{Overview: &store.Overview{TopSessions: []store.SessionUsage{{ID: "fixture", Title: "Fixture session"}}}}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Text)
	}, 900, 1600)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	wait()
	tt.Frame()
	if s.SessionRanking().Err == "" || !tt.HasText("synthetic page refusal") {
		t.Fatal("More modal did not retain failure", tt.Texts())
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	wait()
	if calls.Load() != 2 || s.SessionRanking().Err != "" {
		t.Fatal("visible retry did not reload the same ranking page")
	}
}
