package composer

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// Child images keep their captured navigation owner in the caller. Only the
// shared InputBar presentation and original-image decoder belong here; these
// paths must never be copied into the root composer's attachment collection.
func paintChildAttachments(c *ui.Context, v ChildEditorModel, muted ui.Color) {
	if len(v.Images) == 0 {
		return
	}
	ui.ScrollHorizontal(c.Key("child-attachment-rail")).FillWidth().Height(66).
		Padding(2, 10, 0).Margin(0, 0, -6, 0).Gap(10).Children(func() {
		for _, path := range v.Images {
			a := session.ComposerAttachment{Path: path, Name: filepath.Base(path)}
			card := ui.Box(c.Key("child-image:"+path)).Label("Attachment "+a.Name).
				Width(64).Height(64).Radius(20).Border(.5, c.Theme().Border)
			seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
			seat.load(a, false, c.Services().Invalidate)
			card.Children(func() {
				preview := ui.Box(c).Fill().Radius(20).Clip().Focusable().Label("Preview attachment " + a.Name).Tooltip(a.Name)
				preview.Children(func() {
					if display := seat.result; display != nil && display.bitmap != nil {
						display.animate(c)
						ui.Image(c, display.bitmap).Fill().Fit(ui.Cover)
					} else {
						ui.Text(c, composerText("图片", "Image")).FontSize(12).TextColor(muted)
					}
				})
				if display := seat.result; display != nil {
					if display.err != nil {
						preview.Tooltip(display.err.Error())
					}
					if display.bitmap != nil && (preview.Clicked() || preview.Shortcut(0, ui.KeyEnter)) {
						seat.open = true
					}
				}
				remove := ui.Button(c, "×").Label("Remove attachment " + a.Name).
					Absolute().Top(4).Right(4).Width(18).Height(18).Padding(0).Radius(9).BorderWidth(0).
					Disabled(v.Draft.Sending || v.OnRemoveImage == nil)
				if !card.Hovered() && !card.FocusWithin() {
					remove.Opacity(0)
				}
				if remove.Clicked() && v.OnRemoveImage != nil {
					v.OnRemoveImage(path)
				}
			})
			paintAttachmentOriginal(c, seat, a, a.Name)
		}
	})
}
