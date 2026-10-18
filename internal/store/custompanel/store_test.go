package custompanel

import "testing"

func TestLibraryCopiesRemainIndependentAfterReload(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(`{"title":"Daily","width":420,"height":320,"columns":1,"widgets":[{"id":"note","kind":"input","label":"Notes","text":"seed"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	a := Scope{Host: "host", Bot: "a"}
	b := Scope{Host: "host", Bot: "b"}
	id, err := s.Create(a, d)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Add(b, id); err != nil {
		t.Fatal(err)
	}
	first := s.Snapshot(a).Instances[0]
	first.Definition.Width = 640
	first.Definition.Widgets[0].Text = "local"
	first.State["note"] = "private"
	if err = s.Update(a, first); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Snapshot(a).Instances[0]
	other := reloaded.Snapshot(b).Instances[0]
	if got.Definition.Width != 640 || got.State["note"] != "private" || other.Definition.Width != 420 || other.Definition.Widgets[0].Text != "seed" || other.State["note"] != "" {
		t.Fatalf("copies leaked: %+v %+v", got, other)
	}
	library := reloaded.Snapshot(b).Library[0]
	if library.Widgets[0].Text != "seed" {
		t.Fatal("instance changed library")
	}
	other.Definition.Widgets[0].Text = "snapshot only"
	if reloaded.Snapshot(b).Instances[0].Definition.Widgets[0].Text != "seed" {
		t.Fatal("snapshot aliases store")
	}
	if len(reloaded.Snapshot(Scope{Host: "other", Bot: "a"}).Instances) != 0 {
		t.Fatal("host scope leaked")
	}
}
