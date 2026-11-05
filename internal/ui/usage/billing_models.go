package usage

import (
	"context"
	"github.com/egoist/mygo/ui"
	"sort"
	"strconv"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"
)

func formatPrice(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }
func billingModelsView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	v := m.Store.BillingModels()
	billing := m.Store.Billing()
	if v.Error != "" {
		ui.Text(c, v.Error).TextColor(c.Theme().Danger)
		if ui.Button(c, usageText("重试模型目录", "Retry model directory")).Disabled(v.Loading).Clicked() {
			m.Store.LoadBillingModels(context.Background())
		}
	}
	seat := ui.Row(c.Key("billing-provider-model")).FillWidth().MinWidth(0).Gap(12)
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
	labels := []string{}
	ids := map[string]string{}
	current := ""
	for _, p := range providers {
		label := names[p]
		if label == "" {
			label = p
		}
		labels = append(labels, label)
		ids[label] = p
		if p == *provider {
			current = label
		}
	}
	seat.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, usageText("供应商", "Provider")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			if ui.Select(c, &current, labels).Label(usageText("供应商", "Provider")).FillWidth().Height(36).Disabled(v.Loading || billing.Busy).Changed() {
				*provider = ids[current]
				*model = ""
			}
		})
		choices := models[*provider]
		valid := false
		for _, id := range choices {
			valid = valid || id == *model
		}
		if !valid && len(choices) > 0 {
			*model = choices[0]
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, usageText("模型", "Model")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Select(c, model, choices).Label(usageText("模型", "Model")).FillWidth().Height(36).Disabled(v.Loading || billing.Busy)
		})
	})
	if *provider != "" && *model != "" {
		key := *provider + "/" + *model
		if billing.NewKey == "" || billing.NewKey != key {
			m.Store.SetPriceKey(key)
		}
	}
}
