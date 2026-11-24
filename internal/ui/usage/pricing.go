package usage

import (
	"context"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func pricingView(c *ui.Context, m Model) {
	trigger := settingsIconButton(c, "usage-pricing/"+m.Store.StatisticsOwner(), usageText("设置", "Settings"))
	open := ui.Local(trigger, "open", func() bool { return false })
	submitted := ui.Local(trigger, "submitted", func() bool { return false })
	requested := m.Store.TakeBillingEditorRequest()
	if trigger.Clicked() || requested {
		m.Store.DiscardPrices()
		m.Store.LoadBilling(context.Background())
		*open = true
		*submitted = false
	}
	v := m.Store.Billing()
	wasOpen := *open
	if *submitted && !v.Busy && v.Err == nil && !v.Dirty {
		*submitted = false
		*open = false
	}
	var actions func()
	pricingDialogFooter(c, open, func() {
		pricingEditor(c, m, func(close bool) {
			m.Store.SaveBilling(context.Background())
			if close {
				*submitted = true
			}
		}, func(footer func()) { actions = footer })
	}, func() {
		if coss.Button(c, usageText("关闭", "Close"), coss.ButtonOptions{
			Variant:  coss.ButtonOutline,
			Size:     coss.ButtonSM,
			Disabled: v.Busy,
		}).Clicked() {
			*open = false
		}
		if actions != nil {
			actions()
		}
	}, v.Busy)
	if wasOpen && !*open {
		m.Store.DiscardPrices()
	}
}

func pricingDialog(c *ui.Context, open *bool, content func(), saving ...bool) {
	pricingDialogFooter(c, open, content, nil, len(saving) > 0 && saving[0])
}

func pricingDialogFooter(c *ui.Context, open *bool, content, footer func(), busy bool) {
	coss.Dialog(c, open, coss.DialogOptions{
		Title:           usageText("计费设置", "Billing settings"),
		MaxWidth:        620,
		DismissDisabled: busy,
		NoClose:         busy,
		Footer:          footer,
	}, func() {
		_, h := c.Size()
		reserve := design.Sp(48)
		if footer != nil {
			reserve = design.Sp(64)
		}
		ui.Column(c).FillWidth().MaxHeight(max(design.Sp(30), h-reserve)).Children(func() {
			coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(content)
			}).FillWidth()
		})
	})
}

func pricingEditor(c *ui.Context, m Model, save func(bool), footers ...func(func())) {
	v := m.Store.Billing()
	pagekit.Muted(c, usageText("每百万 Token 单价（人民币）", "Price per million tokens (CNY)")).FillWidth()
	// The editor owns its own outcome: a failed save is drawn above the prices it
	// refused, and a saved change is an acknowledgement line, not a floating bar.
	if v.Err != nil {
		pagekit.FaultSentence(c, "", v.Err.SentenceText(), "", nil)
	} else if v.Notice != "" {
		pagekit.Note(c, coss.AlertSuccess, v.Notice, "", nil)
	}
	if !v.Loaded {
		label := usageText("读取价格", "Load prices")
		if v.Busy {
			label = usageText("正在读取…", "Loading…")
		}
		if coss.Button(c, label, coss.ButtonOptions{
			Variant:  coss.ButtonOutline,
			Size:     coss.ButtonSM,
			Disabled: v.Busy,
			Loading:  v.Busy,
		}).AlignSelf(ui.Start).Clicked() {
			m.Store.LoadBilling(context.Background())
		}
		return
	}
	billingModelsView(c, m)
	key := m.Store.Billing().NewKey
	editor := ui.Column(c.Key("price-current-editor")).FillWidth().MinWidth(0).Gap(design.Sp(3))
	selected := ui.Local(editor, "selected-key", func() string { return "" })
	draft := ui.Local(editor, "draft", func() usagestore.PriceDraft { return usagestore.PriceDraft{Flat: true} })
	if *selected != key {
		*selected = key
		*draft = usagestore.PriceDraft{Flat: true}
		provider, model, _ := strings.Cut(key, "/")
		if price, ok := usagestore.ResolveModelPrice(provider, model, v.Settings.Prices); ok {
			draft.Flat = price.Flat
			draft.Peak = priceRateStrings(price.PriceRates)
			if price.Flat {
				draft.Idle, draft.Peak = draft.Peak, [3]string{}
			} else if price.Idle != nil {
				draft.Idle = priceRateStrings(*price.Idle)
			} else {
				draft.Idle = priceRateStrings(usagestore.PriceRates{InputCacheHit: price.InputCacheHit / 2, InputCacheMiss: price.InputCacheMiss / 2, Output: price.Output / 2})
			}
		}
	}
	commit := func(close bool) {
		if key != "" {
			staged := *draft
			if staged.Flat {
				staged.Peak = staged.Idle
			}
			if !m.Store.EditModelPrice(key, staged) {
				return
			}
		}
		save(close)
	}
	editor.Children(func() {
		if key == "" {
			pagekit.Muted(c, usageText("选择模型后设置价格。", "Choose a model to set its price.")).FillWidth()
			return
		}
		periodPrices := func(title string, values *[3]string) {
			pagekit.Strong(c, title).FillWidth()
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).AlignItems(ui.End).Children(func() {
				for i, label := range []string{usageText("输入（缓存命中）", "Input (cache hit)"), usageText("输入（缓存未命中）", "Input (cache miss)"), usageText("输出", "Output")} {
					i, label := i, label
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						coss.Field(c, coss.FieldOptions{Label: label, Disabled: v.Busy}, func() {
							_ = coss.Input(c, &values[i], coss.InputOptions{
								Size:  coss.InputLG,
								Label: title + " · " + label,
							}).Changed()
						})
					})
				}
			})
		}
		peakValley := !draft.Flat
		if v.Settings.PeakValleyEnabled && coss.Switch(c, &peakValley, usageText("此模型使用峰谷价格", "Use peak/off-peak prices for this model"), coss.CheckOptions{Disabled: v.Busy}).Changed() {
			draft.Flat = !peakValley
			if peakValley {
				for i, value := range draft.Peak {
					if value == "" {
						if number, err := strconv.ParseFloat(draft.Idle[i], 64); err == nil {
							draft.Peak[i] = formatPrice(number * 2)
						}
					}
				}
			}
		}
		title := usageText("谷时价格", "Off-peak prices")
		if draft.Flat {
			title = usageText("单价", "Prices")
		}
		periodPrices(title, &draft.Idle)
		if !draft.Flat {
			periodPrices(usageText("峰时价格", "Peak prices"), &draft.Peak)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Justify(ui.End).Children(func() {
			if coss.Button(c, usageText("添加 / 更新", "Add / update"), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonSM,
				Disabled: v.Busy,
			}).Clicked() {
				commit(false)
			}
		})
	})
	actions := func() {
		label := usageText("保存", "Save")
		if v.Busy {
			label = usageText("保存中…", "Saving…")
		}
		if coss.Button(c, label, coss.ButtonOptions{
			Variant:  coss.ButtonDefault,
			Size:     coss.ButtonSM,
			Disabled: v.Busy,
			Loading:  v.Busy,
		}).Clicked() {
			commit(true)
		}
	}
	if len(footers) > 0 {
		footers[0](actions)
	} else {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Wrap().Justify(ui.End).Children(actions)
	}
}

func priceRateStrings(r usagestore.PriceRates) [3]string {
	return [3]string{formatPrice(r.InputCacheHit), formatPrice(r.InputCacheMiss), formatPrice(r.Output)}
}
