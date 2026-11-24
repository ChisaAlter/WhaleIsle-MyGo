package uipurity

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBoundValueGuard scans the live tree for bound-value misuse: reads or
// compares of a deferred input's bound value before an input query flushes
// pending writes, and stack locals bound into dead variables. The baseline
// is zero — every fix is already in.
func TestBoundValueGuard(t *testing.T) {
	var all []BoundViolation
	for _, dir := range []string{
		filepath.Join("..", "..", "internal", "ui"),
		filepath.Join("..", "..", "internal", "desktop"),
		filepath.Join("..", "..", "cmd", "whaleisle"),
		filepath.Join("..", "..", "internal", "app"),
		filepath.Join("..", "..", "tools"),
	} {
		vs, err := ScanBoundValue(dir)
		if err != nil {
			t.Fatal(err)
		}
		for i := range vs {
			vs[i].File = strings.TrimPrefix(vs[i].File, "../../")
		}
		all = append(all, vs...)
	}
	if len(all) > 0 {
		var b strings.Builder
		for _, v := range all {
			b.WriteString(v.String())
			b.WriteByte('\n')
		}
		t.Fatalf("bound-value violations — deferred inputs write their bound pointer inside applyInputs(), so reads before Changed()/Submitted()/Clicked()/Shortcut() see last frame's value:\n%s", b.String())
	}
}

// scanSrc scans one in-memory source file and returns violation messages.
func scanSrc(t *testing.T, src string) []string {
	t.Helper()
	vs, err := scanBoundSource("case.go", src)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, v := range vs {
		out = append(out, v.String())
	}
	return out
}

func TestBoundValueStackLocalCompare(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	selected := s.Excluded
	ui.Checkbox(c, &selected, "").Label("path")
	if selected == s.Excluded {
		return
	}
}`)
	if len(got) != 1 || !strings.Contains(got[0], `"selected" read before`) {
		t.Fatalf("expected one stale-read violation, got %v", got)
	}
}

func TestBoundValueStackLocalAssign(t *testing.T) {
	got := scanSrc(t, `package p
func dock(c *ui.Context, n *Node) {
	size := float32(100)
	ui.Split(c, &size, func() {}, func() {})
	n.Ratio = size / 99
}`)
	if len(got) != 1 || !strings.Contains(got[0], `"size" read before`) {
		t.Fatalf("expected one stale-read violation, got %v", got)
	}
}

func TestBoundValueStackLocalDropped(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context) {
	enabled := true
	ui.Checkbox(c, &enabled, "on")
}`)
	if len(got) != 1 || !strings.Contains(got[0], `never read`) {
		t.Fatalf("expected one dropped-binding violation, got %v", got)
	}
}

func TestBoundValueSnapshotCompare(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	before := s.Title
	ui.TextInput(c, &s.Title)
	if s.Title != before {
		s.Plan = nil
	}
}`)
	if len(got) != 1 || !strings.Contains(got[0], `compared against snapshot "before"`) {
		t.Fatalf("expected one snapshot-compare violation, got %v", got)
	}
}

func TestBoundValueSnapshotCompareLocal(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	draft := s.Text
	before := draft
	ui.TextArea(c, &draft)
	if before == draft {
		return
	}
}`)
	if len(got) == 0 {
		t.Fatal("expected at least one violation")
	}
}

func TestBoundValueLegalChanged(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	draft := s.Text
	if ui.TextInput(c, &draft).Changed() {
		s.Save(draft)
	}
	selected := 0
	trigger := ui.Tabs(c, &selected, "a", "b")
	if trigger.Changed() && selected >= 0 {
		s.Keep(selected)
	}
}`)
	if len(got) != 0 {
		t.Fatalf("Changed()-guarded reads must be exempt, got %v", got)
	}
}

func TestBoundValueLegalOnChange(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	draft := s.Text
	ui.TextArea(c, &draft).OnChange(func() { s.Edit(draft) })
	path := s.Path
	input := ui.TextInput(c, &path)
	save := func() { s.Browse(path) }
	input.OnShortcut(0, ui.KeyEnter, save)
}`)
	if len(got) != 0 {
		t.Fatalf("deferred-callback reads must be exempt, got %v", got)
	}
}

func TestBoundValueLegalReadOnlyAndLocal(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	text := s.Text
	ui.TextArea(c, &text).ReadOnly(true)
	ptr := ui.Local(c, func() string { return s.Text })
	ui.TextInput(c, ptr)
	if ui.Button(c, "go").Clicked() {
		s.Save(*ptr)
	}
}`)
	if len(got) != 0 {
		t.Fatalf("ReadOnly bindings and ui.Local pointers must be exempt, got %v", got)
	}
}

func TestBoundValueLegalConfigAndReturn(t *testing.T) {
	got := scanSrc(t, `package p
func picker(c *ui.Context, s *State) (int, bool) {
	selected := s.Index
	trigger := ui.Tabs(c, &selected, s.Labels...)
	ui.Button(c, "go").Disabled(selected < 0)
	return selected, trigger.Changed() && selected >= 0
}`)
	if len(got) != 0 {
		t.Fatalf("config reads and flush-bearing return operands must be exempt, got %v", got)
	}
}

func TestBoundValueLegalDetector(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, m *Model) {
	checked := true
	if ui.Checkbox(c, &checked, "remove").Changed() {
		m.Remove()
	}
}`)
	if len(got) != 0 {
		t.Fatalf("a queried-but-unread binding is a change detector, got %v", got)
	}
}

func TestBoundValueLegalChildrenLinear(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, s *State) {
	v := s.V
	ui.Column(c).Children(func() {
		ui.TextInput(c, &v)
		if v != s.V {
			s.Dirty = true
		}
	})
}`)
	if len(got) != 1 {
		t.Fatalf("expected one violation inside a linear Children closure, got %v", got)
	}
}

func TestBoundValueInterproceduralHelpersAndBarePointers(t *testing.T) {
	t.Run("reverted slider helper trips guard while fixed helper passes", func(t *testing.T) {
		bad := scanSrc(t, `package p
func rowHelper(c *ui.Context, fn func()) { fn() }
func mySlider(c *ui.Context, value *float64) {
	rowHelper(c, func() {
		ui.StepSlider(c, value, 1, 5, 1)
	})
}
func view(c *ui.Context, draft *State) {
	before := draft.Count
	mySlider(c, &draft.Count)
	if draft.Count != before {
		draft.Preset = "custom"
	}
}`)
		if len(bad) == 0 || !strings.Contains(bad[0], `"draft.Count" compared against snapshot "before"`) {
			t.Fatalf("expected reverted slider helper to flag, got %v", bad)
		}

		good := scanSrc(t, `package p
func rowHelper(c *ui.Context, fn func()) { fn() }
func mySlider(c *ui.Context, value *float64) bool {
	changed := false
	rowHelper(c, func() {
		changed = ui.StepSlider(c, value, 1, 5, 1).Changed()
	})
	return changed
}
func view(c *ui.Context, draft *State) {
	if mySlider(c, &draft.Count) {
		draft.Preset = "custom"
	}
}`)
		if len(good) != 0 {
			t.Fatalf("flushing helper should pass, got %v", good)
		}
	})

	t.Run("snapshot check helper flags pre-flush compare", func(t *testing.T) {
		got := scanSrc(t, `package p
type WB struct { S *State }
func (w *WB) invalidateChangedAck(before bool) {
	if w.S != nil && before != w.S.LargeAcknowledged {
		w.S.Plan = nil
	}
}
func view(c *ui.Context, w *WB) {
	before := w.S.LargeAcknowledged
	ui.Checkbox(c, &w.S.LargeAcknowledged, "keep")
	w.invalidateChangedAck(before)
}`)
		if len(got) != 1 || !strings.Contains(got[0], `checked via helper snapshot "before"`) {
			t.Fatalf("expected helper snapshot violation, got %v", got)
		}
	})

	t.Run("bare pointer deref flags before flush and passes after Changed", func(t *testing.T) {
		bad := scanSrc(t, `package p
func view(c *ui.Context, m *Model, snap Snap) {
	ui.TextArea(c, m.Draft)
	if m.Store != nil && snap.SelectedID != "" {
		m.Store.SetDraft(snap.SelectedID, *m.Draft)
	}
	send := ui.Button(c, "Send")
	if strings.TrimSpace(*m.Draft) == "" {
		send = send.Disabled(true)
	}
}`)
		if len(bad) == 0 || !strings.Contains(bad[0], `"m.Draft" read before`) {
			t.Fatalf("expected bare-pointer violation, got %v", bad)
		}

		good := scanSrc(t, `package p
func view(c *ui.Context, m *Model, snap Snap) {
	if ui.TextArea(c, m.Draft).Changed() && m.Store != nil && snap.SelectedID != "" {
		m.Store.SetDraft(snap.SelectedID, *m.Draft)
	}
	ui.Text(c, *m.Draft)
	send := ui.Button(c, "Send")
	if strings.TrimSpace(*m.Draft) == "" {
		send = send.Disabled(true)
	}
}`)
		if len(good) != 0 {
			t.Fatalf("Changed()-flushed bare pointer must pass, got %v", good)
		}
	})
}

func TestBoundValueHardenedBypasses(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "B01 parenthesized operand",
			src: `package p
func view(c *ui.Context, s *State) {
	before := s.Title
	ui.TextInput(c, &s.Title)
	if (s.Title) != before { s.Dirty = true }
}`,
		},
		{
			name: "B03 method value",
			src: `package p
func view(c *ui.Context) {
	v := true
	cb := ui.Checkbox
	cb(c, &v, "on")
}`,
		},
		{
			name: "B04 pointer alias",
			src: `package p
func view(c *ui.Context, s *State) {
	v := s.Name
	p := &v
	ui.TextInput(c, p)
	if v != s.Name { s.Save(v) }
}`,
		},
		{
			name: "B05 bare pointer snapshot compare",
			src: `package p
func view(c *ui.Context, m *Model) {
	before := *m.Draft
	ui.TextArea(c, m.Draft)
	if *m.Draft != before { m.Dirty = true }
}`,
		},
		{
			name: "B06 indexed binding",
			src: `package p
func view(c *ui.Context, s *State, i int) {
	before := s.Items[i]
	ui.TextInput(c, &s.Items[i])
	if s.Items[i] != before { s.Dirty = true }
}`,
		},
		{
			name: "B07 stack struct field",
			src: `package p
func view(c *ui.Context) {
	cfg := Config{On: true}
	ui.Checkbox(c, &cfg.On, "on")
	if cfg.On { println("x") }
}`,
		},
		{
			name: "B08-B17 added controls",
			src: `package p
func view(c *ui.Context, s *State) {
	seg := 0
	ui.Segmented(c, &seg, "a", "b")
	if seg == 1 { s.Pick(seg) }
	q := ""
	coss.Textarea(c, &q, coss.TextareaOptions{})
	if q != "" { s.Save(q) }
	num := 0.0
	coss.NumberField(c, &num, coss.NumberFieldOptions{})
	s.Apply(num)
	comp := ""
	coss.Composer(c, &comp, coss.ComposerOptions{})
	s.Send(comp)
	lo, hi := 0.0, 10.0
	ui.RangeSlider(c, &lo, &hi, 0, 10, 1)
	s.Bound(hi)
	open := false
	ui.FindBar(c, &open, &q, 0, nil)
	if open { s.Open() }
}`,
		},
		{
			name: "B18 conditional branch flush does not stick",
			src: `package p
func view(c *ui.Context, s *State) {
	v := s.Title
	ui.TextInput(c, &v)
	if false { ui.Button(c, "go").Clicked() }
	if v != s.Title { s.Dirty = true }
}`,
		},
		{
			name: "B19 domain Changed does not flush",
			src: `package p
func view(c *ui.Context, doc *Doc, s *State) {
	v := s.Title
	ui.TextInput(c, &v)
	if doc.Changed() { s.Save(v) }
}`,
		},
		{
			name: "B20 pure wrapper snapshot",
			src: `package p
func view(c *ui.Context, s *State) {
	before := strings.TrimSpace(s.T)
	ui.TextInput(c, &s.T)
	if strings.TrimSpace(s.T) != before { s.Dirty = true }
}`,
		},
		{
			name: "B21 float ordering snapshot compare",
			src: `package p
func view(c *ui.Context, s *State) {
	before := s.Zoom
	ui.StepSlider(c, &s.Zoom, 1, 3, 0.1)
	if s.Zoom > before { s.Dirty = true }
}`,
		},
		{
			name: "B22 domain call statement reads stale stack local",
			src: `package p
func view(c *ui.Context, store *Store, id string, edit Edit) {
	draft := edit.Draft
	ui.TextInput(c, &draft)
	store.EditRename(id, draft)
}`,
		},
		{
			name: "B23 bare return size reads stale split value",
			src: `package p
func view(c *ui.Context, cur float32) float32 {
	size := cur
	ui.Split(c, &size, func() {}, func() {})
	return size
}`,
		},
		{
			name: "B25 Clicked inside OnChange does not flush outer binding",
			src: `package p
func view(c *ui.Context, s *State) {
	v := s.Title
	ui.TextInput(c, &v)
	ui.Button(c, "other").OnChange(func() { ui.Button(c, "inner").Clicked() })
	if v != s.Title { s.Dirty = true }
}`,
		},
		{
			name: "B27 domain struct literal call reads stale stack local",
			src: `package p
func view(c *ui.Context, s *State) {
	v := s.Title
	ui.TextInput(c, &v)
	s.Apply(Patch{V: v})
}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scanSrc(t, tc.src)
			if len(got) == 0 {
				t.Fatalf("expected violation for %s, got none", tc.name)
			}
		})
	}
}

func TestBoundValueLegalBarePointerDisplayAndShortCircuit(t *testing.T) {
	got := scanSrc(t, `package p
func view(c *ui.Context, checked *bool, pkg Pkg, m *Model, a Actions) {
	ui.Checkbox(c, checked, "on")
	if *checked {
		ui.Box(c).Background(c.Theme().Accent)
	}
	empty := strings.TrimSpace(*m.Draft) == ""
	ui.Button(c, "Send").Disabled(empty)
	value := pkg.Enabled
	if pkg.Enabled && coss.Switch(c, &value, "", coss.CheckOptions{}).Changed() {
		a.Enable(value)
	}
}`)
	if len(got) != 0 {
		t.Fatalf("immediate-mode UI display reads and short-circuit Changed guards must pass, got %v", got)
	}
}

func TestBoundValueHelperExemptionsAndNegatives(t *testing.T) {
	t.Run("ReadOnly_param_transitive_propagation", func(t *testing.T) {
		helpers := `package p
func inner(c *ui.Context, text *string, ro bool) {
	ed := ui.TextArea(c, text)
	ed.ReadOnly(ro)
}
func outer(c *ui.Context, text *string, ro bool) {
	inner(c, text, ro)
}
`
		okGot := scanSrc(t, helpers+`
func viewOK(c *ui.Context) {
	local := "readonly content"
	outer(c, &local, true)
}`)
		if len(okGot) != 0 {
			t.Fatalf("ReadOnly(true) propagated through helper chain must pass, got %v", okGot)
		}
		badGot := scanSrc(t, helpers+`
func viewBad(c *ui.Context) {
	local := "editable content"
	outer(c, &local, false)
}`)
		if len(badGot) == 0 || !strings.Contains(badGot[0], "local") {
			t.Fatalf("ReadOnly(false) propagated through helper chain must flag unread local, got %v", badGot)
		}
	})

	t.Run("CollapsibleBase_bare_pointer_elemVars_exemption", func(t *testing.T) {
		okGot := scanSrc(t, `package p
func viewOK(c *ui.Context, open *bool, trigger func(bool)) {
	parts := ui.CollapsibleBase(c, open)
	parts.Trigger.Children(func() { trigger(*open) })
}`)
		if len(okGot) != 0 {
			t.Fatalf("control-own render closure reading *open must pass, got %v", okGot)
		}
		badGot := scanSrc(t, `package p
func viewBad(c *ui.Context, open *bool, trigger func(bool)) {
	_ = ui.CollapsibleBase(c, open)
	trigger(*open)
}`)
		if len(badGot) == 0 || !strings.Contains(badGot[0], "open") {
			t.Fatalf("standalone pre-flush trigger(*open) outside parts must flag open, got %v", badGot)
		}
	})

	t.Run("Key_and_transitive_methodUIBuilder", func(t *testing.T) {
		decls := `package p
type gen struct { c *ui.Context }
func (g *gen) node(c *ui.Context) { ui.Column(c) }
func (g *gen) nodes(c *ui.Context, s string) {
	g.c.Key(0)
	g.node(c)
}
func (g *gen) save(v int) { _ = v }
`
		okGot := scanSrc(t, decls+`
func viewOK(c *ui.Context, g *gen, sel *int, labels []string, opts coss.TabsOptions) {
	coss.Tabs(c, sel, labels, opts)
	c.Key(*sel)
	g.nodes(c, strconv.Itoa(*sel))
}`)
		if len(okGot) != 0 {
			t.Fatalf(".Key(*sel) and transitive UI builder g.nodes(*sel) must pass, got %v", okGot)
		}
		badGot := scanSrc(t, decls+`
func viewBad(c *ui.Context, g *gen, sel *int, labels []string, opts coss.TabsOptions) {
	coss.Tabs(c, sel, labels, opts)
	g.save(*sel)
}`)
		if len(badGot) == 0 || !strings.Contains(badGot[0], "sel") {
			t.Fatalf("non-UI method g.save(*sel) before flush must flag sel, got %v", badGot)
		}
	})
}
