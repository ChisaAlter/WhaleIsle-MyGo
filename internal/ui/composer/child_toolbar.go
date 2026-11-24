package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// The child uses InputBar's compact intake disclosure, without acquiring the
// root command/model/permission catalogs. Menu callbacks remain the captured
// child callbacks supplied to ChildEditor, including explicit paste intent.
func paintChildToolbar(c *ui.Context, v ChildEditorModel, send func()) {
	locked := v.Disabled || v.Draft.Sending
	row := ui.Row(c.Key("child-toolbar")).FillWidth().Gap(design.Sp(3)).AlignItems(ui.Center).Padding(design.Sp(0.5), design.Sp(2), design.Sp(1.5), design.Sp(2))
	open := ui.Local(row, "child-tools-open", func() bool { return false })
	if locked || (v.OnPickImages == nil && v.OnPasteImage == nil) {
		*open = false
	}
	row.Children(func() {
		toolsBtn := coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonSecondary,
			Size:     coss.ButtonIconSM,
			Icon:     coss.IconPlus,
			Label:    composerText("工具", "Tools"),
			Pressed:  *open,
			Disabled: locked || v.OnPickImages == nil && v.OnPasteImage == nil,
		}).Radius(design.RadiusFull).Tooltip(composerText("工具", "Tools"))
		var items []coss.MenuItem
		if v.OnPickImages != nil {
			items = append(items, coss.MenuItem{ID: "add-images", Label: composerText("添加图片", "Add images"), Disabled: locked})
		}
		if v.OnPasteImage != nil {
			items = append(items, coss.MenuItem{ID: "paste-image", Label: composerText("粘贴图片", "Paste image"), Disabled: locked})
		}
		if len(items) > 0 {
			switch coss.Menu(c, toolsBtn, open, items, coss.MenuOptions{
				Side:  coss.SideTop,
				Align: coss.AlignStart,
				Label: composerText("工具", "Tools"),
			}) {
			case "add-images":
				v.OnPickImages()
			case "paste-image":
				v.OnPasteImage(true)
			}
		}
		if coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonDefault,
			Size:     coss.ButtonIcon,
			Icon:     coss.IconArrowUp,
			Label:    i18n.T(i18n.KeyComposerSend),
			Disabled: locked || v.OnSend == nil || strings.TrimSpace(v.Draft.Text) == "" && len(v.Images) == 0,
		}).Radius(design.RadiusFull).Tooltip(i18n.T(i18n.KeyComposerSend)).Margin(0, 0, 0, ui.Auto).Clicked() {
			send()
		}
	})
}
