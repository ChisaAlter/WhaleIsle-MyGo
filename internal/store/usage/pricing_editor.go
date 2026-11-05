package usage

import "strings"

// EditModelPrice stages the current model editor without changing the generic
// price-draft API. Returning to the pinned default removes its custom override.
func (s *Store) EditModelPrice(key string, draft PriceDraft) bool {
	s.mu.Lock()
	if !s.billing.Loaded || s.billing.Busy || !validPriceKey(key) {
		s.mu.Unlock()
		return false
	}
	next, err := settingsFromDrafts(map[string]PriceDraft{key: draft}, s.billing.Settings.PeakValleyEnabled)
	if err != nil {
		s.billing.Err = err.Error()
		s.mu.Unlock()
		s.fire()
		return false
	}
	provider, model, composite := strings.Cut(key, "/")
	price := next.Prices[key]
	inherited, known := resolveModelPrice(provider, model, nil)
	matchesDefault := known && !price.Flat && price.PriceRates == inherited.PriceRates && price.Idle != nil && inherited.Idle != nil && *price.Idle == *inherited.Idle
	// A failed submission belongs to that editor buffer, not a later model.
	s.billing.Drafts = priceDrafts(s.billing.Settings.Prices)
	if composite {
		delete(s.billing.Drafts, model)
	}
	if matchesDefault {
		delete(s.billing.Drafts, key)
	} else {
		s.billing.Drafts[key] = draft
	}
	s.billing.Dirty, s.billing.Err, s.billing.Notice = true, "", ""
	s.mu.Unlock()
	s.fire()
	return true
}
