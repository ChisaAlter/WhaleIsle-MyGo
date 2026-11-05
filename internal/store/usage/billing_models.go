package usage

import (
	"context"
	"errors"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
)

type BillingModelOption struct {
	Provider     string   `json:"provider"`
	ProviderName string   `json:"providerName"`
	Models       []string `json:"models"`
}

type BillingModelsView struct {
	Options         []BillingModelOption
	Loaded, Loading bool
	Error, Filter   string
}

func (s *Store) BillingModels() BillingModelsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.billing.models
	v.Options = append([]BillingModelOption(nil), v.Options...)
	for i := range v.Options {
		v.Options[i].Models = append([]string(nil), v.Options[i].Models...)
	}
	return v
}

func (s *Store) SetBillingModelFilter(value string) {
	s.mu.Lock()
	s.billing.models.Filter = value
	s.mu.Unlock()
}

func (s *Store) SelectBillingModel(provider, model string) {
	key := strings.TrimSpace(provider) + "/" + strings.TrimSpace(model)
	s.mu.Lock()
	if !s.billing.Loaded || s.billing.Busy || !validPriceKey(key) || provider == "" || model == "" {
		s.mu.Unlock()
		return
	}
	s.billing.NewKey = key
	if _, exists := s.billing.Drafts[key]; !exists {
		draft := PriceDraft{}
		if price, ok := resolveModelPrice(provider, model, s.billing.Settings.Prices); ok {
			draft.Peak, draft.Flat = draftRates(price.PriceRates), price.Flat
			if price.Idle != nil {
				draft.Idle = draftRates(*price.Idle)
			}
		}
		s.billing.Drafts[key] = draft
		s.billing.Dirty = true
	}
	s.billing.Err = ""
	s.billing.Notice = ""
	s.mu.Unlock()
	s.fire()
}

func (s *Store) LoadBillingModels(ctx context.Context) {
	s.mu.Lock()
	if s.client == nil || s.billing.models.Loading {
		s.mu.Unlock()
		return
	}
	client, epoch := s.client, s.statsEpoch
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	s.billing.modelsCancel = cancel
	s.billing.models.Loading, s.billing.models.Error = true, ""
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
		var result struct {
			Options []BillingModelOption `json:"options"`
		}
		err := rpc.Call(ctx, "usage-stats", "billing.models", map[string]any{}, &result)
		if err == nil && result.Options == nil {
			err = errors.New("incomplete billing model directory")
		}
		if err == nil {
			for _, option := range result.Options {
				if strings.TrimSpace(option.Provider) == "" || option.Models == nil {
					err = errors.New("invalid billing model directory entry")
					break
				}
			}
		}
		s.mu.Lock()
		if s.statsEpoch != epoch || s.client != client {
			s.mu.Unlock()
			return
		}
		s.billing.modelsCancel = nil
		s.billing.models.Loading = false
		if err != nil {
			s.billing.models.Error = err.Error()
		} else {
			s.billing.models.Options = result.Options
			s.billing.models.Loaded = true
		}
		s.mu.Unlock()
		s.fire()
	}()
}
