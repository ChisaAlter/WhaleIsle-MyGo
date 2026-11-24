package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/errcopy"

	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/workspacefs"
)

const maxImageBytes int64 = 32 << 20

// filesMedia is the process image service; tests substitute their own.
var filesMedia = media.Default

// imagePreview is a tab's image: the media key and source that make it, and
// what the latest frame read from the service. Build code never reads or
// decodes: the source's Load runs on a media worker, under the tab's lease.
type imagePreview struct {
	owner   imageReadOwner
	key     media.Key
	src     media.Source
	lease   *media.Owner
	handle  *media.Handle
	loadErr atomic.Pointer[error]

	loading bool
	bitmap  *ui.Bitmap
	vector  *ui.SVG
	note    string
	err     error
}

// newImagePreview prepares a request whose bytes come from read, run on a
// media worker. Read's own error (an authority or path refusal) is kept so
// the view can explain it; the media service only classifies.
func newImagePreview(owner imageReadOwner, id, path string, notify func(), read func(context.Context) ([]byte, error)) *imagePreview {
	p := &imagePreview{owner: owner, loading: true}
	p.key = media.Key{ID: id}
	p.src = media.Source{Kind: strings.ToLower(filepath.Ext(path)), MaxBytes: imageByteLimit(path), Load: func(ctx context.Context) ([]byte, error) {
		data, err := read(ctx)
		if err != nil {
			p.loadErr.Store(&err)
		}
		return data, err
	}}
	p.lease = filesMedia().NewOwner(nil, notify)
	return p
}

// release ends the preview's lease: a decode still waiting never runs.
func (p *imagePreview) release() {
	if p != nil && p.lease != nil {
		p.lease.Release()
	}
}

// refresh requests the picture (a cache hit after the first decode) and copies
// its state for this frame. It never decodes and never advances animation.
func (p *imagePreview) refresh() {
	p.sync(time.Time{})
}

// sync is refresh that also shows the animation frame due at now (zero: do not
// advance) and returns the wait until the next. Only the painted tab passes a
// time, so hidden animations do not advance.
func (p *imagePreview) sync(now time.Time) time.Duration {
	if p == nil || p.lease == nil {
		return 0
	}
	p.handle = filesMedia().Request(p.lease, p.key, p.src)
	p.loading, p.err, p.bitmap, p.vector = false, nil, nil, nil
	switch p.handle.State() {
	case media.Pending:
		p.loading = true
		return 0
	case media.Failed:
		if e := p.loadErr.Load(); e != nil {
			p.err = *e
		} else {
			p.err = p.handle.Err()
		}
		return 0
	}
	pic, _ := p.handle.Picture()
	var wait time.Duration
	if !now.IsZero() {
		pic.Bitmap, wait = p.handle.Frame(now)
	}
	p.bitmap, p.vector = pic.Bitmap, pic.SVG
	if p.bitmap != nil {
		w, h := p.bitmap.Size()
		p.note = fmt.Sprintf("%d × %d", w, h)
	} else if p.vector != nil {
		w, h := p.vector.Size()
		p.note = fmt.Sprintf("%.0f × %.0f", w, h)
	}
	return wait
}

// retainImagePreview installs the selected tab's preview. Other tabs keep
// theirs: their pictures stay in the shared cache and come back on revisit.
func retainImagePreview(m Model, selected *FileTab, preview *imagePreview) {
	if selected == nil {
		return
	}
	if selected.image != nil && selected.image != preview {
		selected.image.release()
	}
	selected.image = preview
}

func isPreviewImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif":
		return true
	}
	return false
}

// A pre-existing unsaved text draft (for example an SVG) always wins over
// image presentation. Switching a preview mode must never hide or erase it.
func imageMode(m Model, path string) bool {
	return isPreviewImage(path) && !(m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview)
}

func paintImagePreview(c *ui.Context, m Model, path string, muted ui.Color) {
	t := selectedTab(m)
	preview := readImagePreview(m, t)
	wait := preview.sync(time.Now())
	if preview.loading {
		ui.Text(c, label("正在加载图片…", "Loading image…")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Padding(8, 12, 8, 12)
		return
	}
	if preview.err != nil {
		ui.Text(c, errcopy.Sentence(preview.err)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Padding(8, 12, 8, 12)
		ui.Text(c, label("可用“外部打开”查看此图片；不会自动启动其他应用。", "Use Open external to view this image. No other app is opened automatically.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Padding(4, 12, 4, 12)
		return
	}
	if preview.note != "" {
		ui.Text(c, preview.note).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted).Padding(4, 12, 4, 12)
	}
	view := &imageViewState{}
	if t != nil {
		view = &t.imageView
	}
	if wait > 0 {
		c.After(wait)
	}
	paintImageView(c, preview, view)
}

// readWorkspaceImage reads one workspace image within the authorized root,
// on a media worker.
func readWorkspaceImage(ctx context.Context, root, rel string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	full, err := workspacefs.JoinRoot(root, rel)
	if err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	realFile, err := filepath.EvalSymlinks(full)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label("本地图片不可用", "Local image unavailable"), err)
	}
	within, err := filepath.Rel(realRoot, realFile)
	if err != nil || outsideRoot(within) {
		return nil, errors.New(label("图片路径超出当前工作区", "Image path is outside this workspace"))
	}
	file, err := os.Open(realFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New(label("图片路径不是普通文件", "Image path is not a regular file"))
	}
	limit := imageByteLimit(rel)
	if info.Size() > limit {
		return nil, errors.New(label("图片超过安全预览大小限制", "Image exceeds the preview size limit"))
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New(label("图片超过安全预览大小限制", "Image exceeds the preview size limit"))
	}
	return data, nil
}

func imageByteLimit(path string) int64 {
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return imagepreview.MaxSVGBytes
	}
	return maxImageBytes
}
