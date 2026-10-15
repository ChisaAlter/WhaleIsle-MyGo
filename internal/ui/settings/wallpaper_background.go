package settings

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image"
	"os"
	"strings"
	"time"
	"whaleisle/internal/desktop/wallpapers"
)

type WallpaperEditor struct {
	Prefs                          wallpapers.Preferences
	Saved                          wallpapers.Preferences
	Save                           func(wallpapers.Preferences) error
	Apply                          func(string) error
	Run                            func(func() func())
	Directory                      string
	Source, Search, Category, Year string
	SourceName, SourceURL          string
	Items                          []wallpapers.Item
	Thumbs                         map[string]*ui.Bitmap
	Page, NextPage, LocalPage      int
	Busy                           bool
	Status                         string
	generation                     int
	previewSource                  image.Image
	preview                        *ui.Bitmap
	cropOpen                       bool
}

func NewWallpaperEditor(p wallpapers.Preferences, dir string, save func(wallpapers.Preferences) error, apply func(string) error, run func(func() func())) *WallpaperEditor {
	return &WallpaperEditor{Prefs: p.Clone(), Saved: p.Clone(), Directory: dir, Save: save, Apply: apply, Run: run, Source: "bing", Category: "100", Year: "最近", Page: 1, Thumbs: map[string]*ui.Bitmap{}}
}
func (g *WallpaperEditor) commit() bool {
	if err := g.Save(g.Prefs); err != nil {
		g.Status = err.Error()
		return false
	}
	g.Saved = g.Prefs.Clone()
	g.Status = "已保存背景和图库设置"
	return true
}
func (g *WallpaperEditor) Favorite(item wallpapers.Item) bool {
	for _, v := range g.Prefs.Favorites {
		if v.ID == item.ID && v.SourceID == item.SourceID {
			return true
		}
	}
	return false
}
func (g *WallpaperEditor) toggleFavorite(item wallpapers.Item) {
	out := append([]wallpapers.Item(nil), g.Prefs.Favorites...)
	found := false
	for i, v := range out {
		if v.ID == item.ID && v.SourceID == item.SourceID {
			out = append(out[:i], out[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		out = append(out, item)
	}
	g.Prefs.Favorites = out
	g.commit()
}
func (g *WallpaperEditor) load(page int) {
	if g.Busy {
		return
	}
	if g.Source == "收藏" {
		g.Items = nil
		for _, item := range g.Prefs.Favorites {
			if strings.Contains(strings.ToLower(item.Title+" "+item.ID), strings.ToLower(g.Search)) {
				g.Items = append(g.Items, item)
			}
		}
		g.LocalPage = 0
		g.NextPage = 0
		g.loadThumbs()
		return
	}
	var source wallpapers.Source
	for _, s := range g.Prefs.Sources {
		if s.ID == g.Source {
			source = s
		}
	}
	year := 0
	if g.Year != "最近" {
		fmt.Sscanf(g.Year, "%d", &year)
	}
	q := wallpapers.Query{Source: source, Search: g.Search, Categories: g.Category, Year: year, Page: page}
	g.generation++
	generation := g.generation
	g.Busy = true
	g.Status = "正在读取图库…"
	g.Run(func() func() {
		result := wallpapers.List(context.Background(), q)
		return func() {
			if generation != g.generation {
				return
			}
			g.Items = result.Items
			g.Page = page
			g.LocalPage = 0
			g.NextPage = result.NextPage
			g.Busy = false
			g.Status = result.Warning
			if len(g.Items) == 0 && g.Status == "" {
				g.Status = "没有找到壁纸"
			}
			g.loadThumbs()
		}
	})
}
func (g *WallpaperEditor) loadThumbs() {
	items := append([]wallpapers.Item(nil), g.visible()...)
	generation := g.generation
	g.Run(func() func() {
		images := map[string]*ui.Bitmap{}
		for _, item := range items {
			data, err := wallpapers.Fetch(context.Background(), item.ThumbURL, true)
			if err == nil {
				img, err := ui.DecodeBitmap(data)
				if err == nil {
					images[item.ThumbURL] = img
				}
			}
		}
		return func() {
			if generation != g.generation {
				return
			}
			for key, img := range images {
				g.Thumbs[key] = img
			}
		}
	})
}
func (g *WallpaperEditor) visible() []wallpapers.Item {
	start := min(g.LocalPage*12, len(g.Items))
	return g.Items[start:min(start+12, len(g.Items))]
}
func (g *WallpaperEditor) ImportFile(path string) {
	if g.Busy {
		return
	}
	g.importImage(func() ([]byte, error) { return os.ReadFile(path) })
}
func (g *WallpaperEditor) download(item wallpapers.Item) {
	if g.Busy {
		return
	}
	g.importImage(func() ([]byte, error) { return wallpapers.Fetch(context.Background(), item.ImageURL, true) })
}
func (g *WallpaperEditor) importImage(read func() ([]byte, error)) {
	g.Busy = true
	g.Status = "正在准备壁纸…"
	g.Run(func() func() {
		data, err := read()
		path := ""
		if err == nil {
			path, err = wallpapers.Import(g.Directory, data)
		}
		return func() {
			g.Busy = false
			if err != nil {
				g.Status = err.Error()
				return
			}
			g.Prefs.Original = path
			g.Prefs.Crop = wallpapers.DefaultCrop()
			g.openCrop()
			g.Status = "调整裁剪后应用壁纸"
		}
	})
}
func (g *WallpaperEditor) openCrop() {
	data, err := os.ReadFile(g.Prefs.Original)
	if err != nil {
		g.Status = err.Error()
		return
	}
	g.previewSource, err = wallpapers.Decode(data)
	if err != nil {
		g.Status = err.Error()
		return
	}
	g.cropOpen = true
	g.updatePreview()
}
func (g *WallpaperEditor) updatePreview() {
	if g.previewSource != nil {
		g.preview = ui.NewBitmap(wallpapers.Cropped(g.previewSource, g.Prefs.Crop))
	}
}
func (g *WallpaperEditor) applyCrop() {
	path, err := wallpapers.SaveCrop(g.Directory, g.Prefs.Original, g.Prefs.Crop)
	if err == nil {
		err = g.Apply(path)
	}
	if err != nil {
		g.Status = err.Error()
		return
	}
	if g.commit() {
		g.cropOpen = false
		g.Status = "壁纸已应用；原图副本保留，可重新裁剪"
	}
}
func wallpaperBackgroundPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g == nil {
		return
	}
	ui.Text(c, "壁纸图库与背景").Bold().TextColor(fg)
	ui.Text(c, g.Status).FontSize(11).TextColor(muted)
	sources := []string{"收藏"}
	for _, s := range g.Prefs.Sources {
		sources = append(sources, s.ID)
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Select(c, &g.Source, sources).Changed() {
			g.generation++
			g.Busy = false
			g.Items = nil
			g.LocalPage = 0
			g.NextPage = 0
		}
		ui.TextInput(c, &g.Search).Placeholder("搜索标题 / Wallhaven 关键词").FillWidth()
		if ui.Button(c, "查找 / 刷新").Disabled(g.Busy).Clicked() {
			g.load(1)
		}
	})
	if g.Source == "bing" {
		years := []string{"最近"}
		for year := time.Now().Year(); year >= 2010; year-- {
			years = append(years, fmt.Sprint(year))
		}
		ui.Select(c, &g.Year, years)
	}
	if g.Source == "wallhaven" {
		ui.Text(c, "分类：100 风景 / 010 动漫 / 001 人物；仅 SFW").FontSize(11)
		ui.Select(c, &g.Category, []string{"100", "010", "001"})
	}
	for _, item := range g.visible() {
		item := item
		ui.Row(c).Gap(10).Children(func() {
			if bmp := g.Thumbs[item.ThumbURL]; bmp != nil {
				ui.Image(c, bmp).Width(120).Height(74).Fit(ui.Cover)
			}
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, item.Title).TextColor(fg)
				ui.Text(c, item.Copyright).FontSize(11).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					label := "收藏"
					if g.Favorite(item) {
						label = "取消收藏"
					}
					if ui.Button(c, label).Clicked() {
						g.toggleFavorite(item)
						if g.Source == "收藏" {
							g.load(1)
						}
					}
					if ui.Button(c, "下载并裁剪").Disabled(g.Busy).Clicked() {
						g.download(item)
					}
				})
			})
		})
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "上一页").Disabled(g.Busy || (g.LocalPage == 0 && g.Page <= 1)).Clicked() {
			if g.LocalPage > 0 {
				g.LocalPage--
				g.loadThumbs()
			} else {
				g.load(g.Page - 1)
			}
		}
		ui.Text(c, fmt.Sprintf("第 %d 页 · %d 张", g.Page+g.LocalPage, len(g.Items))).FontSize(11)
		if ui.Button(c, "下一页").Disabled(g.Busy || ((g.LocalPage+1)*12 >= len(g.Items) && g.NextPage == 0)).Clicked() {
			if (g.LocalPage+1)*12 < len(g.Items) {
				g.LocalPage++
				g.loadThumbs()
			} else {
				g.load(g.NextPage)
			}
		}
	})
	ui.Text(c, "图库来源（自定义目录仅 HTTPS JSON，items 或 Bing images 格式）").FontSize(11)
	for sourceIndex := range g.Prefs.Sources {
		removed := false
		s := g.Prefs.Sources[sourceIndex]
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &g.Prefs.Sources[sourceIndex].Name).Label("来源名称")
			ui.Text(c, s.ID).FontSize(11)
			if s.Kind == "catalog" {
				ui.TextInput(c, &g.Prefs.Sources[sourceIndex].URL).Label("目录 URL")
			}
			if ui.Button(c, "保存来源").Clicked() {
				if s.Kind == "catalog" {
					if _, err := wallpapers.URL(g.Prefs.Sources[sourceIndex].URL); err != nil {
						g.Status = err.Error()
					} else {
						g.commit()
					}
				} else {
					g.commit()
				}
			}
			if ui.Button(c, "移除来源").Clicked() {
				out := []wallpapers.Source{}
				for _, v := range g.Prefs.Sources {
					if v.ID != s.ID {
						out = append(out, v)
					}
				}
				g.Prefs.Sources = out
				removed = true
				g.commit()
				if g.Source == s.ID {
					g.Source = "收藏"
					g.Items = nil
				}
			}
		})
		if removed {
			break
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c, &g.SourceName).Placeholder("来源名称")
		ui.TextInput(c, &g.SourceURL).Placeholder("https://…/catalog.json").Grow(1)
		if ui.Button(c, "添加来源").Clicked() {
			count := 0
			for _, s := range g.Prefs.Sources {
				if s.Kind == "catalog" {
					count++
				}
			}
			if _, err := wallpapers.URL(g.SourceURL); err != nil {
				g.Status = err.Error()
			} else if strings.TrimSpace(g.SourceName) == "" {
				g.Status = "请输入来源名称"
			} else if count >= 5 {
				g.Status = "最多 5 个自定义来源"
			} else {
				source := wallpapers.Source{ID: fmt.Sprintf("catalog-%d", time.Now().UnixNano()), Kind: "catalog", Name: strings.TrimSpace(g.SourceName), URL: strings.TrimSpace(g.SourceURL)}
				g.Prefs.Sources = append(append([]wallpapers.Source(nil), g.Prefs.Sources...), source)
				g.Source = source.ID
				if g.commit() {
					g.SourceName, g.SourceURL = "", ""
				}
			}
		}
		if ui.Button(c, "恢复内置来源").Clicked() {
			for _, s := range wallpapers.Sources() {
				found := false
				for _, v := range g.Prefs.Sources {
					if v.ID == s.ID {
						found = true
					}
				}
				if !found {
					g.Prefs.Sources = append(g.Prefs.Sources, s)
				}
			}
			g.commit()
		}
	})
	if g.Prefs.Original != "" && ui.Button(c, "重新裁剪原图").Clicked() {
		g.openCrop()
	}
	if g.cropOpen {
		if g.preview != nil {
			ui.Image(c, g.preview).Width(360).Height(210).Fit(ui.Contain)
		}
		changed := false
		for _, row := range []struct {
			label  string
			v      *float64
			lo, hi float64
		}{{"比例", &g.Prefs.Crop.Aspect, .25, 4}, {"缩放", &g.Prefs.Crop.Zoom, 1, 5}, {"水平位置", &g.Prefs.Crop.X, 0, 1}, {"垂直位置", &g.Prefs.Crop.Y, 0, 1}} {
			ui.Text(c, row.label).FontSize(11)
			if ui.Slider(c, row.v, row.lo, row.hi).Changed() {
				changed = true
			}
		}
		if changed {
			g.updatePreview()
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "应用裁剪壁纸").Clicked() {
				g.applyCrop()
			}
			if ui.Button(c, "取消裁剪").Clicked() {
				g.cropOpen = false
			}
		})
	}
	ui.Text(c, "背景效果").Bold()
	for _, row := range []struct {
		label  string
		v      *float64
		lo, hi float64
	}{{"壁纸模糊", &g.Prefs.Blur, 0, 100}, {"壁纸像素化", &g.Prefs.Pixelate, 0, 100}, {"玻璃不透明度", &g.Prefs.Glass, 0, 100}, {"终端不透明度", &g.Prefs.Terminal, 40, 100}, {"渐变速度", &g.Prefs.Speed, 20, 300}, {"渐变光团数量", &g.Prefs.Count, 1, 5}} {
		ui.Text(c, fmt.Sprintf("%s %.0f", row.label, *row.v)).FontSize(11)
		ui.StepSlider(c, row.v, row.lo, row.hi, 1)
	}
	if ui.Checkbox(c, &g.Prefs.Transparent, "透明主题").Changed() && g.Prefs.Transparent && g.Prefs.Blur < 20 {
		g.Prefs.Blur = 20
	}
	ui.Checkbox(c, &g.Prefs.HideSidebarMask, "隐藏侧栏遮罩")
	ui.Select(c, &g.Prefs.Effect, []string{"none", "gradient"})
	if ui.Select(c, &g.Prefs.Preset, []string{"default", "aurora", "sunset", "ocean", "sakura", "custom"}).Changed() && g.Prefs.Preset != "custom" {
		g.Prefs.Colors = wallpapers.PresetColors(g.Prefs.Preset)
	}
	ui.Select(c, &g.Prefs.Variant, []string{"orbs", "aurora", "chaos", "rays"})
	ui.Text(c, "渐变配色：底色 1/2 与光团 1–5（#rrggbb）；壁纸优先于动态背景").FontSize(11)
	for i := range g.Prefs.Colors {
		if ui.TextInput(c, &g.Prefs.Colors[i]).Label(fmt.Sprintf("背景颜色 %d", i+1)).Changed() {
			g.Prefs.Preset = "custom"
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "保存背景效果").Clicked() {
			g.commit()
		}
		if ui.Button(c, "撤销背景调整").Clicked() {
			g.Prefs = g.Saved.Clone()
		}
		if ui.Button(c, "重置背景效果").Clicked() {
			p := wallpapers.Defaults()
			p.Sources = g.Prefs.Sources
			p.Favorites = g.Prefs.Favorites
			p.Original = g.Prefs.Original
			p.Crop = g.Prefs.Crop
			g.Prefs = p
		}
	})
	ui.Text(c, "指针 WebGL 水波与全局按钮金属光泽暂未移植。").FontSize(11).TextColor(muted)
}
