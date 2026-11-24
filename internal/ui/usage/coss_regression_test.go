package usage

import (
	"context"
	"encoding/json"
	"image/color"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func assertUsageCursors(t *testing.T, viewName string, tt *ui.Tester) {
	t.Helper()
	for _, b := range tt.Interactives() {
		// Dialog containers carry RoleDialog; child controls are audited individually.
		if b.Role == ui.RoleDialog {
			continue
		}
		if slices.Contains(b.Kinds, "text") {
			if b.Cursor != ui.CursorText && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] text input %s has unexpected cursor %v", viewName, b.String(), b.Cursor)
			}
			continue
		}
		if slices.Contains(b.Kinds, "click") {
			if b.Cursor != ui.CursorPointer && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] clickable %s has unexpected cursor %v", viewName, b.String(), b.Cursor)
			}
		}
	}
}

func TestUsageCursorAudit(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}})
		})
	}, 1000, 900)
	tt.Frame()
	assertUsageCursors(t, "embedded-data", tt)

	if err := tt.Click("Export"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	assertUsageCursors(t, "export-menu", tt)
	tt.Key(0, ui.KeyEscape)

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); (s.Billing().Busy || s.BillingModels().Loading) && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	tt.SetTime(time.Now().Add(time.Second))
	tt.Frame()
	assertUsageCursors(t, "billing-dialog", tt)
}

func TestUsageKPIMemoInvalidatesOnTokensThemeLocaleAndStore(t *testing.T) {
	prevLocale := i18n.Locale()
	prevPrefs := theme.CurrentPreferences()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		theme.SetPreferences(prevPrefs)
		design.SetGlass(false, nil)
	})

	i18n.SetLocale(i18n.LocaleEN)
	s1 := usageFixture(t, "data")
	curStore := s1

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: curStore, Embedded: true})
		})
	}, 900, 900)
	tt.Frame()

	labelRect, ok := tt.Find("Total tokens")
	if !ok {
		t.Fatalf("KPI label missing: %v", tt.Texts())
	}
	sampleX, sampleY := int(labelRect.X), int(labelRect.Y+labelRect.H+2)
	before := color.RGBAModel.Convert(tt.Image().At(sampleX, sampleY)).(color.RGBA)

	// 1. design.Revision() change (glass token override) invalidates Memo.
	magenta := ui.RGB(255, 0, 255)
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Card = magenta
	})
	tt.Frame()
	afterGlass := color.RGBAModel.Convert(tt.Image().At(sampleX, sampleY)).(color.RGBA)
	if afterGlass == before || afterGlass.R != 255 || afterGlass.G != 0 || afterGlass.B != 255 {
		t.Fatalf("KPI Memo ignored design.SetGlass Card change: before=%v after=%v", before, afterGlass)
	}
	design.SetGlass(false, nil)
	tt.Frame()

	// 2. Locale change invalidates Memo.
	i18n.SetLocale(i18n.LocaleZH)
	tt.Frame()
	if !tt.HasText("总 Tokens") {
		t.Fatalf("KPI Memo stayed stale after locale switch EN->ZH: %v", tt.Texts())
	}
	i18n.SetLocale(i18n.LocaleEN)
	tt.Frame()

	// 3. theme.Revision() change invalidates Memo.
	p := theme.Defaults()
	p.Interface.Size = 20
	theme.SetPreferences(p)
	tt.Frame()
	if !tt.HasText("Total tokens") {
		t.Fatalf("KPI Memo failed after theme.SetPreferences: %v", tt.Texts())
	}

	// 4. Store swap invalidates Memo via StatisticsOwner().
	s2 := usageFixtureWithTotal(t, 98765)
	curStore = s2
	tt.Frame()
	if !tt.HasText("98765") {
		t.Fatalf("KPI Memo stayed stale after store swap: %v", tt.Texts())
	}
}

func usageFixtureWithTotal(t *testing.T, total float64) *usagestore.Store {
	t.Helper()
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
			totals := usagestore.Totals{Input: total * 0.6, Output: total * 0.3, CacheRead: total * 0.1, Total: total}
			models := []usagestore.ModelUsage{{Model: "model-b", Provider: "fixture", Totals: totals}}
			var o usagestore.Overview
			o.Totals = totals
			o.SessionCount = 1
			o.AllTime.Totals = totals
			o.AllTime.SessionCount = 1
			o.AllTime.ByModel = models
			o.ByModel = models
			o.Days = []usagestore.DayUsage{{Date: "2026-10-14", Total: total, Models: map[string]usagestore.Totals{"model-b": totals}}}
			o.TopSessions = []usagestore.SessionUsage{}
			o.Providers = []usagestore.ProviderUsage{}
			o.UpdatedAt = 1791970000000
			o.Coverage.Mode = "projection"
			o.Coverage.Timezone = "UTC"
			o.Coverage.FailedSessionIDs = []string{}
			value = o
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	})
	s := usagestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: usageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})}}, nil)
	s.RefreshStats(context.Background(), false)
	for deadline := time.Now().Add(2 * time.Second); s.Stats().Loading && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	return s
}

func TestUsageLightAndDarkSchemes(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	s := usageFixture(t, "data")
	var darkCell, lightCell color.RGBA

	for _, dark := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			t := design.For(c)
			ui.Column(c).Fill().Background(t.Background).Children(func() {
				View(c, Model{Store: s, Embedded: true})
			})
		}, 850, 900)
		tt.SetDark(dark)
		r, ok := tt.Find("2026-10-14 UTC")
		if !ok {
			t.Fatalf("calendar cell missing in dark=%v", dark)
		}
		px := tt.Image().RGBAAt(int(r.X+r.W/2), int(r.Y+r.H/2))
		if dark {
			darkCell = px
		} else {
			lightCell = px
		}
	}
	if darkCell == lightCell {
		t.Fatalf("heatmap cell did not adapt between light (%v) and dark (%v)", lightCell, darkCell)
	}
}

func TestUsageStandaloneAccountRecentAndActionError(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	var jobsKilled atomic.Int32
	var closed, refreshed int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/account/getState":
			value = map[string]any{
				"status": "credential-stored",
				"links": map[string]any{
					"usageUrl": "https://usage.example.com",
					"topUpUrl": "https://topup.example.com",
				},
			}
		case "/api/account/hasRunningAccountTasks":
			value = map[string]any{"hasRunningAccountTasks": false}
		case "/api/job/list":
			value = map[string]any{
				"jobs": []any{
					map[string]any{"id": "job-1", "kind": "bash", "label": "Build task", "status": "running", "startedAt": 1700000000},
				},
			}
		case "/api/job/kill":
			jobsKilled.Add(1)
			value = map[string]any{}
		case "/usage-stats/overview":
			var o usagestore.Overview
			o.UpdatedAt = 1791970000000
			o.Days = []usagestore.DayUsage{}
			o.ByModel = []usagestore.ModelUsage{}
			o.AllTime.ByModel = []usagestore.ModelUsage{}
			o.TopSessions = []usagestore.SessionUsage{}
			o.Providers = []usagestore.ProviderUsage{}
			o.Coverage.Timezone = "UTC"
			o.Coverage.FailedSessionIDs = []string{}
			value = o
		case "/usage-stats/projects.more":
			value = map[string]any{"rows": []any{}, "hasMore": false}
		default:
			value = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	})

	s := usagestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: usageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})}}, nil)
	s.Refresh(context.Background(), "sess-1")
	for deadline := time.Now().Add(2 * time.Second); (s.Snapshot().Value.Loading || s.Stats().Loading) && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}

	var openedWeb, openedTopUp int
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, Model{
				Store:          s,
				SessionID:      "sess-1",
				ActionError:    "The operation did not finish. Try again; if it keeps failing, check the log.",
				OnClose:        func() { closed++ },
				OnRefresh:      func() { refreshed++ },
				OnOpenUsageWeb: func() { openedWeb++ },
				OnOpenTopUp:    func() { openedTopUp++ },
				OnCopyUsageURL: func(string) {},
				OnCopyTopUpURL: func(string) {},
			})
		})
	}, 960, 2800)

	joined := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"credential-stored", "https://usage.example.com", "Build task", i18n.T(i18n.KeyUsageKillJob), "The operation did not finish"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("standalone view missing %q in:\n%s", want, joined)
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyUsageOpenWeb)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyUsageOpenTopUp)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyUsageBack)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(settingsPass4UsageRefresh)); err != nil {
		t.Fatal(err)
	}
	if openedWeb != 1 || openedTopUp != 1 || closed != 1 || refreshed != 1 {
		t.Fatalf("standalone actions mismatch: web=%d topup=%d closed=%d refreshed=%d", openedWeb, openedTopUp, closed, refreshed)
	}
	if err := tt.Click(i18n.T(i18n.KeyUsageKillJob)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyUsageFollowJob)); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); jobsKilled.Load() == 0 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	if jobsKilled.Load() != 1 {
		t.Fatalf("kill job action was not dispatched: killed=%d", jobsKilled.Load())
	}
}

func TestUsageRepairAndTimeRangeViews(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/usage-stats/overview":
			var o usagestore.Overview
			o.UpdatedAt = 1791970000000
			o.Days = []usagestore.DayUsage{{Date: "2026-10-14", Total: 100}}
			o.ByModel = []usagestore.ModelUsage{}
			o.AllTime.ByModel = []usagestore.ModelUsage{}
			o.TopSessions = []usagestore.SessionUsage{}
			o.Providers = []usagestore.ProviderUsage{}
			o.Coverage.Timezone = "UTC"
			o.Coverage.SessionsFailed = 1
			o.Coverage.FailedSessionIDs = []string{"broken-sess-1"}
			value = o
		default:
			value = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	})
	s := usagestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: usageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Result(), nil
	})}}, nil)
	s.RefreshStats(context.Background(), false)
	for deadline := time.Now().Add(2 * time.Second); s.Stats().Loading && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(8).Children(func() {
			timeRangeView(c, Model{Store: s})
			repairView(c, Model{Store: s})
		})
	}, 900, 900)

	if err := tt.SetValue("Start date", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if err := tt.SetValue("End date", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Apply"); err != nil {
		t.Fatal(err)
	}
	if !s.Stats().Range.Active() || s.Stats().Range.Start != "2026-10-14" {
		t.Fatalf("timeRangeView did not apply date range: %+v", s.Stats().Range)
	}

	if err := tt.Click("Repair log: broken-sess-1"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Confirm repair of these 1 failed sessions:") {
		t.Fatalf("repair confirmation did not appear: %v", tt.Texts())
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Confirm repair of these 1 failed sessions:") {
		t.Fatalf("repair confirmation did not cancel: %v", tt.Texts())
	}
}
