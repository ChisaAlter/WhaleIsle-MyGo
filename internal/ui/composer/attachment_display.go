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
)

type attachmentDisplay struct {
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
	var decoded image.Image
	switch strings.ToLower(filepath.Ext(a.Path)) {
	case ".svg":
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
		if format == "gif" {
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
func paintAttachmentRail(c *ui.Context, m Model, fg, muted ui.Color) {
	d := m.State
	if len(d.Attachments) > 0 {
		ui.ScrollHorizontal(c).Key("composer-attachment-rail").FillWidth().Height(210).Padding(0, 14).Children(func() {
			for i, a := range d.Attachments {
				i, a := i, a
				label := a.Name
				if label == "" {
					label = filepath.Base(a.Path)
				}
				card := ui.Column(c).Key(fmt.Sprintf("attachment:%d:%d:%s", i, a.PreparationID, a.Path)).Width(200).Height(196).Gap(3).Padding(5).Border(1, c.Theme().Border)
				seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
				seat.load(a, false, c.Services().Invalidate)
				remove := false
				card.Children(func() {
					ui.Text(c, label).FontSize(12).TextColor(fg).MaxLines(1)
					if seat.result == nil {
						ui.Text(c, "读取信息 / Reading…").FontSize(10)
					} else {
						v := seat.result
						if v.bitmap != nil {
							v.animate(c)
							preview := ui.Box(c).FillWidth().Height(48).Focusable().Label("Preview attachment " + label)
							preview.Children(func() { ui.Image(c, v.bitmap).Fill().Fit(ui.Contain) })
							if preview.Clicked() || preview.Shortcut(0, ui.KeyEnter) {
								seat.open = true
							}
						}
						typ := v.mediaType
						if typ == "" {
							typ = "file"
						}
						ui.Text(c, fmt.Sprintf("%s · %d bytes", typ, v.bytes)).FontSize(10).TextColor(muted).MaxLines(1)
						if v.err != nil {
							ui.Text(c, "预览不可用 / Preview unavailable").FontSize(10).TextColor(muted)
						}
					}
					ui.Text(c, attachmentDisplayStatus(d, i)).FontSize(11).TextColor(muted)
					if a.PrepareError != "" {
						ui.Text(c, a.PrepareError).FontSize(10).MaxLines(1)
						if m.OnRetryAttachment != nil && ui.Button(c, "重试上传 / Retry upload: "+label).FontSize(10).Disabled(d.Sending || d.AttachmentPreparing(i)).Clicked() {
							m.OnRetryAttachment(i)
						}
					}
					if ui.Button(c, label+" ×").Label("Remove attachment " + label).FontSize(10).Disabled(d.Sending).Clicked() {
						remove = true
					}
				})
				if remove {
					d.RemoveAttachment(i)
					break
				}
				paintAttachmentOriginal(c, seat, a, label)
			}
		})
	}
	if d.Error != "" {
		ui.Text(c, d.Error).FontSize(12).TextColor(muted).Padding(0, 14)
	}
}
func paintAttachmentOriginal(c *ui.Context, seat *attachmentDisplaySeat, a session.ComposerAttachment, label string) {
	if !seat.open {
		seat.full = nil
		return
	}
	if seat.full == nil {
		seat.full = &attachmentDisplaySeat{}
	}
	seat.full.load(a, true, c.Services().Invalidate)
	ui.DialogBase(c, &seat.open, func(backdrop, panel ui.Element) {
		_, height := c.Size()
		panel.Width(960).MaxWidthPercent(94).Height(max(float32(120), height*.85)).Padding(12).Background(c.Theme().Background).Children(func() {
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
		seat.full = nil
	}
}

func httpImageType(data []byte) string {
	typ := http.DetectContentType(data)
	if strings.HasPrefix(typ, "image/") {
		return typ
	}
	return ""
}
