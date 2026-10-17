// Package markdown turns a small subset of Markdown into MyGo RichText spans.
package markdown

import (
	"bytes"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Style is the base typography for rendered message bodies.
type Style struct {
	FontSize float32
	FG       ui.Color
	Muted    ui.Color
	CodeBG   ui.Color
	Accent   ui.Color
}

// Spans parses markdown source into RichText spans (plain text if parsing fails).
func Spans(source string, st Style) []ui.Span {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	md := goldmark.New()
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

// View renders markdown in a RichText element.
func View(c *ui.Context, source string, st Style) {
	spans := Spans(source, st)
	if len(spans) == 0 {
		return
	}
	ui.RichText(c, spans...).FontSize(st.FontSize).TextColor(st.FG)
}

// Link is one markdown hyperlink extracted for interactive rendering.
type Link struct {
	Label string
	URL   string
}

// Links returns http(s) destinations found in source (order of appearance).
func Links(source string) []Link {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	md := goldmark.New()
	reader := text.NewReader([]byte(source))
	doc := md.Parser().Parse(reader)
	var out []Link
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		url := strings.TrimSpace(string(link.Destination))
		if url == "" || !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			return ast.WalkContinue, nil
		}
		var buf bytes.Buffer
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value([]byte(source)))
			}
		}
		label := strings.TrimSpace(buf.String())
		if label == "" {
			label = url
		}
		out = append(out, Link{Label: label, URL: url})
		return ast.WalkContinue, nil
	})
	return out
}

// ViewInteractive renders markdown RichText plus clickable http(s) link chips.
func ViewInteractive(c *ui.Context, source string, st Style, onLink func(url string)) {
	View(c, source, st)
	links := Links(source)
	if len(links) == 0 || onLink == nil {
		return
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, l := range links {
			link := l
			if ui.Button(c, "↗ "+link.Label).FontSize(st.FontSize - 1).BorderWidth(0).
				Background(ui.Transparent).TextColor(st.Accent).Clicked() {
				onLink(link.URL)
			}
		}
	})
}

func walkBlocks(n ast.Node, src string, st Style, out *[]ui.Span) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Paragraph:
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
			var buf bytes.Buffer
			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				buf.Write(line.Value([]byte(src)))
			}
			lang := string(node.Language([]byte(src)))
			*out = append(*out, highlightedCodeSpans(lang, buf.String(), st)...)
		case *ast.CodeBlock:
			var buf bytes.Buffer
			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				buf.Write(line.Value([]byte(src)))
			}
			*out = append(*out, highlightedCodeSpans("", buf.String(), st)...)
		case *ast.ListItem:
			*out = append(*out, ui.Span{Text: "• ", Size: st.FontSize, Color: st.Muted})
			for inner := node.FirstChild(); inner != nil; inner = inner.NextSibling() {
				walkBlocks(inner, src, st, out)
			}
		case *ast.Blockquote:
			walkInlines(node, src, st, out, func(s *ui.Span) {
				s.Italic = true
				s.Color = st.Muted
			})
			*out = append(*out, ui.Span{Text: "\n", Size: st.FontSize})
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
	case *ast.Text:
		segment := node.Segment
		text := string(segment.Value([]byte(src)))
		if text == "" {
			return
		}
		appendMathAwareText(text, st, out, mutators...)
	case *ast.String:
		text := string(node.Value)
		if text == "" {
			return
		}
		appendMathAwareText(text, st, out, mutators...)
	case *ast.Emphasis:
		level := node.Level
		var extra spanMutator
		if level >= 2 {
			extra = func(s *ui.Span) { s.Weight = 700 }
		} else {
			extra = func(s *ui.Span) { s.Italic = true }
		}
		combined := append(mutators, extra)
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			appendInline(c, src, st, out, combined...)
		}
	case *ast.CodeSpan:
		var buf bytes.Buffer
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value([]byte(src)))
			}
		}
		*out = append(*out, codeSpan(buf.String(), st))
	case *ast.Link:
		var buf bytes.Buffer
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value([]byte(src)))
			}
		}
		label := buf.String()
		if label == "" {
			label = string(node.Destination)
		}
		span := ui.Span{
			Text:      label,
			Size:      st.FontSize,
			Color:     st.Accent,
			Underline: true,
		}
		for _, m := range mutators {
			m(&span)
		}
		*out = append(*out, span)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			appendInline(c, src, st, out, mutators...)
		}
	}
}

func codeSpan(text string, st Style) ui.Span {
	return ui.Span{
		Text:       text,
		Size:       st.FontSize - 1,
		Color:      st.FG,
		Background: st.CodeBG,
		Font:       "Consolas",
	}
}

func trimTrailingNewline(spans *[]ui.Span) {
	for len(*spans) > 0 && (*spans)[len(*spans)-1].Text == "\n" {
		*spans = (*spans)[:len(*spans)-1]
	}
}
