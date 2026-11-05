package settings

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Feature dialogs share the installed modal surface, but keep their own draft
// and dismissal owner. The scroll area never includes the action footer.
func appearanceModal(c *ui.Context, open *bool, title string, width float32, busy bool, dismiss, body, footer func()) {
	ui.DialogBaseWithOptions(c, open, ui.DialogOptions{DismissDisabled: busy, OnDismiss: dismiss}, func(backdrop, panel ui.Element) {
		vw, vh := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(width, max(float32(0), vw-48)), 0).MaxHeight(min(float32(800), vh*.85)).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
				ui.Row(c).FillWidth().Padding(18, 24).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).FontWeight(500)
					if ui.Button(c, "×").Label(providerCopy("关闭", "Close")+title).Size(28, 28).Padding(0).Radius(14).BorderWidth(0).Background(ui.Transparent).Disabled(busy).Clicked() {
						dismiss()
					}
				})
				ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(80), vh*.85-140)).Padding(0, 24, 24, 24).Gap(16).Children(body)
				if footer != nil {
					ui.Row(c).FillWidth().Shrink(0).Padding(16, 24).Gap(8).Justify(ui.End).AlignItems(ui.Center).BorderWidth(.5, 0, 0, 0).BorderColor(c.Theme().Border).Children(footer)
				}
			})
		})
	})
}

func (g *WallpaperEditor) explicitWallpaper() string {
	if g.ExplicitImage != nil {
		return strings.TrimSpace(g.ExplicitImage())
	}
	if g.ImageSource != nil {
		return strings.TrimSpace(g.ImageSource())
	}
	return ""
}

func (g *WallpaperEditor) sourceKind() string {
	for _, source := range g.currentPreferences().Sources {
		if source.ID == g.Source {
			return source.Kind
		}
	}
	return ""
}

func wallpaperImagePreference(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	prefs := theme.RenderPreferences()
	id := prefs.Light
	if c.Theme().Dark {
		id = prefs.Dark
	}
	if prefs.Family(id).Wallpaper != "" {
		skin := g.currentPreferences().SkinWallpaperEnabled
		bodyLocalToggle(c, "skin-wallpaper", providerCopy("使用主题自带壁纸", "Use theme wallpaper"), "", &skin, g.Save == nil, fg, muted, func(value bool) {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.SkinWallpaperEnabled = value })
		})
	}
	row := bodyPreferenceRow(c, "appearance-wallpaper", providerCopy("背景图", "Wallpaper"), providerCopy("选一张图铺在整个界面后面。图要透过界面才能看见：玻璃透明度越高，图越被盖住。设好之后可以拉毛玻璃和像素化。", "Choose an image to sit behind the whole UI. It shows through the chrome: higher glass opacity hides more of it. Frost and pixelate sliders appear after it is set."), fg, muted, func() {
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if g.OnBrowse != nil && ui.Button(c, providerCopy("选择图片", "Choose image")).Disabled(g.Busy || g.Apply == nil).Clicked() {
				g.OnBrowse()
			}
			if ui.Button(c, providerCopy("壁纸图库", "Wallpaper library")).Disabled(g.Run == nil || g.Apply == nil).Clicked() {
				g.galleryOpen = true
				g.sourcesOpen = false
				g.filterKey = ""
			}
		})
	})
	settingsSearchTarget(c, "wallpaper-library", providerCopy("壁纸图库", "Wallpaper library"), row)
	if settingsSearchReveal(c, "wallpaper-library") {
		g.galleryOpen = true
	}
	path := g.explicitWallpaper()
	if path != "" {
		preview := ui.Column(c.Key("current-wallpaper-preview")).FillWidth().MinWidth(0).Gap(8)
		type imageState struct {
			path   string
			bitmap *ui.Bitmap
		}
		state := ui.Local(preview, "image", func() imageState { return imageState{} })
		if state.path != path && g.Run != nil {
			state.path = path
			state.bitmap = nil
			g.Run(func() func() {
				var data []byte
				var err error
				if strings.HasPrefix(path, "data:image/") {
					_, raw, _ := strings.Cut(path, ",")
					data, err = base64.StdEncoding.DecodeString(raw)
				} else {
					data, err = wallpapers.ReadImportFile(path)
				}
				var bitmap *ui.Bitmap
				if err == nil {
					bitmap, err = ui.DecodeBitmap(data)
				}
				return func() {
					if state.path == path && err == nil {
						state.bitmap = bitmap
					}
				}
			})
		}
		preview.Children(func() {
			if g.currentPreferences().Glass >= 90 {
				ui.Text(c, providerCopy("玻璃透明度偏高时背景图会被盖住。把玻璃透明度调低，图会更清楚。", "High glass opacity covers the wallpaper. Lower Glass opacity to let more of the image show through.")).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			}
			if state.bitmap != nil {
				ui.Image(c, state.bitmap).FillWidth().Height(180).Fit(ui.Cover).Radius(12).Clip()
			}
			wallpaperImmediateSlider(c, g, "wallpaper-blur", providerCopy("毛玻璃程度", "Frosted glass"), g.currentPreferences().Blur, 0, 100, 1, false, func(p *wallpapers.Preferences, v float64) { p.Blur = v }, fg, muted)
			wallpaperImmediateSlider(c, g, "wallpaper-pixelate", providerCopy("像素化程度", "Pixelation"), g.currentPreferences().Pixelate, 0, 100, 1, false, func(p *wallpapers.Preferences, v float64) { p.Pixelate = v }, fg, muted)
			ui.Row(c).Wrap().Gap(8).Children(func() {
				if ui.Button(c, providerCopy("重新裁剪", "Recrop")).Disabled(g.Busy || g.Apply == nil).Clicked() {
					g.openCrop()
				}
				if ui.Button(c, providerCopy("清除壁纸", "Clear wallpaper")).Disabled(g.Busy || g.Apply == nil).Clicked() {
					g.ClearImage()
				}
				if ui.Button(c, providerCopy("恢复默认", "Reset")).Disabled(g.Save == nil).Clicked() {
					wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.Blur = 0; p.Pixelate = 0 })
				}
			})
		})
	}
}

func wallpaperModals(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g.galleryOpen {
		g.galleryShown = true
		dismiss := func() {
			g.retire()
			g.galleryOpen = false
			g.sourcesOpen = false
			g.sourceEditOpen = false
			g.downloadItem = nil
			g.filterKey = ""
		}
		title := providerCopy("壁纸图库", "Wallpaper library")
		if g.sourcesOpen {
			title = providerCopy("图库来源", "Wallpaper sources")
		}
		appearanceModal(c, &g.galleryOpen, title, 880, g.cropBusy, dismiss, func() {
			if g.sourcesOpen {
				wallpaperSourcesBody(c, g, fg, muted)
			} else {
				wallpaperGalleryBody(c, g, fg, muted)
			}
		}, func() {
			if g.sourcesOpen {
				if ui.Button(c, providerCopy("返回图库", "Back to library")).Clicked() {
					g.sourcesOpen = false
					g.filterKey = ""
				}
				if ui.PrimaryButton(c, providerCopy("添加来源", "Add source")).Disabled(g.Save == nil).Clicked() {
					g.sourceDraft = wallpapers.Source{Kind: "catalog"}
					g.sourceEditingID = ""
					g.sourceError = ""
					g.sourceEditOpen = true
				}
			} else {
				if ui.Button(c, providerCopy("管理来源", "Manage sources")).Clicked() {
					g.sourcesOpen = true
				}
				if ui.Button(c, providerCopy("完成", "Done")).Clicked() {
					dismiss()
				}
			}
		})
	}
	wallpaperSourceEditModal(c, g, fg, muted)
	if item := g.downloadItem; item != nil {
		open := true
		dismiss := func() {
			if !g.Busy {
				g.downloadItem = nil
			}
		}
		appearanceModal(c, &open, providerCopy("使用这张壁纸？", "Use this wallpaper?"), 600, g.Busy, dismiss, func() {
			if bmp := g.Thumbs[item.ThumbURL]; bmp != nil {
				ui.Image(c, bmp).FillWidth().Height(280).Fit(ui.Contain).Radius(12)
			}
			ui.Text(c, item.Title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(fg)
			ui.Text(c, item.Copyright).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if g.Status != "" {
				ui.Text(c, g.Status).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		}, func() {
			if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(g.Busy).Clicked() {
				dismiss()
			}
			label := providerCopy("下载并裁剪", "Download and crop")
			if g.Busy {
				label = providerCopy("正在准备…", "Preparing…")
			}
			if ui.PrimaryButton(c, label).Disabled(g.Busy).Clicked() {
				g.download(*item)
			}
		})
	}
	wallpaperCropModal(c, g, fg, muted)
}

func wallpaperGalleryBody(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		for _, source := range g.currentPreferences().Sources {
			label := source.Name
			if label == "" {
				label = source.ID
			}
			if settingsPreferenceButton(c, label).Selected(g.Source == source.ID).Clicked() {
				g.Source = source.ID
			}
		}
		if settingsPreferenceButton(c, "☆ "+providerCopy("收藏", "Favorites")).Selected(g.Source == "收藏").Clicked() {
			g.Source = "收藏"
		}
	})
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		ui.TextInput(c, &g.Search).FillWidth().Placeholder(providerCopy("搜索壁纸", "Search wallpapers")).Label(providerCopy("搜索壁纸", "Search wallpapers"))
		if ui.Button(c, providerCopy("刷新", "Refresh")).Disabled(g.Busy).Clicked() {
			g.load(1)
		}
	})
	switch g.sourceKind() {
	case "bing":
		ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
			if settingsPreferenceButton(c, providerCopy("今日", "Today")).Selected(g.Year == "最近").Clicked() {
				g.Year = "最近"
			}
			for year := time.Now().Year(); year > time.Now().Year()-8; year-- {
				value := fmt.Sprint(year)
				if settingsPreferenceButton(c, value).Selected(g.Year == value).Clicked() {
					g.Year = value
				}
			}
		})
	case "wallhaven":
		ui.Row(c).Wrap().Gap(6).Children(func() {
			for _, choice := range [][2]string{{"100", providerCopy("风景", "General")}, {"010", providerCopy("动漫", "Anime")}, {"001", providerCopy("人物", "People")}} {
				if settingsPreferenceButton(c, choice[1]).Selected(g.Category == choice[0]).Clicked() {
					g.Category = choice[0]
				}
			}
		})
	}
	key := g.Source + "\n" + g.Search + "\n" + g.Category + "\n" + g.Year
	if g.filterKey != key {
		g.filterKey = key
		g.load(1)
	}
	if g.Status != "" {
		ui.Text(c, g.Status).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if g.Busy {
		ui.Spinner(c).Size(24, 24)
	}
	grid := ui.Grid(c.Key("wallpaper-image-grid")).FillWidth().MinWidth(0).Gap(12)
	width := grid.Bounds().W
	grid.Draw(func(_ *ui.Painter, r ui.Rect) {
		if width != r.W {
			c.Invalidate()
		}
	})
	grid.Columns(max(1, int((width+12)/188))).Children(func() {
		for _, item := range g.visible() {
			item := item
			card := ui.Column(c.Key(item.SourceID+"/"+item.ID)).MinWidth(0).Radius(12).Clip().Background(c.Theme().Surface).Border(.5, c.Theme().Border)
			card.Children(func() {
				image := ui.ButtonBase(c).Label(item.Title).FillWidth().Height(118).Padding(0).BorderWidth(0).Clip().Disabled(g.Busy)
				image.Children(func() {
					if bmp := g.Thumbs[item.ThumbURL]; bmp != nil {
						ui.Image(c, bmp).Fill().Fit(ui.Cover)
					} else {
						ui.Box(c).Fill().Background(c.Theme().SurfaceHover).Center().Children(func() { ui.Icon(c, bodyIcon("image")).Size(24, 24).TextColor(muted) })
					}
				})
				if image.Clicked() {
					copy := item
					g.downloadItem = &copy
					g.Status = ""
				}
				ui.Row(c).FillWidth().MinWidth(0).Padding(8).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Title).Grow(1).MinWidth(0).SingleLine().Tooltip(item.Title + "\n" + item.Copyright).FontSize(theme.InterfaceSize(12))
					star := "☆"
					if g.Favorite(item) {
						star = "★"
					}
					if ui.Button(c, star).Label(providerCopy("收藏", "Favorite")+" "+item.Title).Size(28, 28).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(g.Save == nil || g.Pending != nil && g.Pending()).Clicked() {
						g.toggleFavorite(item)
						if g.Source == "收藏" {
							g.load(1)
						}
					}
				})
			})
		}
	})
	if len(g.Items) == 0 && !g.Busy && g.Status == "" {
		ui.Text(c, providerCopy("没有找到壁纸", "No wallpapers found")).FillWidth().Padding(24).TextColor(muted)
	}
	if (g.LocalPage+1)*12 < len(g.Items) {
		if ui.Button(c, providerCopy("加载更多", "Load more")).AlignSelf(ui.Center).Disabled(g.Busy).Clicked() {
			g.LocalPage++
			g.loadThumbs()
		}
	} else if g.NextPage > 0 {
		if ui.Button(c, providerCopy("加载更多", "Load more")).AlignSelf(ui.Center).Disabled(g.Busy).Clicked() {
			g.load(g.NextPage)
		}
	}
}

func wallpaperSourcesBody(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	for _, source := range g.currentPreferences().Sources {
		source := source
		bodyPreferenceRow(c, "wallpaper-source-"+source.ID, source.Name, source.URL, fg, muted, func() {
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, providerCopy("编辑", "Edit")).Disabled(g.Save == nil).Clicked() {
					g.sourceDraft = source
					g.sourceEditingID = source.ID
					g.sourceError = ""
					g.sourceEditOpen = true
				}
				if ui.Button(c, providerCopy("移除", "Remove")).Disabled(g.Save == nil).Clicked() {
					out := []wallpapers.Source{}
					for _, value := range g.currentPreferences().Sources {
						if value.ID != source.ID {
							out = append(out, value)
						}
					}
					if g.commitSources(out) && g.Source == source.ID {
						g.Source = "收藏"
						g.Items = nil
					}
				}
			})
		})
	}
	if ui.Button(c, providerCopy("恢复内置来源", "Restore built-in sources")).Disabled(g.Save == nil).Clicked() {
		sources := append([]wallpapers.Source(nil), g.currentPreferences().Sources...)
		for _, builtin := range wallpapers.Sources() {
			found := false
			for _, source := range sources {
				if source.Kind == builtin.Kind {
					found = true
				}
			}
			if !found {
				sources = append(sources, builtin)
			}
		}
		g.commitSources(sources)
	}
}

func wallpaperSourceEditModal(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if !g.sourceEditOpen {
		return
	}
	busy := g.Pending != nil && g.Pending()
	dismiss := func() { g.sourceEditOpen = false; g.sourceError = "" }
	title := providerCopy("添加来源", "Add source")
	if g.sourceEditingID != "" {
		title = providerCopy("编辑来源", "Edit source")
	}
	appearanceModal(c, &g.sourceEditOpen, title, 480, busy, dismiss, func() {
		bodyHeading(c, providerCopy("来源类型", "Source type"), "", fg, muted)
		selectKind := ui.SelectBase(c, &g.sourceDraft.Kind)
		label := g.sourceDraft.Kind
		switch label {
		case "catalog":
			label = providerCopy("自定义 JSON 目录", "Custom JSON catalog")
		case "bing":
			label = "Bing"
		case "wallhaven":
			label = "Wallhaven"
		}
		selectKind.Trigger.FillWidth().Padding(10, 12).Disabled(busy || g.sourceEditingID != "").Children(func() { ui.Text(c, label) })
		selectKind.Popup(func(panel ui.Element) {
			panel.Padding(4)
			for _, choice := range [][2]string{{"bing", "Bing"}, {"wallhaven", "Wallhaven"}, {"catalog", providerCopy("自定义 JSON 目录", "Custom JSON catalog")}} {
				occupied := false
				for _, source := range g.currentPreferences().Sources {
					if source.Kind == choice[0] && choice[0] != "catalog" {
						occupied = true
					}
				}
				selectKind.Item(choice[0]).FillWidth().Padding(10, 12).Disabled(occupied).Children(func() { ui.Text(c, choice[1]) })
			}
		})
		bodyHeading(c, providerCopy("来源名称", "Source name"), "", fg, muted)
		ui.TextInput(c, &g.sourceDraft.Name).FillWidth().Label(providerCopy("来源名称", "Source name")).Disabled(busy)
		if g.sourceDraft.Kind == "catalog" {
			bodyHeading(c, providerCopy("目录地址", "Catalog URL"), providerCopy("HTTPS JSON；支持 items 或 Bing images 格式。", "HTTPS JSON; supports items or Bing images format."), fg, muted)
			ui.TextInput(c, &g.sourceDraft.URL).FillWidth().Label("Catalog URL").Placeholder("https://…/catalog.json").Disabled(busy)
		}
		if g.sourceError != "" {
			ui.Text(c, g.sourceError).FillWidth().TextColor(c.Theme().Danger)
		}
	}, func() {
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(busy).Clicked() {
			dismiss()
		}
		if ui.PrimaryButton(c, providerCopy("保存", "Save")).Disabled(g.Save == nil || busy).Clicked() {
			source := g.sourceDraft
			source.Name = strings.TrimSpace(source.Name)
			source.URL = strings.TrimSpace(source.URL)
			if source.Name == "" {
				g.sourceError = providerCopy("请输入来源名称", "Enter a source name")
				return
			}
			if source.Kind == "catalog" {
				if _, err := wallpapers.URL(source.URL); err != nil {
					g.sourceError = err.Error()
					return
				}
			} else {
				source.URL = ""
			}
			sources := append([]wallpapers.Source(nil), g.currentPreferences().Sources...)
			if g.sourceEditingID != "" {
				for i := range sources {
					if sources[i].ID == g.sourceEditingID {
						source.ID = g.sourceEditingID
						sources[i] = source
					}
				}
			} else {
				if source.Kind != "catalog" {
					for _, existing := range sources {
						if existing.Kind == source.Kind {
							g.sourceError = providerCopy("此内置来源已存在", "This built-in source already exists")
							return
						}
					}
					source.ID = source.Kind
				} else {
					count := 0
					for _, existing := range sources {
						if existing.Kind == "catalog" {
							count++
						}
					}
					if count >= 5 {
						g.sourceError = providerCopy("最多 5 个自定义来源", "Up to 5 custom sources")
						return
					}
					source.ID = fmt.Sprintf("catalog-%d", time.Now().UnixNano())
				}
				sources = append(sources, source)
			}
			if !g.commitSources(sources) {
				g.sourceError = g.Status
				return
			}
			g.Source = source.ID
			g.filterKey = ""
			g.closeWhenSaved(&g.sourceEditOpen)
		}
	})
}

func wallpaperCropModal(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	vw, vh := c.Size()
	if vh > 0 {
		g.cropAspect = float64(vw / vh)
	}
	if !g.cropOpen {
		return
	}
	if g.cropDraft.Aspect != g.cropAspect {
		g.cropDraft.Aspect = g.cropAspect
		g.updatePreview()
	}
	busy := g.cropBusy || g.Pending != nil && g.Pending()
	dismiss := func() { g.retire(); g.cancelCrop() }
	appearanceModal(c, &g.cropOpen, providerCopy("裁剪壁纸", "Crop wallpaper"), 720, busy, dismiss, func() {
		ui.Text(c, providerCopy("拖动图片调整位置，滚轮缩放。裁剪比例跟随当前窗口。", "Drag to position, scroll to zoom. The crop follows the current window's aspect ratio.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if g.preview != nil {
			ui.Image(c, g.preview).Label(providerCopy("壁纸裁剪预览", "Wallpaper crop preview")).FillWidth().Height(min(float32(340), max(float32(140), (min(float32(720), vw-48)-48)/float32(g.cropDraft.Aspect)))).Fit(ui.Cover).Radius(12).Clip().HandleInput(func(e ui.InputEvent) bool {
				if busy {
					return false
				}
				switch e.Kind {
				case ui.InputPointerDown:
					if e.Button == 0 {
						g.dragging = true
						g.dragX, g.dragY = e.X, e.Y
						g.dragCrop = g.cropDraft
						return true
					}
				case ui.InputPointerMove:
					if g.dragging {
						g.cropDraft.X = max(0, min(1, g.dragCrop.X-float64(e.X-g.dragX)/float64(max(1, g.previewWidth))))
						g.cropDraft.Y = max(0, min(1, g.dragCrop.Y-float64(e.Y-g.dragY)/float64(max(1, g.previewHeight))))
						g.updatePreview()
						return true
					}
				case ui.InputPointerUp, ui.InputBlur:
					g.dragging = false
				case ui.InputScroll:
					g.cropDraft.Zoom = max(1, min(3, g.cropDraft.Zoom-float64(e.DY)*.01))
					g.updatePreview()
					return true
				}
				return false
			}).DrawOver(func(_ *ui.Painter, r ui.Rect) { g.previewWidth, g.previewHeight = r.W, r.H })
		}
		before := g.cropDraft.Zoom
		wallpaperSliderStep(c, "wallpaper-crop-zoom", providerCopy("缩放", "Zoom"), &g.cropDraft.Zoom, 1, 3, .05, busy, fg, muted)
		if before != g.cropDraft.Zoom {
			g.updatePreview()
		}
		if g.Status != "" {
			ui.Text(c, g.Status).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	}, func() {
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(busy).Clicked() {
			dismiss()
		}
		label := providerCopy("应用", "Apply")
		if busy {
			label = providerCopy("正在应用…", "Applying…")
		}
		if ui.PrimaryButton(c, label).Disabled(busy || g.previewSource == nil || g.Apply == nil).Clicked() {
			g.applyCrop()
		}
	})
}
