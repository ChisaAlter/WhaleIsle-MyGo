package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"whaleisle/internal/desktop/pluginrpc"
)

// Rates are CNY per million tokens; saved overrides or a pinned reference table, never live quotes.
type PriceRates struct {
	InputCacheHit  float64 `json:"inputCacheHit"`
	InputCacheMiss float64 `json:"inputCacheMiss"`
	Output         float64 `json:"output"`
}
type ModelPrice struct {
	PriceRates
	Idle *PriceRates `json:"idle,omitempty"`
	Flat bool        `json:"flat,omitempty"`
}
type BillingSettings struct {
	Prices            map[string]ModelPrice `json:"prices"`
	PeakValleyEnabled bool                  `json:"peakValleyEnabled"`
}
type PhaseBuckets struct {
	Peak    *Totals `json:"peak"`
	OffPeak *Totals `json:"offPeak"`
}
type PriceDraft struct {
	Peak [3]string
	Idle [3]string
	Flat bool
}
type BillingView struct {
	NewKey              string
	Settings            BillingSettings
	Drafts              map[string]PriceDraft
	Loaded, Busy, Dirty bool
	Err                 string
	Notice              string
}
type billingState struct {
	BillingView
	cancel       context.CancelFunc
	models       BillingModelsView
	modelsCancel context.CancelFunc
}

func (s *Store) resetBillingLocked() {
	if s.billing.cancel != nil {
		s.billing.cancel()
	}
	if s.billing.modelsCancel != nil {
		s.billing.modelsCancel()
	}
	s.billing = billingState{}
}
func (s *Store) Billing() BillingView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.billing.BillingView
	v.Settings.Prices = clonePrices(v.Settings.Prices)
	v.Drafts = map[string]PriceDraft{}
	for k, d := range s.billing.Drafts {
		v.Drafts[k] = d
	}
	return v
}
func clonePrices(p map[string]ModelPrice) map[string]ModelPrice {
	out := make(map[string]ModelPrice, len(p))
	for k, v := range p {
		if v.Idle != nil {
			i := *v.Idle
			v.Idle = &i
		}
		out[k] = v
	}
	return out
}
func draftRates(p PriceRates) [3]string {
	return [3]string{strconv.FormatFloat(p.InputCacheHit, 'f', -1, 64), strconv.FormatFloat(p.InputCacheMiss, 'f', -1, 64), strconv.FormatFloat(p.Output, 'f', -1, 64)}
}
func priceDrafts(p map[string]ModelPrice) map[string]PriceDraft {
	d := map[string]PriceDraft{}
	for k, v := range p {
		row := PriceDraft{Peak: draftRates(v.PriceRates), Flat: v.Flat}
		if v.Idle != nil {
			row.Idle = draftRates(*v.Idle)
		}
		d[k] = row
	}
	return d
}
func validPriceKey(k string) bool {
	return strings.TrimSpace(k) != "" && len(utf16.Encode([]rune(k))) <= 256
}
func parseRates(d [3]string) (PriceRates, error) {
	var n [3]float64
	for i, v := range d {
		x, e := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if e != nil || math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x > 1e6 {
			return PriceRates{}, errors.New("价格须为 0–1000000 有限数值 / Prices must be finite numbers in [0, 1000000]")
		}
		n[i] = x
	}
	return PriceRates{n[0], n[1], n[2]}, nil
}
func settingsFromDrafts(d map[string]PriceDraft, peakValley bool) (BillingSettings, error) {
	v := BillingSettings{Prices: map[string]ModelPrice{}, PeakValleyEnabled: peakValley}
	for k, row := range d {
		if !validPriceKey(k) {
			return v, errors.New("invalid price key")
		}
		p, e := parseRates(row.Peak)
		if e != nil {
			return v, fmt.Errorf("%s: %w", k, e)
		}
		entry := ModelPrice{PriceRates: p, Flat: row.Flat}
		if row.Flat {
			idle := p
			entry.Idle = &idle
		} else if row.Idle != [3]string{} {
			idle, e := parseRates(row.Idle)
			if e != nil {
				return v, fmt.Errorf("%s idle: %w", k, e)
			}
			entry.Idle = &idle
		}
		v.Prices[k] = entry
	}
	return v, nil
}
func decodeBilling(raw json.RawMessage) (BillingSettings, error) {
	var wire struct {
		Prices     map[string]json.RawMessage `json:"prices"`
		PeakValley *bool                      `json:"peakValleyEnabled"`
	}
	if e := json.Unmarshal(raw, &wire); e != nil {
		return BillingSettings{}, e
	}
	if wire.Prices == nil || wire.PeakValley == nil {
		return BillingSettings{}, errors.New("incomplete billing response")
	}
	d := map[string]PriceDraft{}
	for k, r := range wire.Prices {
		var v struct {
			InputCacheHit  *float64        `json:"inputCacheHit"`
			InputCacheMiss *float64        `json:"inputCacheMiss"`
			Output         *float64        `json:"output"`
			Idle           json.RawMessage `json:"idle"`
			Flat           *bool           `json:"flat"`
		}
		if e := json.Unmarshal(r, &v); e != nil {
			return BillingSettings{}, e
		}
		if v.InputCacheHit == nil || v.InputCacheMiss == nil || v.Output == nil {
			return BillingSettings{}, errors.New("incomplete price rates")
		}
		row := PriceDraft{Peak: draftRates(PriceRates{*v.InputCacheHit, *v.InputCacheMiss, *v.Output})}
		if v.Flat != nil {
			if !*v.Flat {
				return BillingSettings{}, errors.New("invalid flat marker")
			}
			row.Flat = true
		}
		if len(v.Idle) > 0 {
			var i struct {
				InputCacheHit  *float64 `json:"inputCacheHit"`
				InputCacheMiss *float64 `json:"inputCacheMiss"`
				Output         *float64 `json:"output"`
			}
			if e := json.Unmarshal(v.Idle, &i); e != nil {
				return BillingSettings{}, e
			}
			if i.InputCacheHit == nil || i.InputCacheMiss == nil || i.Output == nil {
				return BillingSettings{}, errors.New("incomplete idle rates")
			}
			row.Idle = draftRates(PriceRates{*i.InputCacheHit, *i.InputCacheMiss, *i.Output})
		}
		d[k] = row
	}
	return settingsFromDrafts(d, *wire.PeakValley)
}
func (s *Store) EditPrice(key string, d PriceDraft) {
	s.mu.Lock()
	if !s.billing.Loaded || s.billing.Busy || !validPriceKey(key) {
		s.mu.Unlock()
		return
	}
	s.billing.Drafts[key] = d
	s.billing.Dirty = true
	s.billing.Err = ""
	s.billing.Notice = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) RemovePrice(key string) {
	s.mu.Lock()
	if !s.billing.Loaded || s.billing.Busy {
		s.mu.Unlock()
		return
	}
	delete(s.billing.Drafts, key)
	s.billing.Dirty = true
	s.billing.Err = ""
	s.billing.Notice = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) DiscardPrices() {
	s.mu.Lock()
	if !s.billing.Busy {
		s.billing.Drafts = priceDrafts(s.billing.Settings.Prices)
		s.billing.Dirty = false
		s.billing.Err = ""
		s.billing.Notice = ""
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) LoadBilling(ctx context.Context) {
	s.billingRequest(ctx, false)
	s.LoadBillingModels(ctx)
}
func (s *Store) SaveBilling(ctx context.Context) { s.billingRequest(ctx, true) }
func (s *Store) billingRequest(ctx context.Context, save bool) {
	s.mu.Lock()
	if s.client == nil || s.billing.Busy || (!save && s.billing.Dirty) || (save && (!s.billing.Loaded || !s.billing.Dirty)) {
		s.mu.Unlock()
		return
	}
	method := "billing.get"
	var payload any = map[string]any{}
	if save {
		next, e := settingsFromDrafts(s.billing.Drafts, s.billing.Settings.PeakValleyEnabled)
		if e != nil {
			s.billing.Err = e.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		method = "billing.set"
		payload = next
	}
	ep, httpClient := s.client.Endpoint, s.client.HTTP
	epoch := s.statsEpoch
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	s.billing.cancel = cancel
	s.billing.Busy = true
	s.billing.Err = ""
	s.billing.Notice = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie, HTTPClient: httpClient}
		var raw json.RawMessage
		err := rpc.Call(ctx, "usage-stats", method, payload, &raw)
		var settings BillingSettings
		if err == nil {
			settings, err = decodeBilling(raw)
		}
		s.mu.Lock()
		if epoch != s.statsEpoch {
			s.mu.Unlock()
			return
		}
		s.billing.Busy = false
		s.billing.cancel = nil
		if err != nil {
			s.billing.Err = err.Error()
		} else {
			s.resetRankingLocked()
			s.billing.Settings = settings
			s.billing.Drafts = priceDrafts(settings.Prices)
			s.billing.Loaded = true
			s.billing.Dirty = false
			if save {
				s.billing.Notice = "价格已保存 / Prices saved"
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// EstimateModel follows the original panel's per-period cents rounding and cache
// write = miss rate convention. Unknown prices or missing buckets stay unknown.
func EstimateModel(row ModelUsage, settings BillingSettings) (float64, bool) {
	peak, idle, ok := EstimateModelPhases(row, settings)
	return peak + idle, ok
}

// EstimateModelPhases returns independently rounded CNY cents for the host's
// Beijing billing-period buckets. UTC export dates never reclassify those buckets.
func EstimateModelPhases(row ModelUsage, settings BillingSettings) (float64, float64, bool) {
	p, ok := resolveModelPrice(row.Provider, row.Model, settings.Prices)
	if !ok || row.Cost == nil || row.Cost.Peak == nil || row.Cost.OffPeak == nil {
		return 0, 0, false
	}
	idle := PriceRates{p.InputCacheHit / 2, p.InputCacheMiss / 2, p.Output / 2}
	if p.Idle != nil {
		idle = *p.Idle
	}
	if p.Flat || !settings.PeakValleyEnabled {
		idle = p.PriceRates
	}
	period := func(b *Totals, r PriceRates) (float64, bool) {
		for _, n := range []float64{b.Input, b.Output, b.CacheRead, b.CacheWrite, r.InputCacheHit, r.InputCacheMiss, r.Output} {
			if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
				return 0, false
			}
		}
		n := (b.Input+b.CacheWrite)*math.Round(r.InputCacheMiss*1e6) + b.CacheRead*math.Round(r.InputCacheHit*1e6) + b.Output*math.Round(r.Output*1e6)
		if math.IsInf(n, 0) || n > 9007199254740991 {
			return 0, false
		}
		return math.Round(n / 1e10), true
	}
	a, ok := period(row.Cost.Peak, p.PriceRates)
	b, ok2 := period(row.Cost.OffPeak, idle)
	return a, b, ok && ok2
}

func (s *Store) SetPriceKey(key string) { s.mu.Lock(); s.billing.NewKey = key; s.mu.Unlock() }

// Missing bucket fields mean an older/incomplete host contract, not zero cost.
func (p *PhaseBuckets) UnmarshalJSON(raw []byte) error {
	var phases map[string]json.RawMessage
	if err := json.Unmarshal(raw, &phases); err != nil {
		return err
	}
	read := func(raw json.RawMessage) *Totals {
		var fields map[string]*float64
		if json.Unmarshal(raw, &fields) != nil {
			return nil
		}
		for _, key := range []string{"input", "output", "cacheRead", "cacheWrite"} {
			if fields[key] == nil {
				return nil
			}
		}
		return &Totals{Input: *fields["input"], Output: *fields["output"], CacheRead: *fields["cacheRead"], CacheWrite: *fields["cacheWrite"]}
	}
	p.Peak = read(phases["peak"])
	p.OffPeak = read(phases["offPeak"])
	return nil
}
