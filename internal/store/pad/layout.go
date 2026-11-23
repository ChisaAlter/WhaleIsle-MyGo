package pad

import (
	"encoding/json"
	"errors"
	"maps"
)

type PointXY struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}
type Tree struct {
	Kind     string    `json:"kind"`
	ID       string    `json:"id,omitempty"`
	Dir      string    `json:"dir,omitempty"`
	Children []*Tree   `json:"children,omitempty"`
	Weights  []float32 `json:"weights,omitempty"`
}
type Layout struct {
	Expanded bool                                  `json:"expanded"`
	Ball     *PointXY                              `json:"ball,omitempty"`
	Dock     *PointXY                              `json:"dock,omitempty"`
	Width    float32                               `json:"width"`
	Height   float32                               `json:"height,omitempty"`
	Tree     *Tree                                 `json:"tree,omitempty"`
	Heights  map[string]float32                    `json:"heights"`
	States   map[string]map[string]json.RawMessage `json:"states"`
}

func defaultLayout() Layout {
	return Layout{Width: 352, Heights: map[string]float32{}, States: map[string]map[string]json.RawMessage{}}
}
func normalizeTree(root *Tree, panels []Panel) *Tree {
	known := map[string]bool{}
	seen := map[string]bool{}
	for _, p := range panels {
		known[p.ID] = true
	}
	var walk func(*Tree, int) *Tree
	walk = func(n *Tree, depth int) *Tree {
		if n == nil || depth > 8 {
			return nil
		}
		if n.Kind == "leaf" {
			if !known[n.ID] || seen[n.ID] {
				return nil
			}
			seen[n.ID] = true
			return &Tree{Kind: "leaf", ID: n.ID}
		}
		if n.Kind != "split" || n.Dir != "row" && n.Dir != "col" {
			return nil
		}
		out := &Tree{Kind: "split", Dir: n.Dir}
		for i, c := range n.Children {
			v := walk(c, depth+1)
			if v == nil {
				continue
			}
			out.Children = append(out.Children, v)
			w := float32(1)
			if i < len(n.Weights) {
				w = max(.05, min(20, n.Weights[i]))
			}
			out.Weights = append(out.Weights, w)
		}
		if len(out.Children) == 0 {
			return nil
		}
		if len(out.Children) == 1 {
			return out.Children[0]
		}
		return out
	}
	out := walk(root, 0)
	for _, p := range panels {
		if !seen[p.ID] {
			leaf := &Tree{Kind: "leaf", ID: p.ID}
			if out == nil {
				out = leaf
			} else if out.Kind == "split" && out.Dir == "col" {
				out.Children = append(out.Children, leaf)
				out.Weights = append(out.Weights, 1)
			} else {
				out = &Tree{Kind: "split", Dir: "col", Children: []*Tree{out, leaf}, Weights: []float32{1, 1}}
			}
		}
	}
	return out
}
func removeLeaf(n *Tree, id string) *Tree {
	if n == nil {
		return nil
	}
	if n.Kind == "leaf" {
		if n.ID == id {
			return nil
		}
		return n
	}
	out := &Tree{Kind: n.Kind, Dir: n.Dir}
	for i, c := range n.Children {
		v := removeLeaf(c, id)
		if v != nil {
			out.Children = append(out.Children, v)
			w := float32(1)
			if i < len(n.Weights) {
				w = n.Weights[i]
			}
			out.Weights = append(out.Weights, w)
		}
	}
	if len(out.Children) == 0 {
		return nil
	}
	if len(out.Children) == 1 {
		return out.Children[0]
	}
	return out
}
func replaceLeaf(n *Tree, id string, replacement *Tree) *Tree {
	if n == nil {
		return nil
	}
	if n.Kind == "leaf" {
		if n.ID == id {
			return replacement
		}
		return n
	}
	for i, c := range n.Children {
		n.Children[i] = replaceLeaf(c, id, replacement)
	}
	return n
}

// cloneLayout copies the containers a writer may change — the tree, the
// point coordinates and the height/state maps — while sharing the raw JSON
// payloads, which are only ever replaced wholesale.
func cloneLayout(l Layout) Layout {
	l.Tree = cloneTree(l.Tree)
	if l.Ball != nil {
		ball := *l.Ball
		l.Ball = &ball
	}
	if l.Dock != nil {
		dock := *l.Dock
		l.Dock = &dock
	}
	l.Heights = maps.Clone(l.Heights)
	states := make(map[string]map[string]json.RawMessage, len(l.States))
	for k, v := range l.States {
		states[k] = maps.Clone(v)
	}
	l.States = states
	return l
}

func cloneTree(n *Tree) *Tree {
	if n == nil {
		return nil
	}
	out := &Tree{Kind: n.Kind, ID: n.ID, Dir: n.Dir, Weights: append([]float32(nil), n.Weights...)}
	for _, c := range n.Children {
		out.Children = append(out.Children, cloneTree(c))
	}
	return out
}

// DropCard preserves the original nested directional layout, rather than
// translating it to a fixed-column native widget schema.
func (s *Store) DropCard(source, target, zone string) {
	s.mu.Lock()
	tree := cloneTree(s.view.Layout.Tree)
	panels := s.view.Panels
	s.mu.Unlock()
	if source == target {
		return
	}
	if zone == "center" {
		var swap func(*Tree)
		swap = func(n *Tree) {
			if n == nil {
				return
			}
			if n.Kind == "leaf" {
				if n.ID == source {
					n.ID = target
				} else if n.ID == target {
					n.ID = source
				}
			}
			for _, c := range n.Children {
				swap(c)
			}
		}
		swap(tree)
	} else {
		tree = removeLeaf(tree, source)
		leaf := &Tree{Kind: "leaf", ID: source}
		if target == "" {
			tree = &Tree{Kind: "split", Dir: "col", Children: []*Tree{tree, leaf}, Weights: []float32{1, 1}}
		} else {
			dir := "col"
			if zone == "left" || zone == "right" {
				dir = "row"
			}
			old := &Tree{Kind: "leaf", ID: target}
			children := []*Tree{old, leaf}
			if zone == "left" || zone == "top" {
				children = []*Tree{leaf, old}
			}
			tree = replaceLeaf(tree, target, &Tree{Kind: "split", Dir: dir, Children: children, Weights: []float32{1, 1}})
		}
	}
	s.UpdateLayout(func(l *Layout) { l.Tree = normalizeTree(tree, panels) })
}
func (s *Store) UpdateLayout(change func(*Layout)) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	l := cloneLayout(s.view.Layout)
	change(&l)
	if l.Width < 120 {
		l.Width = 352
	}
	l.Tree = normalizeTree(l.Tree, s.view.Panels)
	s.view.Layout = l
	s.restartWatchLocked()
	s.publishLocked()
	s.mu.Unlock()
	s.persistNow()
	s.fire()
}

// DragLayout keeps pointer motion off disk; the release commits one snapshot.
func (s *Store) DragLayout(change func(*Layout), commit bool) {
	s.mu.Lock()
	l := cloneLayout(s.view.Layout)
	change(&l)
	s.view.Layout = l
	s.publishLocked()
	s.mu.Unlock()
	if commit {
		s.persist()
	}
	s.fire()
}
func (s *Store) persist() {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.persistNow()
}

// persistNow runs with persistMu held: it reads the layout under a brief
// state lock and marshals/writes outside it.
func (s *Store) persistNow() {
	s.mu.Lock()
	value, path := cloneLayout(s.view.Layout), s.path
	s.mu.Unlock()
	if path == "" {
		return
	}
	if err := s.writeFn(path, value); err != nil {
		s.SetError(err)
	}
}
func (s *Store) PanelState(id string) map[string]json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view.Layout.States[id]
}
func (s *Store) SetPanelState(id, key string, value json.RawMessage) error {
	return s.SetPanelStateForGeneration(0, id, key, value)
}
func (s *Store) SetPanelStateForGeneration(generation uint64, id, key string, value json.RawMessage) error {
	if !json.Valid(value) {
		return errors.New("Pad: invalid state JSON")
	}
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	if generation != 0 && (s.client == nil || s.view.Generation != generation) {
		s.mu.Unlock()
		return errors.New("Pad: Host changed")
	}
	states := maps.Clone(s.view.Layout.States)
	entry := maps.Clone(states[id])
	if entry == nil {
		entry = map[string]json.RawMessage{}
	}
	entry[key] = append(json.RawMessage(nil), value...)
	states[id] = entry
	s.mu.Unlock()
	raw, err := json.Marshal(entry)
	if err != nil || len(raw) > 64<<10 {
		return errors.New("Pad: panel state exceeds 64 KiB")
	}
	s.mu.Lock()
	s.view.Layout.States = states
	s.publishLocked()
	s.mu.Unlock()
	s.persistNow()
	return nil
}
