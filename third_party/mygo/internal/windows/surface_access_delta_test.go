//go:build windows && (amd64 || arm64)

package windows

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func quietTree() *uiaTree {
	uiaOnce.Do(initUIA)
	t := &uiaTree{s: &surface{}, nodes: map[uint64]*uiaElement{}, quiet: true}
	t.root = newUIAElement(t)
	t.root.root = true
	return t
}

// treeShape describes a uiaTree as its content's nodes, with the IDs of
// each element's children, for comparing two.
func treeShape(t *uiaTree) string {
	s := fmt.Sprintf("focus %d;", t.focus)
	for _, e := range append([]*uiaElement{t.root}, t.order...) {
		var kids []uint64
		for _, k := range e.children {
			kids = append(kids, k.n.ID)
		}
		s += fmt.Sprintf("%+v %v;", e.n, kids)
	}
	return s
}

func randomNodes(rng *rand.Rand, ids []uint64) []platform.AccessNode {
	nodes := make([]platform.AccessNode, len(ids))
	for i, id := range ids {
		parent := -1
		if i > 0 {
			parent = rng.IntN(i+1) - 1
		}
		nodes[i] = platform.AccessNode{ID: id, Parent: parent, Role: platform.RoleButton, Label: fmt.Sprint("n", id),
			Bounds: platform.RectF{X: float64(i), W: 10, H: 10}}
	}
	return nodes
}

func delta(prev, cur []platform.AccessNode, base uint64) *platform.AccessDelta {
	d := &platform.AccessDelta{Base: base}
	if len(prev) != len(cur) {
		d.Structure = true
		return d
	}
	for i := range cur {
		if prev[i].ID != cur[i].ID || prev[i].Parent != cur[i].Parent {
			return &platform.AccessDelta{Base: base, Structure: true}
		}
		if prev[i] != cur[i] {
			d.Changed = append(d.Changed, i)
		}
	}
	return d
}

// Applying the deltas of trees leaves the elements as applying the whole
// trees does, whatever changes from one to the next.
func TestUIATreeDeltaMatchesFull(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	full, part := quietTree(), quietTree()
	ids := []uint64{1, 2, 3, 4, 5, 6, 7, 8}
	nodes := randomNodes(rng, ids)
	var prev []platform.AccessNode
	applied := 0
	for step := range 300 {
		switch rng.IntN(5) {
		case 0: // structure: a node comes or goes
			if rng.IntN(2) == 0 || len(ids) < 3 {
				ids = append(ids, uint64(100+step))
			} else {
				i := rng.IntN(len(ids))
				ids = slices.Delete(ids, i, i+1)
			}
			nodes = randomNodes(rng, ids)
		default: // some nodes change
			nodes = slices.Clone(nodes)
			for range rng.IntN(3) {
				i := rng.IntN(len(nodes))
				nodes[i].Label = fmt.Sprint("label ", step)
				nodes[i].Bounds.Y = float64(rng.IntN(50))
				nodes[i].States ^= platform.AccessChecked
			}
		}
		focus := ids[rng.IntN(len(ids))]
		seq := uint64(step + 1)
		full.update(&platform.AccessTree{Nodes: slices.Clone(nodes), Focus: focus, Seq: seq})
		tree := &platform.AccessTree{Nodes: slices.Clone(nodes), Focus: focus, Seq: seq}
		if prev != nil && step%17 != 5 { // now and then a tree without a delta
			tree.Delta = delta(prev, nodes, seq-1)
		}
		if part.applyDelta(tree) {
			applied++
		} else {
			part.update(tree)
		}
		prev = slices.Clone(nodes)
		if a, b := treeShape(full), treeShape(part); a != b {
			t.Fatalf("step %d: deltas gave\n%s\nwhole trees\n%s", step, b, a)
		}
		for id, e := range full.nodes {
			if part.nodes[id] == nil || part.nodes[id].n != e.n {
				t.Fatalf("step %d: node %d differs", step, id)
			}
		}
	}
	if applied < 100 {
		t.Fatalf("%d deltas applied", applied)
	}
}

// A delta from a tree the surface did not apply is applied as a whole tree.
func TestUIATreeDeltaFromOtherTree(t *testing.T) {
	tr := quietTree()
	nodes := []platform.AccessNode{{ID: 1, Parent: -1, Label: "a"}, {ID: 2, Parent: 0, Label: "b"}}
	tr.update(&platform.AccessTree{Nodes: nodes, Seq: 1})
	next := slices.Clone(nodes)
	next[1].Label = "c"
	// Based on tree 5, which never came: the change of node 0 it does not
	// list must show all the same.
	next[0].Label = "z"
	tr.update(&platform.AccessTree{Nodes: next, Seq: 6, Delta: &platform.AccessDelta{Base: 5, Changed: []int{1}}})
	if tr.order[0].n.Label != "z" || tr.order[1].n.Label != "c" || tr.seq != 6 {
		t.Fatalf("applied %q %q seq %d", tr.order[0].n.Label, tr.order[1].n.Label, tr.seq)
	}
}
