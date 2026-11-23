package media

import (
	"image"
	"image/color"
	"image/gif"
	"io"
)

func encodeTwoFrameGIF(w io.Writer) error {
	pal := color.Palette{color.Black, color.White}
	a := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
	b := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
	for i := range b.Pix {
		b.Pix[i] = 1
	}
	return gif.EncodeAll(w, &gif.GIF{Image: []*image.Paletted{a, b}, Delay: []int{10, 10}})
}
