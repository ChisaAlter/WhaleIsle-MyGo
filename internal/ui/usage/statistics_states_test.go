package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestEmbeddedUsageInitialLoadingErrorAndEmptyStates(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	for _, state := range []string{"loading", "error", "empty"} {
		t.Run(state, func(t *testing.T) {
			s := usageFixture(t, state)
			tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true}) }) }, 680, 1000)
			text := strings.Join(tt.Texts(), "\n")
			if state == "loading" && !strings.Contains(text, "Loading usage") {
				t.Fatal(text)
			}
			if state == "error" && !strings.Contains(text, s.Stats().Err) {
				t.Fatal("initial error hidden", text)
			}
			if state == "empty" && !strings.Contains(text, "No usage in this range.") {
				t.Fatal(text)
			}
			if strings.Contains(text, "Date range (UTC") || strings.Contains(text, "Export json") {
				t.Fatal("empty/loading/error rendered data-only controls", text)
			}
		})
	}
}

func TestEmbeddedUsageCalendarSelectionKeepsOtherDimensionsAndExports(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}}) })
	}, 850, 3600)
	if !tt.HasText("Model share (all time)") || !tt.HasText("Daily model breakdown (recent)") || !tt.HasText("Session ranking") || !tt.HasText("Export") {
		t.Fatal("all-time/recent dimensions missing:" + "\n" + strings.Join(tt.Texts(), "\n"))
	}
	before, err := s.ExportStats("json")
	if err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("2026-10-14 UTC"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !s.Stats().Range.Active() || !tt.HasText("Model share (all time)") || !tt.HasText("Daily model breakdown (recent)") || !tt.HasText("Session ranking") {
		t.Fatal("calendar selection changed independent dashboard dimensions")
	}
	s.SetStatsRangeDraft("2026-10-15", "2026-10-14")
	s.ApplyStatsRangeDraft()
	tt.Frame()
	if s.Stats().RangeError == "" || !s.Stats().Range.Active() {
		t.Fatal("invalid range silently replaced accepted selection")
	}
	if err := tt.Click("Last 7 days"); err != nil {
		t.Fatal(err)
	}
	if s.Stats().Days != 7 || !s.Stats().Range.Active() {
		t.Fatal("recent chart window replaced the calendar selection")
	}
	after, err := s.ExportStats("json")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("calendar/chart controls changed JSON export", err)
	}
	if err := tt.Click("Clear selection"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.Stats().Range.Active() || !tt.HasText("Model share (all time)") || !tt.HasText("Export") {
		t.Fatal("calendar selection was not cleared independently")
	}
}

func TestUsageKPIAndModelBreakdownParityZHAndEN(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)

	// 1. Unpriced model fixture: verify "No priced models yet" / "暂无已定价模型"
	unpricedStore := usageFixture(t, "data")
	i18n.SetLocale(i18n.LocaleEN)
	ttUnpricedEN := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: unpricedStore, Embedded: true}) })
	}, 850, 3600)
	if !ttUnpricedEN.HasText("No priced models yet") {
		t.Fatalf("missing unpriced KPI detail in EN: %v", ttUnpricedEN.Texts())
	}
	i18n.SetLocale(i18n.LocaleZH)
	ttUnpricedZH := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: unpricedStore, Embedded: true}) })
	}, 850, 3600)
	if !ttUnpricedZH.HasText("暂无已定价模型") {
		t.Fatalf("missing unpriced KPI detail in ZH: %v", ttUnpricedZH.Texts())
	}

	// 2. Priced model with peak/valley PhaseBuckets: verify KPI peak/off-peak cost & model breakdown lines
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/usage-stats/billing.get":
			value = map[string]any{"prices": map[string]any{}, "peakValleyEnabled": true}
		case "/usage-stats/billing.models":
			value = map[string]any{"options": []any{}}
		case "/usage-stats/projectRank":
			value = map[string]any{"projects": []any{}, "hasMore": false}
		case "/usage-stats/overview":
			totals := usagestore.Totals{Input: 100000, Output: 50000, CacheRead: 200000, CacheWrite: 10000, Total: 360000}
			costBuckets := &usagestore.PhaseBuckets{
				Peak:    &usagestore.Totals{CacheRead: 120000, Input: 50000, CacheWrite: 10000, Output: 30000, Total: 210000},
				OffPeak: &usagestore.Totals{CacheRead: 80000, Input: 50000, CacheWrite: 0, Output: 20000, Total: 150000},
			}
			models := []usagestore.ModelUsage{{
				Model:    "deepseek-v4-flash",
				Provider: "deepseek",
				Totals:   totals,
				Cost:     costBuckets,
			}}
			o := usagestore.Overview{
				Totals:       totals,
				SessionCount: 2,
				Days: []usagestore.DayUsage{{
					Date:       "2026-10-14",
					Total:      360000,
					Models:     map[string]usagestore.Totals{"deepseek-v4-flash": totals},
					ModelCosts: map[string]*usagestore.PhaseBuckets{"deepseek-v4-flash": costBuckets},
				}},
				ByModel:     models,
				TopSessions: []usagestore.SessionUsage{},
				Providers:   []usagestore.ProviderUsage{},
				UpdatedAt:   1791970000000,
			}
			o.AllTime.Totals = totals
			o.AllTime.SessionCount = 2
			o.AllTime.ByModel = models
			o.Coverage.Mode = "projection"
			o.Coverage.Timezone = "UTC"
			o.Coverage.SessionsTotal = 2
			o.Coverage.SessionsOk = 2
			o.Coverage.UsageSessionsMain = 1
			o.Coverage.UsageSessionsSubagent = 1
			value = o
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	})
	s := usagestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: usageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}}, nil)
	s.RefreshStats(context.Background(), false)
	for deadline := time.Now().Add(2 * time.Second); s.Stats().Loading && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}

	i18n.SetLocale(i18n.LocaleZH)
	ttZH := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Store:    s,
			Embedded: false,
			Preferences: func(c *ui.Context, fg, muted ui.Color) {
				ui.Text(c, "首选项挂钩行").TextColor(fg)
			},
		})
	}, 900, 3600)
	zhJoined := strings.Join(ttZH.Texts(), "\n")
	for _, want := range []string{
		"首选项挂钩行",
		"输入 310000 · 输出 50000 · 缓存读取 200000",
		"峰段 ¥",
		"谷段 ¥",
		"缓存读取 200000 · 缓存写入 10000",
		"输入 100000 · 输出 50000 · 缓存读取 200000 · 缓存写入 10000",
		"高峰 命中 120000 · 未命中 60000 · 输出 30000",
		"空闲 命中 80000 · 未命中 50000 · 输出 20000",
	} {
		if !strings.Contains(zhJoined, want) {
			t.Fatalf("zh usage view missing %q in:\n%s", want, zhJoined)
		}
	}

	i18n.SetLocale(i18n.LocaleEN)
	ttEN := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: s, Embedded: true})
	}, 900, 3600)
	enJoined := strings.Join(ttEN.Texts(), "\n")
	for _, want := range []string{
		"Input 310000 · Output 50000 · Cache read 200000",
		"Peak ¥",
		"Off-peak ¥",
		"Cache read 200000 · Cache write 10000",
		"Input 100000 · Output 50000 · Cache read 200000 · Cache write 10000",
		"Peak hit 120000 · miss 60000 · output 30000",
		"Off-peak hit 80000 · miss 50000 · output 20000",
	} {
		if !strings.Contains(enJoined, want) {
			t.Fatalf("en usage view missing %q in:\n%s", want, enJoined)
		}
	}
}
