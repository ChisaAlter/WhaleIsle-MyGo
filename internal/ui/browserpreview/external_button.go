package browserpreview

import (
	"github.com/egoist/mygo/ui"
)

// externalLinkIcon is the 16 DIP external-link glyph on the address bar's
// external-open button: a box with an arrow leaving it.
var externalLinkIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.4 3.6h-2.9v8.9h8.9V9.6" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/><path d="M9.6 3.5h2.9v2.9" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/><path d="M12.4 3.6L7.6 8.4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>`))

// externalOpenButton paints the address bar's 28×28 "open externally" control.
// It only exists while the tab holds a valid HTTP(S) URL, and it hands exactly
// that URL to the system browser — no Browser cookies or profile travel with it.
// open is injected so tests can observe the hand-off without a desktop backend.
func externalOpenButton(c *ui.Context, draft string, open func(string) error) (clicked bool, err error) {
	url := NormalizeURL(draft)
	if url == "" || open == nil {
		return false, nil
	}
	label := browserChromeLabel("在外部打开", "Open externally")
	button := ui.ButtonBase(c).Size(28, 28).Padding(6).Radius(6).BorderWidth(0).
		Background(ui.Transparent).Role(ui.RoleButton).Label(label).
		Tooltip(browserChromeLabel("用系统浏览器打开当前地址；不传递 Browser 分区 Cookie。", "Open the current address in the system browser; Browser profile cookies are not transferred."))
	button.Children(func() { ui.Icon(c, externalLinkIcon).Size(16, 16) })
	if !button.Clicked() {
		return false, nil
	}
	return true, open(url)
}
