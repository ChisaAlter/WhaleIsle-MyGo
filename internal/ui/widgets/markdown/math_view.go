package markdown

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/go-text/typesetting/segmenter"
	"whaleisle/internal/mathrender"
	"whaleisle/internal/ui/theme"
)

var mathWorkers = make(chan struct{}, 2)

type mathImage struct {
	svg                  *ui.SVG
	width, height, depth float32
	err                  error
	ready                bool
}

type mathViewState struct {
	mu           sync.Mutex
	images       map[mathFormula]mathImage
	revision     uint64
	layout       mathLayout
	lastRevision uint64
	lastKey      string
	lastWidth    float32
	width        float32
}

func (s *mathViewState) request(formula mathFormula, services ui.Services) {
	s.mu.Lock()
	if s.images == nil {
		s.images = map[mathFormula]mathImage{}
	}
	if _, exists := s.images[formula]; exists || len(s.images) >= 64 {
		s.mu.Unlock()
		return
	}
	s.images[formula] = mathImage{}
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		image := mathImage{ready: true}
		select {
		case mathWorkers <- struct{}{}:
			defer func() { <-mathWorkers }()
			if node, err := mathrender.BundledNode(); err != nil {
				image.err = err
			} else {
				result, err := mathrender.Render(ctx, node, mathrender.Formula{TeX: formula.TeX, Display: formula.Display})
				image.err = err
				if err == nil {
					image.svg, image.err = ui.ParseSVG([]byte(result.SVG))
					image.width, image.height, image.depth = float32(result.WidthEm), float32(result.HeightEm), float32(result.DepthEm)
				}
			}
		case <-ctx.Done():
			image.err = ctx.Err()
		}
		s.mu.Lock()
		s.images[formula] = image
		s.revision++
		s.mu.Unlock()
		services.Invalidate()
	}()
}

func viewMathSpans(c *ui.Context, st Style, spans []ui.Span, targets map[int]string, offset int) {
	root := ui.Box(c).FillWidth().MinWidth(0)
	state := ui.Local(root, "offline-math", func() mathViewState { return mathViewState{width: 320} })
	services := c.Services()
	for index, formula := range st.maths {
		if index >= offset && index < offset+len(spans) {
			state.request(formula, services)
		}
	}
	state.mu.Lock()
	images := make(map[mathFormula]mathImage, len(state.images))
	for key, image := range state.images {
		images[key] = image
	}
	revision := state.revision
	state.mu.Unlock()
	width := state.width
	if previous := root.Bounds().W; previous > 0 {
		width = previous
	}
	key := fmt.Sprint(spans, st.FontSize, st.lineHeight, theme.RenderPreferences().Chat.Family, offset, targets, st.urlTargets)
	if state.lastKey != key || state.lastWidth != width || state.lastRevision != revision {
		atoms := mathAtoms(spans, st, targets, offset, images, width)
		lineHeight := st.FontSize * 1.25
		if st.lineHeight > 0 {
			lineHeight = st.lineHeight
		}
		state.layout = layoutMathAtoms(atoms, width, lineHeight)
		state.lastKey, state.lastWidth, state.lastRevision = key, width, revision
	}
	layout := state.layout
	var label strings.Builder
	for _, span := range spans {
		label.WriteString(span.Text)
	}
	root.Height(layout.height).Role(ui.RoleText).Label(label.String()).Draw(func(p *ui.Painter, bounds ui.Rect) {
		if math.Abs(float64(bounds.W-state.width)) > .5 && bounds.W > 0 {
			state.width = bounds.W
			services.Invalidate()
		}
		for _, item := range layout.items {
			r := item.rect
			r.X, r.Y = r.X+bounds.X, r.Y+bounds.Y
			if item.atom.svg != nil {
				p.Icon(item.atom.svg, r, item.atom.color)
			} else if item.atom.text != nil {
				p.TextLayout(item.atom.text, r.X, r.Y, item.atom.color)
			}
		}
	}).Children(func() {
		for _, item := range layout.items {
			if item.atom.target == "" && item.atom.url == "" && item.atom.label == "" {
				continue
			}
			r := item.rect
			el := ui.Box(c).Absolute().Left(r.X).Top(r.Y).Width(r.W).Height(r.H).Tooltip(item.atom.label)
			if item.atom.target != "" && st.OnOpenFile != nil {
				el = el.Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink).Label(item.atom.target)
				if el.Clicked() || el.Shortcut(0, ui.KeyEnter) || el.Shortcut(0, ui.KeySpace) {
					st.OnOpenFile(item.atom.target)
				}
			} else if item.atom.url != "" && st.OnOpenURL != nil {
				el = el.Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink).Label(item.atom.label).Tooltip(item.atom.url)
				if externalURLActivated(el) {
					st.OnOpenURL(item.atom.url)
				}
			}
		}
	})
}

func mathAtoms(spans []ui.Span, st Style, targets map[int]string, offset int, images map[mathFormula]mathImage, width float32) []mathAtom {
	var atoms []mathAtom
	for index, span := range spans {
		formula, isMath := st.maths[offset+index]
		if isMath {
			href := st.urlTargets[offset+index]
			image := images[formula]
			if image.ready && image.err == nil && image.svg != nil {
				atoms = append(atoms, mathAtom{svg: image.svg, width: image.width * st.FontSize, ascent: (image.height - image.depth) * st.FontSize, descent: image.depth * st.FontSize, color: st.FG, label: formula.Raw, url: href, display: formula.Display})
				continue
			}
			label := "Rendering equation offline…"
			if image.err != nil {
				label = "Equation kept as TeX: " + image.err.Error()
			}
			start := len(atoms)
			atoms = appendTextAtoms(atoms, span, st, "", width)
			for i := start; i < len(atoms); i++ {
				atoms[i].label = label
				atoms[i].url = href
			}
			continue
		}
		start := len(atoms)
		atoms = appendTextAtoms(atoms, span, st, targets[offset+index], width)
		if href := st.urlTargets[offset+index]; href != "" {
			for i := start; i < len(atoms); i++ {
				atoms[i].url = href
				atoms[i].label = span.Text
			}
		}
	}
	return atoms
}

func appendTextAtoms(atoms []mathAtom, span ui.Span, st Style, target string, width float32) []mathAtom {
	font := ui.Font{Family: span.Font, Size: span.Size, Weight: span.Weight, Italic: span.Italic, Features: span.Features}
	if font.Family == "" {
		font.Family = theme.RenderPreferences().Chat.Family
	}
	if font.Size <= 0 {
		font.Size = st.FontSize
	}
	metrics := font.Metrics()
	appendPart := func(value string) {
		part := span
		part.Text = value
		shaped := ui.ShapeRichText([]ui.Span{part}, font, 0)
		w, h := shaped.Size()
		ascent := (h-metrics.Ascent-metrics.Descent)/2 + metrics.Ascent
		atoms = append(atoms, mathAtom{text: shaped, width: w, ascent: ascent, descent: h - ascent, color: st.FG, target: target, label: target})
	}
	lines := strings.Split(span.Text, "\n")
	for index, line := range lines {
		if index > 0 {
			atoms = append(atoms, mathAtom{newline: true})
		}
		var breaks segmenter.Segmenter
		breaks.Init([]rune(line))
		iterator := breaks.LineIterator()
		for iterator.Next() {
			value := string(iterator.Line().Text)
			part := span
			part.Text = value
			w, _ := ui.ShapeRichText([]ui.Span{part}, font, 0).Size()
			if w <= width {
				appendPart(value)
				continue
			}
			// Emergency wrapping respects grapheme clusters, including emoji and
			// combining marks; normal opportunities follow Unicode line breaking.
			var graphemes segmenter.Segmenter
			graphemes.Init([]rune(value))
			it := graphemes.GraphemeIterator()
			for it.Next() {
				appendPart(string(it.Grapheme().Text))
			}
		}
	}
	return atoms
}
