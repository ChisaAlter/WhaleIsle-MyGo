package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	xdraw "golang.org/x/image/draw"

	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/svgrender"
)

// rasterVersion changes whenever a rasteriser's output for the same input
// could change, invalidating the disk cache.
const rasterVersion = "resvg-0.46.0/1"

// animationBudget bounds one animation's frame cache; larger sequences are
// scaled down to fit rather than kept at full size.
const animationBudget = 48 << 20

// rasterHook observes resvg output (tests read the pixels back).
var rasterHook func([]byte)

type worker struct {
	svc    *Service
	raster *svgrender.Rasterizer
}

func newWorker(s *Service) *worker { return &worker{svc: s} }

func (w *worker) close() {
	if w.raster != nil {
		w.raster.Close()
		w.raster = nil
	}
}

func (w *worker) run(ctx context.Context, h *Handle) *result {
	if ctx.Err() != nil {
		return nil
	}
	data, fail := w.load(ctx, h.src)
	if ctx.Err() != nil {
		return nil
	}
	if fail != FailNone {
		return &result{fail: fail}
	}
	w.svc.decodes.Add(1)
	r := w.decode(ctx, h.key, h.src.Kind, data)
	if ctx.Err() != nil {
		return nil
	}
	return r
}

func (w *worker) load(ctx context.Context, src Source) ([]byte, Failure) {
	limit := src.MaxBytes
	if limit <= 0 {
		limit = imagepreview.MaxBytes
	}
	switch {
	case src.Bytes != nil:
		if int64(len(src.Bytes)) > limit {
			return nil, FailTooLarge
		}
		return src.Bytes, FailNone
	case src.DataURL != "":
		return decodeDataURL(src.DataURL, limit)
	case src.Load != nil:
		data, err := src.Load(ctx)
		if err != nil {
			return nil, classifyLoad(err)
		}
		if int64(len(data)) > limit {
			return nil, FailTooLarge
		}
		return data, FailNone
	case src.URL != "":
		if w.svc.cfg.Fetch == nil {
			return nil, FailUnavailable
		}
		path, err := w.svc.cfg.Fetch(ctx, src.URL)
		if err != nil {
			return nil, classifyLoad(err)
		}
		return readFile(path, limit)
	case src.Path != "":
		return readFile(src.Path, limit)
	}
	return nil, FailUnavailable
}

func classifyLoad(err error) Failure {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return FailCanceled
	case errors.Is(err, errTooLarge):
		return FailTooLarge
	}
	return FailUnavailable
}

var errTooLarge = errors.New("image exceeds the preview budget")

// ErrTooLarge lets a Source.Load report the byte budget.
var ErrTooLarge = errTooLarge

func readFile(path string, limit int64) ([]byte, Failure) {
	f, err := os.Open(path)
	if err != nil {
		return nil, FailUnavailable
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, FailUnavailable
	}
	if info.Size() > limit {
		return nil, FailTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, FailUnavailable
	}
	if int64(len(data)) > limit {
		return nil, FailTooLarge
	}
	return data, FailNone
}

func decodeDataURL(source string, limit int64) ([]byte, Failure) {
	if !strings.HasPrefix(source, "data:") {
		return nil, FailUnsupported
	}
	head, payload, ok := strings.Cut(source, ",")
	if !ok {
		return nil, FailCorrupt
	}
	if strings.HasSuffix(head, ";base64") {
		if int64(base64.StdEncoding.DecodedLen(len(payload))) > limit+2 {
			return nil, FailTooLarge
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, FailCorrupt
		}
		if int64(len(data)) > limit {
			return nil, FailTooLarge
		}
		return data, FailNone
	}
	if int64(len(payload)) > 3*limit {
		return nil, FailTooLarge
	}
	decoded, err := url.PathUnescape(payload)
	if err != nil {
		return nil, FailCorrupt
	}
	if int64(len(decoded)) > limit {
		return nil, FailTooLarge
	}
	return []byte(decoded), FailNone
}

// kind sniffs the format, preferring content over the hint.
func kind(data []byte, hint string) string {
	hint = strings.ToLower(hint)
	switch {
	case len(data) >= 4 && data[0] == 0 && data[1] == 0 && data[2] == 1 && data[3] == 0:
		return "ico"
	case len(data) >= 12 && string(data[4:8]) == "ftyp" && (bytes.Contains(data[8:min(len(data), 64)], []byte("avif")) || bytes.Contains(data[8:min(len(data), 64)], []byte("avis"))):
		return "avif"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "gif"
	case bytes.HasPrefix(data, []byte("\x89PNG")), bytes.HasPrefix(data, []byte("\xff\xd8")),
		bytes.HasPrefix(data, []byte("BM")), len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "bitmap"
	}
	if strings.HasSuffix(hint, ".svg") || strings.Contains(hint, "svg") || bytes.Contains(data[:min(len(data), 4096)], []byte("<svg")) {
		return "svg"
	}
	return "bitmap"
}

func (w *worker) decode(ctx context.Context, key Key, hint string, data []byte) *result {
	switch kind(data, hint) {
	case "svg":
		return w.decodeSVG(ctx, key, data)
	case "ico":
		return w.decodeCached(ctx, key, "ico", data, func() (image.Image, error) { return imagepreview.DecodeIcon(data) })
	case "avif":
		if imagepreview.IsAnimatedAVIF(data) {
			motion, err := imagepreview.DecodeAVIFAnimationContext(ctx, data)
			if err != nil {
				return &result{fail: decodeFailure(err)}
			}
			return animationResult(motion, key)
		}
		return w.decodeCached(ctx, key, "avif", data, func() (image.Image, error) { return imagepreview.DecodeAVIFContext(ctx, data) })
	case "gif":
		motion, err := imagepreview.DecodeAnimation(data)
		if err != nil {
			return &result{fail: decodeFailure(err)}
		}
		return animationResult(motion, key)
	}
	if imagepreview.IsAnimatedWebP(data) {
		motion, err := imagepreview.DecodeAnimation(data)
		if err != nil {
			return &result{fail: decodeFailure(err)}
		}
		return animationResult(motion, key)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return &result{fail: FailUnsupported}
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width) > imagepreview.MaxPixels/int64(cfg.Height) {
		return &result{fail: FailTooLarge}
	}
	if key.W > 0 && key.H > 0 && fitScale(cfg.Width, cfg.Height, key) < 1 {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return &result{fail: FailCorrupt}
		}
		if format == "jpeg" {
			img = avatarimage.ApplyJPEGOrientation(img, data)
		}
		return bitmapResult(fit(img, key))
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		return &result{fail: FailCorrupt}
	}
	return bitmapFromUI(bmp)
}

func decodeFailure(err error) Failure {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return FailCanceled
	case strings.Contains(err.Error(), "budget") || strings.Contains(err.Error(), "exceed"):
		return FailTooLarge
	}
	return FailCorrupt
}

func (w *worker) decodeSVG(ctx context.Context, key Key, data []byte) *result {
	info := ClassifySVG(data)
	switch info.Class {
	case SVGUnsupported:
		return &result{fail: FailUnsupported}
	case SVGNativeVector:
		if svg, err := ui.ParseSVG(data); err == nil {
			width, height := svg.Size()
			if width <= 0 || height <= 0 {
				width, height = float32(info.Width), float32(info.Height)
			}
			return &result{pic: Picture{SVG: svg, Width: width, Height: height}}
		}
		// The native parser disagreed with the classifier: rasterise.
	}
	width, height := key.W, key.H
	if width <= 0 || height <= 0 {
		width, height = int(math.Round(info.Width)), int(math.Round(info.Height))
	}
	if width <= 0 || height <= 0 || int64(width) > imagepreview.MaxPixels/int64(height) {
		return &result{fail: FailTooLarge}
	}
	name := w.svc.diskName("svg", key, data, width, height)
	if cached := w.svc.disk.get(name); cached != nil {
		if bmp, err := ui.DecodeBitmap(cached); err == nil {
			return bitmapFromUI(bmp)
		}
	}
	if w.raster == nil {
		w.raster = svgrender.NewRasterizer()
	}
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	w.svc.rasters.Add(1)
	out, err := w.raster.Render(rctx, info.Prepared(data, key.Appearance), uint32(width), uint32(height), info.NeedsFonts)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return &result{fail: FailCorrupt}
	}
	if rasterHook != nil {
		rasterHook(out)
	}
	bmp, err := ui.DecodeBitmap(out)
	if err != nil {
		return &result{fail: FailCorrupt}
	}
	w.svc.disk.put(name, out)
	return bitmapFromUI(bmp)
}

// decodeCached decodes a still through the disk cache.
func (w *worker) decodeCached(ctx context.Context, key Key, format string, data []byte, decode func() (image.Image, error)) *result {
	name := w.svc.diskName(format, key, data, key.W, key.H)
	if cached := w.svc.disk.get(name); cached != nil {
		if bmp, err := ui.DecodeBitmap(cached); err == nil {
			return bitmapFromUI(bmp)
		}
	}
	img, err := decode()
	if err != nil {
		return &result{fail: decodeFailure(err)}
	}
	if ctx.Err() != nil {
		return nil
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || int64(b.Dx()) > imagepreview.MaxPixels/int64(b.Dy()) {
		return &result{fail: FailTooLarge}
	}
	if key.W > 0 && key.H > 0 {
		img = fit(img, key)
	}
	if w.svc.disk != nil {
		var buf bytes.Buffer
		if (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img) == nil {
			w.svc.disk.put(name, buf.Bytes())
		}
	}
	return bitmapResult(img)
}

func (s *Service) diskName(format string, key Key, data []byte, width, height int) string {
	if s.disk == nil {
		return ""
	}
	sum := sha256.Sum256(data)
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%x|%d|%d|%s", rasterVersion, format, sum, width, height, key.Appearance)
	return hex.EncodeToString(h.Sum(nil))
}

func bitmapResult(img image.Image) *result {
	bmp := ui.NewBitmap(img)
	return bitmapFromUI(bmp)
}

func bitmapFromUI(bmp *ui.Bitmap) *result {
	w, h := bmp.Size()
	return &result{pic: Picture{Bitmap: bmp, Width: float32(w), Height: float32(h)}}
}

// fitScale is the scale that fits a w×h image to key: within W×H, or
// covering it for Cover keys. It never enlarges.
func fitScale(w, h int, key Key) float64 {
	if key.W <= 0 || key.H <= 0 || w <= 0 || h <= 0 {
		return 1
	}
	sx, sy := float64(key.W)/float64(w), float64(key.H)/float64(h)
	if key.Cover {
		return math.Min(1, math.Max(sx, sy))
	}
	return math.Min(1, math.Min(sx, sy))
}

// fit scales img down to key's target; a smaller image is returned as is.
func fit(img image.Image, key Key) image.Image {
	b := img.Bounds()
	scale := fitScale(b.Dx(), b.Dy(), key)
	if scale >= 1 {
		return img
	}
	dw, dh := max(1, int(math.Round(float64(b.Dx())*scale))), max(1, int(math.Round(float64(b.Dy())*scale)))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.ApproxBiLinear.Scale(dst, dst.Rect, img, b, draw.Src, nil)
	return dst
}

// animation is a picture's frame cache: premultiplied frames shown through
// one bitmap (one texture) by swapping its pixels.
type animation struct {
	timing  *imagepreview.Animation
	frames  [][]byte
	bytes   int64
	started time.Time
	current int
}

func animationResult(motion *imagepreview.Animation, key Key) *result {
	if motion == nil || len(motion.Frames) == 0 {
		return &result{fail: FailCorrupt}
	}
	first := motion.Frames[0].Bounds()
	var total int64
	for _, f := range motion.Frames {
		total += int64(f.Bounds().Dx()) * int64(f.Bounds().Dy()) * 4
	}
	scale := fitScale(first.Dx(), first.Dy(), key)
	if over := float64(total) * scale * scale; over > animationBudget {
		scale *= math.Sqrt(animationBudget / over)
	}
	dw, dh := max(1, int(math.Round(float64(first.Dx())*scale))), max(1, int(math.Round(float64(first.Dy())*scale)))
	a := &animation{timing: motion}
	for i, f := range motion.Frames {
		dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
		if dw == first.Dx() && dh == first.Dy() {
			draw.Draw(dst, dst.Rect, f, f.Bounds().Min, draw.Src)
		} else {
			xdraw.ApproxBiLinear.Scale(dst, dst.Rect, f, f.Bounds(), draw.Src, nil)
		}
		a.frames = append(a.frames, dst.Pix)
		a.bytes += int64(len(dst.Pix))
		motion.Frames[i] = nil
	}
	motion.Frames = nil
	// The first frame's pixels back the bitmap; the others are swapped in.
	bmp := ui.NewBitmap(&image.RGBA{Pix: a.frames[0], Stride: 4 * dw, Rect: image.Rect(0, 0, dw, dh)})
	if len(a.frames) == 1 {
		return bitmapFromUI(bmp)
	}
	r := bitmapFromUI(bmp)
	r.anim = a
	return r
}

func (a *animation) show(bmp *ui.Bitmap, now time.Time) time.Duration {
	if a.started.IsZero() {
		a.started = now
	}
	index, wait := a.timing.FrameAt(now.Sub(a.started))
	if index != a.current && index < len(a.frames) {
		bmp.SetPixels(a.frames[index])
		a.current = index
	}
	return wait
}
