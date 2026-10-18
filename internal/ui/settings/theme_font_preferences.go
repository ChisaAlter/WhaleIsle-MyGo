package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"time"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func themeFontPane(c *ui.Context, e *theme.Editor) {
	if e == nil {
		return
	}
	ui.Text(c, "主题库与字体 / Theme library and fonts").Bold()
	ids := []string{}
	for _, f := range e.Draft.Families() {
		ids = append(ids, f.ID)
	}
	ui.Text(c, "浅色主题 / Light theme")
	ui.Select(c, &e.Draft.Light, ids).FillWidth()
	ui.Text(c, "深色主题 / Dark theme")
	ui.Select(c, &e.Draft.Dark, ids).FillWidth()
	for _, f := range e.Draft.Families() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, f.Name).Grow(1)
			for _, s := range []theme.Seed{f.Light, f.Dark} {
				ui.Text(c, "Aa").Padding(6, 12).Background(chrome.Color(s.Background)).TextColor(chrome.Color(s.Foreground))
				ui.Box(c).Size(18, 18).Background(chrome.Color(s.Accent))
			}
		})
	}
	ui.Text(c, "自定义主题 / Custom themes").Bold()
	customIDs := []string{""}
	for _, f := range e.Draft.Custom {
		customIDs = append(customIDs, f.ID)
	}
	ui.Select(c, &e.Selected, customIDs).FillWidth()
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "从浅色主题复制 / Duplicate light family").Clicked() {
			f := e.Draft.Family(e.Draft.Light)
			f.ID = fmt.Sprintf("custom-%d", time.Now().UnixNano())
			f.Name += " copy"
			e.Draft.Custom = append(e.Draft.Custom, f)
			e.Selected = f.ID
		}
		if ui.Button(c, "删除自定义 / Delete custom").Disabled(e.Selected == "").Clicked() {
			e.DeleteSelected()
		}
	})
	for i := range e.Draft.Custom {
		f := &e.Draft.Custom[i]
		if f.ID != e.Selected {
			continue
		}
		ui.Text(c, "名称 / Name")
		ui.TextInput(c, &f.Name).FillWidth()
		for _, v := range []struct {
			name string
			s    *theme.Seed
		}{{"浅色 / Light", &f.Light}, {"深色 / Dark", &f.Dark}} {
			ui.Text(c, v.name)
			ui.Text(c, "背景 / Background (#RRGGBB)")
			ui.TextInput(c, &v.s.Background).FillWidth()
			ui.Text(c, "文字 / Foreground (#RRGGBB)")
			ui.TextInput(c, &v.s.Foreground).FillWidth()
			ui.Text(c, "强调色 / Accent (#RRGGBB)")
			ui.TextInput(c, &v.s.Accent).FillWidth()
		}
	}
	ui.Text(c, "字体与字号 / Fonts and sizes").Bold()
	ui.Text(c, "字体名称可逗号分隔；留空使用系统字体 / Comma-separated font fallback names; blank uses system font")
	for _, v := range []struct {
		name string
		f    *theme.FontPreference
	}{{"界面 / Interface", &e.Draft.Interface}, {"聊天 / Chat", &e.Draft.Chat}, {"代码 / Code", &e.Draft.Code}, {"输入框 / Composer", &e.Draft.Composer}, {"终端 / Terminal", &e.Draft.Terminal}} {
		ui.Text(c, v.name)
		ui.TextInput(c, &v.f.Family).FillWidth()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "−").Disabled(v.f.Size <= 8).Clicked() {
				v.f.Size--
			}
			ui.Text(c, strconv.FormatFloat(float64(v.f.Size), 'f', 0, 32)+" px")
			if ui.Button(c, "+").Disabled(v.f.Size >= 48).Clicked() {
				v.f.Size++
			}
			ui.Text(c, "预览 Aa 字体 0123").Font(v.f.Family).FontSize(v.f.Size)
		})
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "恢复默认字体 / Reset fonts").Clicked() {
			d := theme.Defaults()
			e.Draft.Interface = d.Interface
			e.Draft.Chat = d.Chat
			e.Draft.Code = d.Code
			e.Draft.Composer = d.Composer
			e.Draft.Terminal = d.Terminal
		}
		if ui.Button(c, "预览 / Preview").Clicked() {
			e.ApplyPreview()
			c.Invalidate()
		}
		if ui.Button(c, "取消预览 / Cancel").Clicked() {
			e.Cancel()
			c.Invalidate()
		}
		if ui.Button(c, "保存主题与字体 / Save themes and fonts").Clicked() {
			e.Save()
			c.Invalidate()
		}
	})
	if e.Status != "" {
		ui.Text(c, e.Status)
	}
}
