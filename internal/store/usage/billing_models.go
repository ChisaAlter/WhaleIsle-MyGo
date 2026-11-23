package usage

import (
	"context"
	"errors"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
)

type BillingModelOption struct {
	Provider     string   `json:"provider"`
	ProviderName string   `json:"providerName"`
	Models       []string `json:"models"`
}

type BillingModelsView struct {
	Options         []BillingModelOption
	Loaded, Loading bool
	Error           *state.Fault
	Filter          string
}

func (s *Store) SetBillingModelFilter(value string) {
	s.mu.Lock()
	s.billing.models.Filter = value
	s.unlockPublish()
}

func (s *Store) SelectBillingModel(provider, model string) {
	key := strings.TrimSpace(provider) + "/" + strings.TrimSpace(model)
	s.mu.Lock()
	if !s.billing.Loaded || s.billing.Busy || !validPriceKey(key) || provider == "" || model == "" {
		s.unlockPublish()
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
		s.billing.Drafts = cloneDrafts(s.billing.Drafts)
		s.billing.Drafts[key] = draft
		s.billing.Dirty = true
	}
	s.billing.Err = nil
	s.billing.Notice = ""
	s.unlockPublish()
	s.fire()
}

func (s *Store) LoadBillingModels(ctx context.Context) {
	s.mu.Lock()
	if s.client == nil || s.billing.models.Loading {
		s.unlockPublish()
		return
	}
	client, epoch := s.client, s.attachL.Generation()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	s.billing.modelsCancel = cancel
	s.billing.models.Loading, s.billing.models.Error = true, nil
	s.unlockPublish()
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
		if s.attachL.Generation() != epoch || s.client != client {
			s.unlockPublish()
			return
		}
		s.billing.modelsCancel = nil
		s.billing.models.Loading = false
		if err != nil {
			s.billing.models.Error = faultOf(err)
		} else {
			s.billing.models.Options = result.Options
			s.billing.models.Loaded = true
		}
		s.unlockPublish()
		s.fire()
	}()
}
