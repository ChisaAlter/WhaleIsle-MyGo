package pluginmanager

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
)

var searchArtworkOnce sync.Once
var searchArtworkSource string

// The original search artwork uses foreignObject solely for its conic-gradient
// ring. Paint that fixed local geometry here; package images keep the existing
// SVG policy and do not gain arbitrary HTML/foreignObject support.
func searchArtwork() string {
	searchArtworkOnce.Do(func() {
		const scale = 4
		pixels := image.NewNRGBA(image.Rect(0, 0, 36*scale, 36*scale))
		rotation := math.Atan2(-.00173904, .0119394)
		for y := 0; y < 36*scale; y++ {
			for x := 0; x < 36*scale; x++ {
				px, py := (float64(x)+.5)/scale, (float64(y)+.5)/scale
				ink := color.NRGBA{}
				// Original handle: a 3px, butt-capped stroke, behind the ring.
				dx, dy := 26.5362-22.3813, 26.9865-22.8317
				t := ((px-22.3813)*dx + (py-22.8317)*dy) / (dx*dx + dy*dy)
				if t >= 0 && t <= 1 && math.Abs((px-22.3813)*dy-(py-22.8317)*dx)/math.Hypot(dx, dy) <= 1.5 {
					ink = color.NRGBA{101, 142, 255, 255}
				}
				xc, yc := px-15.7661, py-16.2159
				radius := math.Hypot(xc, yc)
				if radius >= 4.2056 && radius <= 8.2056 {
					angle := math.Mod((math.Atan2(yc, xc)-rotation)*180/math.Pi+360, 360)
					fraction := angle / 62.0619
					if angle > 62.0619 {
						fraction = (360 - angle) / (360 - 62.0619)
					}
					mix := func(a, b float64) uint8 { return uint8(math.Round(a + (b-a)*fraction)) }
					ink = color.NRGBA{mix(65, 101), mix(225, 142), mix(172, 255), 255}
				}
				pixels.SetNRGBA(x, y, ink)
			}
		}
		var encoded bytes.Buffer
		if png.Encode(&encoded, pixels) == nil {
			searchArtworkSource = "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
		}
	})
	return searchArtworkSource
}
