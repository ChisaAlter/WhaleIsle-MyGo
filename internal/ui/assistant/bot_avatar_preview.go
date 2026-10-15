package assistant

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

// Static pose of the original eight-point cubic blobPath(shape,0,0).
var botAvatarRadii = map[string][8]float64{
	"circle": {22, 22, 22, 22, 22, 22, 22, 22}, "soft": {20, 25, 21, 24, 19, 23, 22, 26},
	"square": {18, 26, 18, 26, 18, 26, 18, 26}, "pill": {14, 18, 26, 18, 14, 18, 26, 18},
	"triangle": {26, 16, 12, 22, 13, 22, 12, 16}, "hex": {20, 24, 24, 20, 24, 24, 20, 24},
	"cloud": {14, 26, 22, 18, 16, 18, 22, 26}, "drop": {12, 16, 20, 24, 26, 24, 20, 16},
}

func botAvatarColor(name string, fg, muted ui.Color) ui.Color {
	black, white := ui.RGB(15, 17, 21), ui.RGB(255, 255, 255)
	blue, red, green := ui.RGB(65, 118, 230), ui.RGB(239, 68, 68), ui.RGB(34, 197, 94)
	switch name {
	case "ink":
		return fg
	case "brown":
		return ui.RGB(221, 134, 41).Mix(black, .3)
	case "red":
		return red
	case "orange":
		return ui.RGB(245, 158, 11)
	case "yellow":
		return ui.RGB(247, 173, 49)
	case "green":
		return green
	case "teal":
		return green.Mix(blue, .45)
	case "blue":
		return blue
	case "purple":
		return red.Mix(ui.RGB(72, 104, 178), .6)
	case "pink":
		return ui.RGB(242, 90, 90).Mix(white, .3)
	default:
		return muted
	}
}
func botAvatarOutline(shape string, r ui.Rect) *ui.Path {
	radii := botAvatarRadii[shape]
	k := 4.0 / 3.0 * math.Tan(math.Pi/16)
	type point struct{ x, y, a, rad float64 }
	var pts [8]point
	for i, radius := range radii {
		a := -math.Pi/2 + float64(i)*math.Pi/4
		pts[i] = point{32 + radius*math.Cos(a), 32 + radius*math.Sin(a), a, radius}
	}
	x := func(v float64) float32 { return r.X + float32(v)*r.W/64 }
	y := func(v float64) float32 { return r.Y + float32(v)*r.H/64 }
	p := &ui.Path{}
	p.MoveTo(x(pts[0].x), y(pts[0].y))
	for i, p0 := range pts {
		p1 := pts[(i+1)%8]
		a0, a1 := p0.a+math.Pi/2, p1.a+math.Pi/2
		p.CubeTo(x(p0.x+math.Cos(a0)*p0.rad*k), y(p0.y+math.Sin(a0)*p0.rad*k), x(p1.x-math.Cos(a1)*p1.rad*k), y(p1.y-math.Sin(a1)*p1.rad*k), x(p1.x), y(p1.y))
	}
	return p.Close()
}
func botAvatarEllipse(cx, cy, rx, ry float32) *ui.Path {
	const k = float32(.5522847498)
	p := &ui.Path{}
	p.MoveTo(cx+rx, cy)
	p.CubeTo(cx+rx, cy+ry*k, cx+rx*k, cy+ry, cx, cy+ry)
	p.CubeTo(cx-rx*k, cy+ry, cx-rx, cy+ry*k, cx-rx, cy)
	p.CubeTo(cx-rx, cy-ry*k, cx-rx*k, cy-ry, cx, cy-ry)
	p.CubeTo(cx+rx*k, cy-ry, cx+rx, cy-ry*k, cx+rx, cy)
	return p.Close()
}
func botAvatarDrawBlob(c *ui.Context, v pluginrpc.DshbotAvatarValues, size float32, label string, fg, muted ui.Color) {
	ui.Box(c).Size(size, size).Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillPath(botAvatarOutline(v.Shape, r), botAvatarColor(v.Color, fg, muted))
		scale := r.W / 64
		black, white := ui.RGB(15, 17, 21), ui.RGB(255, 255, 255)
		for _, cx := range []float32{25, 39} {
			eye := botAvatarEllipse(r.X+cx*scale, r.Y+30*scale, 5.2*scale, 6.4*scale)
			p.FillPath(eye, white)
			p.StrokePath(eye, 1.15*scale, black)
			p.FillPath((&ui.Path{}).Circle(r.X+(cx+1.2)*scale, r.Y+31*scale, 2.45*scale), black)
			p.FillPath((&ui.Path{}).Circle(r.X+(cx+.2)*scale, r.Y+29.9*scale, .8*scale), white)
		}
	})
}

// At most one decoded inline image is retained. Content-addressed lookup never
// substitutes one account's image for another input; no file or URL is opened.
var botAvatarImageCache struct {
	sync.Mutex
	key    [32]byte
	set    bool
	bitmap *ui.Bitmap
	err    error
}

func decodeBotAvatarImage(v pluginrpc.DshbotAvatarValues) (*ui.Bitmap, error) {
	if _, err := pluginrpc.ParseDshbotAvatar(v); err != nil || v.Kind != "image" {
		return nil, fmt.Errorf("invalid inline image record")
	}
	comma := strings.IndexByte(v.DataURL, ',')
	data, err := base64.StdEncoding.Strict().DecodeString(v.DataURL[comma+1:])
	if err != nil || len(data) > 90000 {
		return nil, fmt.Errorf("invalid or oversized inline image")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	expected := "jpeg"
	if strings.HasPrefix(strings.ToLower(v.DataURL), "data:image/png;") {
		expected = "png"
	}
	if err != nil || format != expected || config.Width < 1 || config.Height < 1 || config.Width > 2048 || config.Height > 2048 || int64(config.Width)*int64(config.Height) > 1048576 {
		return nil, fmt.Errorf("inline image preview exceeds safe JPEG/PNG dimensions")
	}
	// The exact same bounded bytes are decoded only after dimensions and format.
	return ui.DecodeBitmap(data)
}
func botAvatarImage(v pluginrpc.DshbotAvatarValues) (*ui.Bitmap, error) {
	key := sha256.Sum256([]byte(v.DataURL))
	botAvatarImageCache.Lock()
	defer botAvatarImageCache.Unlock()
	if !botAvatarImageCache.set || botAvatarImageCache.key != key {
		botAvatarImageCache.set = true
		botAvatarImageCache.key = key
		botAvatarImageCache.bitmap, botAvatarImageCache.err = decodeBotAvatarImage(v)
	}
	return botAvatarImageCache.bitmap, botAvatarImageCache.err
}
func botAvatarSummary(v pluginrpc.DshbotAvatarValues) string {
	if v.Kind == "blob" {
		return fmt.Sprintf("kind=blob · shape=%s · color=%s · dataUrl/crop=empty", v.Shape, v.Color)
	}
	digest := sha256.Sum256([]byte(v.DataURL))
	return fmt.Sprintf("kind=image · crop=%s · inline data URL chars=%d · SHA-256=%x · shape/color=empty", v.Crop, len(v.DataURL), digest)
}
func botAvatarPreview(c *ui.Context, v pluginrpc.DshbotAvatarValues, size float32, label string, fg, muted ui.Color) error {
	if _, err := pluginrpc.ParseDshbotAvatar(v); err != nil {
		return err
	}
	if v.Kind == "blob" {
		botAvatarDrawBlob(c, v, size, label, fg, muted)
		return nil
	}
	bmp, err := botAvatarImage(v)
	if err != nil {
		return err
	}
	radius := size * 10 / 64
	if v.Crop == "circle" {
		radius = size / 2
	}
	ui.Image(c, bmp).Size(size, size).Fit(ui.Cover).Radius(radius).Label(label)
	return nil
}
