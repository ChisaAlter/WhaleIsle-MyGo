package chat

import (
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func toolTreeLayoutFixture(t *testing.T, width int, dark bool) (*ui.Tester, *InspectorNavigation, *session.Message, *map[string]bool) {
	t.Helper()
	n := &InspectorNavigation{}
	msg := session.ProjectToolMessages(inspectorNestedToolMessages(t, 1))[0]
	expanded := map[string]bool{}
	tt := ui.NewTester(func(c *ui.Context) {
		clear(n.reveal.toolTrees)
		n.reveal.toolTrees = n.reveal.toolTrees[:0]
		colors := theme.Resolve(theme.DefaultID).Active(dark)
		style := DefaultToolCardStyle(chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border), chrome.Color(colors.Field), chrome.Color(colors.Accent))
		ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Padding(20).Children(func() {
			shared := &expanded
			if expanded == nil {
				shared = nil
			}
			ToolMessageView(c, msg, style, ToolImageGalleryOptions{ID: msg.ID, Inspector: n, Expanded: shared})
			ui.Box(c).Height(1).Label("after tool tree")
		})
	}, width, 600)
	tt.SetTime(time.Unix(7300, 0))
	return tt, n, &msg, &expanded
}

func toolTreeBoxes(n *InspectorNavigation) map[string]ui.Rect {
	boxes := map[string]ui.Rect{}
	for _, tree := range n.reveal.toolTrees {
		for _, box := range tree.boxes {
			boxes[box.call.CallID] = box.rect
		}
	}
	return boxes
}

func toolTreeNear(t *testing.T, label string, actual, want float32) {
	t.Helper()
	if math.Abs(float64(actual-want)) > .02 {
		t.Fatalf("%s: got %v, want %v", label, actual, want)
	}
}

func TestToolTreeNestedSpacingAndContinuousConnectors(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			tt, n, _, _ := toolTreeLayoutFixture(t, 520, dark)
			boxes := toolTreeBoxes(n)
			root, child, grand, sibling := boxes["opaque:root"], boxes["unrelated-child"], boxes["opaque/grandchild"], boxes["sibling-0"]
			for label, pair := range map[string][2]float32{
				"root width": {root.W, 480}, "root height": {root.H, 110},
				"child inset": {child.X - root.X, 31}, "grandchild inset": {grand.X - child.X, 31},
				"child width": {child.W, root.W - 31}, "grandchild width": {grand.W, child.W - 31},
				"child top gap": {child.Y - root.Y, 28}, "grandchild top gap": {grand.Y - child.Y, 28},
				"child subtree": {child.H, 54}, "grandchild leaf": {grand.H, 24},
				"sibling gap": {sibling.Y - child.Y - child.H, 4}, "sibling bottom": {root.Y + root.H - sibling.Y - sibling.H, 0},
			} {
				toolTreeNear(t, label, pair[0], pair[1])
			}
			flow, ok := tt.Find("after tool tree")
			if !ok {
				t.Fatal("missing following flow marker")
			}
			toolTreeNear(t, "root escaping bottom margin", flow.Y-root.Y, 112)
			if strings.Contains(strings.Join(tt.Texts(), "\n"), "↳ ") {
				t.Fatal("wire parent identity added an extra tool-tree line")
			}
			// The outer connector spans the full child subtree, sibling gap and
			// sibling. A blank gutter must contain actual border pixels there.
			image := tt.Image()
			x := int(root.X + 22)
			for _, y := range []int{int(child.Y + 8), int(grand.Y + 8), int(child.Y + child.H + 2), int(sibling.Y + 8)} {
				if image.RGBAAt(x, y) == image.RGBAAt(x-2, y) {
					t.Fatalf("continuous ancestor connector missing at %d,%d", x, y)
				}
			}
			toolTreeExport(t, tt, boxes, fmt.Sprintf("collapsed-dark-%v", dark))
		})
	}
}

func TestToolTreeExpandParentPreservesChildrenAndSiblingSpacing(t *testing.T) {
	tt, n, _, expanded := toolTreeLayoutFixture(t, 520, false)
	before := toolTreeBoxes(n)
	root := before["opaque:root"]
	tt.ClickAt(root.X+12, root.Y+12)
	tt.Frame()
	if !(*expanded)["tool:opaque:root|tool:opaque:root"] {
		t.Fatal("root disclosure did not open")
	}
	after := toolTreeBoxes(n)
	delta := after["unrelated-child"].Y - before["unrelated-child"].Y
	if delta <= 0 {
		t.Fatal("expanded root body did not move its children")
	}
	for _, id := range []string{"unrelated-child", "opaque/grandchild", "sibling-0"} {
		toolTreeNear(t, id+" body shift", after[id].Y-before[id].Y, delta)
		toolTreeNear(t, id+" subtree height", after[id].H, before[id].H)
	}
	toolTreeExport(t, tt, after, "expanded-root")
	tt.ClickAt(root.X+12, root.Y+12)
	tt.Frame()
	// The original keeps the inert exiting body in flow for200ms.
	tt.SetTime(time.Unix(7300, 0).Add(250 * time.Millisecond))
	for id, box := range toolTreeBoxes(n) {
		if box != before[id] {
			t.Fatalf("closing parent hid or repositioned child: %s %+v want %+v", id, box, before[id])
		}
	}
}

func TestToolTreeDeepNarrowAndMalformedLinks(t *testing.T) {
	tt, n, msg, _ := toolTreeLayoutFixture(t, 520, false)
	root := &session.ToolCallView{CallID: "level-0", Name: "read", State: "ok"}
	parent := root
	for i := 1; i <= 10; i++ {
		child := &session.ToolCallView{CallID: fmt.Sprintf("level-%d", i), Name: "read", State: "ok"}
		parent.Children = []*session.ToolCallView{child}
		parent = child
	}
	msg.Tool = root
	tt.Frame()
	for _, width := range []int{520, 350, 220, 100} {
		tt.SetSize(width, 600)
		boxes := toolTreeBoxes(n)
		for i := 1; i <= 10; i++ {
			parent, child := boxes[fmt.Sprintf("level-%d", i-1)], boxes[fmt.Sprintf("level-%d", i)]
			toolTreeNear(t, fmt.Sprintf("width%d depth%d indent", width, i), child.X-parent.X, 31)
			if child.W < 0 || child.H <= 0 {
				t.Fatalf("narrow deep descendant lost measured height: width%d depth%d %+v", width, i, child)
			}
			toolTreeNear(t, "nested top gap", child.Y-parent.Y, 28)
		}
		toolTreeExport(t, tt, boxes, fmt.Sprintf("deep-width-%d", width))
	}
	// Duplicate pointers, nils and a back edge must not mount a phantom empty
	// subcalls group or loop. The valid first occurrence keeps one card.
	leaf := &session.ToolCallView{CallID: "leaf", Name: "read", State: "ok"}
	root.Children = []*session.ToolCallView{nil, leaf, leaf, root}
	leaf.Children = []*session.ToolCallView{root, nil}
	tt.SetSize(520, 600)
	boxes := toolTreeBoxes(n)
	if len(n.reveal.toolTrees[0].boxes) != 2 || len(boxes) != 2 || boxes["leaf"].H != 24 || boxes["level-0"].H != 52 {
		t.Fatalf("malformed pointer graph mounted extra content: %+v", boxes)
	}
	root.Children = []*session.ToolCallView{nil, root}
	tt.Frame()
	boxes = toolTreeBoxes(n)
	if len(n.reveal.toolTrees[0].boxes) != 1 || boxes["level-0"].H != 24 {
		t.Fatalf("leaf gained a phantom child gutter: %+v", boxes)
	}
}

func TestToolTreeSiblingInsertKeepsOwnDisclosureState(t *testing.T) {
	tt, n, msg, expanded := toolTreeLayoutFixture(t, 520, false)
	// Exercise the widget's local disclosure state, independent of the shared
	// map used by production Chat and its full owner key.
	*expanded = nil
	tt.Frame()
	child := toolTreeBoxes(n)["unrelated-child"]
	tt.ClickAt(child.X+12, child.Y+12)
	tt.Frame()
	before := toolTreeBoxes(n)
	if before["unrelated-child"].H <= 54 {
		t.Fatal("child body did not open")
	}
	added := &session.ToolCallView{CallID: "inserted", Name: "read", State: "ok"}
	msg.Tool.Children = append([]*session.ToolCallView{added}, msg.Tool.Children...)
	tt.Frame()
	after := toolTreeBoxes(n)
	toolTreeNear(t, "existing open child after insertion", after["unrelated-child"].H, before["unrelated-child"].H)
	if after["inserted"].H != 24 || after["sibling-0"].H != 24 {
		t.Fatal("new sibling inherited another call body")
	}
}

func TestToolTreeDeviceScaleReservesUsedHairlineWidth(t *testing.T) {
	tt, n, _, _ := toolTreeLayoutFixture(t, 520, false)
	for _, tc := range []struct{ scale, border float32 }{{1, 1}, {1.25, .8}, {1.5, 2.0 / 3}, {2, .5}, {3, 1.0 / 3}, {4, .5}} {
		tt.SetScale(tc.scale)
		boxes := toolTreeBoxes(n)
		root, child, grand := boxes["opaque:root"], boxes["unrelated-child"], boxes["opaque/grandchild"]
		toolTreeNear(t, "used border child inset", child.X-root.X, 30+tc.border)
		toolTreeNear(t, "used border grandchild inset", grand.X-child.X, 30+tc.border)
		toolTreeNear(t, "child right edge", child.X+child.W, root.X+root.W)
		toolTreeNear(t, "grandchild right edge", grand.X+grand.W, root.X+root.W)
		toolTreeExport(t, tt, boxes, fmt.Sprintf("device-scale-%g", tc.scale))
	}
}

func toolTreeExport(t *testing.T, tt *ui.Tester, boxes map[string]ui.Rect, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_TREE_LAYOUT_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tool-tree-pass257-native-software-"+name)
	f, err := os.Create(path + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(map[string]any{"scope": "MyGo software renderer; no OS backend/font/GPU or live Electron comparison", "boxes": boxes, "texts": tt.Texts()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".json", raw, 0644); err != nil {
		t.Fatal(err)
	}
}
