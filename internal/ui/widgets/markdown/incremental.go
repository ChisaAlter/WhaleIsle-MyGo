package markdown

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const sourceStartAttribute = "whale-source-start"

// positionedParser records the grammar's own opening position, before heading,
// quote and fence parsers consume their delimiters. Paragraph transformations
// (tables and setext headings) recover their enclosing source line below.
type positionedParser struct{ parser.BlockParser }

func (p positionedParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	_, segment := reader.Position()
	previous := pc.LastOpenedBlock().Node
	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil {
		start := segment.Start
		if state&parser.RequireParagraph != 0 && previous != nil {
			if value, ok := previous.AttributeString(sourceStartAttribute); ok {
				start = value.(int)
			}
		}
		node.SetAttributeString(sourceStartAttribute, start)
	}
	return node, state
}

func positionedBlockParsers() []util.PrioritizedValue {
	blocks := parser.DefaultBlockParsers()
	for index := range blocks {
		blocks[index].Value = positionedParser{blocks[index].Value.(parser.BlockParser)}
	}
	return blocks
}

type documentBlock struct {
	node   ast.Node
	source string
	base   int
	key    int
}

type cachedDocumentInlines struct {
	source     string
	key        bodyKey
	spans      []ui.Span
	blocks     []richBlock
	targets    map[int]string
	maths      map[int]mathFormula
	urlTargets map[int]string
}

type streamDocument struct {
	previous    string
	frozen      []documentBlock
	tail        []documentBlock
	combined    []documentBlock
	inlines     map[ast.Node]cachedDocumentInlines
	tailStart   int
	settled     bool
	initialized bool
	// parsedBytes counts grammar input, not the cheap append-prefix comparison.
	// Kept here so behavior tests can verify the long-reply work bound.
	parsedBytes int
	openFence   *streamFence
}

type streamFence struct {
	blockIndex int
	marker     byte
	length     int
	indent     int
	pending    int
}

func sourceNodeStart(node ast.Node, source string) (int, bool) {
	if value, ok := node.AttributeString(sourceStartAttribute); ok {
		return value.(int), true
	}
	if node.Type() == ast.TypeBlock && node.Lines().Len() > 0 {
		start := node.Lines().At(0).Start
		return strings.LastIndex(source[:min(start, len(source))], "\n") + 1, true
	}
	start := len(source)
	found := false
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if at, ok := sourceNodeStart(child, source); ok && at < start {
			start, found = at, true
		}
	}
	return start, found
}

func (s *streamDocument) parse(source string, base int, streaming bool) []documentBlock {
	// A suffix substring otherwise pins an old complete reply for each frozen
	// batch. Keep only the bounded parse slice alongside its source offsets.
	source = strings.Clone(source)
	s.parsedBytes += len(source)
	doc := markdownGrammar(streaming).Parser().Parse(text.NewReader([]byte(source)))
	var blocks []documentBlock
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {
		start, ok := sourceNodeStart(node, source)
		key := -(len(blocks) + 1)
		if ok {
			key = base + start
		}
		blocks = append(blocks, documentBlock{node: node, source: source, base: base, key: key})
	}
	return blocks
}

func (s *streamDocument) rebuildCombined() []documentBlock {
	total := len(s.frozen) + len(s.tail)
	if cap(s.combined) < total {
		s.combined = make([]documentBlock, 0, total)
	} else {
		s.combined = s.combined[:0]
	}
	s.combined = append(s.combined, s.frozen...)
	s.combined = append(s.combined, s.tail...)
	return s.combined
}

func (s *streamDocument) update(source string, streaming bool) []documentBlock {
	if s.inlines == nil {
		s.inlines = make(map[ast.Node]cachedDocumentInlines)
	}
	if s.initialized && source == s.previous && s.settled == !streaming {
		return s.combined
	}
	if !streaming || s.settled || !strings.HasPrefix(source, s.previous) {
		s.frozen, s.tail, s.combined, s.tailStart, s.openFence = nil, nil, nil, 0, nil
		clear(s.inlines)
	} else if len(s.inlines) > 512 {
		clear(s.inlines)
	} else {
		for _, oldTail := range s.tail {
			s.forgetInlines(oldTail.node)
		}
	}
	s.initialized, s.settled = true, !streaming
	if streaming && s.openFence != nil && strings.HasPrefix(source, s.previous) && s.extendFence(source) {
		s.previous = source
		return s.rebuildCombined()
	}
	s.openFence = nil
	s.previous = source
	base := s.tailStart
	blocks := s.parse(source[base:], base, streaming)
	freeze := len(blocks) - 2
	if streaming && freeze > 0 {
		// A footnote section is reordered to the document end by Goldmark. Keep
		// its complete current parse together so its numbers never duplicate.
		for _, block := range blocks {
			if _, footnotes := block.node.(*extast.FootnoteList); footnotes {
				freeze = 0
				break
			}
		}
		if freeze > 0 {
			cut, ok := sourceNodeStart(blocks[freeze].node, blocks[freeze].source)
			if ok && cut > 0 {
				s.frozen = append(s.frozen, blocks[:freeze]...)
				s.tailStart = base + cut
				blocks = blocks[freeze:]
			}
		}
	}
	s.tail = blocks
	if streaming {
		s.recognizeFence(source)
	}
	return s.rebuildCombined()
}

// forgetInlines drops the inline cache entries of node and of everything below
// it. A list, quote or table caches its nested paragraphs and cells, never the
// top-level node itself, so deleting only the root left every discarded tail's
// descendants behind: a long list streamed in grew the cache past its bound
// within a few hundred chunks and the bound then cleared the whole cache,
// including the settled blocks above the tail.
func (s *streamDocument) forgetInlines(node ast.Node) {
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			delete(s.inlines, n)
		}
		return ast.WalkContinue, nil
	})
}

// recognizeFence installs a second frontier for the last unclosed top-level
// fence. Completed content lines remain in the AST; only the partial line and
// newly appended lines return through the grammar.
func (s *streamDocument) recognizeFence(source string) {
	if len(s.tail) == 0 {
		return
	}
	index := len(s.tail) - 1
	block := s.tail[index]
	node, ok := block.node.(*ast.FencedCodeBlock)
	if !ok || node.Lines().Len() == 0 {
		return
	}
	start, ok := sourceNodeStart(node, block.source)
	if !ok {
		return
	}
	end := strings.IndexByte(block.source[start:], '\n')
	if end < 0 {
		return
	}
	opening := strings.TrimRight(block.source[start:start+end], "\r")
	indent := len(opening) - len(strings.TrimLeft(opening, " "))
	if indent > 3 || indent >= len(opening) {
		return
	}
	marker := opening[indent]
	if marker != '`' && marker != '~' {
		return
	}
	run := 0
	for indent+run < len(opening) && opening[indent+run] == marker {
		run++
	}
	if run < 3 {
		return
	}
	content := block.base + start + end + 1
	if closingFence(source[content:], marker, run) {
		return
	}
	pending := content
	if line := strings.LastIndexByte(source[content:], '\n'); line >= 0 {
		pending = content + line + 1
	}
	s.openFence = &streamFence{blockIndex: index, marker: marker, length: run, indent: indent, pending: pending}
}

func closingFence(source string, marker byte, length int) bool {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSuffix(line, "\r")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent > 3 {
			continue
		}
		run := 0
		for indent+run < len(line) && line[indent+run] == marker {
			run++
		}
		if run >= length && strings.TrimSpace(line[indent+run:]) == "" {
			return true
		}
	}
	return false
}

func (s *streamDocument) extendFence(source string) bool {
	state := s.openFence
	pending := source[state.pending:]
	if closingFence(pending, state.marker, state.length) {
		return false
	}
	block := s.tail[state.blockIndex]
	node := block.node.(*ast.FencedCodeBlock)
	prefix := strings.Repeat(" ", state.indent) + strings.Repeat(string(state.marker), state.length) + "\n"
	parsed := s.parse(prefix+pending, 0, true)
	if len(parsed) != 1 {
		return false
	}
	suffix, ok := parsed[0].node.(*ast.FencedCodeBlock)
	if !ok {
		return false
	}
	// Retain all completed lines; the parser's synthetic EOF newline belongs
	// only to the replaced partial line.
	lines := node.Lines()
	kept := text.NewSegments()
	for index := 0; index < lines.Len(); index++ {
		segment := lines.At(index)
		if block.base+segment.Start >= state.pending {
			break
		}
		kept.Append(segment)
	}
	for index := 0; index < suffix.Lines().Len(); index++ {
		segment := suffix.Lines().At(index)
		segment.Start += state.pending - block.base - len(prefix)
		segment.Stop += state.pending - block.base - len(prefix)
		kept.Append(segment)
	}
	node.SetLines(kept)
	block.source = source[block.base:]
	s.tail[state.blockIndex] = block
	if end := strings.LastIndexByte(pending, '\n'); end >= 0 {
		state.pending += end + 1
	}
	return true
}

// viewStreamingDocument keeps one parser per retained message. Stable source
// keys survive freezing, final settlement and tail edits, preserving code-wrap
// choices and image/control ownership through stream updates.
func viewStreamingDocument(c *ui.Context, source string, st Style) {
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(0).Selectable()
	state := ui.Local(root, "markdown-document", func() streamDocument { return streamDocument{} })
	blocks := state.update(source, st.Streaming)
	st.lineHeight = BodyLineHeight(st)
	st.docInlines = state.inlines
	root.Children(func() {
		var previous ast.Node
		var previousBottom float32
		for _, block := range blocks {
			top, bottom := documentMargins(block.node)
			gap := max(previousBottom, top)
			if previous == nil {
				gap = 0
			}
			if heading, ok := previous.(*ast.Heading); ok && heading.Level >= 4 {
				if _, list := block.node.(*ast.List); list {
					gap = 8
				}
			}
			ui.Column(c.Key("md-block-"+strconv.Itoa(block.key))).FillWidth().MinWidth(0).Padding(gap, 0, 0, 0).Children(func() {
				blockStyle := st
				if st.RenderFence != nil {
					blockStyle.RenderFence = func(ctx *ui.Context, language, program string, offset int) bool {
						return st.RenderFence(ctx, language, program, block.base+offset)
					}
				}
				viewDocumentBlock(c, block.node, block.source, blockStyle)
			})
			previous, previousBottom = block.node, bottom
		}
	})
}
