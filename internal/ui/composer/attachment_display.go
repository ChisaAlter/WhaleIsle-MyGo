package composer

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// composerMedia is the process image service; tests substitute their own.
var composerMedia = media.Default

// firstError prefers the staging error and falls back to the preview's.
func firstError(prepare string, v *attachmentDisplay) string {
	if prepare != "" {
		return prepare
	}
	if err := v.failure(); err != nil {
		return errcopy.Sentence(err)
	}
	return ""
}

// attachmentDisplay is one attachment's metadata, read off the UI thread, and
// the media request for its pixels. Build code only reads the handle: the
// bytes are read, base64-decoded and decoded on media workers.
type attachmentDisplay struct {
	key       media.Key
	src       media.Source
	image     bool
	bytes     int64
	mediaType string
	err       error
	readErr   error

	handle *media.Handle
	bitmap *ui.Bitmap
	vector *ui.SVG
}
type attachmentDisplaySeat struct {
	result             *attachmentDisplay
	pending            chan *attachmentDisplay
	open, originalSize bool
	full               *attachmentDisplaySeat
}

var attachmentDecodeSlots = make(chan struct{}, 2)

// Thumbnails are decoded for the rail's slot, not at the source's full size.
const attachmentThumbW, attachmentThumbH = 240, 160

// readAttachmentDisplay reads metadata only (a stat and a 512-byte sniff) and
// prepares the pixel request. It runs on a goroutine, never in build code.
func readAttachmentDisplay(a session.ComposerAttachment, full ...bool) *attachmentDisplay {
	out := &attachmentDisplay{mediaType: a.MediaType}
	whole := len(full) > 0 && full[0]
	if a.Upload.ReceiptId != "" && a.ImageData == "" && !strings.HasPrefix(a.MediaType, "image/") {
		out.bytes = int64(a.Upload.File.Bytes)
		if out.mediaType == "" {
			out.mediaType = mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Path)))
		}
		return out
	}
	kind := strings.ToLower(filepath.Ext(a.Path))
	if a.ImageData != "" {
		if int64(base64.StdEncoding.DecodedLen(len(a.ImageData))) > imagepreview.MaxBytes+2 {
			out.err = errors.New("image exceeds safe preview byte budget")
			return out
		}
		out.bytes = int64(base64.StdEncoding.DecodedLen(len(a.ImageData)))
		encoded := a.ImageData
		out.image = true
		out.key = media.Key{ID: "attachment-b64:" + encoded}
		out.src = media.Source{Kind: kind, MaxBytes: imagepreview.MaxBytes, Load: func(context.Context) ([]byte, error) {
			return base64.StdEncoding.DecodeString(encoded)
		}}
		out.sized(whole)
		return out
	}
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
		out.mediaType = mime.TypeByExtension(kind)
	}
	f, e := os.Open(a.Path)
	if e != nil {
		out.err = e
		return out
	}
	var header [512]byte
	n, e := f.Read(header[:])
	f.Close()
	if e != nil && e != io.EOF {
		out.err = e
		return out
	}
	if detected := httpImageType(header[:n]); detected != "" {
		out.mediaType = detected
	}
	if !strings.HasPrefix(out.mediaType, "image/") {
		return out
	}
	if info.Size() > imagepreview.MaxBytes {
		out.err = errors.New("image exceeds safe preview byte budget")
		return out
	}
	out.image = true
	// A file's content identity is its path, size and modification time.
	out.key = media.Key{ID: "attachment-file:" + a.Path + "|" + strconv.FormatInt(info.Size(), 10) + "|" + strconv.FormatInt(info.ModTime().UnixNano(), 10)}
	out.src = media.Source{Path: a.Path, Kind: kind, MaxBytes: imagepreview.MaxBytes}
	out.sized(whole)
	return out
}

func (v *attachmentDisplay) sized(whole bool) {
	if !whole {
		v.key = v.key.Sized(attachmentThumbW, attachmentThumbH)
	}
}

// refresh reads this frame's picture from the media service; with a non-zero
// now it also shows the animation frame due and returns the wait to the next.
func (v *attachmentDisplay) refresh(owner *media.Owner, now time.Time) time.Duration {
	if v == nil || !v.image {
		return 0
	}
	v.handle = composerMedia().Request(owner, v.key, v.src)
	v.bitmap, v.vector = nil, nil
	switch v.handle.State() {
	case media.Failed:
		v.readErr = v.handle.Err()
		return 0
	case media.Pending:
		return 0
	}
	v.readErr = nil
	pic, _ := v.handle.Picture()
	var wait time.Duration
	if !now.IsZero() {
		pic.Bitmap, wait = v.handle.Frame(now)
	}
	v.bitmap, v.vector = pic.Bitmap, pic.SVG
	return wait
}

// failure is the metadata or decode failure to explain, nil when none.
func (v *attachmentDisplay) failure() error {
	if v == nil {
		return nil
	}
	if v.err != nil {
		return v.err
	}
	return v.readErr
}

// picture is the drawable source once ready.
func (v *attachmentDisplay) picture() ui.ImageSource {
	if v == nil {
		return nil
	}
	if v.bitmap != nil {
		return v.bitmap
	}
	if v.vector != nil {
		return v.vector
	}
	return nil
}

// animate shows the frame due now and schedules the next only while painted.
func (v *attachmentDisplay) animate(c *ui.Context, owner *media.Owner) {
	if wait := v.refresh(owner, time.Now()); wait > 0 {
		c.After(wait)
	}
}

func (v *attachmentDisplaySeat) load(a session.ComposerAttachment, full bool, owner *media.Owner, notify func()) {
	if v.result == nil {
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
	if v.result != nil {
		v.result.refresh(owner, time.Time{})
	}
}
func attachmentDisplayStatus(d *DraftState, i int) string {
	a := d.Attachments[i]
	if a.PrepareError != "" {
		return i18n.T(i18n.KeyComposerAttachmentFailedStatus)
	}
	if d.AttachmentPreparing(i) {
		if a.TotalBytes > 0 {
			return fmt.Sprintf("%d%% · %d / %d bytes", 100*a.SentBytes/a.TotalBytes, a.SentBytes, a.TotalBytes)
		}
		return i18n.T(i18n.KeyComposerAttachmentUploadingStatus)
	}
	if d.Sending {
		return i18n.T(i18n.KeyComposerAttachmentSendingStatus)
	}
	if a.ImageData != "" || a.Upload.ReceiptId != "" {
		return i18n.T(i18n.KeyComposerAttachmentReadyStatus)
	}
	return i18n.T(i18n.KeyComposerAttachmentPendingStatus)
}

// Draft files share the original 64-DIP rail: files are 240 wide and images 64.
// Metadata reads and upload state never reserve an additional vertical block.
func paintAttachmentRail(c *ui.Context, m Model, fg, muted ui.Color) {
	t := design.For(c)
	d := m.State
	paintFileIntake(c, d, muted)
	if len(d.Attachments) > 0 {
		ui.ScrollHorizontal(c).Key("composer-attachment-rail").FillWidth().Height(66).Padding(design.Sp(0.5), design.Sp(2.5), 0).Margin(0, 0, -design.Sp(1.5), 0).Gap(design.Sp(2.5)).Children(func() {
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
				card := ui.Row(c).Key(fmt.Sprintf("attachment:%d:%d:%s", i, a.PreparationID, a.Path)).Label("Attachment "+label).Width(width).Height(64).Gap(design.Sp(2.5)).Radius(design.RadiusXL2).Border(design.BorderWidth*0.5, t.Border).AlignItems(ui.Center)
				if !imageFile {
					card.Padding(0, design.Sp(3))
				}
				seat := ui.Local(card, "preview", func() attachmentDisplaySeat { return attachmentDisplaySeat{} })
				seat.load(a, false, m.Media, c.Services().Invalidate)
				if seat.result != nil && strings.HasPrefix(seat.result.mediaType, "image/") {
					imageFile = true
					card.Width(64).Padding(0)
				}
				if imageFile {
					// The empty seat keeps a quiet placeholder fill while the
					// preview decodes, instead of flashing a white square.
					card.Radius(design.RadiusXL2).Background(t.Muted)
				}
				remove := false
				card.Children(func() {
					v := seat.result
					if imageFile {
						preview := ui.Box(c).Fill().Radius(design.RadiusXL2).Clip().Focusable().Cursor(ui.CursorPointer).Label("Preview attachment " + label).Tooltip(label)
						preview.Children(func() {
							if v != nil {
								v.animate(c, m.Media)
							}
							if src := v.picture(); src != nil {
								ui.Image(c, src).Fill().Fit(ui.Cover)
							} else {
								ui.Text(c, i18n.T(i18n.KeyComposerAttachmentImage)).FontSize(design.TextXS.Size).TextColor(muted)
							}
						})
						if v.failure() != nil {
							preview.Tooltip(errcopy.Sentence(v.failure()))
						}
						if v != nil && v.picture() != nil && (preview.Clicked() || preview.Shortcut(0, ui.KeyEnter)) {
							seat.open = true
						}
					} else {
						ui.Box(c).Width(28).Center().Children(func() {
							cossIcon(c, coss.IconFile, design.IconLG, muted)
						})
						body := ui.Column(c).Grow(1).MinWidth(0).Gap(0).Tooltip(label)
						if a.PrepareError != "" && m.OnRetryAttachment != nil {
							body.Focusable().Cursor(ui.CursorPointer).Label("Retry upload: " + label).Tooltip(a.PrepareError)
							if !d.Sending && !d.AttachmentPreparing(i) && (body.Clicked() || body.Shortcut(0, ui.KeyEnter)) {
								m.OnRetryAttachment(i)
							}
						}
						body.Children(func() {
							ui.Text(c, label).FontSize(design.TextSM.Size).FixedLineHeight(22).TextColor(fg).MaxLines(1)
							meta := i18n.T(i18n.KeyComposerAttachmentPending)
							if d.AttachmentPreparing(i) {
								meta = attachmentDisplayStatus(d, i)
							} else if a.PrepareError != "" {
								meta = i18n.T(i18n.KeyComposerAttachmentFailed)
							} else if v.failure() != nil {
								meta = i18n.T(i18n.KeyComposerAttachmentUnavailable)
								body.Tooltip(errcopy.Sentence(v.failure()))
							} else if v != nil && a.Upload.ReceiptId != "" {
								meta = fmt.Sprintf("%s · %d bytes", strings.ToUpper(strings.TrimPrefix(filepath.Ext(label), ".")), v.bytes)
							}
							ui.Text(c, meta).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(muted).MaxLines(1)
						})
					}
					// Contrast-fill discs sit on top of an unpredictable preview, so
					// both the retry and remove buttons use the same dark seat and high-contrast ink.
					discFill := design.TW.Black.Alpha(0.55)
					discInk := design.TW.White
					failed := a.PrepareError != "" || v.failure() != nil
					if failed {
						card.Tooltip(firstError(a.PrepareError, v)).Border(design.BorderWidth*0.5, t.DestructiveForeground)
						if imageFile && m.OnRetryAttachment != nil && !d.Sending && !d.AttachmentPreparing(i) {
							retry := ui.ButtonBase(c).Label("Retry upload: " + label).
								Tooltip(firstError(a.PrepareError, v)).
								Absolute().Bottom(design.Sp(1)).Left(design.Sp(1)).Width(18).Height(18).Padding(0).
								Radius(design.RadiusFull).BorderWidth(0).Background(discFill).Cursor(ui.CursorPointer).Center()
							retry.Children(func() {
								cossIcon(c, coss.IconRefresh, design.Sp(3), discInk)
							})
							if retry.Clicked() {
								m.OnRetryAttachment(i)
							}
						}
					}
					close := ui.ButtonBase(c).Absolute().Top(design.Sp(1)).Right(design.Sp(1)).Width(18).Height(18).Padding(0).
						Radius(design.RadiusFull).BorderWidth(0).Background(discFill).Cursor(ui.CursorPointer).Center().
						Label("Remove attachment " + label).Disabled(d.Sending)
					close.Children(func() {
						cossIcon(c, coss.IconX, design.Sp(3), discInk)
					})
					if !card.Hovered() && !card.FocusWithin() && a.PrepareError == "" && !failed {
						close.Opacity(0)
					}
					remove = close.Clicked()
				})
				if remove {
					d.RemoveAttachment(i)
					break
				}
				paintAttachmentOriginal(c, seat, a, label, m.Media)
			}
		})
	}
	if d.Error != "" {
		ui.Text(c, FormatComposerError(d.Error)).FontSize(design.TextXS.Size).TextColor(muted).Padding(0, design.Sp(3.5))
	}
}
func paintAttachmentOriginal(c *ui.Context, seat *attachmentDisplaySeat, a session.ComposerAttachment, label string, owner *media.Owner) {
	if !seat.open {
		seat.full = nil
		return
	}
	if seat.full == nil {
		seat.full = &attachmentDisplaySeat{}
	}
	seat.full.load(a, true, owner, c.Services().Invalidate)
	coss.Dialog(c, &seat.open, coss.DialogOptions{
		Title:         label,
		MaxWidth:      960,
		NoClose:       true,
		FooterVariant: coss.FooterBare,
		Footer: func() {
			if coss.Button(c, "1:1 / Fit", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				seat.originalSize = !seat.originalSize
			}
			if coss.Button(c, i18n.T(i18n.KeyComposerAttachmentClosePreview), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM}).Clicked() {
				seat.open = false
			}
		},
	}, func() {
		_, height := c.Size()
		ui.Column(c).FillWidth().Height(max(float32(120), height*.65)).Children(func() {
			v := seat.full.result
			if v == nil || v.image && v.handle.State() == media.Pending {
				ui.Text(c, i18n.T(i18n.KeyComposerAttachmentLoadingOriginal))
				return
			}
			v.animate(c, owner)
			src := v.picture()
			if src == nil {
				ui.Text(c, i18n.T(i18n.KeyComposerAttachmentOriginalUnavailable))
				return
			}
			if seat.originalSize {
				w, h := v.handle.Size()
				ui.ScrollBoth(c).Fill().Children(func() { ui.Image(c, src).Width(w).Height(h) })
			} else {
				ui.Image(c, src).Fill().Fit(ui.Contain)
			}
		})
	})
	if !seat.open {
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
