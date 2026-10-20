package usage

import (
	"context"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
)

func billingModelsView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	v := m.Store.BillingModels()
	ui.Row(c).Gap(6).Children(func() {
		ui.Text(c, usageText("选择服务商与模型", "Choose provider and model")).Bold().TextColor(fg).Grow(1)
		if widgets.Button(c, colors, usageText("刷新模型目录", "Refresh model directory"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(v.Loading).Clicked() {
			m.Store.LoadBillingModels(context.Background())
		}
	})
	filter := v.Filter
	ui.TextInput(c, &filter).FillWidth().Placeholder(usageText("搜索服务商或模型", "Search providers or models")).OnChange(func() { m.Store.SetBillingModelFilter(filter) })
	if v.Loading {
		ui.Text(c, usageText("正在读取模型目录…", "Loading model directory…")).TextColor(muted)
	}
	if v.Error != "" {
		ui.Text(c, v.Error).TextColor(accent)
	}
	b := m.Store.Billing()
	query := strings.ToLower(strings.TrimSpace(v.Filter))
	count := 0
	ui.Scroll(c).Height(160).Children(func() {
		ui.Column(c).Gap(5).Children(func() {
			for _, option := range v.Options {
				for _, model := range option.Models {
					option, model := option, model
					if strings.TrimSpace(model) == "" {
						continue
					}
					key := option.Provider + "/" + model
					title := option.ProviderName
					if title == "" {
						title = option.Provider
					}
					if query != "" && !strings.Contains(strings.ToLower(title+" "+key), query) {
						continue
					}
					count++
					ui.Row(c).Gap(6).Children(func() {
						ui.Text(c, title+" · "+model).TextColor(fg).Grow(1)
						title := usageText("添加到价格表", "Add to price table")
						if _, exists := b.Drafts[key]; exists {
							title = usageText("选择已有价格", "Select existing price")
						}
						if ui.Button(c, title).Disabled(b.Busy || !b.Loaded).Clicked() {
							m.Store.SelectBillingModel(option.Provider, model)
						}
					})
				}
			}
		})
	})
	if v.Loaded && count == 0 {
		ui.Text(c, usageText("没有匹配的模型，可在下方手动输入模型键。", "No matching models. You can enter a model key below.")).TextColor(muted)
	}
	ui.Text(c, usageText("模型目录只提供名称；价格需自行填写后保存。", "The directory supplies model names; enter and save your own prices.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
}
