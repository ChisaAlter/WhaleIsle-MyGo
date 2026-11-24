package text

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unsafe"
	"weak"
)

func TestShapeCacheHitsAndBounds(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}

	p := Params{Text: "New conversation", Style: Style{Size: 13}, Width: 180, MaxLines: 1}
	first := s.Shape(p)
	for range 10 {
		if got := s.Shape(p); got != first {
			t.Fatal("Shape missed cache for identical Params")
		}
	}
	if made := s.LayoutsMade(); made != 1 {
		t.Fatalf("expected 1 layout built, got %d", made)
	}

	// Fill beyond maxShapes and verify strict LRU budget enforcement.
	for i := range maxShapes + 64 {
		s.Shape(Params{Text: fmt.Sprintf("hot-item-%d", i)})
		if s.shapeBytes > maxShapeBytes || len(s.shapes) > maxShapes {
			t.Fatalf("shape cache exceeded budget: bytes=%d entries=%d", s.shapeBytes, len(s.shapes))
		}
		// Keep p hot so LRU never evicts it under pressure.
		if got := s.Shape(p); got != first {
			t.Fatalf("LRU evicted hot Shape entry at iteration %d", i)
		}
	}

	// Oversized paragraph must bypass the Shape cache without displacing hot labels.
	large := s.Shape(Params{Text: strings.Repeat("x", maxShapeEntryBytes/16)})
	if len(large.Runes) != maxShapeEntryBytes/16 {
		t.Fatal("oversized Shape did not lay out")
	}
	if got := s.Shape(p); got != first {
		t.Fatal("oversized Shape displaced hot entry")
	}

	// EndFrame age eviction and clearLayouts must clear the Shape cache cleanly.
	for range 245 {
		s.EndFrame()
	}
	if len(s.shapes) != 0 || s.shapeBytes != 0 {
		t.Fatalf("stale shapes not evicted after 240+ frames: entries=%d bytes=%d", len(s.shapes), s.shapeBytes)
	}
}

func cacheShapeExcerpt(s *System) weak.Pointer[byte] {
	doc := strings.Repeat("y", 1<<20)
	old := weak.Make(unsafe.StringData(doc))
	s.Shape(Params{Text: doc[100:115]})
	return old
}

func TestShapeCacheDoesNotRetainExcerptSource(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	old := cacheShapeExcerpt(s)
	runtime.GC()
	if old.Value() != nil {
		t.Fatal("cached Shape excerpt retained its large source document")
	}
	runtime.KeepAlive(s)
}

func TestShapeCacheConcurrentGeometrySafe(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	p := Params{Text: "hello world caret geometry test", Width: 200}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				l := s.Shape(p)
				l.Caret(5)
				l.CaretAt(CaretPosition{Index: 5, Affinity: Downstream})
				l.CaretAt(CaretPosition{Index: 5, Affinity: Upstream})
				l.PositionAt(12, 4)
				l.MoveCaret(CaretPosition{Index: 5, Affinity: Downstream}, 1)
				l.SelectionRanges(CaretPosition{Index: 2, Affinity: Downstream}, CaretPosition{Index: 10, Affinity: Upstream})
				_ = l.GraphemeOffsets()
			}
		}()
	}
	wg.Wait()
}

func BenchmarkShapeCached(b *testing.B) {
	s := Shared()
	p := Params{Text: "Sidebar conversation title with CJK 中文 123", Style: Style{Size: 13}, Width: 200, MaxLines: 1}
	_ = s.Shape(p)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = s.Shape(p)
	}
}
