package usage

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func formatPrice(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func billingModelsView(c *ui.Context, m Model) {
	v := m.Store.BillingModels()
	billing := m.Store.Billing()
	if v.Error != nil {
		pagekit.FaultSentence(c, "", v.Error.SentenceText(), usageText("重试模型目录", "Retry model directory"), func() {
			m.Store.LoadBillingModels(context.Background())
		})
	}
	seat := ui.Row(c.Key("billing-provider-model")).FillWidth().MinWidth(0).Gap(design.Sp(3))
	provider := ui.Local(seat, "provider", func() string { return "" })
	model := ui.Local(seat, "model", func() string { return "" })
	previous := ui.Local(seat, "selected-key", func() string { return "" })
	if billing.NewKey != *previous && billing.NewKey != "" {
		p, md, ok := strings.Cut(billing.NewKey, "/")
		if ok {
			*provider, *model = p, md
		}
		*previous = billing.NewKey
	}
	providers := []string{}
	names := map[string]string{}
	models := map[string][]string{}
	for _, option := range v.Options {
		providers = append(providers, option.Provider)
		names[option.Provider] = option.ProviderName
		models[option.Provider] = append([]string(nil), option.Models...)
	}
	for key := range billing.Settings.Prices {
		p, md, ok := strings.Cut(key, "/")
		if !ok {
			continue
		}
		if _, exists := names[p]; !exists {
			providers = append(providers, p)
			names[p] = p
		}
		found := false
		for _, id := range models[p] {
			found = found || id == md
		}
		if !found {
			models[p] = append(models[p], md)
		}
	}
	sort.Strings(providers)
	if *provider == "" && len(providers) > 0 {
		*provider = providers[0]
	}
	providerOptions := make([]coss.SelectOption, 0, len(providers))
	for _, p := range providers {
		label := names[p]
		if label == "" {
			label = p
		}
		providerOptions = append(providerOptions, coss.SelectOption{Value: p, Label: label})
	}
	choices := models[*provider]
	valid := false
	for _, id := range choices {
		valid = valid || id == *model
	}
	if !valid && len(choices) > 0 {
		*model = choices[0]
	}
	modelOptions := make([]coss.SelectOption, 0, len(choices))
	for _, id := range choices {
		modelOptions = append(modelOptions, coss.SelectOption{Value: id, Label: id})
	}
	disabled := v.Loading || billing.Busy
	seat.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			coss.Field(c, coss.FieldOptions{Label: usageText("供应商", "Provider"), Disabled: disabled}, func() {
				if coss.Select(c, provider, providerOptions, coss.SelectOptions{Size: coss.InputLG}).FillWidth().Changed() {
					*model = ""
					nextChoices := models[*provider]
					if len(nextChoices) > 0 {
						*model = nextChoices[0]
					}
				}
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			coss.Field(c, coss.FieldOptions{Label: usageText("模型", "Model"), Disabled: disabled}, func() {
				_ = coss.Select(c, model, modelOptions, coss.SelectOptions{Size: coss.InputLG}).FillWidth().Changed()
			})
		})
	})
	if *provider != "" && *model != "" {
		key := *provider + "/" + *model
		if billing.NewKey == "" || billing.NewKey != key {
			m.Store.SetPriceKey(key)
		}
	}
}
