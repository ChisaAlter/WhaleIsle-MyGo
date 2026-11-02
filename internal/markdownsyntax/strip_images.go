package markdownsyntax

import (
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// WithoutImages removes parsed image occurrences from prompt text. Code and
// escaped examples remain verbatim; source ranges come from the link grammar.
func WithoutImages(source string) string {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, Images))
	pc := parser.NewContext()
	doc := md.Parser().Parse(text.NewReader([]byte(source)), parser.WithContext(pc))
	ranges, _ := pc.Get(imageRangeKey).([]imageRange)
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if placeholder, ok := node.(*ImagePlaceholder); entering && ok && placeholder.Lines().Len() > 0 {
			segment := placeholder.Lines().At(0)
			ranges = append(ranges, imageRange{segment.Start, segment.Stop})
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	var out strings.Builder
	at := 0
	for _, span := range ranges {
		if span.start < at || span.end < span.start || span.end > len(source) {
			continue
		}
		out.WriteString(source[at:span.start])
		out.WriteByte(' ')
		at = span.end
	}
	out.WriteString(source[at:])
	return out.String()
}
