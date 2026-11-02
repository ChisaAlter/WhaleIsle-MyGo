package ui

import "testing"

// The interface assertion recorded a buildable RED before the query landed;
// retain it as a direct contract check for the private native selection API.
type textSelectionIntersectionQuery interface {
	HasTextSelection() bool
}

func requireTextSelectionIntersection(t *testing.T, n *node) bool {
	t.Helper()
	query, ok := any(n).(textSelectionIntersectionQuery)
	if !ok {
		t.Fatal("missing private node.HasTextSelection query: a card cannot detect an intersecting noncollapsed selection")
	}
	return query.HasTextSelection()
}

func TestTextSelectionIntersectionQuery(t *testing.T) {
	for _, tc := range []struct {
		name          string
		anchor, caret textPoint
		want          bool
	}{
		{"collapsed inside", textPoint{20, 1}, textPoint{20, 1}, false},
		{"same descendant paragraph", textPoint{20, 1}, textPoint{20, 3}, true},
		{"reverse descendant range", textPoint{20, 3}, textPoint{20, 1}, true},
		{"unrelated sibling", textPoint{10, 0}, textPoint{10, 4}, false},
		{"both endpoints outside spanning card", textPoint{10, 1}, textPoint{30, 2}, true},
		{"reverse spanning card", textPoint{30, 2}, textPoint{10, 1}, true},
		{"starts in card ends outside", textPoint{20, 1}, textPoint{30, 2}, true},
		{"starts outside ends in card", textPoint{10, 1}, textPoint{20, 2}, true},
		{"removed endpoint", textPoint{999, 0}, textPoint{20, 2}, false},
		{"different scope endpoint", textPoint{40, 0}, textPoint{20, 2}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &engine{states: map[uint64]*state{}, windowFocused: false}
			c := &context{rt: rt}
			add := func(id, parent, scope uint64) *state {
				s := &state{id: id, parent: parent, textScope: scope}
				rt.states[id] = s
				return s
			}
			add(1, 0, 0)
			card := add(2, 1, 0)
			// Paragraph order crosses the card even when neither endpoint is in
			// it. A bounds-only or endpoint-only check would miss this case.
			before := add(10, 1, 1)
			inside := add(20, 2, 1)
			after := add(30, 1, 1)
			otherScope := add(40, 1, 40)
			rt.texts = []*state{before, inside, after, otherScope}
			rt.selection = textSelection{scope: 1, anchor: tc.anchor, caret: tc.caret}
			n := &node{id: 2, c: c, st: card}
			flags, selection := n.flags, rt.selection
			if got := requireTextSelectionIntersection(t, n); got != tc.want {
				t.Fatalf("intersection=%v want=%v", got, tc.want)
			}
			if n.flags != flags || rt.selection != selection || rt.consumed || rt.focused != 0 {
				t.Fatal("selection query mutated flags, range, focus or consumed input")
			}
		})
	}
}

func TestTextSelectionIntersectionStandaloneAndAbsentOwner(t *testing.T) {
	rt := &engine{states: map[uint64]*state{}, focused: 3}
	c := &context{rt: rt}
	card := &state{id: 1}
	ed := newEditor()
	ed.anchor, ed.caret = 1, 4
	paragraph := &state{id: 3, parent: 1, flags: flagSelectable, editor: ed}
	rt.states[1], rt.states[3] = card, paragraph
	rt.texts = []*state{paragraph}
	n := &node{id: 1, c: c, st: card}
	if !requireTextSelectionIntersection(t, n) {
		t.Fatal("focused standalone selected text was not included")
	}
	ed.caret = ed.anchor
	if requireTextSelectionIntersection(t, n) {
		t.Fatal("standalone collapsed caret counted as selection")
	}
	ed.caret = 4
	rt.focused = 99
	if requireTextSelectionIntersection(t, n) {
		t.Fatal("a retained editor outside the active selection was included")
	}
	if requireTextSelectionIntersection(t, nil) {
		t.Fatal("nil owner has a selection")
	}
	rt.closed = true
	if requireTextSelectionIntersection(t, n) {
		t.Fatal("closed engine retained selection authority")
	}
}

// No native window, pointer event, focus action, shaping or clipboard is used.
// This checks that Selectable on a clickable container wins over its own control
// barrier while descendants of an ordinary button remain controls' text.
func TestTextSelectionIntersectionSelectableButtonScopePreparation(t *testing.T) {
	for _, selectable := range []bool{false, true} {
		rt := &engine{states: map[uint64]*state{}}
		c := &context{rt: rt}
		card := &node{id: 1, kind: kindBox, c: c, flags: flagClickable | flagFocusable}
		if selectable {
			card.Selectable()
		}
		paragraph := &node{id: 2, kind: kindText, c: c, parent: card, st: &state{id: 2}, text: "Selectable body"}
		paragraph.prepareSelectable()
		want := uint64(0)
		if selectable {
			want = 1
		}
		if paragraph.st.textScope != want {
			t.Fatalf("Selectable=%v scope=%d want=%d", selectable, paragraph.st.textScope, want)
		}
	}
}
