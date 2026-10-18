package imagepreview

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"time"
)

// Animation contains every composited frame of a prevalidated bounded GIF.
// It is independent of files, scripts, WebViews and external decoder processes.
// Total retained canvas pixels cannot exceed MaxPixels.
type Animation struct {
	Frames   []*image.NRGBA
	ends     []time.Duration
	duration time.Duration
	loops    int
}

func DecodeAnimation(data []byte) (*Animation, error) {
	// Validate all compressed frames and cumulative canvas allocation before
	// image/gif can allocate its frame list.
	if err := validateGIF(data); err != nil {
		return nil, err
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	a := &Animation{loops: g.LoopCount}
	canvas := image.NewNRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	draw.Draw(canvas, canvas.Rect, image.NewUniform(gifFrameBackground(g, g.Image[0])), image.Point{}, draw.Src)
	for i, frame := range g.Image {
		var previous *image.NRGBA
		if g.Disposal[i] == gif.DisposalPrevious {
			previous = image.NewNRGBA(canvas.Rect)
			copy(previous.Pix, canvas.Pix)
		}
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		snapshot := image.NewNRGBA(canvas.Rect)
		copy(snapshot.Pix, canvas.Pix)
		a.Frames = append(a.Frames, snapshot)
		delay := time.Duration(g.Delay[i]) * 10 * time.Millisecond
		// Match browser-like GIF handling of zero/one-centisecond delays.
		if delay < 20*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		a.duration += delay
		a.ends = append(a.ends, a.duration)
		switch g.Disposal[i] {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), image.NewUniform(gifFrameBackground(g, frame)), image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, previous.Pix)
		}
	}
	return a, nil
}

// Opaque frames restore the logical-screen background from the GLOBAL color
// table, even when their own pixels use a different local palette. Frames with
// transparency retain the reader's transparent-canvas behavior; the wire GIF
// global table cannot itself store alpha.
func gifFrameBackground(g *gif.GIF, frame *image.Paletted) color.Color {
	for _, c := range frame.Palette {
		if _, _, _, alpha := c.RGBA(); alpha == 0 {
			return color.Transparent
		}
	}
	if palette, ok := g.Config.ColorModel.(color.Palette); ok && int(g.BackgroundIndex) < len(palette) {
		return palette[g.BackgroundIndex]
	}
	return color.Transparent
}

// FrameAt skips missed frames without replaying work after an inactive tab.
// A zero wait means the last frame of a finite animation is final.
func (a *Animation) FrameAt(elapsed time.Duration) (index int, wait time.Duration) {
	if a == nil || len(a.ends) == 0 {
		return 0, 0
	}
	if elapsed < 0 {
		elapsed = 0
	}
	cycles := a.loops + 1
	if a.loops < 0 {
		cycles = 1
	}
	if a.loops != 0 && elapsed/a.duration >= time.Duration(cycles) {
		return len(a.ends) - 1, 0
	}
	position := elapsed % a.duration
	for i, end := range a.ends {
		if position < end {
			return i, end - position
		}
	}
	return len(a.ends) - 1, 0
}
