package usage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
)

type pricingFeedbackTransport func(*http.Request) (*http.Response, error)

func (f pricingFeedbackTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestUpstreamDeltaPricingFeedback(t *testing.T) {
	var fail atomic.Bool
	c := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1"}}
	c.HTTP = &http.Client{Transport: pricingFeedbackTransport(func(r *http.Request) (*http.Response, error) {
		var req struct {
			RPCID   string          `json:"rpcId"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		status := 200
		payload := req.Payload
		if strings.HasSuffix(r.URL.Path, "billing.get") {
			payload = json.RawMessage(`{"prices":{"p/m":{"inputCacheHit":1,"inputCacheMiss":2,"output":4}},"peakValleyEnabled":true}`)
		}
		if fail.Load() {
			status = 503
		}
		data, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": payload}})
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})}
	s := New()
	s.Attach(c, nil)
	s.billingRequest(context.Background(), false)
	v := waitBilling(t, s)
	if v.Err != "" || !v.Loaded || v.Notice != "" {
		t.Fatalf("load: %+v", v)
	}
	d := v.Drafts["p/m"]
	d.Peak[1] = "8"
	s.EditPrice("p/m", d)
	fail.Store(true)
	s.SaveBilling(context.Background())
	v = waitBilling(t, s)
	if v.Err == "" || v.Notice != "" || !v.Dirty || v.Drafts["p/m"].Peak[1] != "8" {
		t.Fatalf("failed save: %+v", v)
	}
	fail.Store(false)
	s.SaveBilling(context.Background())
	v = waitBilling(t, s)
	if v.Err != "" || v.Dirty || !strings.Contains(v.Notice, storelocale.T(storelocale.KeyStoreUsagePricesSaved)) || v.Settings.Prices["p/m"].InputCacheMiss != 8 {
		t.Fatalf("saved: %+v", v)
	}
	s.EditPrice("p/m", d)
	if s.Billing().Notice != "" {
		t.Fatal("success feedback persisted into new edit")
	}
	s.SaveBilling(context.Background())
	_ = waitBilling(t, s)
	s.SelectBillingModel("other", "model")
	if s.Billing().Notice != "" {
		t.Fatal("success feedback persisted into new model draft")
	}
	s.Detach()
	if s.Billing().Notice != "" {
		t.Fatal("feedback crossed host boundary")
	}
}
