package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestAccessibilityDisclosureKeepsNestedActions(t *testing.T) {
	opened, toggled := 0, 0
	tt := coreNewTester(func(c *context) {
		row := coreButtonBase(c).Role(RoleDisclosure).Expanded(false).Height(24)
		if row.Clicked() {
			toggled++
		}
		row.Children(func() {
			coreText(c, "Read")
			path := coreButtonBase(c).Label("src/read.go").Height(24)
			if path.Clicked() {
				opened++
			}
			path.Children(func() { coreText(c, "src/read.go") })
		})
	}, 400, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	row := accessNode(t, tt.h.access, platform.RoleDisclosure, "Read src/read.go")
	path := accessNode(t, tt.h.access, platform.RoleButton, "src/read.go")
	if tt.h.access.Nodes[path.Parent].ID != row.ID || path.Actions&platform.ActionPress == 0 {
		t.Fatalf("missing independent action: row%+v path%+v", row, path)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: path.ID, Action: platform.AccessPress})
	if opened != 1 || toggled != 0 {
		t.Fatalf("path action opened%d toggled%d", opened, toggled)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: row.ID, Action: platform.AccessPress})
	if opened != 1 || toggled != 1 {
		t.Fatalf("row action opened%d toggled%d", opened, toggled)
	}
	// Ordinary buttons continue to expose their label as a leaf.
	for _, n := range tt.h.access.Nodes {
		if n.Role == platform.RoleText && n.Label == "src/read.go" {
			t.Fatal("nested button exposed its decorative text twice")
		}
	}
}
