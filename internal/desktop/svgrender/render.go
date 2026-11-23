package svgrender

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

var fontsOnce sync.Once
var defaultFonts [][]byte

// fontLoads counts font databases loaded into renderer instances. Tests use it
// to prove that text-free artwork never pays for the ~57 MB of system fonts.
var fontLoads atomic.Int64

// FontLoads reports how many renderer instances have loaded the font set.
func FontLoads() int64 { return fontLoads.Load() }

// fonts are application-selected system fonts, never paths supplied by SVG.
// The WASI instance receives bytes only, no filesystem, network or environment.
func fonts() [][]byte {
	fontsOnce.Do(func() {
		defaultFonts = [][]byte{goregular.TTF, gobold.TTF, goitalic.TTF}
		var paths []string
		switch runtime.GOOS {
		case "windows":
			base := filepath.Join(os.Getenv("WINDIR"), "Fonts")
			if !filepath.IsAbs(base) {
				base = `C:\Windows\Fonts`
			}
			for _, name := range []string{"segoeui.ttf", "segoeuib.ttf", "segoeuii.ttf", "msyh.ttc", "msyhbd.ttc", "simsun.ttc", "arial.ttf", "times.ttf"} {
				paths = append(paths, filepath.Join(base, name))
			}
		case "darwin":
			paths = []string{"/System/Library/Fonts/Helvetica.ttc", "/System/Library/Fonts/Times.ttc", "/System/Library/Fonts/PingFang.ttc"}
		default:
			paths = []string{"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", "/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf", "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"}
		}
		budget := int64(64 << 20)
		for _, path := range paths {
			f, err := os.Open(path)
			if err != nil {
				continue
			}
			st, err := f.Stat()
			if err != nil || !st.Mode().IsRegular() || st.Size() <= 0 || st.Size() > budget {
				f.Close()
				continue
			}
			// Bound the read even if the installed font is replaced while being read.
			data, err := readFont(f, budget)
			f.Close()
			if err != nil {
				continue
			}
			budget -= int64(len(data))
			defaultFonts = append(defaultFonts, data)
		}
	})
	return defaultFonts
}

// recycleBytes is the linear-memory size past which an instance is dropped
// after a render. WASM memory never shrinks, so one huge document must not pin
// its peak for the rest of the process.
const recycleBytes = 192 << 20

type instance struct {
	c     *Context
	r     *Renderer
	fonts bool
}

func newInstance(ctx context.Context, withFonts bool) (*instance, error) {
	c, err := NewContext(ctx)
	if err != nil {
		return nil, err
	}
	r, err := c.NewRenderer()
	if err != nil {
		c.Close()
		return nil, err
	}
	in := &instance{c: c, r: r}
	if withFonts {
		for _, font := range fonts() {
			if err := r.LoadFontData(font); err != nil {
				in.close()
				return nil, err
			}
		}
		in.fonts = true
		fontLoads.Add(1)
	}
	return in, nil
}

func (in *instance) close() {
	if in == nil {
		return
	}
	_ = in.r.Close()
	_ = in.c.Close()
}

// Rasterizer is a long-lived resvg renderer for one goroutine. It reuses the
// process-wide compiled module and keeps up to two instantiated renderers:
// one without fonts for vector artwork, and one whose font database is loaded
// lazily, once, the first time a document actually contains text. resvg parses
// a fresh tree per call, so a renderer can be reused across documents.
//
// A Rasterizer is NOT safe for concurrent use.
type Rasterizer struct {
	plain, text *instance
}

// NewRasterizer returns an empty rasterizer; instances are created on demand.
func NewRasterizer() *Rasterizer { return &Rasterizer{} }

// Render rasterises svg into a PNG of width x height (zero for the natural
// size). needsFonts selects the font-carrying instance.
func (z *Rasterizer) Render(ctx context.Context, svg []byte, width, height uint32, needsFonts bool) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slot := &z.plain
	if needsFonts {
		slot = &z.text
	}
	if *slot == nil {
		in, err := newInstance(ctx, needsFonts)
		if err != nil {
			return nil, err
		}
		*slot = in
	}
	in := *slot
	in.c.ctx = ctx
	out, err := in.r.RenderWithSize(svg, width, height)
	// A cancelled call closes the module (WithCloseOnContextDone); any failure
	// or an inflated heap retires the instance instead of reusing it.
	if err != nil || ctx.Err() != nil || in.c.memory.Size() > recycleBytes {
		in.close()
		*slot = nil
	} else {
		in.c.ctx = context.Background()
	}
	return out, err
}

// Close releases both instances.
func (z *Rasterizer) Close() {
	z.plain.close()
	z.text.close()
	z.plain, z.text = nil, nil
}

// Render is a one-shot convenience with fonts. Long-running callers keep a
// Rasterizer instead, so the module instance and font database are reused.
func Render(ctx context.Context, svg []byte, width, height uint32) ([]byte, error) {
	z := NewRasterizer()
	defer z.Close()
	return z.Render(ctx, svg, width, height, true)
}

func readFont(f *os.File, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("font exceeds image renderer budget")
	}
	return data, nil
}
