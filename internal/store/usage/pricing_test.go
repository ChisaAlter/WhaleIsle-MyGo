package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func waitBilling(t *testing.T, s *Store) BillingView {
	t.Helper()
	for end := time.Now().Add(3 * time.Second); time.Now().Before(end); {
		v := s.Billing()
		if !v.Busy {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("billing timed out")
	return BillingView{}
}
func TestPricingFixture(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string          `json:"rpcId"`
			Payload json.RawMessage `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.Header.Get("Cookie") != "fixture=yes" {
			t.Error("cookie missing")
		}
		switch r.URL.Path {
		case "/usage-stats/billing.models":
			writeRPC(w, req.RPCID, json.RawMessage(`{"options":[{"provider":"p","providerName":"Fixture","models":["m"]}]}`))
		case "/usage-stats/billing.get":
			writeRPC(w, req.RPCID, json.RawMessage(`{"prices":{"p/m":{"inputCacheHit":1,"inputCacheMiss":2,"output":4}},"peakValleyEnabled":true}`))
		case "/usage-stats/billing.set":
			if fail.Load() {
				http.Error(w, "offline", 503)
				return
			}
			writeRPC(w, req.RPCID, req.Payload)
		default:
			t.Error("unexpected path", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "fixture=yes"}}, nil)
	s.LoadBilling(context.Background())
	v := waitBilling(t, s)
	if !v.Loaded || v.Err != "" {
		t.Fatalf("load: %+v", v)
	}
	d := v.Drafts["p/m"]
	d.Peak[1] = "8"
	d.Flat = true
	s.EditPrice("p/m", d)
	fail.Store(true)
	s.SaveBilling(context.Background())
	v = waitBilling(t, s)
	if v.Err == "" || !v.Dirty || v.Drafts["p/m"].Peak[1] != "8" || v.Settings.Prices["p/m"].InputCacheMiss != 2 {
		t.Fatalf("failed save lost draft: %+v", v)
	}
	fail.Store(false)
	s.SaveBilling(context.Background())
	v = waitBilling(t, s)
	p := v.Settings.Prices["p/m"]
	if v.Err != "" || v.Dirty || p.Idle == nil || p.Idle.InputCacheMiss != 8 {
		t.Fatalf("flat save: %+v", v)
	}
	d = v.Drafts["p/m"]
	d.Peak[0] = "NaN"
	s.EditPrice("p/m", d)
	s.SaveBilling(context.Background())
	v = waitBilling(t, s)
	if v.Err == "" || !v.Dirty {
		t.Fatal("accepted invalid price")
	}
	s.Detach()
	if s.Billing().Loaded || len(s.Billing().Drafts) > 0 {
		t.Fatal("billing leaked across hosts")
	}
}
func TestPricingEstimate(t *testing.T) {
	settings := BillingSettings{Prices: map[string]ModelPrice{"m": {PriceRates: PriceRates{1, 2, 4}}, "p/m": {PriceRates: PriceRates{2, 6, 8}}}, PeakValleyEnabled: true}
	row := ModelUsage{Model: "m", Provider: "p", Cost: &PhaseBuckets{Peak: &Totals{Input: 1e6, CacheWrite: 1e6, CacheRead: 1e6, Output: 1e6}, OffPeak: &Totals{Input: 1e6}}}
	cents, ok := EstimateModel(row, settings)
	if !ok || cents != 2500 {
		t.Fatalf("estimate %v %v", cents, ok)
	}
	row.Provider = "other"
	cents, ok = EstimateModel(row, settings)
	if !ok || cents != 1000 {
		t.Fatalf("legacy fallback %v %v", cents, ok)
	}
	row.Model = "unknown"
	if _, ok = EstimateModel(row, settings); ok {
		t.Fatal("invented unknown price")
	}
	row.Model = "m"
	row.Cost = nil
	if _, ok = EstimateModel(row, settings); ok {
		t.Fatal("invented missing buckets")
	}
}

func TestPricingContract(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"prices":{},"peakValleyEnabled":null}`, `{"prices":{"m":{"inputCacheHit":1,"output":2}},"peakValleyEnabled":true}`, `{"prices":{"m":{"inputCacheHit":1,"inputCacheMiss":2,"output":3,"idle":null}},"peakValleyEnabled":true}`} {
		if _, err := decodeBilling(json.RawMessage(raw)); err == nil {
			t.Fatalf("accepted malformed response %s", raw)
		}
	}
	var row ModelUsage
	if err := json.Unmarshal([]byte(`{"model":"m","cost":{"peak":{},"offPeak":{}}}`), &row); err != nil {
		t.Fatal(err)
	}
	if _, ok := EstimateModel(row, BillingSettings{Prices: map[string]ModelPrice{"m": {PriceRates: PriceRates{1, 2, 3}}}}); ok {
		t.Fatal("treated absent fields as zero usage")
	}
}
