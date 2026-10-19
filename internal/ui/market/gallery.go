package market

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
)

type screenshotGalleryState struct {
	sources string
	index   int
	failed  map[string]bool
}

// MarketDetails accepts public HTTPS images without URL credentials. Relative
// paths are never resolved against a README or the local filesystem.
func marketScreenshots(values []string) []string {
	images := []string{}
	for _, value := range values {
		parsed, err := url.Parse(strings.TrimSpace(value))
		if err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil {
			parsed.Scheme = "https"
			images = append(images, parsed.String())
			if len(images) == 12 {
				break
			}
		}
	}
	return images
}

func marketScreenshotGallery(c *ui.Context, m Model, item desktopmarket.CatalogItem, fg, muted, field ui.Color) {
	images := marketScreenshots(item.Screenshots)
	if len(images) == 0 {
		ui.Text(c, marketText("暂无截图", "No screenshots available.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		return
	}
	root := ui.Column(c.Key("market-screenshot-gallery:" + item.ID)).FillWidth().MinWidth(0).Gap(8)
	state := ui.Local(root, "gallery", func() screenshotGalleryState { return screenshotGalleryState{} })
	sources := strings.Join(images, "\x00")
	if state.sources != sources {
		*state = screenshotGalleryState{sources: sources, failed: map[string]bool{}}
	}
	image := images[state.index]
	root.Children(func() {
		marketScreenshot(c, image, state, field, muted, m.SVGImages)
		ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "‹").Label(marketText("上一张截图", "Previous image")).Disabled(state.index == 0).Clicked() {
				state.index--
			}
			ui.Text(c, fmt.Sprintf("%d / %d", state.index+1, len(images))).FontSize(theme.InterfaceSize(12)).TextColor(fg)
			if ui.Button(c, "›").Label(marketText("下一张截图", "Next image")).Disabled(state.index == len(images)-1).Clicked() {
				state.index++
			}
			if ui.Button(c, marketText("打开原图", "Open image")).Disabled(m.OnOpenURL == nil).Clicked() && m.OnOpenURL != nil {
				m.OnOpenURL(image)
			}
		})
	})
}

func marketScreenshot(c *ui.Context, source string, state *screenshotGalleryState, field, muted ui.Color, owners ...*svgimage.Owner) {
	root := ui.Box(c.Key("market-screenshot:" + source)).Label(marketText("插件截图", "Plugin screenshot") + fmt.Sprintf(" %d", state.index+1)).FillWidth().MinWidth(0).AspectRatio(1.6).Radius(6).Clip().Background(field)
	decoded := ui.Local(root, "decoded-image", func() readmeDecodedImage { return readmeDecodedImage{} })
	readmeRemoteImages.RLock()
	cache := readmeRemoteImages.cache
	readmeRemoteImages.RUnlock()
	if !state.failed[source] && cache != nil {
		status, path, _ := cache.State(source)
		switch status {
		case imgcache.StatusMissing:
			cache.Ensure(source, c.Services().Invalidate)
		case imgcache.StatusError:
			state.failed[source] = true
		case imgcache.StatusReady:
			if decoded.path != path {
				if decoded.svgSeat != nil {
					decoded.svgSeat.Close()
				}
				*decoded = readmeDecodedImage{path: path}
				decoded.load(path, c.Services().Invalidate)
			}
			decoded.poll(path)
			if decoded.svgSeat != nil {
				var owner *svgimage.Owner
				if len(owners) > 0 {
					owner = owners[0]
				}
				decoded.svgSeat.Prepare(c, owner)
			}
			if decoded.pending == nil && (decoded.width <= 0 || decoded.height <= 0) {
				state.failed[source] = true
			}
		}
	}
	if wait := decoded.advance(time.Now()); !state.failed[source] && wait > 0 {
		c.After(wait)
	}
	root.Children(func() {
		if state.failed[source] || cache == nil {
			ui.Text(c, marketText("截图加载失败", "Image could not be loaded.")).Fill().Padding(16).TextColor(muted)
		} else if decoded.svgSeat != nil {
			ui.Box(c).Fill().DrawOver(func(p *ui.Painter, r ui.Rect) { decoded.svgSeat.PaintClipped(p, r, ui.Contain, 6, nil, field) })
		} else if decoded.bitmap != nil {
			ui.Image(c, decoded.bitmap).Fill().Fit(ui.Contain)
		} else if decoded.svg != nil {
			ui.Image(c, decoded.svg).Fill().Fit(ui.Contain)
		}
	})
}
