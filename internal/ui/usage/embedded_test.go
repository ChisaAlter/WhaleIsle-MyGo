package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func usageFixture(t *testing.T, state string) *usagestore.Store {
	t.Helper()
	gate := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage-stats/overview" {
			http.NotFound(w, r)
			return
		}
		if state == "loading" {
			<-gate
		}
		if state == "error" {
			http.Error(w, "fixture service unavailable", 503)
			return
		}
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		o := usagestore.Overview{Days: []usagestore.DayUsage{}, ByModel: []usagestore.ModelUsage{}, TopSessions: []usagestore.SessionUsage{}, Providers: []usagestore.ProviderUsage{}, UpdatedAt: 1791970000000}
		o.AllTime.ByModel = []usagestore.ModelUsage{}
		o.Coverage.Mode = "projection"
		o.Coverage.Timezone = "UTC"
		o.Coverage.FailedSessionIDs = []string{}
		if state == "data" {
			totals := usagestore.Totals{Input: 12000, Output: 6000, CacheRead: 2000, Total: 20000}
			models := []usagestore.ModelUsage{{Model: "fixture-model-with-long-metadata", Provider: "fixture", Totals: totals}}
			o.Totals = totals
			o.SessionCount = 3
			o.AllTime.Totals = totals
			o.AllTime.SessionCount = 3
			o.AllTime.ByModel = models
			o.ByModel = models
			o.Days = []usagestore.DayUsage{{Date: "2026-10-14", Total: 20000, Models: map[string]usagestore.Totals{models[0].Model: totals}}}
			o.Coverage.SessionsTotal = 3
			o.Coverage.SessionsOk = 3
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": o}})
	})

	t.Cleanup(func() { close(gate) })
	s := usagestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: usageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}}, nil)
	s.RefreshStats(context.Background(), false)
	if state != "loading" {
		deadline := time.Now().Add(3 * time.Second)
		for s.Stats().Loading && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if s.Stats().Loading {
			t.Fatal("fixture usage timed out")
		}
	}
	return s
}
func TestEmbeddedUsageHasOnlyStatistics(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true}) }) }, 780, 900)
	text := strings.Join(tt.Texts(), "\n")
	for _, forbidden := range []string{"Account & login", "Recent activity", "Kill job", "Top up"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unrelated settings body %q", forbidden)
		}
	}
	if !strings.Contains(text, "Total tokens") {
		t.Fatal(text)
	}
}
func TestUsageBodySoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set SECONDARY_SNAPSHOT_DIR for body renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, state := range []string{"data", "empty", "loading", "error", "no-permission"} {
		m := Model{Embedded: true}
		if state != "no-permission" {
			m.Store = usageFixture(t, state)
		}
		for _, width := range []int{360, 780} {
			tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Padding(20).Children(func() { View(c, m) }) }, width, 760)
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("settings-usage-%s-%d.png", state, width)))
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
	}
}

type usageFixtureTransport func(*http.Request) (*http.Response, error)

func (f usageFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
