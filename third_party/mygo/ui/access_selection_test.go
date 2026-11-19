package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A radio button, a tab and a tree item are given the selection-item pattern
// unconditionally (surface_access.go:267-274), and that pattern is how an
// accessibility client both chooses one — Select presses it — and asks which
// one is chosen, through IsSelected, which reads AccessChecked. So a control
// built from those roles must report which of its parts is the chosen one, or
// its selection is invisible and its choice may not move at all.
//
// The rows are the primitives that carry those roles. The outline is absent on
// purpose: its rows are a List's, built through the table path that already
// marks the chosen row (list.go:558-575, which is what outline.go:249 reads),
// so there is no separate widget to drive.
func TestEverySelectionControlReportsWhichPartIsChosen(t *testing.T) {
	var (
		radioChosen int
		tabChosen   int
		treeChosen  int
	)
	builders := []struct {
		name  string
		reset func()
		build func(c *context)
		chose func() int
	}{
		{"Radio", func() { radioChosen = 0 }, func(c *context) {
			for i := 0; i < 3; i++ {
				part := coreRadioBase[int](c, &radioChosen, i)
				part.Children(func() { coreText(c, "option") })
			}
		}, func() int { return radioChosen }},
		{"Tabs", func() { tabChosen = 0 }, func(c *context) {
			p := coreTabsBase(c, &tabChosen, 3)
			for i := 0; i < 3; i++ {
				tab := p.Tab(i)
				tab.Children(func() { coreText(c, "tab") })
			}
		}, func() int { return tabChosen }},
		{"Tree", func() { treeChosen = -1 }, func(c *context) {
			coreTree(c, func() {
				for i := 0; i < 3; i++ {
					i := i
					item := coreTreeItem(c, "node", nil, nil)
					item.afterInput(func() {
						if item.Clicked() {
							treeChosen = i
						}
					})
				}
			})
		}, func() int { return treeChosen }},
	}
	for _, tc := range builders {
		t.Run(tc.name, func(t *testing.T) {
			tc.reset()
			tt := coreNewTester(tc.build, 420, 420)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			items := selectionItems(tt)
			if len(items) != 3 {
				t.Fatalf("%s offers %d selection items, want 3", tc.name, len(items))
			}
			if got := chosenCount(items); got > 1 {
				t.Fatalf("%s reports %d parts chosen at rest", tc.name, got)
			}

			// The second part, pressed the way the selection pattern presses it.
			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: items[1].ID, Action: platform.AccessPress})
			tt.Frame()
			items = selectionItems(tt)
			if got := chosenCount(items); got != 1 {
				t.Fatalf("%s reports %d parts chosen after a press, want 1", tc.name, got)
			}
			if items[1].States&platform.AccessChecked == 0 {
				t.Fatalf("%s: the pressed part does not report itself chosen", tc.name)
			}
			if tc.chose() != 1 {
				t.Fatalf("%s: the press chose %d", tc.name, tc.chose())
			}
		})
	}
}

// selectionItems are the parts the surface gives the selection-item pattern to
// by role alone, in tree order.
func selectionItems(tt *Tester) []platform.AccessNode {
	var items []platform.AccessNode
	for _, n := range tt.h.access.Nodes {
		switch n.Role {
		case platform.RoleRadio, platform.RoleTab, platform.RoleTreeItem:
			items = append(items, n)
		}
	}
	return items
}

func chosenCount(items []platform.AccessNode) int {
	n := 0
	for _, item := range items {
		if item.States&platform.AccessChecked != 0 {
			n++
		}
	}
	return n
}
