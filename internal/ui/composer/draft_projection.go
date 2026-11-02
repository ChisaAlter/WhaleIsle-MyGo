package composer

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// draftReference occupies one object in the editable display and its complete
// UTF-16 range in the clipboard/persistence text. Offsets never address its label.
type draftReference struct {
	Start int
	End   int
	Kind  DraftMentionKind
	Text  string
}

type draftInlineChip struct {
	reference draftReference
	at        int // UTF-16 display offset of the single object
	label     string
	width     float32
}

const draftObject = "\ufffc"

func firstDraftRune(text string) (rune, int) { return utf8.DecodeRuneInString(text) }
func startsDraftSpace(text string) bool {
	r, _ := utf8.DecodeRuneInString(text)
	return unicode.IsSpace(r)
}

func draftReferences(text string) []draftReference {
	var out []draftReference
	for _, m := range ParseDraftMentions(text) {
		out = append(out, draftReference{ui.UTF16Len(text[:m.Start]), ui.UTF16Len(text[:m.End]), m.Kind, m.Text})
	}
	return out
}

// Durable occurrence metadata preserves objects moved beside prose. Older
// drafts without it still import the established plain-text reference grammar.
func checkedDraftReferences(text string, refs []draftReference) ([]draftReference, error) {
	out := append([]draftReference(nil), refs...)
	previous := 0
	for _, ref := range out {
		a, z := ui.UTF16ByteOffset(text, ref.Start), ui.UTF16ByteOffset(text, ref.End)
		parsed := ParseDraftMentions(ref.Text)
		if ref.Start < previous || ref.Start >= ref.End || ref.End > ui.UTF16Len(text) ||
			ui.UTF16Len(text[:a]) != ref.Start || ui.UTF16Len(text[:z]) != ref.End || text[a:z] != ref.Text ||
			len(parsed) != 1 || parsed[0].Text != ref.Text || parsed[0].Kind != ref.Kind {
			return nil, fmt.Errorf("saved reference at UTF-16 offset %d is invalid", ref.Start)
		}
		previous = ref.End
	}
	return out, nil
}

func (c *DraftClient) mention(ref draftReference) (DraftMention, bool) {
	text := c.text()
	a, z := ui.UTF16ByteOffset(text, ref.Start), ui.UTF16ByteOffset(text, ref.End)
	m := DraftMention{Kind: ref.Kind, Text: ref.Text, Start: a, End: z}
	return m, a < z && text[a:z] == ref.Text
}

func (c *DraftClient) normalizeRange(r ui.TextInputRange) ui.TextInputRange {
	collapsed := r.Start == r.End
	text := c.text()
	n := ui.UTF16Len(text)
	r.Start, r.End = max(0, min(r.Start, n)), max(0, min(r.End, n))
	if r.Start > r.End {
		r.Start, r.End = r.End, r.Start
	}
	r.Start = ui.UTF16Len(text[:ui.UTF16ByteOffset(text, r.Start)])
	if collapsed {
		r.End = r.Start
	}
	endByte := ui.UTF16ByteOffset(text, r.End)
	if ui.UTF16Len(text[:endByte]) < r.End {
		_, size := firstDraftRune(text[endByte:])
		endByte += size
	}
	r.End = ui.UTF16Len(text[:endByte])
	if r.Start == r.End {
		for _, ref := range c.references {
			if r.Start > ref.Start && r.Start < ref.End {
				r.Start, r.End = ref.End, ref.End
				break
			}
		}
		return r
	}
	for _, ref := range c.references {
		if r.Start < ref.End && r.End > ref.Start {
			r.Start, r.End = min(r.Start, ref.Start), max(r.End, ref.End)
		}
	}
	return r
}

// documentToDisplay maps both edges exactly. A native offset inside an object
// snaps outward for ranges, or to its trailing edge for a collapsed caret.
func (c *DraftClient) documentToDisplay(at int, trailing bool) int {
	removed := 0
	for _, chip := range c.inlineChips {
		ref := chip.reference
		if at <= ref.Start {
			break
		}
		if at < ref.End {
			if trailing {
				return chip.at + 1
			}
			return chip.at
		}
		removed += ref.End - ref.Start - 1
	}
	return max(0, min(at-removed, ui.UTF16Len(c.displayText)))
}

func (c *DraftClient) displayToDocument(at int) int {
	added := 0
	for _, chip := range c.inlineChips {
		if at <= chip.at {
			break
		}
		added += chip.reference.End - chip.reference.Start - 1
	}
	return max(0, min(at+added, ui.UTF16Len(c.lastText)))
}

func (c *DraftClient) previousBoundary(at int) int {
	return c.displayToDocument(c.layout.PreviousBoundary(c.documentToDisplay(at, true)))
}

func (c *DraftClient) nextBoundary(at int) int {
	return c.displayToDocument(c.layout.NextBoundary(c.documentToDisplay(at, false)))
}

func (c *DraftClient) visualIndex(p ui.Point) int {
	return c.displayToDocument(c.layout.IndexAt(c.rawPoint(p)))
}

func inlineChipLabel(ref draftReference) string {
	m := DraftMention{Kind: ref.Kind, Text: ref.Text}
	if ref.Kind == DraftMentionSession {
		return strings.TrimPrefix(ChipLabel(m), "session ")
	}
	if ref.Kind == DraftMentionSkill {
		return ref.Text
	}
	path := strings.Trim(strings.TrimPrefix(ref.Text, "@"), `"`)
	trimmed := strings.TrimRight(path, `/\`)
	if at := strings.LastIndexAny(trimmed, `/\`); at >= 0 {
		trimmed = trimmed[at+1:]
	}
	if strings.HasSuffix(path, "/") || strings.HasSuffix(path, `\`) {
		trimmed += "/"
	}
	return "@" + trimmed
}

func (c *DraftClient) chipFont() ui.Font {
	font := c.font
	font.Size = max(11, c.font.Size-1)
	return font
}

func (c *DraftClient) buildProjection() {
	c.inlineChips = c.inlineChips[:0]
	var text strings.Builder
	var spans []ui.Span
	previous, display := 0, 0
	objectWidth, _ := ui.ShapeText(draftObject, c.font, 0).Size()
	for _, ref := range c.references {
		m, valid := c.mention(ref)
		if !valid {
			continue
		}
		plain := c.lastText[previous:m.Start]
		text.WriteString(plain)
		spans = append(spans, ui.Span{Text: plain})
		display += ui.UTF16Len(plain)
		label := inlineChipLabel(ref)
		width, _ := ui.ShapeText(label, c.chipFont(), 0).Size()
		width = min(240, max(42, width+30))
		if c.wrapWidth > 0 {
			width = min(width, c.wrapWidth)
		}
		c.inlineChips = append(c.inlineChips, draftInlineChip{ref, display, label, width})
		text.WriteString(draftObject)
		// The object is one nonbreaking shaping unit. Its advance includes the
		// label and remove control; the object's own glyph is never painted.
		spans = append(spans, ui.Span{Text: draftObject, LetterSpacing: width - objectWidth})
		display++
		previous = m.End
	}
	plain := c.lastText[previous:]
	text.WriteString(plain)
	spans = append(spans, ui.Span{Text: plain})
	c.displayText = text.String()
	c.layout = ui.ShapeRichText(spans, c.font, c.wrapWidth)
}

func (c *DraftClient) editReferences(r ui.TextInputRange, text string, importText, native bool) {
	delta := ui.UTF16Len(text) - (r.End - r.Start)
	refs := make([]draftReference, 0, len(c.references))
	for _, ref := range c.references {
		switch {
		case ref.End <= r.Start:
			refs = append(refs, ref)
		case ref.Start >= r.End:
			ref.Start, ref.End = ref.Start+delta, ref.End+delta
			refs = append(refs, ref)
		}
	}
	c.references = refs
	if !importText {
		return
	}
	c.importReferences(r.Start, r.Start+ui.UTF16Len(text), native, startsDraftSpace(text))
}

func (c *DraftClient) importReferences(start, insertEnd int, native, separator bool) {
	for _, ref := range draftReferences(c.lastText) {
		// Ordinary native typing keeps its unfinished trigger editable. A
		// separator commits it; explicit replacement/paste imports complete
		// tokens even at the document end.
		if ref.End < start || ref.Start >= insertEnd || ref.End == start && !separator {
			continue
		}
		if native && ref.End >= insertEnd {
			continue
		}
		overlaps := false
		for _, existing := range c.references {
			if ref.Start < existing.End && ref.End > existing.Start {
				overlaps = true
				break
			}
		}
		if !overlaps {
			c.references = append(c.references, ref)
		}
	}
	c.sortReferences()
}

func (c *DraftClient) sortReferences() {
	slices.SortFunc(c.references, func(a, b draftReference) int { return a.Start - b.Start })
}

func (c *DraftClient) chipRect(chip draftInlineChip) ui.Rect {
	rects := c.layout.SelectionRects(ui.TextInputRange{Start: chip.at, End: chip.at + 1})
	if len(rects) > 0 {
		return c.projectRect(rects[0])
	}
	b := c.projectRect(c.layout.Caret(chip.at))
	b.W = chip.width
	return b
}

// paintProjectedText uses the retained paragraph shaping for both text and
// objects, projecting line advance once. Clips omit object glyphs entirely.
func (c *DraftClient) paintProjectedText(p *ui.Painter, r ui.Rect, ink ui.Color) {
	left, _, top := c.ins()
	for line, raw := range c.visualLines {
		y := float32(line)*c.lineHeight() + top - c.scrollY
		if y+c.lineHeight() <= 0 || y >= r.H {
			continue
		}
		var objects []ui.Rect
		for _, chip := range c.inlineChips {
			b := c.chipRect(chip)
			if b.Y == float32(line)*c.lineHeight() {
				objects = append(objects, b)
			}
		}
		slices.SortFunc(objects, func(a, b ui.Rect) int {
			if a.X < b.X {
				return -1
			}
			if a.X > b.X {
				return 1
			}
			return 0
		})
		x := float32(0)
		paint := func(end float32) {
			if end > x {
				p.Clip(ui.Rect{X: r.X + left + x, Y: r.Y + y, W: end - x, H: c.lineHeight()}, 0, func() {
					p.TextLayout(c.layout, r.X+left, r.Y+y-raw.rawY+(c.lineHeight()-raw.rawHeight)/2, ink)
				})
			}
		}
		for _, object := range objects {
			paint(object.X)
			x = max(x, object.X+object.W)
		}
		paint(max(c.wrapWidth, r.W-left))
	}
}
