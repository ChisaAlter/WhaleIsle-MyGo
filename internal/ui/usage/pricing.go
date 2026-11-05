package usage

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func pricingView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	trigger := widgets.Button(c.Key("usage-pricing/"+m.Store.StatisticsOwner()), colors, usageText("设置", "Settings"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall})
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
	if *submitted && !v.Busy && v.Err == "" && !v.Dirty {
		*submitted = false
		*open = false
	}
	var actions func()
	pricingDialogFooter(c, open, colors, func() {
		pricingEditor(c, m, fg, muted, accent, func(close bool) {
			m.Store.SaveBilling(context.Background())
			if close {
				*submitted = true
			}
		}, func(footer func()) { actions = footer })
	}, func() {
		if widgets.Button(c, colors, usageText("关闭", "Close"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(v.Busy).Clicked() {
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
func pricingDialog(c *ui.Context, open *bool, colors fonttheme.Colors, content func(), saving ...bool) {
	pricingDialogFooter(c, open, colors, content, nil, len(saving) > 0 && saving[0])
}
func pricingDialogFooter(c *ui.Context, open *bool, colors fonttheme.Colors, content, footer func(), busy bool) {
	ui.DialogBaseWithOptions(c, open, ui.DialogOptions{DismissDisabled: busy}, func(back, panel ui.Element) {
		widgets.RevealOverlay(c, back, panel, ui.RGBA(0, 0, 0, .4))
		w, h := c.Size()
		panel.Width(max(0, min(620, w-48))).MaxHeight(max(0, h-48)).Radius(16).Background(c.Theme().Background).Padding(24).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, usageText("计费设置", "Billing settings")).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(16)).FontWeight(600)
					if widgets.Button(c, colors, "×", widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Label(usageText("关闭", "Close")).Disabled(busy).Clicked() {
						*open = false
					}
				})
				bodyHeight := max(0, h-160)
				if footer != nil {
					bodyHeight = max(0, bodyHeight-52)
				}
				ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(bodyHeight).Shrink(1).Children(func() { ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(content) })
				if footer != nil {
					ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Gap(8).Wrap().Justify(ui.End).Children(footer)
				}
			})
		})
	})
}
func pricingContent(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	pricingEditor(c, m, fg, muted, accent, func(bool) { m.Store.SaveBilling(context.Background()) })
}
func pricingEditor(c *ui.Context, m Model, fg, muted, accent ui.Color, save func(bool), footers ...func(func())) {
	v := m.Store.Billing()
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Text(c, usageText("每百万 Token 单价（人民币）", "Price per million tokens (CNY)")).FillWidth().FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
	seen := ui.Local(c.Root(), "billing-editor-feedback", func() string { return "" })
	feedback := v.Err
	if feedback == "" {
		feedback = v.Notice
	}
	if feedback != "" && feedback != *seen {
		c.AddToast(ui.Toast{ID: "billing-editor-feedback", Description: feedback})
	}
	*seen = feedback
	if !v.Loaded {
		label := usageText("读取价格", "Load prices")
		if v.Busy {
			label = usageText("正在读取…", "Loading…")
		}
		if ui.Button(c, label).Disabled(v.Busy).Clicked() {
			m.Store.LoadBilling(context.Background())
		}
		return
	}
	billingModelsView(c, m, fg, muted, accent)
	key := m.Store.Billing().NewKey
	editor := ui.Column(c.Key("price-current-editor")).FillWidth().MinWidth(0).Gap(12)
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
		*seen = ""
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
			ui.Text(c, usageText("选择模型后设置价格。", "Choose a model to set its price.")).TextColor(muted)
			return
		}
		periodPrices := func(title string, values *[3]string) {
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
			ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.End).Children(func() {
				for i, label := range []string{usageText("输入（缓存命中）", "Input (cache hit)"), usageText("输入（缓存未命中）", "Input (cache miss)"), usageText("输出", "Output")} {
					i, label := i, label
					ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
						ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
						ui.TextInput(c, &values[i]).Label(title+" · "+label).FillWidth().Height(36).Radius(8).Border(.5, c.Theme().Border.Alpha(.45)).Disabled(v.Busy)
					})
				}
			})
		}
		peakValley := !draft.Flat
		if v.Settings.PeakValleyEnabled && ui.Switch(c, &peakValley).Label(usageText("此模型使用峰谷价格", "Use peak/off-peak prices for this model")).Disabled(v.Busy).Changed() {
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
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Justify(ui.End).Children(func() {
			if widgets.Button(c, colors, usageText("添加 / 更新", "Add / update"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(v.Busy).Clicked() {
				commit(false)
			}
		})
	})
	actions := func() {
		label := usageText("保存", "Save")
		if v.Busy {
			label = usageText("保存中…", "Saving…")
		}
		if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(v.Busy).Clicked() {
			commit(true)
		}
	}
	if len(footers) > 0 {
		footers[0](actions)
	} else {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Justify(ui.End).Children(actions)
	}
}
func priceRateStrings(r usagestore.PriceRates) [3]string {
	return [3]string{formatPrice(r.InputCacheHit), formatPrice(r.InputCacheMiss), formatPrice(r.Output)}
}
