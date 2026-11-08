// Package markdown turns a small subset of Markdown into MyGo RichText spans.
package markdown

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"whaleisle/internal/markdownsyntax"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Style is the base typography for rendered message bodies.
type Style struct {
	// RenderFence is an owner-bound optional native renderer. Returning false retains code.
	RenderFence func(*ui.Context, string, string, int) bool
	// RenderImage optionally renders an image in document order. False keeps alt text.
	RenderImage func(*ui.Context, string, string) bool
	// OnOpenURL opts a document into inline external links; rendering never navigates.
	OnOpenURL func(string)
	// LiteralHTML displays raw markup as text instead of interpreting or dropping it.
	LiteralHTML bool
	// Streaming retains stable parsed blocks and renders TeX literally until settled.
	Streaming bool
	// DocumentLayout uses the original MarkdownText block typography and margins.
	// Existing conversation renderers retain their current layout unless opted in.
	DocumentLayout bool
	// Market's later-loaded .readme h3 rule overrides only these two metrics.
	// Zero keeps the shared Markdown heading; explicit values are absolute DIPs.
	DocumentH3FontSize   float32
	DocumentH3LineHeight float32
	QuoteBorder          ui.Color
	RuleBorder           ui.Color
	// FontFamily scopes prose without changing code or mathematical font faces.
	FontFamily string
	FontSize   float32
	FG         ui.Color
	Muted      ui.Color
	CodeBG     ui.Color
	Accent     ui.Color
	// Dark selects the dark syntax palette. A light palette on a dark
	// transcript renders unreadable tokens; the Original follows the theme.
	Dark bool
	// LineNumbers shows the code gutter. Off by default, as in the Original.
	LineNumbers bool
	// CodeBanner is the code-card header fill; zero derives a muted tint.
	CodeBanner ui.Color
	// ToolCodeToolbar uses distinct language, action and hover tokens; zero
	// retains the existing markdown preset for callers without this toolbar.
	ToolCodeLanguage ui.Color
	ToolCodeHover    ui.Color
	// TableHeader is the table header fill; zero derives a muted tint.
	TableHeader ui.Color
	// AvailableWidth is the width a table may use before it scrolls. Zero keeps
	// the Original 748 DIP document axis.
	AvailableWidth float32
	// QuoteInset is the blockquote text inset next to the 2px rule. Zero keeps
	// the market/readme baseline of 16px; the conversation passes 14px.
	QuoteInset float32
	// Code-card toolbar copy overrides the package's English defaults, so the
	// conversation can localize them without changing other surfaces.
	CopyLabel   string
	CopiedLabel string
	WrapLabel   string
	NoWrapLabel string
	CodeLabel   string
	// CodeWrapDefault opts a surface into wrapped code initially. The retained
	// per-block toggle still wins after the user chooses wrap or unwrap.
	CodeWrapDefault bool
	// File mentions resolve only inline code, never ordinary prose or code blocks.
	ResolveFile func(string) string
	OnOpenFile  func(string)
	codeSpanAt  func(int, string)
	fileSpanAt  func(int, string)
	blockAt     func(int, int, ast.Node)
	maths       map[int]mathFormula
	urlSpanAt   func(int, string)
	urlTargets  map[int]string
	lineHeight  float32
	textWeight  int
	docInlines  map[ast.Node]cachedDocumentInlines
}

func segmentText(seg text.Segment, src string) string {
	start := max(0, min(seg.Start, len(src)))
	stop := max(start, min(seg.Stop, len(src)))
	needNL := seg.ForceNewline && (stop == start || src[stop-1] != '\n')
	if seg.Padding == 0 && !needNL {
		return src[start:stop]
	}
	var b strings.Builder
	b.Grow(seg.Padding + (stop - start) + 1)
	for i := 0; i < seg.Padding; i++ {
		b.WriteByte(' ')
	}
	b.WriteString(src[start:stop])
	if needNL {
		b.WriteByte('\n')
	}
	return b.String()
}

func fencedLanguage(fenced *ast.FencedCodeBlock, src string) string {
	if fenced.Info == nil {
		return ""
	}
	info := segmentText(fenced.Info.Segment, src)
	for i := 0; i < len(info); i++ {
		switch info[i] {
		case ' ', '\t', '\n', '\r':
			return info[:i]
		}
	}
	return info
}

// Spans parses markdown source into RichText spans (plain text if parsing fails).
func Spans(source string, st Style) []ui.Span {
	if st.FontSize <= 0 {
		st.FontSize = theme.CurrentPreferences().Chat.Size
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	md := markdownGrammar(st.Streaming)
	reader := text.NewReader([]byte(source))
	doc := md.Parser().Parse(reader)
	var out []ui.Span
	walkBlocks(doc, source, st, &out)
	if len(out) == 0 {
		return []ui.Span{{Text: source, Size: st.FontSize, Color: st.FG}}
	}
	trimTrailingNewline(&out)
	return out
}

// parsedBody is a memoized goldmark parse of one message body. Parsing a
// whole reply is the dominant per-frame cost of a transcript row; caching it
// on the row's element keeps scrolling smooth while the body is unchanged.
// spans, blocks, maths, urlTargets and fileTargets are all pure products of
// (source, Style): the walk fills the index-keyed maps, and the OnOpenURL /
// OnOpenFile / ResolveFile callbacks only run at click time in viewSpans.
type parsedBody struct {
	source      string
	key         bodyKey
	spans       []ui.Span
	blocks      []richBlock
	maths       map[int]mathFormula
	urlTargets  map[int]string
	fileTargets map[int]string
}

// bodyKey fingerprints every Style field that changes the spans or the
// index-keyed maps Spans emits, so a theme, font or palette change re-parses
// instead of serving stale output. Function fields (RenderFence, OnOpenURL,
// ResolveFile, OnOpenFile, RenderImage) bind at click/render time, never at
// parse time, so they are not part of the key.
type bodyKey struct {
	fontSize        float32
	fg, muted       ui.Color
	codeBG, accent  ui.Color
	quoteBorder     ui.Color
	ruleBorder      ui.Color
	tableHeader     ui.Color
	codeBanner      ui.Color
	fontFamily      string
	streaming       bool
	literalHTML     bool
	documentLayout  bool
	dark            bool
	lineNumbers     bool
	codeWrapDefault bool
	availableWidth  float32
	quoteInset      float32
	lineHeight      float32
	textWeight      int
	hasOpenURL      bool
	hasOpenFile     bool
	hasRenderImage  bool
}

func makeBodyKey(st Style) bodyKey {
	return bodyKey{
		fontSize: st.FontSize, fg: st.FG, muted: st.Muted, codeBG: st.CodeBG,
		accent: st.Accent, quoteBorder: st.QuoteBorder, ruleBorder: st.RuleBorder,
		tableHeader: st.TableHeader, codeBanner: st.CodeBanner, fontFamily: st.FontFamily,
		streaming: st.Streaming, literalHTML: st.LiteralHTML, documentLayout: st.DocumentLayout,
		dark: st.Dark, lineNumbers: st.LineNumbers, codeWrapDefault: st.CodeWrapDefault,
		availableWidth: st.AvailableWidth, quoteInset: st.QuoteInset,
		lineHeight: st.lineHeight, textWeight: st.textWeight,
		hasOpenURL: st.OnOpenURL != nil, hasOpenFile: st.OnOpenFile != nil,
		hasRenderImage: st.RenderImage != nil,
	}
}

func (k bodyKey) matches(source string, st Style) bool {
	return k == makeBodyKey(st)
}

// View renders markdown in a RichText element.
func View(c *ui.Context, source string, st Style) {
	if st.FontSize <= 0 {
		st.FontSize = theme.CurrentPreferences().Chat.Size
	}
	if st.DocumentLayout {
		viewDocument(c, source, st)
		return
	}
	// A stable wrapper gives ui.Local an element whose node persists while the
	// row is mounted, so the parsed body survives frame to frame during scroll.
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(0).Selectable()
	root.Children(func() {
		cached := ui.Local(root, "parsed-body", func() parsedBody { return parsedBody{} })
		key := makeBodyKey(st)
		var spans []ui.Span
		var blocks []richBlock
		var targets map[int]string
		if cached.key.matches(source, st) && cached.source == source {
			spans, blocks, targets = cached.spans, cached.blocks, cached.fileTargets
			st.maths, st.urlTargets = cached.maths, cached.urlTargets
		} else {
			targets = map[int]string{}
			st.maths = map[int]mathFormula{}
			collectURLTargets(&st)
			collectFileTargets(&st, targets)
			st.blockAt = func(start, end int, node ast.Node) {
				blocks = append(blocks, richBlock{start: start, end: end, node: node})
			}
			spans = Spans(source, st)
			*cached = parsedBody{
				source: source, key: key, spans: spans, blocks: blocks,
				maths: st.maths, urlTargets: st.urlTargets, fileTargets: targets,
			}
		}
		if len(spans) == 0 && len(blocks) == 0 {
			return
		}
		if len(blocks) > 0 {
			viewBlocks(c, strings.TrimSpace(source), st, spans, targets, blocks)
			return
		}
		viewSpans(c, st, spans, targets, 0)
	})
}

type fileLinkIconPlacement struct {
	rect  ui.Rect
	color ui.Color
}

func linkColor(st Style) ui.Color {
	if st.Accent.A != 0 {
		return st.Accent
	}
	if st.Dark {
		return ui.Hex("#6ea8ff")
	}
	return ui.Hex("#4176e6")
}

func viewSpans(c *ui.Context, st Style, spans []ui.Span, targets map[int]string, offset int) {
	if hasMath(st.maths, offset, len(spans)) {
		viewMathSpans(c, st, spans, targets, offset)
		return
	}
	if len(spans) == 0 {
		return
	}
	if len(targets) == 0 && len(st.urlTargets) == 0 {
		styledRichText(c, spans, st)
		return
	}
	var iconPlacements []fileLinkIconPlacement
	root := styledRichText(c, nil, st)
	root.Children(func() {
		for index, span := range spans {
			path := targets[offset+index]
			href := st.urlTargets[offset+index]
			displaySpan := span
			showDocIcon := path != "" && span.Background.A == 0 && (index == 0 || targets[offset+index-1] != path)
			if showDocIcon {
				displaySpan.Text = "\u2003 " + span.Text
			}
			el := ui.RichText(c, displaySpan)
			if path != "" {
				el = el.Label(span.Text).Tooltip(path).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink)
				if el.Hovered() {
					el.Underline()
				}
				if showDocIcon {
					iconColor := displaySpan.Color
					if iconColor.A == 0 {
						iconColor = linkColor(st)
					}
					slot := len(iconPlacements)
					iconPlacements = append(iconPlacements, fileLinkIconPlacement{color: iconColor})
					el.ObserveLayout(func(rect ui.Rect) {
						if slot < len(iconPlacements) {
							iconPlacements[slot].rect = rect
						}
					})
				}
				if st.OnOpenFile != nil && (el.Clicked() || el.Shortcut(0, ui.KeyEnter) || el.Shortcut(0, ui.KeySpace)) {
					st.OnOpenFile(path)
				}
			} else if href != "" && st.OnOpenURL != nil {
				el = el.Label(span.Text).Tooltip(href).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink)
				if el.Hovered() {
					el.Underline()
				}
				if externalURLActivated(el) {
					st.OnOpenURL(href)
				}
			}
		}
	})
	if len(iconPlacements) > 0 {
		root.DrawOver(func(p *ui.Painter, rootRect ui.Rect) {
			for _, item := range iconPlacements {
				rect := item.rect
				if rect.W <= 0 || rect.H <= 0 {
					rect = rootRect
				}
				if rect.W <= 0 || rect.H <= 0 {
					continue
				}
				iconY := rect.Y + (rect.H-14)/2
				p.Icon(codeIconDocument, ui.Rect{X: rect.X, Y: iconY, W: 14, H: 14}, item.color)
			}
		})
	}
}

func proseFont(st Style) string {
	if st.FontFamily != "" {
		return st.FontFamily
	}
	return theme.RenderPreferences().Chat.Family
}

func styledRichText(c *ui.Context, spans []ui.Span, st Style) ui.Element {
	el := ui.RichText(c, spans...).Font(proseFont(st)).FontSize(st.FontSize).TextColor(st.FG)
	if st.lineHeight > 0 {
		el = el.FixedLineHeight(st.lineHeight)
	}
	if st.textWeight > 0 {
		el = el.FontWeight(st.textWeight)
	}
	return el
}

func collectURLTargets(st *Style) {
	st.urlTargets = map[int]string{}
	if st.OnOpenURL == nil {
		return
	}
	st.urlSpanAt = func(index int, raw string) {
		if href := documentURL(raw); href != "" {
			st.urlTargets[index] = href
		}
	}
}

func documentURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Hostname() != "" && u.User == nil {
			return raw
		}
	case "mailto":
		if u.Opaque != "" {
			return raw
		}
	}
	return ""
}

// Link is one markdown hyperlink extracted for interactive rendering.
type Link struct {
	Label string
	URL   string
}

// Links returns http(s) destinations found in source (order of appearance).
// The result depends only on the source text, so transcripts that rescan each
// row every frame share one small content-keyed memo instead of reparsing.
func Links(source string) []Link {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	return linkMemo.get(source)
}

// linkMemoStore is a tiny MRU cache of link scans, bounded by retained source
// bytes. Tool cards and user messages rescan the same body every render pass;
// the exact source string is the only safe key (length or seq never imply it).
type linkMemoStore struct {
	mu      sync.Mutex
	entries []linkMemoEntry
	bytes   int
}

type linkMemoEntry struct {
	source string
	links  []Link
}

const linkMemoEntryLimit = 32
const linkMemoByteLimit = 4 << 20

var linkMemo = linkMemoStore{}

func (memo *linkMemoStore) get(source string) []Link {
	if len(source) > linkMemoByteLimit {
		return parseLinks(source)
	}
	memo.mu.Lock()
	for i, entry := range memo.entries {
		if entry.source == source {
			if i > 0 {
				copy(memo.entries[1:i+1], memo.entries[:i])
				memo.entries[0] = entry
			}
			memo.mu.Unlock()
			return entry.links
		}
	}
	memo.mu.Unlock()
	links := parseLinks(source)
	memo.mu.Lock()
	defer memo.mu.Unlock()
	for len(memo.entries) >= linkMemoEntryLimit || memo.bytes+len(source) > linkMemoByteLimit {
		last := len(memo.entries) - 1
		memo.bytes -= len(memo.entries[last].source)
		memo.entries[last] = linkMemoEntry{}
		memo.entries = memo.entries[:last]
	}
	memo.entries = append(memo.entries, linkMemoEntry{})
	copy(memo.entries[1:], memo.entries[:len(memo.entries)-1])
	memo.entries[0] = linkMemoEntry{source: source, links: links}
	memo.bytes += len(source)
	return links
}

func parseLinks(source string) []Link {
	md := newMarkdown()
	reader := text.NewReader([]byte(source))
	doc := md.Parser().Parse(reader)
	var out []Link
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination, label string
		switch link := n.(type) {
		case *ast.Link:
			destination = string(link.Destination)
			var buf bytes.Buffer
			for c := link.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					buf.Write(t.Segment.Value([]byte(source)))
				}
			}
			label = strings.TrimSpace(buf.String())
		case *ast.AutoLink:
			destination = string(link.URL([]byte(source)))
			label = string(link.Label([]byte(source)))
		default:
			return ast.WalkContinue, nil
		}
		destination = strings.TrimSpace(destination)
		parsed, err := url.Parse(destination)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
			return ast.WalkContinue, nil
		}
		for _, existing := range out {
			if existing.URL == destination {
				return ast.WalkContinue, nil
			}
		}
		if label == "" {
			label = destination
		}
		out = append(out, Link{Label: label, URL: destination})
		return ast.WalkContinue, nil
	})
	return out
}

// ViewInteractive renders markdown RichText plus clickable http(s) link chips.
func ViewInteractive(c *ui.Context, source string, st Style, onLink func(url string)) {
	View(c, source, st)
	LinkActions(c, Links(source), st, onLink)
}

// LinkActions only invokes navigation after an explicit user click.
func LinkActions(c *ui.Context, links []Link, st Style, onLink func(string)) {
	if len(links) == 0 || onLink == nil {
		return
	}
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for _, link := range links {
			if ui.Button(c, "").Label("↗ " + link.Label).Tooltip(link.URL).
				MaxWidthPercent(100).MinWidth(0).FontSize(theme.InterfaceSize(st.FontSize - 1)).
				BorderWidth(0).Background(ui.Transparent).TextColor(st.Accent).Clicked() {
				onLink(link.URL)
			}
		}
	})
}

func walkBlocks(n ast.Node, src string, st Style, out *[]ui.Span) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *blockMath:
			if node.closed {
				appendFormula(node.formula, st, out)
			} else {
				appendPlainSpan(node.formula.Raw, st, out)
			}
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
		case *markdownsyntax.ImagePlaceholder:
			start := len(*out)
			appendPlainSpan("[image] "+node.Destination, st, out)
			if st.RenderImage != nil && st.blockAt != nil {
				st.blockAt(start, len(*out), node)
			}
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
		case *extast.Footnote:
			appendPlainSpan(strconv.Itoa(node.Index)+". ", st, out)
			walkBlocks(node, src, st, out)
		case *ast.Paragraph, *ast.TextBlock:
			walkInlines(node, src, st, out)
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
		case *ast.Heading:
			level := node.Level
			if level < 1 {
				level = 1
			}
			size := st.FontSize + float32(7-level*2)
			if size < st.FontSize {
				size = st.FontSize
			}
			walkInlines(node, src, st, out, func(s *ui.Span) {
				s.Size = size
				s.Weight = 700
			})
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
		case *ast.FencedCodeBlock:
			start := len(*out)
			lang, code := codeBlockSource(node, src)
			if lang == "math" && !st.Streaming {
				appendFormula(mathFormula{TeX: code, Raw: code, Display: true}, st, out)
			} else {
				*out = append(*out, highlightedCodeSpans(lang, code, st)...)
			}
			if st.blockAt != nil {
				st.blockAt(start, len(*out), node)
			}
		case *ast.CodeBlock:
			start := len(*out)
			_, code := codeBlockSource(node, src)
			*out = append(*out, highlightedCodeSpans("", code, st)...)
			if st.blockAt != nil {
				st.blockAt(start, len(*out), node)
			}
		case *extast.Table:
			start := len(*out)
			for row := node.FirstChild(); row != nil; row = row.NextSibling() {
				for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
					if cell.PreviousSibling() != nil {
						*out = append(*out, ui.Span{Text: " | ", Size: st.FontSize, Color: st.Muted})
					}
					cellStyle := st
					cellStyle.blockAt = nil // The table renderer owns images inside its cells.
					walkInlines(cell, src, cellStyle, out)
				}
				*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
			}
			if st.blockAt != nil {
				st.blockAt(start, len(*out), node)
			}
		case *ast.ListItem:
			marker := "• "
			if list, ok := node.Parent().(*ast.List); ok && list.IsOrdered() {
				number := list.Start
				for sibling := node.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
					number++
				}
				marker = strconv.Itoa(number) + ". "
			}
			depth := 0
			for parent := node.Parent(); parent != nil; parent = parent.Parent() {
				if _, ok := parent.(*ast.ListItem); ok {
					depth++
				}
			}
			*out = append(*out, ui.Span{Text: strings.Repeat("  ", depth) + marker, Size: st.FontSize, Color: st.Muted})
			// Walk the item itself so its TextBlock/Paragraph children are rendered.
			walkBlocks(node, src, st, out)
		case *ast.Blockquote:
			walkInlines(node, src, st, out, func(s *ui.Span) {
				s.Italic = true
				s.Color = st.Muted
			})
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
		case *ast.HTMLBlock:
			if st.LiteralHTML {
				appendPlainSpan(string(node.Text([]byte(src))), st, out)
			}
		default:
			if n, ok := child.(ast.Node); ok {
				walkBlocks(n, src, st, out)
			}
		}
	}
}

type spanMutator func(*ui.Span)

func walkInlines(n ast.Node, src string, st Style, out *[]ui.Span, mutators ...spanMutator) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		appendInline(child, src, st, out, mutators...)
	}
}

func appendInline(n ast.Node, src string, st Style, out *[]ui.Span, mutators ...spanMutator) {
	switch node := n.(type) {
	case *inlineMath:
		appendFormula(node.formula, st, out)
	case *ast.Text:
		text := segmentText(node.Segment, src)
		if text == "" {
			return
		}
		appendPlainSpan(text, st, out, mutators...)
	case *ast.String:
		text := string(node.Value)
		if text == "" {
			return
		}
		appendPlainSpan(text, st, out, mutators...)
	case *extast.FootnoteLink:
		span := ui.Span{Text: superscriptNumber(node.Index), Size: st.FontSize * .8, Color: st.FG}
		for _, m := range mutators {
			m(&span)
		}
		*out = append(*out, span)
	case *extast.FootnoteBacklink:
		appendPlainSpan(" ↩", st, out, mutators...)
	case *extast.TaskCheckBox:
		label := "☐ "
		if node.IsChecked {
			label = "☑ "
		}
		*out = append(*out, ui.Span{Text: label, Size: st.FontSize, Color: st.FG})
	case *extast.Strikethrough:
		combined := append(mutators, func(s *ui.Span) { s.Strikethrough = true })
		walkInlines(node, src, st, out, combined...)
	case *ast.Emphasis:
		level := node.Level
		var extra spanMutator
		if level >= 2 {
			extra = func(s *ui.Span) { s.Weight = 700 }
			if st.DocumentLayout {
				extra = func(s *ui.Span) { s.Weight = 600 }
			}
		} else {
			extra = func(s *ui.Span) { s.Italic = true }
		}
		combined := append(mutators, extra)
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			appendInline(c, src, st, out, combined...)
		}
	case *ast.CodeSpan:
		var buf strings.Builder
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.WriteString(segmentText(t.Segment, src))
			}
		}
		codeText := buf.String()
		if st.codeSpanAt != nil {
			st.codeSpanAt(len(*out), codeText)
		}
		span := codeSpan(codeText, st)
		for _, m := range mutators {
			m(&span)
		}
		*out = append(*out, span)
	case *ast.AutoLink:
		span := ui.Span{Text: string(node.Label([]byte(src))), Size: st.FontSize, Color: linkColor(st)}
		for _, m := range mutators {
			m(&span)
		}
		if st.urlSpanAt != nil {
			st.urlSpanAt(len(*out), string(node.URL([]byte(src))))
		}
		*out = append(*out, span)
	case *ast.Link:
		if st.OnOpenURL != nil || st.OnOpenFile != nil {
			start := len(*out)
			destination := string(node.Destination)
			href := documentURL(destination)
			_, _, file := ParseFileLink(destination)
			active := href != "" && st.OnOpenURL != nil || file && st.OnOpenFile != nil && !st.Streaming
			linkMutators := mutators
			if active {
				c := linkColor(st)
				linkMutators = append(append([]spanMutator(nil), mutators...), func(s *ui.Span) { s.Color = c })
			}
			walkInlines(node, src, st, out, linkMutators...)
			for index := start; index < len(*out); index++ {
				if href != "" && st.urlSpanAt != nil {
					st.urlSpanAt(index, href)
				} else if file && st.fileSpanAt != nil {
					st.fileSpanAt(index, destination)
				}
			}
			return
		}
		destination := string(node.Destination)
		_, _, file := ParseFileLink(destination)
		if file && !st.Streaming && st.fileSpanAt != nil && isDeliverableDocumentPath(destination) {
			start := len(*out)
			c := linkColor(st)
			linkMutators := append(append([]spanMutator(nil), mutators...), func(s *ui.Span) { s.Color = c })
			walkInlines(node, src, st, out, linkMutators...)
			for index := start; index < len(*out); index++ {
				st.fileSpanAt(index, destination)
			}
			return
		}
		var buf strings.Builder
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.WriteString(segmentText(t.Segment, src))
			}
		}
		label := buf.String()
		if label == "" {
			label = destination
		}
		span := ui.Span{
			Text:  label,
			Size:  st.FontSize,
			Color: linkColor(st),
		}
		for _, m := range mutators {
			m(&span)
		}
		*out = append(*out, span)
	case *ast.Image:
		start := len(*out)
		walkInlines(node, src, st, out, mutators...)
		if st.RenderImage != nil && st.blockAt != nil {
			st.blockAt(start, len(*out), node)
		}
	case *ast.RawHTML:
		if st.LiteralHTML {
			appendPlainSpan(string(node.Text([]byte(src))), st, out, mutators...)
		}
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			appendInline(c, src, st, out, mutators...)
		}
	}
}

func codeSpan(text string, st Style) ui.Span {
	return ui.Span{
		Text:       text,
		Size:       InlineCodeSize(st),
		Color:      st.FG,
		Background: st.CodeBG,
		Font:       theme.RenderPreferences().Code.Family,
	}
}

func trimTrailingNewline(spans *[]ui.Span) {
	for len(*spans) > 0 && (*spans)[len(*spans)-1].Text == "\n" {
		*spans = (*spans)[:len(*spans)-1]
	}
}
