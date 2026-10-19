package files

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/workspacefs"
)

const maxImageBytes int64 = 16 << 20
const maxImagePixels = 32_000_000

type imagePreview struct {
	loading   bool
	bitmap    *ui.Bitmap
	vector    *ui.SVG
	animation *imagepreview.Animation
	frames    []*ui.Bitmap
	started   time.Time
	note      string
	err       error
}

// advance uses a paint-driven deadline: hidden or closed tabs schedule no new
// wakeups, and elapsed time skips missed frames without accumulating timers.
func (p *imagePreview) advance(now time.Time) time.Duration {
	if p.animation == nil || len(p.frames) == 0 {
		return 0
	}
	if p.started.IsZero() {
		p.started = now
	}
	index, wait := p.animation.FrameAt(now.Sub(p.started))
	p.bitmap = p.frames[index]
	return wait
}

// Release decoded pixels for inactive tabs without touching their zoom/pan state.
func retainImagePreview(m Model, selected *FileTab, preview *imagePreview) {
	if selected == nil {
		return
	}
	if w := tabsFor(m); w != nil {
		for _, tab := range w.Open {
			if tab != selected {
				tab.image = nil
			}
		}
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
	if preview.loading {
		ui.Text(c, label("正在加载图片…", "Loading image…")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Padding(8, 12, 8, 12)
		c.After(50 * time.Millisecond)
		return
	}
	if preview.err != nil {
		ui.Text(c, preview.err.Error()).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Padding(8, 12, 8, 12)
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
	if wait := preview.advance(time.Now()); wait > 0 {
		c.After(wait)
	}
	paintImageView(c, preview, view)
}

func loadImagePreview(root, rel string) *imagePreview {
	return loadImagePreviewContext(context.Background(), root, rel)
}

func loadImagePreviewContext(ctx context.Context, root, rel string) *imagePreview {
	p := &imagePreview{}
	if err := ctx.Err(); err != nil {
		p.err = err
		return p
	}
	full, err := workspacefs.JoinRoot(root, rel)
	if err != nil {
		p.err = err
		return p
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		p.err = err
		return p
	}
	realFile, err := filepath.EvalSymlinks(full)
	if err != nil {
		p.err = fmt.Errorf("%s: %w", label("本地图片不可用", "Local image unavailable"), err)
		return p
	}
	within, err := filepath.Rel(realRoot, realFile)
	if err != nil || outsideRoot(within) {
		p.err = errors.New(label("图片路径超出当前工作区", "Image path is outside this workspace"))
		return p
	}
	file, err := os.Open(realFile)
	if err != nil {
		p.err = err
		return p
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		p.err = err
		return p
	}
	if !info.Mode().IsRegular() {
		p.err = errors.New(label("图片路径不是普通文件", "Image path is not a regular file"))
		return p
	}
	limit := maxImageBytes
	ext := strings.ToLower(filepath.Ext(rel))
	if ext == ".svg" {
		limit = 1 << 20
	}
	if info.Size() > limit {
		p.err = errors.New(label("图片超过安全预览大小限制", "Image exceeds the preview size limit"))
		return p
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		p.err = err
		return p
	}
	if int64(len(data)) > limit {
		p.err = errors.New(label("图片超过安全预览大小限制", "Image exceeds the preview size limit"))
		return p
	}
	if err := ctx.Err(); err != nil {
		p.err = err
		return p
	}
	var decoded image.Image
	switch {
	case ext == ".gif" || imagepreview.IsAnimatedWebP(data):
		p.animation, p.err = imagepreview.DecodeAnimation(data)
		if p.err == nil {
			for _, frame := range p.animation.Frames {
				if err := ctx.Err(); err != nil {
					return &imagePreview{err: err}
				}
				p.frames = append(p.frames, ui.NewBitmap(frame))
			}
			// Keep timing metadata, but release the decoder staging canvases.
			p.animation.Frames = nil
			p.bitmap = p.frames[0]
		}
	case ext == ".svg":
		decoded, p.err = imagepreview.DecodeSVGContext(ctx, data)
	case ext == ".ico":
		decoded, p.err = imagepreview.DecodeIcon(data)
	case ext == ".avif":
		decoded, p.err = imagepreview.DecodeAVIFContext(ctx, data)
	default:
		config, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		if decodeErr != nil {
			p.err = decodeErr
			return p
		}
		if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
			p.err = errors.New(label("图片尺寸超过安全预览限制", "Image dimensions exceed the preview limit"))
			return p
		}
		p.bitmap, p.err = ui.DecodeBitmap(data)
	}
	if err := ctx.Err(); err != nil {
		return &imagePreview{err: err}
	}
	if decoded != nil && p.err == nil {
		p.bitmap = ui.NewBitmap(decoded)
	}
	if p.bitmap != nil {
		w, h := p.bitmap.Size()
		p.note = fmt.Sprintf("%d × %d", w, h)
	}
	return p
}
