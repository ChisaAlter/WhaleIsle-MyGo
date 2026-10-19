package projects

import (
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/theme"
)

// VisualPagePreview contains only native presentation state. HTML and Host
// credentials never enter the projects view or its retained observer state.
type VisualPagePreview struct {
	Available, Loading            bool
	Error                         string
	InteractiveError              string
	Surface                       session.VisualReplySurface
	Bitmap                        *ui.Bitmap
	ContentHeight, CapturedHeight int
	Retry                         func()
}

func renderVisualPages(c *ui.Context, m Model, source string, pages []projectflow.DeliveryPageReference) {
	for _, page := range pages {
		page := page
		slot := ui.Column(c.Key("project-visual-" + source + "/" + page.CallID)).FillWidth().MinWidth(0).Gap(6)
		width := slot.Bounds().W // Public Bounds reports the previous frame in DIPs.
		slot.Draw(func(_ *ui.Painter, bounds ui.Rect) {
			if bounds.W != width {
				c.Invalidate() // Read the settled layout in the next build, never capture during paint.
			}
		})
		slot.Children(func() {
			ui.Text(c, page.Title).FontSize(theme.InterfaceSize(13)).FontWeight(600)
			ui.Text(c, page.File.Name).FontSize(theme.InterfaceSize(12)).Selectable()
			state := VisualPagePreview{}
			if m.VisualPage != nil {
				// The helper normalizes device-scale PNGs to CSS dimensions. DIPs
				// are the native CSS width here; multiplying by display DPI would
				// change the page's responsive breakpoint rather than its sharpness.
				state = m.VisualPage(source, page, int(math.Round(float64(width))))
			}
			if !state.Available {
				ui.Text(c, tr("打开原任务详情以预览已发布页面。", "Open the original request details to preview this published page.")).FontSize(theme.InterfaceSize(12))
				return
			}
			if state.Loading {
				ui.Text(c, tr("正在读取并生成页面预览…", "Reading and rendering the published page…")).FontSize(theme.InterfaceSize(12))
			}
			if state.Error != "" {
				ui.Text(c, state.Error).FontSize(theme.InterfaceSize(12)).Selectable()
			}
			if state.InteractiveError != "" {
				ui.Text(c, tr("交互页面不可用，正在使用静态预览：", "Interactive page unavailable; using a static preview: ")+state.InteractiveError).FontSize(theme.InterfaceSize(12)).Selectable()
			}
			if (state.Error != "" || state.InteractiveError != "") && state.Retry != nil && ui.Button(c, tr("重试页面预览", "Retry page preview")).Clicked() {
				state.Retry()
			}
			if state.Surface != nil {
				chat.PublishedPageSurface(c, state.Surface, m.ThemeID, m.SystemDark, float32(max(80, min(2000, page.Height))))
				return
			}
			if state.Bitmap != nil {
				height := float32(max(80, min(2000, page.Height)))
				ui.Image(c, state.Bitmap).FillWidth().Height(height).Fit(ui.Contain)
				if state.CapturedHeight < state.ContentHeight {
					ui.Text(c, tr("页面较长，预览未包含底部内容。", "The page is longer than this preview; the bottom is not shown.")).FontSize(theme.InterfaceSize(12))
				}
			}
		})
	}
}
