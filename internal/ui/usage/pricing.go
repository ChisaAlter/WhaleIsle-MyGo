package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	usagestore "whaleisle/internal/store/usage"
)

func pricingView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	v := m.Store.Billing()
	ui.Text(c, usageText("价格配置与费用估算", "Price settings and cost estimates")).Bold().TextColor(fg)
	ui.Text(c, usageText("单位：CNY / 百万 tokens。来自用户配置；估算不是账单，也不是实时供应商报价。", "CNY per million tokens. User-configured prices; estimates are not bills or live vendor quotes.")).FontSize(11).TextColor(muted)
	if v.Err != "" {
		ui.Text(c, v.Err).TextColor(accent)
	}
	if v.Busy {
		ui.Text(c, usageText("正在读写价格…", "Loading/saving prices…")).TextColor(muted)
		return
	}
	ui.Row(c).Gap(6).Children(func() {
		if !v.Dirty && ui.Button(c, usageText("读取价格", "Load prices")).Clicked() {
			m.Store.LoadBilling(context.Background())
		}
		if v.Loaded && v.Dirty {
			if ui.Button(c, usageText("保存价格", "Save prices")).Clicked() {
				m.Store.SaveBilling(context.Background())
			}
			if ui.Button(c, usageText("放弃草稿", "Discard draft")).Clicked() {
				m.Store.DiscardPrices()
			}
		}
	})
	if !v.Loaded {
		ui.Text(c, usageText("先读取已有价格后才能编辑；未配置价格的模型不估算。", "Load existing prices before editing. Models without configured prices stay unpriced.")).FontSize(11).TextColor(muted)
		return
	}
	billingModelsView(c, m, fg, muted, accent)
	if v.Dirty {
		ui.Text(c, usageText("有未保存草稿；估算仍使用已保存价格。保存失败可重试。", "Unsaved draft; estimates still use saved prices. Failed saves can be retried.")).TextColor(accent)
	}
	ui.Text(c, usageText("峰值顺序：缓存命中 / 未命中（含缓存写入）/ 输出。谷值三项留空时沿原面板按峰值一半估算，此为配置约定。", "Rate order: cache hit / miss (including cache writes) / output. Three blank off-peak fields use half of peak, following the original panel convention.")).FontSize(11).TextColor(muted)
	keys := make([]string, 0, len(v.Drafts))
	for k := range v.Drafts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ui.Scroll(c).Height(220).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			for _, key := range keys {
				key := key
				d := v.Drafts[key]
				ui.Row(c).Gap(6).Children(func() {
					title := key
					if v.NewKey == key { title = "● " + title }
					ui.Text(c, title).TextColor(fg).Grow(1)
					if ui.Button(c, usageText("删除价格", "Remove price")).Clicked() {
						m.Store.RemovePrice(key)
					}
				})
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, usageText("峰值", "Peak")).TextColor(muted)
					for i := range d.Peak {
						i := i
						ui.TextInput(c, &d.Peak[i]).Grow(1).OnChange(func() { m.Store.EditPrice(key, d) })
					}
				})
				ui.Checkbox(c, &d.Flat, usageText("平价（峰谷使用同价）", "Flat (same price for both periods)")).OnChange(func() { m.Store.EditPrice(key, d) })
				if !d.Flat {
					ui.Row(c).Gap(6).Children(func() {
						ui.Text(c, usageText("谷值", "Off-peak")).TextColor(muted)
						for i := range d.Idle {
							i := i
							ui.TextInput(c, &d.Idle[i]).Grow(1).Placeholder(usageText("留空=峰值一半", "Blank = half peak")).OnChange(func() { m.Store.EditPrice(key, d) })
						}
					})
				}
			}
		})
	})
	key := v.NewKey
	ui.Row(c).Gap(6).Children(func() {
		ui.TextInput(c, &key).Grow(1).Placeholder("provider/model").OnChange(func() { m.Store.SetPriceKey(key) })
		if ui.Button(c, usageText("添加价格", "Add price")).Clicked() {
			key = strings.TrimSpace(key)
			if _, exists := v.Drafts[key]; !exists {
				m.Store.EditPrice(key, usagestore.PriceDraft{})
			}
		}
	})
	st := m.Store.Stats()
	if st.Overview == nil {
		return
	}
	ui.Text(c, usageText("全部历史费用估算（不随日期选区变化）", "All-time cost estimates (independent of date selection)")).Bold().TextColor(fg)
	total := 0.0
	priced, unpriced := 0, 0
	for _, row := range st.Overview.AllTime.ByModel {
		cents, ok := usagestore.EstimateModel(row, v.Settings)
		label := usageText("未计价（缺价格或分时数据）", "Unpriced (missing price or period data)")
		if ok {
			total += cents
			priced++
			label = fmt.Sprintf("≈ CNY %.2f", cents/100)
		} else {
			unpriced++
		}
		ui.Text(c, fmt.Sprintf("%s / %s · %s", row.Provider, row.Model, label)).FontSize(11).TextColor(muted)
	}
	if priced > 0 {
		ui.Text(c, usageText("全部历史已计价部分估算", "All-time priced portion estimate")+fmt.Sprintf(": ≈ CNY %.2f · %d priced · %d unpriced", total/100, priced, unpriced)).TextColor(fg)
	}
	cv := st.Overview.Coverage
	if unpriced > 0 || cv.SessionsFailed > 0 || cv.SessionsPending > 0 || st.Overview.Stale || st.Err != "" {
		ui.Text(c, usageText("价格或用量覆盖不完整 / 数据可能过期，不代表完整费用。", "Price/usage coverage is incomplete or data may be stale; this is not a complete cost.")).FontSize(11).TextColor(accent)
	}
}
