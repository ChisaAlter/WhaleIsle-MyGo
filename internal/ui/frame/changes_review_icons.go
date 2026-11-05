package frame

import "github.com/egoist/mygo/ui"

var reviewChevronIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor"/></svg>`))
var reviewSplitIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><rect x="1.5" y="2.5" width="13" height="11" rx="1.5" stroke="currentColor"/><path d="M8 2.5v11M3.5 5.5h2.5m-2.5 2h2.5m4-2h2.5m-2.5 2h2.5" stroke="currentColor"/></svg>`))
var reviewInspectIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M9.5 1.5H3a1 1 0 0 0-1 1v11a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1V6L9.5 1.5Z" stroke="currentColor"/><path d="M9.5 1.5V6H14M5 9h6m-6 2.5h4" stroke="currentColor"/></svg>`))

// Same regular wrap glyphs used by the original document toolbar.
var reviewWrapIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M10.9999 8C10.9999 6.89543 10.1046 6 9 6H4.5V5H9C10.6568 5 11.9999 6.34315 11.9999 8C11.9999 9.65685 10.6568 11 9 11H6.20703L6.85351 11.6465L6.14648 12.3535L4.64652 10.8536C4.45126 10.6583 4.45126 10.3417 4.64652 10.1464L6.14648 8.64648L6.85351 9.35352L6.20703 10H9C10.1046 10 10.9999 9.10457 10.9999 8Z" fill="currentColor"/><path d="M2 15H1V1H2V15ZM15 15H14V1H15V15Z" fill="currentColor"/></svg>`))
var reviewNowrapIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M2 15H1V1H2V15ZM12.3535 7.64645C12.5487 7.84171 12.5487 8.15829 12.3535 8.35355L9.85352 10.8535L9.14648 10.1465L10.793 8.5H3.5V7.5H10.793L9.14648 5.85352L9.85352 5.14648L12.3535 7.64645ZM15 15H14V1H15V15Z" fill="currentColor"/></svg>`))

func reviewToolbarButton(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	button := ui.Button(c, "").Size(28, 28).Shrink(0).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Label(label).Tooltip(label)
	if button.Hovered() {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() { ui.Icon(c, icon).Size(16, 16).PassThrough() })
	return button
}
