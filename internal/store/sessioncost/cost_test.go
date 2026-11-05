package sessioncost

import (
	"encoding/json"
	"testing"
	"time"
	"whaleisle/internal/store/usage"
)

func TestSessionCostFixture(t *testing.T) {
	raw := json.RawMessage(`{"billedUsage":{"models":[{"provider":"p","model":"m","peak":{"missInputTokens":1000000,"cacheReadTokens":0,"outputTokens":0},"offPeak":{"missInputTokens":0,"cacheReadTokens":0,"outputTokens":0}}]}}`)
	p := Decode(raw)
	if p == nil {
		t.Fatal("missing projection")
	}
	settings := usage.BillingSettings{Prices: map[string]usage.ModelPrice{"p/m": {PriceRates: usage.PriceRates{InputCacheMiss: 2}, Flat: true}}}
	r := Calculate("root", []Source{{ID: "root", Usage: p}, {ID: "child", Parent: "root", Origin: "subagent", Usage: p}, {ID: "branch", Parent: "root", Origin: "fork", Usage: p}, {ID: "unknown", Parent: "child", Origin: "subagent"}}, settings)
	if r.Cents != 400 || len(r.Rows) != 1 || r.Children != 1 || !r.Partial {
		t.Fatalf("aggregate: %+v", r)
	}
	if Calculate("root", []Source{{ID: "root", Usage: p}}, usage.BillingSettings{}).Priced != 0 {
		t.Fatal("invented price")
	}
	if Decode(json.RawMessage(`{"billedUsage":{"models":[{"provider":"p","model":"m","peak":{},"offPeak":{}}]}}`)) != nil {
		t.Fatal("accepted missing counters")
	}
	peak, next := Phase(time.Date(2026, 10, 14, 3, 59, 59, 0, time.UTC))
	if !peak || next.Sub(time.Date(2026, 10, 14, 3, 59, 59, 0, time.UTC)) != time.Second {
		t.Fatal("Beijing boundary")
	}
}

func TestOfficialPricesAndCustomIdleDerivation(t *testing.T) {
	projection := &Projection{Models: []Route{{Provider: "deepseek-official", Model: "deepseek-v4-flash", Peak: &Bucket{Miss: 1000000}, OffPeak: &Bucket{Miss: 1000000}}}}
	sources := []Source{{ID: "root", Usage: projection}}
	got := Calculate("root", sources, usage.BillingSettings{})
	if got.Priced != 1 || got.Unpriced != 0 || got.Cents != 450 {
		t.Fatalf("official peak/idle prices: %+v", got)
	}
	custom := usage.BillingSettings{Prices: map[string]usage.ModelPrice{"deepseek-official/deepseek-v4-flash": {PriceRates: usage.PriceRates{InputCacheMiss: 4}}}}
	if got := Calculate("root", sources, custom); got.Cents != 600 || got.Priced != 1 {
		t.Fatalf("custom missing idle must derive half peak: %+v", got)
	}
}
