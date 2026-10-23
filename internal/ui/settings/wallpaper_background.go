package settings

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image"
	"image/png"
	"os"
	"reflect"
	"strings"
	"time"
	"whaleisle/internal/desktop/wallpapers"
	fonttheme "whaleisle/internal/ui/theme"
)

type WallpaperEditor struct {
	Prefs                          wallpapers.Preferences
	Saved                          wallpapers.Preferences
	Save                           func(wallpapers.Preferences) error
	Pending                        func() bool
	Apply                          func(string) error
	Run                            func(func() func())
	Directory                      string
	Source, Search, Category, Year string
	SourceName, SourceURL          string
	Items                          []wallpapers.Item
	Thumbs                         map[string]*ui.Bitmap
	Page, NextPage, LocalPage      int
	Busy                           bool
	Inline                         bool
	Status                         string
	generation                     int
	previewSource                  image.Image
	preview                        *ui.Bitmap
	cropOpen                       bool
}

func (g *WallpaperEditor) Cropping() bool { return g.cropOpen }

func (g *WallpaperEditor) RetireOwner() {
	g.generation++
	g.Busy = false
	g.cropOpen = false
	g.previewSource = nil
	g.preview = nil
	g.Items = nil
	g.Thumbs = map[string]*ui.Bitmap{}
}

func NewWallpaperEditor(p wallpapers.Preferences, dir string, save func(wallpapers.Preferences) error, apply func(string) error, run func(func() func())) *WallpaperEditor {
	return &WallpaperEditor{Prefs: p.Clone(), Saved: p.Clone(), Directory: dir, Save: save, Apply: apply, Run: run, Source: "bing", Category: "100", Year: "最近", Page: 1, Thumbs: map[string]*ui.Bitmap{}}
}
func (g *WallpaperEditor) commit() bool {
	if g.Save == nil {
		g.Status = providerCopy("背景设置为只读。", "Background settings are read-only.")
		return false
	}
	if err := g.Save(g.Prefs); err != nil {
		g.Status = err.Error()
		return false
	}
	g.Saved = g.Prefs.Clone()
	g.Status = "已保存背景和图库设置"
	if g.Pending != nil && g.Pending() {
		g.Status = providerCopy("正在保存背景设置…", "Saving background settings…")
	}
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
	g.generation++
	generation, inline := g.generation, g.Inline
	g.Busy = true
	g.Status = "正在准备壁纸…"
	g.Run(func() func() {
		data, err := read()
		path := ""
		var decoded image.Image
		if err == nil && inline {
			decoded, err = wallpapers.Decode(data)
		} else if err == nil {
			path, err = wallpapers.Import(g.Directory, data)
		}
		return func() {
			if generation != g.generation {
				return
			}
			g.Busy = false
			if err != nil {
				g.Status = err.Error()
				return
			}
			g.Prefs.Original = path
			g.Prefs.Crop = wallpapers.DefaultCrop()
			if decoded != nil {
				g.previewSource, g.cropOpen = decoded, true
				g.updatePreview()
			} else {
				g.openCrop()
			}
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
	path, err := "", error(nil)
	if g.Inline {
		var data bytes.Buffer
		err = png.Encode(&data, wallpapers.Cropped(g.previewSource, g.Prefs.Crop))
		path = "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
	} else {
		path, err = wallpapers.SaveCrop(g.Directory, g.Prefs.Original, g.Prefs.Crop)
	}
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
// wallpaperChoiceSelect shows localized labels while writing the renderer's
// stable enum IDs back into preferences.
func wallpaperChoiceSelect(c *ui.Context, current *string, label string, options [][2]string, disabled bool) bool {
	labels := make([]string, 0, len(options))
	selected := ""
	for _, option := range options {
		labels = append(labels, option[1])
		if option[0] == *current {
			selected = option[1]
		}
	}
	if selected == "" {
		selected = *current
		labels = append(labels, selected)
	}
	if !ui.Select(c, &selected, labels).Label(label).FillWidth().Disabled(disabled).Changed() {
		return false
	}
	for _, option := range options {
		if option[1] == selected {
			*current = option[0]
			return true
		}
	}
	return false
}

func wallpaperGalleryPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {	if g == nil {
		return
	}
	ui.Text(c, "壁纸图库与背景").Bold().TextColor(fg)
	ui.Text(c, g.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	labels := []string{providerCopy("收藏夹", "Favorites")}
	ids := []string{"收藏"}
	for _, s := range g.Prefs.Sources {
		name := strings.TrimSpace(s.Name)
		switch s.ID {
		case "bing":
			name = providerCopy("必应", "Bing")
		case "wallhaven":
			name = "Wallhaven"
		default:
			if name == "" {
				name = s.ID
			}
		}
		labels = append(labels, name)
		ids = append(ids, s.ID)
	}
	selectedSource := labels[0]
	for index, id := range ids {
		if id == g.Source {
			selectedSource = labels[index]
		}
	}
	ui.Row(c).Wrap().Gap(8).Children(func() {
		if ui.Select(c, &selectedSource, labels).Changed() {
			for index, label := range labels {
				if label == selectedSource {
					g.Source = ids[index]
					break
				}
			}
			g.generation++
			g.Busy = false
			g.Items = nil
			g.LocalPage = 0
			g.NextPage = 0
		}
		ui.TextInput(c, &g.Search).Placeholder(providerCopy("搜索标题或 Wallhaven 关键词", "Search titles or Wallhaven keywords")).FillWidth()
		if ui.Button(c, providerCopy("查找或刷新", "Find or refresh")).Disabled(g.Busy).Clicked() {
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
		ui.Text(c, providerCopy("分类：100 风景、010 动漫、001 人物；仅 SFW", "Categories: 100 scenery, 010 anime, 001 people; SFW only")).FontSize(fonttheme.InterfaceSize(11))
		ui.Select(c, &g.Category, []string{"100", "010", "001"})
	}
	for _, item := range g.visible() {
		item := item
		ui.Row(c).Wrap().Gap(10).Children(func() {
			if bmp := g.Thumbs[item.ThumbURL]; bmp != nil {
				ui.Image(c, bmp).Width(120).Height(74).Fit(ui.Cover)
			}
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, item.Title).FillWidth().TextColor(fg)
				ui.Text(c, item.Copyright).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Row(c).Wrap().Gap(8).Children(func() {
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
	ui.Row(c).Wrap().Gap(8).Children(func() {
		if ui.Button(c, "上一页").Disabled(g.Busy || (g.LocalPage == 0 && g.Page <= 1)).Clicked() {
			if g.LocalPage > 0 {
				g.LocalPage--
				g.loadThumbs()
			} else {
				g.load(g.Page - 1)
			}
		}
		ui.Text(c, fmt.Sprintf("第 %d 页 · %d 张", g.Page+g.LocalPage, len(g.Items))).FontSize(fonttheme.InterfaceSize(11))
		if ui.Button(c, "下一页").Disabled(g.Busy || ((g.LocalPage+1)*12 >= len(g.Items) && g.NextPage == 0)).Clicked() {
			if (g.LocalPage+1)*12 < len(g.Items) {
				g.LocalPage++
				g.loadThumbs()
			} else {
				g.load(g.NextPage)
			}
		}
	})
	ui.Text(c, "图库来源（自定义目录仅 HTTPS JSON，items 或 Bing images 格式）").FontSize(fonttheme.InterfaceSize(11))
	for sourceIndex := range g.Prefs.Sources {
		removed := false
		s := g.Prefs.Sources[sourceIndex]
		ui.Row(c).Wrap().Gap(8).Children(func() {
			ui.TextInput(c, &g.Prefs.Sources[sourceIndex].Name).Label("来源名称").Width(140).Shrink(0)
			ui.Text(c, s.ID).FontSize(fonttheme.InterfaceSize(11))
			if s.Kind == "catalog" {
				ui.TextInput(c, &g.Prefs.Sources[sourceIndex].URL).Label("目录 URL").Grow(1).MinWidth(200)
			}
			if ui.Button(c, "保存来源").Height(30).Padding(0, 10).Clicked() {
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
			if ui.Button(c, "移除来源").Height(30).Padding(0, 10).Clicked() {
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
	ui.Row(c).Wrap().Gap(8).Children(func() {
		ui.TextInput(c, &g.SourceName).Placeholder("来源名称").Width(140).Shrink(0)
		ui.TextInput(c, &g.SourceURL).Placeholder("https://…/catalog.json").Grow(1).MinWidth(200)
		if ui.Button(c, "添加来源").Height(30).Padding(0, 10).Clicked() {
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
			ui.Image(c, g.preview).Width(360).MaxWidthPercent(100).Height(210).Fit(ui.Contain)
		}
		changed := false
		for _, row := range []struct {
			label  string
			v      *float64
			lo, hi float64
		}{{"比例", &g.Prefs.Crop.Aspect, .25, 4}, {"缩放", &g.Prefs.Crop.Zoom, 1, 5}, {"水平位置", &g.Prefs.Crop.X, 0, 1}, {"垂直位置", &g.Prefs.Crop.Y, 0, 1}} {
			ui.Text(c, row.label).FontSize(fonttheme.InterfaceSize(11))
			if ui.Slider(c, row.v, row.lo, row.hi).Changed() {
				changed = true
			}
		}
		if changed {
			g.updatePreview()
		}
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if ui.Button(c, "应用裁剪壁纸").Clicked() {
				g.applyCrop()
			}
			if ui.Button(c, "取消裁剪").Clicked() {
				g.cropOpen = false
			}
		})
	}
}

func wallpaperBackgroundPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g == nil {
		return
	}
	before := g.Prefs.Clone()
	if g.Status != "" {
		ui.Text(c, g.Status).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if g.Busy {
		ui.Text(c, providerCopy("正在读取图库…", "Loading wallpaper library…")).FillWidth().TextColor(muted)
	}
	bodyDisclosure(c, "wallpaper-library", providerCopy("壁纸图库", "Wallpaper library"), func() { wallpaperGalleryPane(c, g, fg, muted) })
	bodyDisclosure(c, "background-effects", providerCopy("背景特效", "Background effects"), func() {
		bodyPreferenceRow(c, "background-effect-mode", providerCopy("动态背景", "Animated background"), providerCopy("壁纸优先于动态背景。", "Wallpaper takes precedence over the animated background."), fg, muted, func() {
			wallpaperChoiceSelect(c, &g.Prefs.Effect, providerCopy("动态背景", "Animated background"), [][2]string{{"none", providerCopy("无", "None")}, {"gradient", providerCopy("渐变", "Gradient")}}, g.Save == nil)
		})
		if g.Prefs.Effect == "gradient" {
			bodyPreferenceRow(c, "background-preset", providerCopy("配色方案", "Color preset"), "", fg, muted, func() {
				changed := wallpaperChoiceSelect(c, &g.Prefs.Preset, providerCopy("配色方案", "Color preset"), [][2]string{
					{"default", providerCopy("默认", "Default")}, {"aurora", providerCopy("极光", "Aurora")}, {"sunset", providerCopy("日落", "Sunset")},
					{"ocean", providerCopy("海洋", "Ocean")}, {"sakura", providerCopy("樱花", "Sakura")}, {"custom", providerCopy("自定义", "Custom")},
				}, g.Save == nil)
				if changed && g.Prefs.Preset != "custom" {
					g.Prefs.Colors = wallpapers.PresetColors(g.Prefs.Preset)
				}
			})
			bodyPreferenceRow(c, "background-variant", providerCopy("动画样式", "Animation style"), "", fg, muted, func() {
				wallpaperChoiceSelect(c, &g.Prefs.Variant, providerCopy("动画样式", "Animation style"), [][2]string{
					{"orbs", providerCopy("光团", "Orbs")}, {"aurora", providerCopy("极光", "Aurora")}, {"chaos", providerCopy("混沌", "Chaos")}, {"rays", providerCopy("射线", "Rays")},
				}, g.Save == nil)
			})
			wallpaperSlider(c, "background-speed", providerCopy("速度", "Speed"), &g.Prefs.Speed, 20, 300, g.Save == nil, fg, muted)
			wallpaperSlider(c, "background-count", providerCopy("光团数量", "Orb count"), &g.Prefs.Count, 1, 5, g.Save == nil, fg, muted)
			for i := range g.Prefs.Colors {
				bodyPreferenceRow(c, fmt.Sprintf("background-color-%d", i), fmt.Sprintf(providerCopy("颜色 %d", "Color %d"), i+1), "#RRGGBB", fg, muted, func() {
					if ui.TextInput(c, &g.Prefs.Colors[i]).FillWidth().Disabled(g.Save == nil).Changed() {
						g.Prefs.Preset = "custom"
					}
				})
			}
		}
	})
	pointerEffectsPane(c, g, fg, muted)
	bodyLocalToggle(c, "metallic-buttons", providerCopy("按钮悬停光泽", "Button hover sheen"), providerCopy("减少动态效果时显示静态光泽。", "Uses a static sheen when reduced motion is enabled."), &g.Prefs.MetallicButtons, g.Save == nil, fg, muted, nil)
	bodyHeading(c, providerCopy("玻璃与透明度", "Glass and transparency"), "", fg, muted)
	wallpaperSlider(c, "wallpaper-blur", providerCopy("壁纸模糊", "Wallpaper blur"), &g.Prefs.Blur, 0, 100, g.Save == nil, fg, muted)
	wallpaperSlider(c, "wallpaper-pixelate", providerCopy("壁纸像素化", "Wallpaper pixelation"), &g.Prefs.Pixelate, 0, 100, g.Save == nil, fg, muted)
	bodyPreferenceRow(c, "glass-opacity", providerCopy("玻璃不透明度", "Glass opacity"), "", fg, muted, func() {
		ui.Text(c, fmt.Sprintf("%.0f%%", g.Prefs.Glass)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.StepSlider(c, &g.Prefs.Glass, 40, 100, 5).FillWidth().Label(providerCopy("玻璃不透明度", "Glass opacity")).Disabled(g.Save == nil || g.Prefs.Transparent && g.Prefs.Original != "")
		if ui.Button(c, providerCopy("恢复默认", "Reset")).Disabled(g.Save == nil).Clicked() {
			g.Prefs.Glass = 80
		}
	})
	wallpaperSlider(c, "terminal-opacity", providerCopy("终端不透明度", "Terminal opacity"), &g.Prefs.Terminal, 40, 100, g.Save == nil, fg, muted)
	bodyLocalToggle(c, "transparent-theme", providerCopy("透明主题", "Transparent theme"), providerCopy("需要壁纸；模糊至少为 20。", "Requires wallpaper with a blur of at least 20."), &g.Prefs.Transparent, g.Save == nil, fg, muted, func(enabled bool) {
		if enabled && g.Prefs.Blur < 20 {
			g.Prefs.Blur = 20
		}
	})
	bodyLocalToggle(c, "sidebar-mask", providerCopy("隐藏侧栏遮罩", "Hide sidebar mask"), "", &g.Prefs.HideSidebarMask, g.Save == nil, fg, muted, nil)
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if ui.Button(c, providerCopy("保存背景效果", "Save background effects")).Disabled(g.Save == nil || g.Busy).Clicked() {
			g.commit()
		}
		if ui.Button(c, providerCopy("撤销背景调整", "Discard background changes")).Clicked() {
			g.Prefs = g.Saved.Clone()
		}
		if ui.Button(c, providerCopy("重置背景效果", "Reset background effects")).Disabled(g.Save == nil).Clicked() {
			p := wallpapers.Defaults()
			p.Sources = g.Prefs.Sources
			p.Favorites = g.Prefs.Favorites
			p.Original = g.Prefs.Original
			p.Crop = g.Prefs.Crop
			g.Prefs = p
		}
	})
	if !g.Busy && !reflect.DeepEqual(before, g.Prefs) {
		g.commit()
	}
}
func wallpaperSlider(c *ui.Context, key, label string, value *float64, lo, hi float64, disabled bool, fg, muted ui.Color) {
	bodyPreferenceRow(c, key, label, "", fg, muted, func() {
		ui.Text(c, fmt.Sprintf("%.0f", *value)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.StepSlider(c, value, lo, hi, 1).FillWidth().Label(label).Disabled(disabled)
	})
}
