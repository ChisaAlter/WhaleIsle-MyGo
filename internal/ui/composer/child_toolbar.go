package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// The child uses InputBar's compact intake disclosure, without acquiring the
// root command/model/permission catalogs. Menu callbacks remain the captured
// child callbacks supplied to ChildEditor, including explicit paste intent.
func paintChildToolbar(c *ui.Context, v ChildEditorModel, send func()) {
	locked := v.Disabled || v.Draft.Sending
	ui.Row(c.Key("child-toolbar")).FillWidth().Gap(12).AlignItems(ui.Center).Padding(2, 8, 6, 8).Children(func() {
		ui.Button(c, "+").Label(composerText("工具", "Tools")).Tooltip(composerText("工具", "Tools")).
			Width(28).Height(28).Padding(0).Radius(14).BorderWidth(0).
			Background(c.Theme().Surface).TextColor(c.Theme().Text).FontSize(18).
			Disabled(locked || v.OnPickImages == nil && v.OnPasteImage == nil).
			Menu(func(menu *ui.Menu) {
				if v.OnPickImages != nil && menu.Item(composerText("添加图片", "Add images")).Disabled(locked).Chosen() {
					v.OnPickImages()
				}
				if v.OnPasteImage != nil && menu.Item(composerText("粘贴图片", "Paste image")).Disabled(locked).Chosen() {
					v.OnPasteImage(true)
				}
			})
		if ui.Button(c, "↑").Label(i18n.T(i18n.KeyComposerSend)).Tooltip(i18n.T(i18n.KeyComposerSend)).
			Width(34).Height(34).Padding(0).Radius(17).BorderWidth(0).Top(-2).
			Margin(0, 0, 0, ui.Auto).Background(c.Theme().Accent).TextColor(c.Theme().AccentText).FontSize(16).
			Disabled(locked || v.OnSend == nil || strings.TrimSpace(v.Draft.Text) == "" && len(v.Images) == 0).Clicked() {
			send()
		}
	})
}
