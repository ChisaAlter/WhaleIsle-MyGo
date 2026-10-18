package svgrender

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

var fontsOnce sync.Once
var defaultFonts [][]byte

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

// Render uses an isolated WASI instance per operation and shares only compiled
// trusted renderer code. Width/height zero request the validated natural size.
func Render(ctx context.Context, svg []byte, width, height uint32) ([]byte, error) {
	c, err := NewContext(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	r, err := c.NewRenderer()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for _, font := range fonts() {
		if err := r.LoadFontData(font); err != nil {
			return nil, err
		}
	}
	return r.RenderWithSize(svg, width, height)
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
