package text

import (
	"errors"
	"math"

	"github.com/egoist/mygo/internal/scene"
)

// stubEngine lays out text without fonts where the system has no text
// engine MyGo can use: every rune takes half an em and draws nothing.
type stubEngine struct {
	shapeScratch
	fonts      map[float32]*Font
	boundaries Boundaries
	breaks     []bool
}

func (e *stubEngine) font(style Style) *Font {
	size := style.FontSize()
	if f, ok := e.fonts[size]; ok {
		return f
	}
	if e.fonts == nil {
		e.fonts = map[float32]*Font{}
	}
	f := &Font{Size: size, Ascent: size * 0.8, Descent: size * 0.2}
	e.fonts[size] = f
	return f
}

func (e *stubEngine) shape(text []rune, style Style, spans []Span, width float32, rtl, wholeWords, breakAll bool) []shapedLine {
	if len(text) == 0 {
		return nil
	}
	f := e.font(style)
	e.boundaries.Reset(text)
	graphemes := e.boundaries.graphemes
	if width > 0 && !breakAll {
		e.breaks = lineBreaks(&e.boundaries.seg, e.breaks, text)
	}
	mark := len(e.lines)
	var lines []shapedLine
	for g := 0; g < len(graphemes)-1; {
		first, end, lastBreak := g, g, g
		w := float32(0)
		for end < len(graphemes)-1 {
			next := end + 1
			advance := float32(graphemes[next]-graphemes[end]) * f.Size / 2
			if width > 0 && w+advance > width && end > first {
				if breakAll {
					break
				}
				if lastBreak > first {
					end = lastBreak
					break
				}
				if !wholeWords {
					break
				}
			}
			w += advance
			end = next
			if width > 0 && !breakAll && e.breaks[graphemes[end]] {
				lastBreak = end
				if wholeWords && w > width {
					break
				}
			}
		}
		start, stop := graphemes[first], graphemes[end]
		run := shapedRun{font: f, start: start, end: stop, glyphs: e.glyphRoom(stop - start)}
		for i := start; i < stop; i++ {
			x := float32(i-start) * f.Size / 2
			if rtl {
				x = float32(stop-i-1) * f.Size / 2
			}
			run.glyphs[i-start] = Glyph{Font: f, X: x, Advance: f.Size / 2, Cluster: i, RTL: rtl}
		}
		runs := e.addRun(len(e.runs), run)
		lines = e.addLine(mark, shapedLine{start: start, end: stop, runs: runs})
		g = end
	}
	return lines
}

func (e *stubEngine) glyph(f *Font, id uint32, scale, dx float32, _ Shade, _ bool) bitmap {
	return bitmap{}
}

func (e *stubEngine) textParams() (scene.TextParams, bool) { return scene.TextParams{}, false }

func (e *stubEngine) positions(*Font, float32, bool) (int, bool) { return 1, true }

func (e *stubEngine) decorate(decoRange) []Stroke { return nil }

func (e *stubEngine) join() bool { return false }

func (e *stubEngine) baseline(y float32) float32 { return float32(math.Round(float64(y))) }

func (e *stubEngine) register(data []byte, family string) error {
	return errors.New("mygo: this system has no text engine to add fonts to")
}
