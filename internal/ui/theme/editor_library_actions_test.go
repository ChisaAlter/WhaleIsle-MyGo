package theme

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDeleteFamilyKeepsOtherDraftAndFallsBackOnlyAddressedHalf(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	prefs := Defaults()
	for _, id := range []string{"custom-a", "custom-b"} {
		f := prefs.DuplicateFamily(prefs.Family("celadon"))
		f.ID, f.Name = id, id
		prefs.Custom = append(prefs.Custom, f)
	}
	prefs.Light = "custom-b"
	var saved Preferences
	e := NewEditor(prefs, func(p Preferences) error { saved = p; return nil })
	e.StartFamily(e.Draft.Family("custom-a"))
	e.Draft.Custom[0].Name = "Unsaved A"
	if !e.DeleteFamily("custom-b") {
		t.Fatal("delete failed")
	}
	if saved.Light != Defaults().Light || saved.Dark != prefs.Dark || len(saved.Custom) != 1 || saved.Custom[0].Name != "custom-a" {
		t.Fatalf("delete persisted another draft or wrong fallback: %+v", saved)
	}
	if e.Selected != "custom-a" || e.Draft.Custom[0].Name != "Unsaved A" {
		t.Fatal("delete discarded unrelated editor draft")
	}
	if e.DeleteFamily("deepseek") || e.DeleteFamily("missing") {
		t.Fatal("deleted builtin/missing family")
	}
}

func TestDeleteFamilyFailureKeepsSelectedFamilyAndPreview(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	prefs := Defaults()
	f := prefs.DuplicateFamily(prefs.Family("violet"))
	prefs.Custom = []Family{f}
	e := NewEditor(prefs, func(Preferences) error { return fmt.Errorf("write failed") })
	e.StartFamily(f)
	before := e.Draft.Clone()
	if e.DeleteFamily(f.ID) {
		t.Fatal("delete reported success")
	}
	if e.Selected != f.ID || !e.Preview || !reflect.DeepEqual(e.Draft, before) || !reflect.DeepEqual(e.Saved, prefs) || e.Status != "write failed" {
		t.Fatal("failed delete changed editor")
	}
}
