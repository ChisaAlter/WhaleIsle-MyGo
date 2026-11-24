package usage

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func rankDialog(c *ui.Context, open *bool, title string, content func()) {
	coss.Dialog(c, open, coss.DialogOptions{
		Title:    title,
		MaxWidth: 640,
	}, func() {
		_, h := c.Size()
		ui.Column(c).FillWidth().MaxHeight(max(design.Sp(30), h-design.Sp(44))).Children(func() {
			coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(content)
			}).FillWidth()
		})
	})
}

func rankingSort(c *ui.Context, value *string, busy bool, changed func()) {
	options := []coss.SelectOption{
		{Value: "tokens", Label: usageText("Token", "Tokens")},
		{Value: "cost", Label: usageText("费用", "Cost")},
	}
	ui.Row(c).FillWidth().Gap(design.Sp(2)).Justify(ui.End).AlignItems(ui.Center).Children(func() {
		pagekit.Caption(c, usageText("排序", "Sort by"))
		if coss.Select(c, value, options, coss.SelectOptions{
			Size:     coss.InputSM,
			Disabled: busy,
			Label:    usageText("排序", "Sort by"),
		}).Width(108).Changed() {
			changed()
		}
	})
}

func rankTableHeader(c *ui.Context, sessions bool) {
	t := design.For(c)
	ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).PaddingY(design.Sp(1.5)).BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border).Children(func() {
		pagekit.Caption(c, "#").Width(22)
		label := usageText("工作目录", "Project")
		if sessions {
			label = usageText("会话", "Session")
		}
		pagekit.Caption(c, label).Grow(1).MinWidth(0)
		if sessions {
			pagekit.Caption(c, usageText("类型", "Type")).Width(56)
			pagekit.Caption(c, usageText("活跃日期", "Last active")).Width(78)
		}
		pagekit.Caption(c, usageText("估算费用", "Est. cost")).Width(62).TextAlign(ui.End)
		pagekit.Caption(c, "Tokens").Width(62).TextAlign(ui.End)
	})
}

func rankMore(c *ui.Context, loading, started, hasMore bool, err string, load func()) {
	if err != "" {
		retryLabel := ""
		var retry func()
		if !loading {
			retryLabel = usageText("重试", "Retry")
			retry = load
		}
		pagekit.FaultSentence(c, "", err, retryLabel, retry)
	}
	if loading {
		ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			coss.Spinner(c, coss.SpinnerOptions{})
			pagekit.Muted(c, usageText("正在加载…", "Loading…"))
		})
		return
	}
	if err == "" && (!started || hasMore) {
		if coss.Button(c, usageText("更多", "More"), coss.ButtonOptions{
			Variant: coss.ButtonOutline,
			Size:    coss.ButtonSM,
		}).AlignSelf(ui.Start).Clicked() {
			load()
		}
	}
}
