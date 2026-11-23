package market

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/ui/i18n"
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
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketNoScreenshots), "No screenshots available.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
		marketScreenshot(c, image, state, field, muted, m.Media)
		ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "‹").Label(marketText(i18n.T(i18n.KeyW4MarketPreviousScreenshot), "Previous image")).Disabled(state.index == 0).Clicked() {
				state.index--
			}
			ui.Text(c, fmt.Sprintf("%d / %d", state.index+1, len(images))).FontSize(theme.InterfaceSize(12)).TextColor(fg)
			if ui.Button(c, "›").Label(marketText(i18n.T(i18n.KeyW4MarketNextScreenshot), "Next image")).Disabled(state.index == len(images)-1).Clicked() {
				state.index++
			}
			if ui.Button(c, marketText(i18n.T(i18n.KeyW4MarketOpenOriginalImage), "Open image")).Disabled(m.OnOpenURL == nil).Clicked() && m.OnOpenURL != nil {
				m.OnOpenURL(image)
			}
		})
	})
}

func marketScreenshot(c *ui.Context, source string, state *screenshotGalleryState, field, muted ui.Color, owner *media.Owner) {
	root := ui.Box(c.Key("market-screenshot:" + source)).Label(marketText(i18n.T(i18n.KeyW4MarketPluginScreenshot), "Plugin screenshot") + fmt.Sprintf(" %d", state.index+1)).FillWidth().MinWidth(0).AspectRatio(1.6).Radius(6).Clip().Background(field)
	h := requestImage(owner, source, 0, 0)
	if h == nil || h.State() == media.Failed {
		state.failed[source] = true
	}
	root.Children(func() {
		if state.failed[source] {
			ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketScreenshotFailedToLoad), "Image could not be loaded.")).Fill().Padding(16).TextColor(muted)
		} else if h.State() == media.Ready {
			showImage(c, h).Fill().Fit(ui.Contain)
		}
	})
}
