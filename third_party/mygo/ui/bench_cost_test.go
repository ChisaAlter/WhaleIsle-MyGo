package ui

import (
	"strconv"
	"testing"
)

// costTree is a view of about 5000 elements: 34 groups of a header and
// 24 rows of six elements, one of them a text.
// It keeps to some 850 texts, whose layouts the text system's cache holds, so
// the frames measure the engine rather than shaping.
type costTree struct {
	rev    [costGroups]uint64
	labels [costGroups][costRows]string
	memo   bool
}

const costGroups, costRows = 34, 24

func newCostTree() *costTree {
	t := &costTree{}
	for g := range costGroups {
		for r := range costRows {
			t.labels[g][r] = "Group " + strconv.Itoa(g) + " row " + strconv.Itoa(r)
		}
	}
	return t
}

func (t *costTree) group(c *context, g int) {
	coreColumn(c).Gap(2).Children(func() {
		coreText(c, "Header "+strconv.Itoa(g)).Bold()
		for r := range costRows {
			coreRow(c).Padding(2, 4).Gap(4).Children(func() {
				coreBox(c).Size(12, 12).Background(RGB(200, 200, 200))
				coreBox(c).Size(8, 8)
				coreBox(c).Grow(1).Row().Children(func() {
					coreBox(c).Size(4, 4)
					coreText(c, t.labels[g][r]).FontSize(12)
				})
			})
		}
	})
}

func (t *costTree) view(c *context) {
	coreColumn(c).Fill().ClipY().Children(func() {
		for g := range costGroups {
			t.group(c, g)
		}
	})
}

func costTester(t *costTree) *Tester {
	tt := coreNewTester(t.view, 1280, 800)
	for range 5 {
		tt.Frame()
	}
	return tt
}

// BenchmarkCostTreeSteady builds a frame of a 5000-element tree where
// nothing changed.
func BenchmarkCostTreeSteady(b *testing.B) {
	t := newCostTree()
	tt := costTester(t)
	b.ReportAllocs()
	for b.Loop() {
		tt.Frame()
	}
}

// BenchmarkCostTreeLeaf builds frames of the tree where one leaf's text
// changes each frame.
func BenchmarkCostTreeLeaf(b *testing.B) {
	t := newCostTree()
	tt := costTester(t)
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		n++
		t.labels[3][5] = "changed " + strconv.Itoa(n&7)
		t.rev[3]++
		tt.Frame()
	}
}
