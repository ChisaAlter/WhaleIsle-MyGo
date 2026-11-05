package composer

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
)

// attachmentErrorInk marks a failed attachment. theme.Colors carries no danger
// token yet, so the shell's error red is kept here for the composer's rail.
var attachmentErrorInk = ui.RGB(213, 73, 65)

// firstError prefers the staging error and falls back to the preview's.
func firstError(prepare string, v *attachmentDisplay) string {
	if prepare != "" {
		return prepare
	}
	if v != nil && v.err != nil {
		return v.err.Error()
	}
	return ""
}

type attachmentDisplay struct {
	svg       *svgimage.Seat
	bitmap    *ui.Bitmap
	motion    *attachmentMotion
	bytes     int64
	mediaType string
	err       error
}
type attachmentDisplaySeat struct {
	result             *attachmentDisplay
	pending            chan *attachmentDisplay
	open, originalSize bool
	full               *attachmentDisplaySeat
}

var attachmentDecodeSlots = make(chan struct{}, 2)

func readAttachmentDisplay(a session.ComposerAttachment, full ...bool) *attachmentDisplay {
	out := &attachmentDisplay{mediaType: a.MediaType}
	var data []byte
	var err error
	if a.Upload.ReceiptId != "" && a.ImageData == "" && !strings.HasPrefix(a.MediaType, "image/") {
		out.bytes = int64(a.Upload.File.Bytes)
		if out.mediaType == "" {
			out.mediaType = mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Path)))
		}
		return out
	}
	if a.ImageData != "" {
		if int64(base64.StdEncoding.DecodedLen(len(a.ImageData))) > imagepreview.MaxBytes+2 {
			out.err = errors.New("image exceeds safe preview byte budget")
			return out
		}
		data, err = base64.StdEncoding.DecodeString(a.ImageData)
		out.bytes = int64(len(data))
	} else {
		info, e := os.Stat(a.Path)
		if e != nil {
			out.err = e
			return out
		}
		out.bytes = info.Size()
		if !info.Mode().IsRegular() {
			out.err = errors.New("preview requires a regular file")
			return out
		}
		if out.mediaType == "" {
			out.mediaType = mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Path)))
		}
		f, e := os.Open(a.Path)
		if e != nil {
			out.err = e
			return out
		}
		defer f.Close()
		var header [512]byte
		n, e := f.Read(header[:])
		if e != nil && e != io.EOF {
			out.err = e
			return out
		}
		detected := httpImageType(header[:n])
		if detected != "" {
			out.mediaType = detected
		}
		if !strings.HasPrefix(out.mediaType, "image/") {
			return out
		}
		if info.Size() > imagepreview.MaxBytes {
			out.err = errors.New("image exceeds safe preview byte budget")
			return out
		}
		if _, e = f.Seek(0, 0); e != nil {
			out.err = e
			return out
		}
		data, err = io.ReadAll(io.LimitReader(f, imagepreview.MaxBytes+1))
	}
	if err != nil {
		out.err = err
		return out
	}
	if int64(len(data)) > imagepreview.MaxBytes {
		out.err = errors.New("image exceeds safe preview byte budget")
		return out
	}
	if imagepreview.IsAnimatedAVIF(data) {
		out.decodeAnimation(data, len(full) > 0 && full[0])
		return out
	}
	var decoded image.Image
	switch strings.ToLower(filepath.Ext(a.Path)) {
	case ".svg":
		if imagepreview.NeedsBrowserSVG(data) {
			out.svg, out.err = svgimage.NewSeat(data)
			return out
		}
		decoded, err = imagepreview.DecodeSVGContext(context.Background(), data)
	case ".ico":
		decoded, err = imagepreview.DecodeIcon(data)
	case ".avif":
		decoded, err = imagepreview.DecodeAVIFContext(context.Background(), data)
	default:
		cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
		if e != nil {
			out.err = e
			return out
		}
		if format == "gif" || imagepreview.IsAnimatedWebP(data) {
			out.decodeAnimation(data, len(full) > 0 && full[0])
			return out
		}
		if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width) > imagepreview.MaxPixels/int64(cfg.Height) {
			out.err = errors.New("image exceeds safe preview pixel budget")
			return out
		}
		// The existing bitmap decoder preserves JPEG EXIF orientation.
		if len(full) > 0 && full[0] {
			out.bitmap, out.err = ui.DecodeBitmap(data)
			return out
		}
		decoded, _, err = image.Decode(bytes.NewReader(data))
		if err == nil && format == "jpeg" {
			decoded = avatarimage.ApplyJPEGOrientation(decoded, data)
		}
	}
	if err != nil {
		out.err = err
		return out
	}
	if decoded == nil {
		out.err = errors.New("image unavailable")
		return out
	}
	out.bitmap = attachmentBitmap(decoded, len(full) > 0 && full[0])
	return out
}
func (v *attachmentDisplaySeat) load(a session.ComposerAttachment, full bool, notify func()) {
	if v.result != nil {
		return
	}
	if v.pending == nil {
		v.pending = make(chan *attachmentDisplay, 1)
		result := v.pending
		go func() {
			attachmentDecodeSlots <- struct{}{}
			out := readAttachmentDisplay(a, full)
			<-attachmentDecodeSlots
			result <- out
			notify()
		}()
	}
	select {
	case v.result = <-v.pending:
	default:
	}
}
func attachmentDisplayStatus(d *DraftState, i int) string {
	a := d.Attachments[i]
	if a.PrepareError != "" {
		return "失败 / Failed"
	}
	if d.AttachmentPreparing(i) {
		if a.TotalBytes > 0 {
			return fmt.Sprintf("%d%% · %d / %d bytes", 100*a.SentBytes/a.TotalBytes, a.SentBytes, a.TotalBytes)
		}
		return "上传中 / Uploading"
	}
	if d.Sending {
		return "发送中 / Sending"
	}
	if a.ImageData != "" || a.Upload.ReceiptId != "" {
		return "就绪 / Ready"
	}
	return "待上传 / Pending"
}

// Draft files share the original 64-DIP rail: files are 240 wide and images 64.
// Metadata reads and upload state never reserve an additional vertical block.
func paintAttachmentRail(c *ui.Context, m Model, fg, muted ui.Color) {
	d := m.State
	if len(d.Attachments) > 0 {
		ui.ScrollHorizontal(c).Key("composer-attachment-rail").FillWidth().Height(66).Padding(2, 10, 0).Margin(0, 0, -6, 0).Gap(10).Children(func() {
			for i, a := range d.Attachments {
				i, a := i, a
				label := a.Name
				if label == "" {
					label = filepath.Base(a.Path)
				}
				imageFile := a.ImageData != "" || strings.HasPrefix(a.MediaType, "image/") || strings.HasPrefix(mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Path))), "image/")
				width := float32(240)
				if imageFile {
					width = 64
				}
				card := ui.Row(c).Key(fmt.Sprintf("attachment:%d:%d:%s", i, a.PreparationID, a.Path)).Label("Attachment "+label).Width(width).Height(64).Gap(10).Radius(16).Border(.5, c.Theme().Border).AlignItems(ui.Center)
				if !imageFile {
					card.Padding(0, 12)
				}
				seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
				seat.load(a, false, c.Services().Invalidate)
				if seat.result != nil && strings.HasPrefix(seat.result.mediaType, "image/") {
					imageFile = true
					card.Width(64).Padding(0)
				}
				if imageFile {
					// The empty seat keeps a quiet placeholder fill while the
					// preview decodes, instead of flashing a white square.
					card.Radius(20).Background(muted.Alpha(0.12))
				}
				remove := false
				card.Children(func() {
					v := seat.result
					if imageFile {
						preview := ui.Box(c).Fill().Radius(20).Clip().Focusable().Label("Preview attachment " + label).Tooltip(label)
						preview.Children(func() {
							if v != nil && v.svg != nil {
								v.svg.Prepare(c, m.SVGImages)
								showClose := card.Hovered() || card.FocusWithin() || a.PrepareError != ""
								ui.Box(c).Fill().DrawOver(func(p *ui.Painter, r ui.Rect) {
									var holes []svgimage.Exclusion
									if showClose {
										holes = append(holes, svgimage.Exclusion{Bounds: ui.Rect{X: r.X + r.W - 22, Y: r.Y + 4, W: 18, H: 18}, Radius: 9})
									}
									if a.PrepareError != "" && m.OnRetryAttachment != nil {
										holes = append(holes, svgimage.Exclusion{Bounds: ui.Rect{X: r.X + 4, Y: r.Y + r.H - 22, W: 18, H: 18}, Radius: p.Theme().Radius})
									}
									v.svg.PaintClipped(p, r, ui.Cover, 20, holes, chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).InputMajor))
								})
							} else if v != nil && v.bitmap != nil {
								v.animate(c)
								ui.Image(c, v.bitmap).Fill().Fit(ui.Cover)
							} else {
								ui.Text(c, i18n.T(i18n.KeyComposerAttachmentImage)).FontSize(12).TextColor(muted)
							}
						})
						if v != nil && v.err != nil {
							preview.Tooltip(v.err.Error())
						}
						if v != nil && (v.bitmap != nil || v.svg != nil) && (preview.Clicked() || preview.Shortcut(0, ui.KeyEnter)) {
							seat.open = true
						}
					} else {
						ui.Text(c, "▤").Width(28).FontSize(22).TextColor(muted)
						body := ui.Column(c).Grow(1).MinWidth(0).Gap(0).Tooltip(label)
						if a.PrepareError != "" && m.OnRetryAttachment != nil {
							body.Focusable().Label("Retry upload: " + label).Tooltip(a.PrepareError)
							if !d.Sending && !d.AttachmentPreparing(i) && (body.Clicked() || body.Shortcut(0, ui.KeyEnter)) {
								m.OnRetryAttachment(i)
							}
						}
						body.Children(func() {
							ui.Text(c, label).FontSize(14).FixedLineHeight(22).TextColor(fg).MaxLines(1)
							meta := i18n.T(i18n.KeyComposerAttachmentPending)
							if d.AttachmentPreparing(i) {
								meta = attachmentDisplayStatus(d, i)
							} else if a.PrepareError != "" {
								meta = i18n.T(i18n.KeyComposerAttachmentFailed)
							} else if v != nil && v.err != nil {
								meta = i18n.T(i18n.KeyComposerAttachmentUnavailable)
								body.Tooltip(v.err.Error())
							} else if v != nil && a.Upload.ReceiptId != "" {
								meta = fmt.Sprintf("%s · %d bytes", strings.ToUpper(strings.TrimPrefix(filepath.Ext(label), ".")), v.bytes)
							}
							ui.Text(c, meta).FontSize(12).FixedLineHeight(15).TextColor(muted).MaxLines(1)
						})
					}
					// Contrast-fill discs sit on top of an unpredictable preview, so
					// both the retry and remove buttons use the same dark seat and a
					// glyph colour checked against it.
					discFill := ui.RGB(0, 0, 0).Alpha(0.55)
					discInk := chrome.EnsureContrast(ui.RGB(255, 255, 255), ui.RGB(0, 0, 0))
					failed := a.PrepareError != "" || v != nil && v.err != nil
					if failed {
						// The theme exposes no danger token yet; this is the shell's
						// error ink, kept in one place.
						card.Tooltip(firstError(a.PrepareError, v)).Border(.5, attachmentErrorInk)
						if imageFile && m.OnRetryAttachment != nil && !d.Sending && !d.AttachmentPreparing(i) &&
							ui.Button(c, "↻").Label("Retry upload: "+label).
								Tooltip(firstError(a.PrepareError, v)).
								Absolute().Bottom(4).Left(4).Width(18).Height(18).Padding(0).Radius(9).BorderWidth(0).
								Background(discFill).TextColor(discInk).FontSize(12).Clicked() {
							m.OnRetryAttachment(i)
						}
					}
					close := ui.Button(c, "×").Absolute().Top(4).Right(4).Width(18).Height(18).Padding(0).Radius(9).
						BorderWidth(0).Background(discFill).TextColor(discInk).FontSize(12).
						Label("Remove attachment " + label).Disabled(d.Sending)
					if !card.Hovered() && !card.FocusWithin() && a.PrepareError == "" && !failed {
						close.Opacity(0)
					}
					remove = close.Clicked()
				})
				if remove {
					if seat.result != nil && seat.result.svg != nil {
						seat.result.svg.Close()
					}
					if seat.full != nil && seat.full.result != nil && seat.full.result.svg != nil {
						seat.full.result.svg.Close()
					}
					d.RemoveAttachment(i)
					break
				}
				paintAttachmentOriginal(c, seat, a, label, m.SVGImages)
			}
		})
	}
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(12).TextColor(muted).Padding(0, 14)
	}
}
func paintAttachmentOriginal(c *ui.Context, seat *attachmentDisplaySeat, a session.ComposerAttachment, label string, owners ...*svgimage.Owner) {
	if !seat.open {
		if seat.full != nil && seat.full.result != nil && seat.full.result.svg != nil {
			seat.full.result.svg.Close()
		}
		seat.full = nil
		return
	}
	if seat.full == nil {
		seat.full = &attachmentDisplaySeat{}
	}
	seat.full.load(a, true, c.Services().Invalidate)
	ui.DialogBase(c, &seat.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		_, height := c.Size()
		panel.Width(960).MaxWidthPercent(94).Height(max(float32(120), height*.85)).Padding(16).Radius(20).Clip().
			Background(c.Theme().Background).Children(func() {
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, label).Grow(1).MaxLines(1)
				if ui.Button(c, "1:1 / Fit").Clicked() {
					seat.originalSize = !seat.originalSize
				}
				if ui.Button(c, "关闭预览 / Close preview").Clicked() {
					seat.open = false
				}
			})
			if seat.full.result == nil {
				ui.Text(c, "加载原图 / Loading original…")
				return
			}
			v := seat.full.result
			if v.svg != nil {
				var owner *svgimage.Owner
				if len(owners) > 0 {
					owner = owners[0]
				}
				v.svg.Prepare(c, owner)
				paint := func() { ui.Box(c).Fill().DrawOver(func(p *ui.Painter, r ui.Rect) { v.svg.Paint(p, r, ui.Contain) }) }
				if seat.originalSize {
					w, h := v.svg.Size()
					ui.ScrollBoth(c).Fill().Children(func() {
						ui.Box(c).Width(w).Height(h).DrawOver(func(p *ui.Painter, r ui.Rect) { v.svg.Paint(p, r, ui.Contain) })
					})
				} else {
					paint()
				}
				return
			}
			if v.bitmap == nil {
				ui.Text(c, "原图预览不可用 / Original preview unavailable")
				return
			}
			v.animate(c)
			if seat.originalSize {
				w, h := v.bitmap.Size()
				ui.ScrollBoth(c).Fill().Children(func() { ui.Image(c, v.bitmap).Width(float32(w)).Height(float32(h)) })
			} else {
				ui.Image(c, v.bitmap).Fill().Fit(ui.Contain)
			}
		})
	})
	if !seat.open {
		if seat.full != nil && seat.full.result != nil && seat.full.result.svg != nil {
			seat.full.result.svg.Close()
		}
		seat.full = nil
	}
}

func httpImageType(data []byte) string {
	if imagepreview.IsAnimatedAVIF(data) {
		return "image/avif"
	}
	typ := http.DetectContentType(data)
	if strings.HasPrefix(typ, "image/") {
		return typ
	}
	return ""
}
