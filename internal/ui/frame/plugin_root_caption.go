package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// PluginRootCaption retains the native window owner when an elected extension
// replaces the application's navigation and body, rather than a conversation.
func PluginRootCaption(c *ui.Context, colors chrome.TitleBarColors) {
	controls := max(float32(112), c.TitleBar().Right+8)
	left := max(float32(20), c.TitleBar().Left+12)
	ui.Row(c.Key("plugin-root-caption")).Label("Plugin root native caption").
		FillWidth().Height(workspaceCaptionHeight).Shrink(0).
		Padding(workspaceCaptionInsetTop, controls, workspaceCaptionInsetBottom, left).
		AlignItems(ui.Center).Background(colors.Background).
		BorderWidth(0, 0, .5, 0).BorderColor(colors.Border).DragWindow().Children(func() {
		ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(13)).FontWeight(500).
			TextColor(colors.Foreground).MinWidth(0).NoWrap().Ellipsis("…")
		captionWindowButtons(c, colors.Muted)
	})
}
