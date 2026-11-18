package ui

// ListRow gives a row its shared context, index, selection and list focus.
type ListRow struct {
	Context  *Context
	Index    int
	owner    Element
	selected bool
}

func (r ListRow) Selected() bool    { return r.owner.Valid() && r.selected }
func (r ListRow) ListFocused() bool { return r.owner.FocusWithin() }

type listSpec struct {
	state *ListState
	count int
	built bool
}

// List creates a virtual list. The original row callback remains supported.
// Omit it to configure ItemKey and Selection, then call Rows.
func List(c *Context, s *ListState, n int, row ...func(int)) Element {
	raw := c.build()
	if raw == nil {
		return Element{}
	}
	if len(row) > 1 {
		panic("ui: List accepts one row builder")
	}
	if len(row) == 1 {
		return wrapElement(coreList(raw, s, n, row[0]))
	}
	e := coreScroll(raw)
	e.widget, e.role = "List", RoleList
	if s == nil {
		s = coreLocal(e, "list", func() ListState { return ListState{} })
	}
	*valueBinding[listSpec](e) = listSpec{state: s, count: n}
	return wrapElement(e)
}
func (e Element) ItemKey(key func(int) any) Element {
	if n := e.node(); n != nil {
		if s, ok := n.st.valueBinding.(*listSpec); ok {
			s.state.Key = key
		}
	}
	return e
}
func (e Element) Selection(selection Selector) Element {
	if n := e.node(); n != nil {
		if s, ok := n.st.valueBinding.(*listSpec); ok {
			s.state.Selection = selection
		}
	}
	return e
}

// SelectionText supplies a virtual list's complete logical text. Paragraphs
// returns only the row's selectable text, in the same order as Rows builds it;
// gutters, buttons, hidden content and nested selection scopes are excluded.
// Return nil for a row with no selectable text, and []string{""} for an empty
// selectable paragraph. The callback reads the current immutable view model and
// must not perform I/O or build UI. Change revision when that model changes.
// Offsets are rune indices, just like Text and RichText selection offsets.
func (e Element) SelectionText(revision uint64, paragraphs func(row int) []string) Element {
	if n := e.node(); n != nil {
		source := ListTextSource{Revision: revision, Paragraphs: paragraphs}
		if spec, ok := n.st.valueBinding.(*listSpec); ok {
			spec.state.TextSource = source
		} else if n.st.list != nil {
			n.st.list.TextSource = source
		}
	}
	return e
}

func (e Element) Rows(row func(ListRow)) Element {
	n := e.node()
	if n == nil {
		return e
	}
	s, ok := n.st.valueBinding.(*listSpec)
	if !ok {
		return e
	}
	if s.built {
		panic("ui: List.Rows called twice in one pass")
	}
	s.built = true
	buildList(n.c, n, n, s.state, s.count, func(i int) {
		if row == nil {
			return
		}
		selected := false
		if n.c.row != nil {
			selected = n.c.row.chosen
		}
		row(ListRow{Context: makeContext(n.c), Index: i, owner: e, selected: selected})
	}, plainList)
	return e
}
