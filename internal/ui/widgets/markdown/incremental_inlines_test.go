package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
)

// cacheInlinesLikeRendering stores an entry for every paragraph and text block
// under the blocks that has none, which is what viewDocumentInlines does for each
// block it draws (a list caches its items' paragraphs, a table its cells). It
// returns how many nodes are live and how many of them were cache misses.
func cacheInlinesLikeRendering(state *streamDocument, blocks []documentBlock) (live, misses int) {
	for _, block := range blocks {
		_ = ast.Walk(block.node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch n.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				live++
				if _, ok := state.inlines[n]; !ok {
					misses++
					state.inlines[n] = cachedDocumentInlines{source: block.source}
				}
			}
			return ast.WalkContinue, nil
		})
	}
	return live, misses
}

func settledPrefix(paragraphs int) *strings.Builder {
	var source strings.Builder
	source.WriteString("# Title\n\n")
	for i := 0; i < paragraphs; i++ {
		fmt.Fprintf(&source, "Settled paragraph %d.\n\n", i)
	}
	return &source
}

// While a long list streams, every chunk reparses the whole list (it is one
// top-level block) and discards the previous parse. The discarded parse's nested
// paragraphs must leave the cache with it: after every frame the cache holds
// exactly the nodes that are alive, and the settled paragraphs above the list are
// never recomputed because of the tail.
func TestStreamingListDoesNotLeakNestedInlineCacheEntries(t *testing.T) {
	var state streamDocument
	source := settledPrefix(100)
	var settled ast.Node
	lostSettled := 0
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(source, "- item number %d with some words\n", i)
		blocks := state.update(source.String(), true)
		if settled == nil {
			for _, block := range blocks {
				if _, ok := block.node.(*ast.Paragraph); ok {
					settled = block.node
					break
				}
			}
		} else if _, ok := state.inlines[settled]; !ok {
			lostSettled++
		}
		live, _ := cacheInlinesLikeRendering(&state, blocks)
		if got := len(state.inlines); got != live {
			t.Fatalf("chunk %d: cache holds %d entries for %d live nodes: discarded parses leaked", i, got, live)
		}
	}
	if lostSettled != 0 {
		t.Fatalf("a settled paragraph lost its cached inlines in %d chunks because of the streaming tail", lostSettled)
	}
}

// The same holds for a table, whose cells are cached rather than the table.
func TestStreamingTableDoesNotLeakCellInlineCacheEntries(t *testing.T) {
	var state streamDocument
	source := settledPrefix(20)
	source.WriteString("| A | B |\n| --- | --- |\n")
	for i := 1; i <= 120; i++ {
		fmt.Fprintf(source, "| cell %d | value %d |\n", i, i)
		blocks := state.update(source.String(), true)
		cacheInlinesLikeRendering(&state, blocks)
		alive := 0
		for _, block := range blocks {
			_ = ast.Walk(block.node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if _, ok := state.inlines[n]; entering && ok {
					alive++
				}
				return ast.WalkContinue, nil
			})
		}
		if got := len(state.inlines); got != alive {
			t.Fatalf("row %d: %d cached entries but only %d belong to live nodes", i, got, alive)
		}
	}
}
