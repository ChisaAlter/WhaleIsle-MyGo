package usage

import "strings"

// Pinned to WhaleIsle 90465b67ec16765631d54c195af19d9ff3c3a770,
// vendor/dsh-usage-panel/src/shared/pricing.ts. This is an estimate table,
// not a current quote or historical tariff inference from event timestamps.
const OfficialPricesAsOf = "2026-08-17"
const OfficialPricesSource = "https://api-docs.deepseek.com/zh-cn/quick_start/pricing"

func resolveModelPrice(provider, model string, custom map[string]ModelPrice) (ModelPrice, bool) {
	if model == "" {
		return ModelPrice{}, false
	}
	// Map membership matters: an explicit zero is an override, not missing.
	if p, ok := custom[provider+"/"+model]; ok {
		return p, true
	}
	if p, ok := custom[model]; ok {
		return p, true
	}
	var peak, idle PriceRates
	switch strings.ToLower(model) {
	case "deepseek-v4-flash", "deepseek-v4-flash-vision-exp":
		peak, idle = PriceRates{0.1, 3, 9}, PriceRates{0.05, 1.5, 4.5}
	case "deepseek-v4-pro":
		peak, idle = PriceRates{0.3, 9, 27}, PriceRates{0.15, 4.5, 13.5}
	default:
		return ModelPrice{}, false
	}
	return ModelPrice{PriceRates: peak, Idle: &idle}, true
}
