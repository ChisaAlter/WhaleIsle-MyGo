package interaction

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"whaleisle/internal/ui/planpreview"
)

// The approval takeover carries the first line and first paragraph only. The
// complete Markdown stays in the independently opened plan resource.
type planReviewSummary struct {
	source, title, description string
	ready                      bool
}

func (s *planReviewSummary) resolve(source string) {
	if s.ready && s.source == source {
		return
	}
	*s = planReviewSummary{source: source, ready: true}
	raw := []byte(source)
	doc := goldmark.New().Parser().Parse(text.NewReader(raw))
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() != ast.KindHeading && node.Kind() != ast.KindParagraph {
			return ast.WalkContinue, nil
		}
		var plain strings.Builder
		_ = ast.Walk(node, func(inline ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				switch value := inline.(type) {
				case *ast.Text:
					plain.Write(value.Segment.Value(raw))
					if value.SoftLineBreak() || value.HardLineBreak() {
						plain.WriteByte('\n')
					}
				case *ast.String:
					plain.Write(value.Value)
				}
			}
			return ast.WalkContinue, nil
		})
		value := strings.TrimSpace(plain.String())
		if s.title == "" && value != "" {
			s.title = strings.SplitN(value, "\n", 2)[0]
		}
		if node.Kind() == ast.KindParagraph && value != "" && s.description == "" {
			s.description = strings.Join(strings.Fields(value), " ")
			return ast.WalkStop, nil
		}
		return ast.WalkSkipChildren, nil
	})
	if s.title == "" {
		s.title = planpreview.Text("提交的计划", "Submitted plan")
	}
	if s.description == s.title {
		s.description = ""
	}
}
