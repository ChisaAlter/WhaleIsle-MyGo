// Package markdownsyntax shares authored image syntax between session projection
// and Markdown rendering. Source offsets remain unchanged by recovery.
package markdownsyntax

import (
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Images preserves legacy attachment lines and recovers unambiguous local image
// paths containing bare spaces. Ordinary Markdown uses Goldmark's link grammar.
var Images goldmark.Extender = imagesExtension{}

type imagesExtension struct{}

func (imagesExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		imageRangesOption{},
		parser.WithInlineParsers(util.Prioritized(localImageParser{}, 150)),
		parser.WithBlockParsers(util.Prioritized(placeholderParser{}, 50)),
	)
}

type imageRangesOption struct{}

func (imageRangesOption) SetParserOption(config *parser.Config) {
	for index := range config.InlineParsers {
		if config.InlineParsers[index].Value == parser.NewLinkParser() {
			config.InlineParsers[index].Value = imageLinkParser{}
		}
	}
}

type imageRange struct{ start, end int }

var imageRangeKey = parser.NewContextKey()
var imageOpensKey = parser.NewContextKey()

type imageOpen struct {
	start int
	image bool
}
type imageLinkParser struct{}

func (imageLinkParser) Trigger() []byte { return []byte{'!', '[', ']'} }
func (imageLinkParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	_, before := reader.Position()
	marker := reader.Peek()
	n := parser.NewLinkParser().Parse(parent, reader, pc)
	stack, _ := pc.Get(imageOpensKey).([]imageOpen)
	if n != nil && (marker == '!' || marker == '[') {
		stack = append(stack, imageOpen{before.Start, marker == '!'})
	} else if marker == ']' && len(stack) > 0 {
		open := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, image := n.(*ast.Image); image && open.image {
			_, after := reader.Position()
			ranges, _ := pc.Get(imageRangeKey).([]imageRange)
			pc.Set(imageRangeKey, append(ranges, imageRange{open.start, after.Start}))
		}
	}
	pc.Set(imageOpensKey, stack)
	return n
}

func (imageLinkParser) CloseBlock(parent ast.Node, reader text.Reader, pc parser.Context) {
	parser.NewLinkParser().(parser.CloseBlocker).CloseBlock(parent, reader, pc)
	pc.Set(imageOpensKey, nil)
}

var placeholderKind = ast.NewNodeKind("ImagePlaceholder")

// ImagePlaceholder is a legacy [image] attachment at its original block position.
type ImagePlaceholder struct {
	ast.BaseBlock
	Destination string
}

func (n *ImagePlaceholder) Kind() ast.NodeKind { return placeholderKind }
func (n *ImagePlaceholder) IsRaw() bool        { return true }
func (n *ImagePlaceholder) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type placeholderParser struct{}

func (placeholderParser) Trigger() []byte { return []byte{'['} }
func (placeholderParser) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pos >= len(line) || !strings.HasPrefix(string(line[pos:]), "[image] ") {
		return nil, parser.NoChildren
	}
	path := strings.TrimSpace(string(line[pos+len("[image] "):]))
	if path == "" {
		return nil, parser.NoChildren
	}
	n := &ImagePlaceholder{Destination: path}
	n.Lines().Append(segment)
	return n, parser.NoChildren
}
func (placeholderParser) Continue(ast.Node, text.Reader, parser.Context) parser.State {
	return parser.Close
}
func (placeholderParser) Close(ast.Node, text.Reader, parser.Context) {}
func (placeholderParser) CanInterruptParagraph() bool                 { return true }
func (placeholderParser) CanAcceptIndentedLine() bool                 { return false }

type localImageParser struct{}

func (localImageParser) Trigger() []byte { return []byte{'!'} }
func (localImageParser) Parse(_ ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	line, _ := reader.PeekLine()
	value := string(line)
	if !strings.HasPrefix(value, "![") {
		return nil
	}
	endAlt := strings.Index(value, "](")
	if endAlt < 2 || strings.ContainsAny(value[2:endAlt], "[\n\r") {
		return nil
	}
	start := endAlt + 2
	depth := 0
	for end := start; end < len(value); end++ {
		if value[end] == '\\' && end+1 < len(value) && strings.ContainsRune("()[]", rune(value[end+1])) {
			end++
			continue
		}
		switch value[end] {
		case '\n', '\r', '<', '>', '"', '\'':
			return nil
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
				continue
			}
			path := value[start:end]
			if !strings.ContainsRune(path, ' ') || strings.TrimSpace(path) != path || !localPath(path) || !imageExtension(path) {
				return nil
			}
			link := ast.NewLink()
			link.Destination = util.UnescapePunctuations([]byte(path))
			link.AppendChild(link, ast.NewString(util.UnescapePunctuations([]byte(value[2:endAlt]))))
			n := ast.NewImage(link)
			_, before := reader.Position()
			reader.Advance(end + 1)
			_, after := reader.Position()
			ranges, _ := pc.Get(imageRangeKey).([]imageRange)
			pc.Set(imageRangeKey, append(ranges, imageRange{before.Start, after.Start}))
			return n
		}
	}
	return nil
}

func localPath(path string) bool {
	return (strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")) ||
		strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") ||
		(len(path) >= 3 && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) && path[1] == ':' && (path[2] == '/' || path[2] == '\\'))
}

func imageExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".tif", ".tiff", ".avif", ".heic":
		return true
	}
	return false
}

// ImageReferences returns all image occurrences in authored order, including
// reference images and placeholders, while ignoring code and escaped examples.
func ImageReferences(source string) []string {
	if !strings.Contains(source, "[") {
		return nil
	}
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, Images))
	doc := md.Parser().Parse(text.NewReader([]byte(source)))
	var refs []string
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := node.(type) {
			case *ast.Image:
				refs = append(refs, string(util.UnescapePunctuations(n.Destination)))
			case *ImagePlaceholder:
				refs = append(refs, n.Destination)
			}
		}
		return ast.WalkContinue, nil
	})
	return refs
}

// OnlyImages reports whether a parsed body contains images and whitespace only.
// It preserves each image, including repeated references.
func OnlyImages(source string) ([]string, bool) {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, Images))
	doc := md.Parser().Parse(text.NewReader([]byte(source)))
	var refs []string
	valid := true
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Document, *ast.Paragraph:
		case *ast.Image:
			refs = append(refs, string(util.UnescapePunctuations(n.Destination)))
			return ast.WalkSkipChildren, nil
		case *ImagePlaceholder:
			refs = append(refs, n.Destination)
		case *ast.Text:
			valid = valid && strings.TrimSpace(string(n.Segment.Value([]byte(source)))) == ""
		default:
			valid = false
		}
		return ast.WalkContinue, nil
	})
	return refs, valid && len(refs) > 0
}
