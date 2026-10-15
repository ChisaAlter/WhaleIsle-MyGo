package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/workspacefs"
)

const maxImageBytes int64 = 16 << 20
const maxImagePixels = 32_000_000

type imagePreview struct {
	bitmap *ui.Bitmap
	vector *ui.SVG
	note   string
	err    error
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
	if imagepreview.Supports(path) && m.OnOpenDoc != nil {
		ui.Row(c).Gap(6).Padding(4, 12, 4, 12).Children(func() {
			if ui.Button(c, label("隔离兼容预览", "Isolated compatibility preview")).FontSize(11).Clicked() {
				full, err := workspacefs.JoinRoot(m.WorkspaceRoot, path)
				if err != nil {
					setFileStatus(m, err.Error())
				} else {
					m.OnOpenDoc(full, workspacefs.DocNone, true)
				}
			}
		})
		ui.Text(c, label("GIF 动画 / ICO / AVIF / SVG：先校验大小、尺寸和资源边界，再用无脚本、无网络的隔离图片页打开。", "GIF animation / ICO / AVIF / SVG: validate size, dimensions and resource bounds, then open an isolated image page with no scripts or network.")).FontSize(10).TextColor(muted).Padding(0, 12, 4, 12)
	}
	t := selectedTab(m)
	var preview *imagePreview
	if t != nil {
		preview = t.image
	}
	if preview == nil {
		preview = loadImagePreview(m.WorkspaceRoot, path)
		if t != nil {
			// Retain only the visible decoded image, not one bitmap per open tab.
			if w := tabsFor(m); w != nil {
				for _, other := range w.Open {
					other.image = nil
				}
			}
			t.image = preview
		}
	}
	if preview.err != nil {
		ui.Text(c, preview.err.Error()).FontSize(12).TextColor(muted).Padding(8, 12, 8, 12)
		ui.Text(c, label("可用“外部打开”查看此图片；不会自动启动其他应用。", "Use Open external to view this image. No other app is opened automatically.")).FontSize(11).TextColor(muted).Padding(4, 12, 4, 12)
		return
	}
	if preview.note != "" {
		ui.Text(c, preview.note).FontSize(10).TextColor(muted).Padding(4, 12, 4, 12)
	}
	if preview.bitmap != nil {
		ui.Image(c, preview.bitmap).Fill().Grow(1).MinHeight(0).Fit(ui.Contain)
	} else if preview.vector != nil {
		ui.Image(c, preview.vector).Fill().Grow(1).MinHeight(0).Fit(ui.Contain)
	}
}

func loadImagePreview(root, rel string) *imagePreview {
	p := &imagePreview{}
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
	switch ext {
	case ".svg":
		p.vector, p.err = ui.ParseSVG(data)
		p.note = label("SVG 原生预览；文字、嵌入图片与滤镜等元素可能省略。", "Native SVG preview; text, embedded images and filters may be omitted.")
		return p
	case ".avif":
		p.err = errors.New(label("当前原生解码器不支持 AVIF", "The native decoder does not support AVIF"))
		return p
	case ".ico":
		data, err = icoPNG(data)
		if err != nil {
			p.err = err
			return p
		}
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		p.err = err
		return p
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
		p.err = errors.New(label("图片尺寸超过安全预览限制", "Image dimensions exceed the preview limit"))
		return p
	}
	p.bitmap, p.err = ui.DecodeBitmap(data)
	if p.bitmap != nil {
		w, h := p.bitmap.Size()
		p.note = fmt.Sprintf("%d × %d", w, h)
		if ext == ".gif" {
			p.note += label(" · GIF 首帧", " · GIF first frame")
		}
	}
	return p
}

// Modern ICO files contain PNG frames. Choose the largest valid embedded PNG,
// without trusting directory offsets or starting an external decoder process.
func icoPNG(data []byte) ([]byte, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil, errors.New("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if count == 0 || count > (len(data)-6)/16 {
		return nil, errors.New("invalid ICO directory")
	}
	var selected []byte
	best := 0
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		size, offset := uint64(binary.LittleEndian.Uint32(entry[8:])), uint64(binary.LittleEndian.Uint32(entry[12:]))
		if offset > uint64(len(data)) || size > uint64(len(data))-offset || size < 8 {
			continue
		}
		frame := data[int(offset):int(offset+size)]
		if !bytes.HasPrefix(frame, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			continue
		}
		w, h := int(entry[0]), int(entry[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w*h > best {
			selected, best = frame, w*h
		}
	}
	if selected == nil {
		return nil, errors.New(label("此 ICO 没有原生解码器支持的 PNG 图层", "This ICO has no PNG frame supported by the native decoder"))
	}
	return selected, nil
}
