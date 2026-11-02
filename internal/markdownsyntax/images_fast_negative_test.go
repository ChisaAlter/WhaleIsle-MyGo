package markdownsyntax

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// imageReferencesBeforeFastNegative preserves the complete pre-change parser
// path as an oracle, including parser construction and the authored-order walk.
func imageReferencesBeforeFastNegative(source string) []string {
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

func TestImageReferencesFastNegativeMatchesPreviousParser(t *testing.T) {
	fixtures := []struct {
		name, source string
	}{
		{"empty", ""},
		{"whitespace", " \t\r\n"},
		{"plain", "Synthetic question 399. No image here! (answer.png)"},
		{"unicode", "中文说明、café e\u0301、日本語、🧪🐋，结束。"},
		{"markdown_without_images", "# Heading\n\n> **bold** and _emphasis_\n\n- one\n- two\n\nhttps://example.test/a.png"},
		{"html_image", "<img src=\"image.png\" alt=\"only HTML\">"},
		{"encoded_opening", "!&#91;encoded](image.png) !%5Burl](encoded.png)"},
		{"fullwidth_bracket", "!［不是语法］(image.png)"},
		{"ordinary_link", "[open](picture.png)"},
		{"legacy_without_bang", "[image] attachment:fixture\n[image] C:\\My plots\\photo (1).png"},
		{"inline_images", "![one](./a_(1).png) ![two](<C:/My plots/b (2).png> \"title\")"},
		{"recovered_spaces", "![local](./My plots/c (3).png)"},
		{"reference_images", "![one][image-ref] ![image-ref][] ![image-ref]\n\n[image-ref]: ./reference.png"},
		{"escaped_and_code", "\\![escaped](escaped.png) `![inline](inline.png)`\n\n```md\n![fenced](fenced.png)\n[image] attachment:fenced\n```\n\n![real](real.png)"},
		{"duplicates_order", "![a](same.png)\n\n[image] attachment:second\n\n![b](same.png)"},
		{"broken_syntax", "![missing]( ![open [image] \ntext ]"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			want := imageReferencesBeforeFastNegative(fixture.source)
			if got := ImageReferences(fixture.source); !reflect.DeepEqual(got, want) {
				t.Fatalf("references=%#v, previous parser=%#v", got, want)
			}
		})
	}
	for i := 0; i < 64; i++ {
		// Vary Unicode, punctuation and line boundaries without introducing '['.
		source := fmt.Sprintf("问题 %03d: %s\n答案 (路径_%d.png)! café 🐋。", i, strings.Repeat("说明 αβγ—日本語 ", i%11+1), i*i)
		want := imageReferencesBeforeFastNegative(source)
		if got := ImageReferences(source); !reflect.DeepEqual(got, want) {
			t.Fatalf("generated case %d: references=%#v, previous parser=%#v", i, got, want)
		}
	}
}

var imageReferencesBenchmarkSink []string

func BenchmarkImageReferencesPlainText(b *testing.B) {
	for _, fixture := range []struct {
		name, source string
	}{
		{"short", "Synthetic question 399"},
		{"unicode", strings.Repeat("中文说明、café 日本語 🐋。No image!\n", 32)},
	} {
		b.Run(fixture.name, func(b *testing.B) {
			b.Run("previous_parser", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					imageReferencesBenchmarkSink = imageReferencesBeforeFastNegative(fixture.source)
				}
			})
			b.Run("fast_negative", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					imageReferencesBenchmarkSink = ImageReferences(fixture.source)
				}
			})
		})
	}
}
