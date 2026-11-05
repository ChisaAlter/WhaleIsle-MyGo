package settings

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image"
	"reflect"
	"strings"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

type WallpaperEditor struct {
	failure                                  string
	hostFailure                              string
	pendingPreferences                       *wallpapers.Preferences
	OnBrowse                                 func()
	ExplicitImage                            func() string
	galleryOpen, sourcesOpen, sourceEditOpen bool
	sourceDraft                              wallpapers.Source
	sourceEditingID                          string
	sourceError                              string
	cropBusy                                 bool
	pendingClose                             *bool
	finishCropOnAck                          bool

	Prefs, Saved                                          wallpapers.Preferences
	Save                                                  func(wallpapers.Preferences) error
	Pending                                               func() bool
	Apply                                                 func(string) error
	ImageSource                                           func() string
	EffectiveImage                                        func(wallpapers.Preferences) string
	Run                                                   func(func() func())
	Directory                                             string
	Source, Search, Category, Year, SourceName, SourceURL string
	Items                                                 []wallpapers.Item
	Thumbs                                                map[string]*ui.Bitmap
	Page, NextPage, LocalPage                             int
	Busy, Inline                                          bool
	Status                                                string
	generation                                            int
	cancel                                                context.CancelFunc
	operation                                             context.Context
	filterKey                                             string
	galleryShown                                          bool
	pointerShown                                          bool
	previewSource                                         image.Image
	preview                                               *ui.Bitmap
	cropOpen                                              bool
	cropDraft                                             wallpapers.Crop
	cropOriginal                                          string
	cropAspect                                            float64
	dragging                                              bool
	dragX, dragY                                          float32
	dragCrop                                              wallpapers.Crop
	previewWidth, previewHeight                           float32
	downloadItem                                          *wallpapers.Item
	pointerPreview                                        *ui.DecorativePreview
	backgroundPreview                                     *chrome.BackgroundEffectPreview
}

func (g *WallpaperEditor) Cropping() bool { return g.cropOpen }
func (g *WallpaperEditor) Dirty() bool    { return !reflect.DeepEqual(g.Prefs, g.currentPreferences()) }

// EffectivePreferences is the preview overlay for queued changes; Saved stays
// the last accepted document used for rollback and retry.
func (g *WallpaperEditor) EffectivePreferences() wallpapers.Preferences {
	return g.currentPreferences()
}
func (g *WallpaperEditor) currentPreferences() wallpapers.Preferences {
	if g.pendingPreferences != nil {
		return *g.pendingPreferences
	}
	return g.Saved
}
func (g *WallpaperEditor) retire() {
	g.generation++
	if g.cancel != nil {
		g.cancel()
		g.cancel = nil
	}
	g.Busy = false
	g.cropBusy = false
	g.operation = nil
}
func (g *WallpaperEditor) Hide() {
	g.retire()
	g.cancelCrop()
	g.Prefs = g.currentPreferences().Clone()
	g.filterKey = ""
	g.downloadItem = nil
	g.galleryOpen, g.sourcesOpen, g.sourceEditOpen = false, false, false
	g.pendingClose = nil
	g.finishCropOnAck = false
	if g.pointerPreview != nil {
		g.pointerPreview.Close()
		g.pointerPreview = nil
	}
	if g.backgroundPreview != nil {
		g.backgroundPreview.Close()
		g.backgroundPreview = nil
	}
}
func (g *WallpaperEditor) RetireOwner() {
	g.Hide()
	g.pendingPreferences = nil
	g.Items = nil
	g.Thumbs = map[string]*ui.Bitmap{}
}
func (g *WallpaperEditor) AcceptHost(p wallpapers.Preferences) {
	dirty := g.Dirty()
	g.pendingPreferences = nil
	g.Saved = p.Clone()
	if !dirty {
		g.Prefs = p.Clone()
	}
}
func NewWallpaperEditor(p wallpapers.Preferences, dir string, save func(wallpapers.Preferences) error, apply func(string) error, run func(func() func())) *WallpaperEditor {
	return &WallpaperEditor{Prefs: p.Clone(), Saved: p.Clone(), Directory: dir, Save: save, Apply: apply, Run: run, Source: "bing", Category: "100", Year: "最近", Page: 1, Thumbs: map[string]*ui.Bitmap{}}
}
func (g *WallpaperEditor) commit() bool {
	if g.Prefs.Effect != g.currentPreferences().Effect {
		g.Prefs.EffectExplicit = true
	}
	if g.Save == nil {
		g.Status = providerCopy("背景设置为只读。", "Background settings are read-only.")
		return false
	}
	if err := g.Save(g.Prefs); err != nil {
		g.Status = err.Error()
		g.failure = g.Status
		return false
	}
	if g.Pending != nil && g.Pending() {
		pending := g.Prefs.Clone()
		g.pendingPreferences = &pending
	} else {
		g.Saved = g.Prefs.Clone()
	}
	g.Status = ""
	return true
}

// Catalog actions save their own fields without committing an unfinished
// effects draft. Crop data likewise lives outside preferences until Apply.
func (g *WallpaperEditor) commitSources(sources []wallpapers.Source) bool {
	return g.changePreference(func(p *wallpapers.Preferences) { p.Sources = append([]wallpapers.Source(nil), sources...) })
}
func (g *WallpaperEditor) closeWhenSaved(open *bool) {
	if g.Pending != nil && g.Pending() {
		g.pendingClose = open
		return
	}
	*open = false
}
func (g *WallpaperEditor) AcceptHostResult(p wallpapers.Preferences, failure string) {
	g.AcceptHost(p)
	if failure != "" {
		g.Status = failure
		if g.hostFailure != failure {
			g.failure = failure
		}
		g.hostFailure = failure
		if g.pendingClose == &g.sourceEditOpen {
			g.sourceError = failure
		}
	} else {
		g.hostFailure = ""
		if g.pendingClose != nil {
			*g.pendingClose = false
		}
		if g.finishCropOnAck {
			g.cancelCrop()
		}
	}
	g.pendingClose = nil
	g.finishCropOnAck = false
}
func (g *WallpaperEditor) Favorite(item wallpapers.Item) bool {
	for _, v := range g.currentPreferences().Favorites {
		if v.ID == item.ID && v.SourceID == item.SourceID {
			return true
		}
	}
	return false
}
func (g *WallpaperEditor) toggleFavorite(item wallpapers.Item) {
	out := append([]wallpapers.Item(nil), g.currentPreferences().Favorites...)
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
	g.changePreference(func(p *wallpapers.Preferences) { p.Favorites = out })
}
func (g *WallpaperEditor) begin() (context.Context, int) {
	g.retire()
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.operation = ctx
	g.Busy = true
	return ctx, g.generation
}
func (g *WallpaperEditor) load(page int) {
	ctx, generation := g.begin()
	if g.Source == "收藏" {
		g.Items = nil
		for _, item := range g.currentPreferences().Favorites {
			if strings.Contains(strings.ToLower(item.Title+" "+item.ID), strings.ToLower(g.Search)) {
				g.Items = append(g.Items, item)
			}
		}
		g.LocalPage = 0
		g.Page = 1
		g.NextPage = 0
		g.Busy = false
		g.loadThumbs()
		return
	}
	var source wallpapers.Source
	for _, v := range g.currentPreferences().Sources {
		if v.ID == g.Source {
			source = v
		}
	}
	year := 0
	if g.Year != "最近" {
		fmt.Sscanf(g.Year, "%d", &year)
	}
	q := wallpapers.Query{Source: source, Search: g.Search, Categories: g.Category, Year: year, Page: page}
	g.Status = providerCopy("正在读取图库…", "Loading wallpaper library…")
	g.Run(func() func() {
		result := wallpapers.List(ctx, q)
		return func() {
			if generation != g.generation || ctx.Err() != nil {
				return
			}
			if page > 1 {
				g.Items = append(g.Items, result.Items...)
				g.LocalPage = max(0, (len(g.Items)-1)/12)
			} else {
				g.Items = result.Items
				g.LocalPage = 0
			}
			g.Page = page
			g.NextPage = result.NextPage
			g.Busy = false
			g.Status = result.Warning
			if len(g.Items) == 0 && g.Status == "" {
				g.Status = providerCopy("没有找到壁纸", "No wallpapers found")
			}
			g.loadThumbs()
		}
	})
}
func (g *WallpaperEditor) loadThumbs() {
	items := append([]wallpapers.Item(nil), g.visible()...)
	generation := g.generation
	ctx := g.operation
	if ctx == nil {
		ctx = context.Background()
	}
	g.Run(func() func() {
		images := map[string]*ui.Bitmap{}
		for _, item := range items {
			data, err := wallpapers.Fetch(ctx, item.ThumbURL, true)
			if err == nil {
				img, err := ui.DecodeBitmap(data)
				if err == nil {
					images[item.ThumbURL] = img
				}
			}
			if ctx.Err() != nil {
				break
			}
		}
		return func() {
			if generation != g.generation || ctx.Err() != nil {
				return
			}
			for key, img := range images {
				g.Thumbs[key] = img
			}
		}
	})
}
func (g *WallpaperEditor) visible() []wallpapers.Item {
	return g.Items[:min((g.LocalPage+1)*12, len(g.Items))]
}
func (g *WallpaperEditor) ImportFile(path string) {
	g.importImage(func(context.Context) ([]byte, error) { return wallpapers.ReadImportFile(path) })
}
func (g *WallpaperEditor) download(item wallpapers.Item) {
	g.importImage(func(ctx context.Context) ([]byte, error) { return wallpapers.Fetch(ctx, item.ImageURL, true) })
}
func (g *WallpaperEditor) importImage(read func(context.Context) ([]byte, error)) {
	ctx, generation := g.begin()
	inline := g.Inline
	g.Status = providerCopy("正在准备壁纸…", "Preparing wallpaper…")
	g.Run(func() func() {
		data, err := read(ctx)
		path := ""
		var decoded image.Image
		if err == nil {
			var encoded []byte
			decoded, encoded, err = wallpapers.PrepareImport(data)
			if err == nil && inline {
				path = wallpapers.ImageDataURL(encoded)
			}
		}
		if err == nil && !inline {
			path, err = wallpapers.Import(g.Directory, data)
		}
		return func() {
			if generation != g.generation || ctx.Err() != nil {
				return
			}
			g.Busy = false
			if err != nil {
				g.Status = err.Error()
				g.failure = g.Status
				return
			}
			g.cropOriginal = path
			g.cropDraft = wallpapers.DefaultCrop()
			if g.cropAspect > 0 {
				g.cropDraft.Aspect = g.cropAspect
			}
			g.downloadItem = nil
			g.galleryOpen, g.sourcesOpen = false, false
			g.filterKey = ""
			g.previewSource = decoded
			g.cropOpen = true
			g.updatePreview()
			g.Status = providerCopy("拖动或滚轮调整，应用后生效", "Drag or scroll to crop, then apply")
		}
	})
}
func (g *WallpaperEditor) openCrop() {
	path := g.currentPreferences().Original
	if path == "" && g.ImageSource != nil {
		path = g.ImageSource()
	}
	if path == "" {
		return
	}
	g.importImage(func(context.Context) ([]byte, error) {
		if strings.HasPrefix(path, "data:image/") {
			_, raw, ok := strings.Cut(path, ",")
			if !ok {
				return nil, fmt.Errorf("invalid wallpaper data")
			}
			return base64.StdEncoding.DecodeString(raw)
		}
		return wallpapers.ReadImportFile(path)
	})
}
func (g *WallpaperEditor) updatePreview() {
	if g.previewSource != nil {
		g.preview = ui.NewBitmap(wallpapers.Cropped(g.previewSource, g.cropDraft))
	}
}
func (g *WallpaperEditor) cancelCrop() {
	g.cropOpen = false
	g.previewSource = nil
	g.preview = nil
	g.cropOriginal = ""
	g.dragging = false
}
func (g *WallpaperEditor) ClearImage() {
	g.retire()
	g.cancelCrop()
	if g.Apply == nil {
		return
	}
	if err := g.Apply(""); err != nil {
		g.Status = err.Error()
		g.failure = g.Status
		return
	}
	g.changePreference(func(p *wallpapers.Preferences) {
		p.Original = ""
		p.Crop = wallpapers.DefaultCrop()
	})
}
func (g *WallpaperEditor) applyCrop() {
	if g.previewSource == nil || g.Apply == nil || g.cropBusy {
		return
	}
	source, crop, original, inline := g.previewSource, g.cropDraft, g.cropOriginal, g.Inline
	hadWallpaper := g.hasEffectiveWallpaper(g.currentPreferences())
	ctx, generation := g.begin()
	g.cropBusy = true
	g.Run(func() func() {
		data, err := wallpapers.EncodeJPEG(wallpapers.Cropped(source, crop))
		path := ""
		if err == nil {
			if inline {
				path = wallpapers.ImageDataURL(data)
			} else {
				path, err = wallpapers.Import(g.Directory, data)
			}
		}
		return func() {
			if generation != g.generation || ctx.Err() != nil {
				return
			}
			g.Busy, g.cropBusy = false, false
			if err == nil {
				err = g.Apply(path)
			}
			if err != nil {
				g.Status = err.Error()
				g.failure = g.Status
				return
			}
			if !g.changePreference(func(p *wallpapers.Preferences) {
				p.Original = original
				p.Crop = crop
				if !hadWallpaper && p.Transparent && p.Blur < 20 {
					p.Blur = 20
				}
			}) {
				return
			}
			if g.Pending != nil && g.Pending() {
				g.finishCropOnAck = true
			} else {
				g.cancelCrop()
			}
			g.Status = providerCopy("壁纸已应用", "Wallpaper applied")
		}
	})
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

func wallpaperBackgroundPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g == nil {
		return
	}
	g.galleryShown = false
	g.pointerShown = false
	wallpaperImagePreference(c, g, fg, muted)
	backgroundEffectPreference(c, g, fg, muted)
	pointerEffectsPane(c, g, fg, muted)
	metallic := g.currentPreferences().MetallicButtons
	bodyLocalToggle(c, "metallic-buttons", providerCopy("按钮悬停光泽", "Button sheen"), providerCopy("鼠标悬停在按钮上时叠加一层金属扫光；关闭后只保留原本的悬停效果。", "Overlay a metallic sheen sweep on hovered buttons; off keeps only the plain hover fill."), &metallic, g.Save == nil, fg, muted, func(value bool) {
		wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.MetallicButtons = value })
	})
	// AppearanceSection §glass: one 12 DIP block — heading + current percent,
	// the description, a full-width slider, a ghost reset, the terminal
	// sub-block, then the two switch-leading rows each owning a hint.
	prefs := g.currentPreferences()
	hasWallpaper := g.hasEffectiveWallpaper(prefs)
	glass := ui.Column(c.Key("appearance-glass")).FillWidth().MinWidth(0).Gap(12)
	settingsSearchTarget(c, "appearance-glass", providerCopy("玻璃透明度", "Glass opacity"), glass)
	glass.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, providerCopy("玻璃透明度", "Glass opacity")).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(16)).FixedLineHeight(fonttheme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
			ui.Text(c, fmt.Sprintf("%.0f%%", prefs.Glass)).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).FontFeatures("tnum").TextColor(muted)
		})
		ui.Text(c, providerCopy("控制界面表层的实心程度：菜单、对话框、输入框，以及设置背景图后的侧栏和聊天区域。数值越低越通透。有背景图时画布不会完全实心，图始终能透出一层。", "Controls how solid UI surfaces are: menus, dialogs, the composer, and — with a wallpaper set — the sidebar and chat area. Lower values are more see-through. With a wallpaper, the canvas never goes fully opaque, so the image always shows through a little.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		opacity := prefs.Glass
		if ui.StepSlider(c, &opacity, 40, 100, 5).FillWidth().Label(providerCopy("玻璃透明度", "Glass opacity")).Disabled(g.Save == nil || prefs.Transparent && hasWallpaper).Changed() {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.Glass = opacity })
		}
		if appearanceGhostButton(c, providerCopy("重置", "Reset"), g.Save == nil).Clicked() {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.Glass = 80 })
		}
		wallpaperImmediateSlider(c, g, "terminal-opacity", providerCopy("终端透明度", "Terminal opacity"), prefs.Terminal, 40, 100, 5, false, func(p *wallpapers.Preferences, v float64) { p.Terminal = v }, fg, muted)
		terminalHint := providerCopy("终端面板自己的实心程度，不跟随上面的玻璃滑杆。有背景图或流动渐变时，按这个比例透出背景。", "The terminal pane’s own solidity, independent of the glass slider above. With a wallpaper or the flowing gradient set, the backdrop shows through at this rate.")
		if !hasWallpaper && prefs.Effect != "gradient" {
			terminalHint = providerCopy("设置背景图或开启「流动渐变」背景特效后生效。", "Takes effect once a wallpaper or the flowing gradient backdrop effect is set.")
		} else if prefs.Terminal < 75 {
			terminalHint = providerCopy("低于 75% 时，只用加粗和颜色标选中行的终端程序可能看不清选中了哪一行。", "Below 75%, terminal apps that mark the selected row with only bold text and a color may make it hard to tell which row is selected.")
		}
		ui.Text(c, terminalHint).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		transparent := prefs.Transparent
		wallpaperBlockSwitch(c, "transparent-theme", providerCopy("透明主题", "Transparent theme"), &transparent, g.Save == nil, fg, func(value bool) {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.Transparent = value })
		})
		transparentHint := providerCopy("开启后界面表层完全透明：侧栏、输入框、菜单和对话框都不再有遮罩，壁纸色彩全量透出。文字可读性可以用「毛玻璃程度」调节；生效时若毛玻璃低于 20% 会自动提到 20%。", "Makes every UI surface fully transparent: the sidebar, composer, menus, and dialogs drop their masks so the wallpaper colors show through everywhere. Use the frosted-glass slider to keep text readable; taking effect raises a frosted glass below 20% to 20% automatically.")
		if transparent && !hasWallpaper {
			transparentHint = providerCopy("透明主题需要先设置背景图才会生效。", "The transparent theme takes effect after a wallpaper is set.")
		} else if transparent && prefs.Blur < 20 {
			transparentHint = providerCopy("毛玻璃程度低于 20%，文字会直接贴在清晰壁纸上，可能难以阅读。建议把「毛玻璃程度」调高。", "Frosted glass is below 20%, so text sits directly on the sharp wallpaper and may be hard to read. Raise the frosted-glass slider.")
		}
		ui.Text(c, transparentHint).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		mask := prefs.HideSidebarMask
		wallpaperBlockSwitch(c, "sidebar-mask", providerCopy("隐藏侧栏遮罩", "Hide sidebar mask"), &mask, g.Save == nil, fg, func(value bool) {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.HideSidebarMask = value })
		})
		ui.Text(c, providerCopy("开启后左侧栏不再叠加遮罩，与工作区底色保持一致，只保留中间的分割线。", "The left sidebar drops its extra surface fill and shares the workspace canvas color; only the divider between them stays.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
	})
	if !g.pointerShown && g.pointerPreview != nil {
		g.pointerPreview.Close()
		g.pointerPreview = nil
	}
	wallpaperModals(c, g, fg, muted)
	if g.failure != "" {
		c.AddToast(ui.Toast{ID: "appearance-preference", Title: g.failure, Type: "error"})
		g.failure = ""
	}
	if !g.galleryOpen && g.filterKey != "" {
		g.retire()
		g.filterKey = ""
		g.downloadItem = nil
	}
}
func wallpaperSlider(c *ui.Context, key, label string, value *float64, lo, hi float64, disabled bool, fg, muted ui.Color) {
	wallpaperSliderStep(c, key, label, value, lo, hi, 1, disabled, fg, muted)
}

func wallpaperSliderStep(c *ui.Context, key, label string, value *float64, lo, hi, step float64, disabled bool, fg, muted ui.Color) {
	bodyPreferenceRow(c, key, label, "", fg, muted, func() {
		ui.Text(c, fmt.Sprintf("%.0f", *value)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		ui.StepSlider(c, value, lo, hi, step).FillWidth().Label(label).Disabled(disabled)
	})
}

// appearanceGhostButton is the reference's borderless block action (重置 /
// 高级): transparent, 36 DIP tall, primary text, only a hover wash.
func appearanceGhostButton(c *ui.Context, label string, disabled bool) ui.Element {
	button := ui.Button(c, label).Height(36).Padding(0, 14).Radius(12).BorderWidth(0).Background(ui.Transparent).FontSize(fonttheme.InterfaceSize(14)).TextColor(c.Theme().Text).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	return button
}

// wallpaperBlockSwitch draws the reference's switch-leading row (.switchRow):
// the control sits left of its label inside a block, unlike bodyLocalToggle's
// label-first Setting-Cell.
func wallpaperBlockSwitch(c *ui.Context, key, label string, value *bool, disabled bool, fg ui.Color, changed func(bool)) {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0)
	settingsSearchTarget(c, key, label, row)
	row.Children(func() {
		ui.Row(c).MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Switch(c, value).Label(label).Disabled(disabled).Changed() && changed != nil {
				changed(*value)
			}
			ui.Text(c, label).MinWidth(0).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(fg)
		})
	})
}
