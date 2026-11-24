package composer

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// Child images keep their captured navigation owner in the caller. Only the
// shared InputBar presentation and original-image decoder belong here; these
// paths must never be copied into the root composer's attachment collection.
func paintChildAttachments(c *ui.Context, v ChildEditorModel, muted ui.Color) {
	if len(v.Images) == 0 {
		return
	}
	t := design.For(c)
	ui.ScrollHorizontal(c.Key("child-attachment-rail")).FillWidth().Height(66).
		Padding(design.Sp(0.5), design.Sp(2.5), 0).Margin(0, 0, -design.Sp(1.5), 0).Gap(design.Sp(2.5)).Children(func() {
		for _, path := range v.Images {
			a := session.ComposerAttachment{Path: path, Name: filepath.Base(path)}
			card := ui.Box(c.Key("child-image:"+path)).Label("Attachment "+a.Name).
				Width(64).Height(64).Radius(design.RadiusXL2).Border(design.BorderWidth, t.Border).Background(t.Muted)
			seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
			seat.load(a, false, nil, c.Services().Invalidate)
			card.Children(func() {
				preview := ui.Box(c).Fill().Radius(design.RadiusXL2).Clip().Focusable().Cursor(ui.CursorPointer).Label("Preview attachment " + a.Name).Tooltip(a.Name)
				preview.Children(func() {
					if display := seat.result; display != nil && display.picture() != nil {
						display.animate(c, nil)
						ui.Image(c, display.picture()).Fill().Fit(ui.Cover)
					} else {
						ui.Text(c, composerText("图片", "Image")).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
					}
				})
				if display := seat.result; display != nil {
					if display.failure() != nil {
						preview.Tooltip(errcopy.Sentence(display.failure()))
					}
					if display.picture() != nil && (preview.Clicked() || preview.Shortcut(0, ui.KeyEnter)) {
						seat.open = true
					}
				}
				remove := ui.ButtonBase(c).Label("Remove attachment " + a.Name).
					Absolute().Top(design.Sp(1)).Right(design.Sp(1)).Width(18).Height(18).Padding(0).Radius(design.RadiusFull).
					Background(design.TW.Black.Alpha(0.55)).FocusRing(false).Cursor(ui.CursorPointer).
					Disabled(v.Draft.Sending || v.OnRemoveImage == nil).Children(func() {
					ui.Icon(c, coss.IconX.SVG()).Size(design.IconXS, design.IconXS).TextColor(design.TW.White).PassThrough()
				})
				if !card.Hovered() && !card.FocusWithin() {
					remove.Opacity(0)
				}
				if remove.Clicked() && v.OnRemoveImage != nil {
					v.OnRemoveImage(path)
				}
			})
			paintAttachmentOriginal(c, seat, a, a.Name, nil)
		}
	})
}
