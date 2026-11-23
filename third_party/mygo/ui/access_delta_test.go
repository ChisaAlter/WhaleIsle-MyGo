package ui

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A host applying the deltas of the trees it gets has the tree a full
// description of every frame gives, on views changing at random.
func TestAccessDeltaMatchesFullTree(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	n, checked, moved := 6, 3, 0
	labels := map[int]string{}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(float32(moved)).Children(func() {
			for i := range n {
				coreRow(c).Key(i).Children(func() {
					l := labels[i]
					if l == "" {
						l = fmt.Sprint("Button ", i)
					}
					coreButton(c, l)
					v := i == checked
					coreCheckbox(c, &v, fmt.Sprint("Check ", i))
				})
			}
		})
	}, 400, 600)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	var mirror []platform.AccessNode
	var seq uint64
	var focus uint64
	deltas, skipped := 0, 0
	for step := range 400 {
		switch rng.IntN(6) {
		case 0:
			n = 1 + rng.IntN(10)
		case 1:
			labels[rng.IntN(10)] = fmt.Sprint("Label ", step)
		case 2:
			checked = rng.IntN(10)
		case 3:
			moved = rng.IntN(4)
		case 4:
			tt.Key(0, KeyTab)
		}
		tt.Frame()
		got := tt.h.access
		switch {
		case got.Seq == seq:
			skipped++
		case got.Delta != nil && !got.Delta.Structure && got.Delta.Base == seq:
			deltas++
			for _, i := range got.Delta.Changed {
				mirror[i] = got.Nodes[i]
			}
			seq, focus = got.Seq, got.Focus
		default:
			mirror, seq, focus = slices.Clone(got.Nodes), got.Seq, got.Focus
		}
		full := tt.rt.accessTree()
		if !slices.Equal(mirror, full.Nodes) || focus != full.Focus {
			t.Fatalf("step %d: the host has\n%+v\nfocus %d, the frame\n%+v\nfocus %d", step, mirror, focus, full.Nodes, full.Focus)
		}
	}
	if deltas < 50 || skipped < 20 {
		t.Fatalf("%d deltas, %d trees not sent again", deltas, skipped)
	}
}
