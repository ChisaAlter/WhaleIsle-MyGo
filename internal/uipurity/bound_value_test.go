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
		filepath.Join("..", "..", "cmd", "whaleisle"),
		filepath.Join("..", "..", "internal", "app"),
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
	// The stale read of draft inside the compare flags via the stack-local
	// path; the snapshot path keys on the bound expression too.
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
		t.Fatalf("config reads and return operands must be exempt, got %v", got)
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
	// Reads inside a Children closure run during the enclosing call — the
	// staleness is real, so they still flag.
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
