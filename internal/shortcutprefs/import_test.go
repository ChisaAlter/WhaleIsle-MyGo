package shortcutprefs

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestPreviewImportReportsExplicitOverridesAndConflicts(t *testing.T) {
	s, data := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.same":{"code":"KeyK","modifiers":["control"]},"app.changed":{"code":"KeyP","modifiers":["primary"]},"app.disabled":null},"web:windows":{"local.keep":null}}}`)
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"app.same":{"code":"KeyK","modifiers":["primary"]},"app.changed":null,"app.disabled":{"code":"KeyD","modifiers":[]},"plugin.unknown":{"code":"KeyJ","secondCode":"KeyK","modifiers":["alt"]}},"desktop:macos":{"other.ignore":null}}}`)
	sourceBefore := append([]byte(nil), source...)
	before := append([]byte(nil), (*data)...)
	preview := s.PreviewImport(source, "web:windows")
	if preview.Status != "ready" || preview.SourceProfile != "web:windows" || preview.TargetProfile != "desktop:windows" {
		t.Fatalf("preview %+v", preview)
	}
	if !reflect.DeepEqual(preview.Conflicts, []string{"app.changed", "app.disabled"}) {
		t.Fatal(preview.Conflicts)
	}
	if len(preview.Entries) != 4 {
		t.Fatal(preview.Entries)
	}
	byID := map[string]ImportEntry{}
	for _, entry := range preview.Entries {
		byID[entry.ID] = entry
	}
	if byID["app.same"].Conflict || !byID["app.same"].HasExisting {
		t.Fatal(byID["app.same"])
	}
	if byID["plugin.unknown"].HasExisting || byID["plugin.unknown"].Incoming.SecondCode != "KeyK" {
		t.Fatal(byID["plugin.unknown"])
	}
	if !bytes.Equal(before, *data) || !bytes.Equal(source, sourceBefore) {
		t.Fatal("preview changed source or target")
	}
	byID["app.same"].Existing.Code = "KeyZ"
	if s.Get().Document.Profiles["desktop:windows"]["app.same"].Code != "KeyK" {
		t.Fatal("preview leaked stored binding")
	}
	encoded, err := json.Marshal(preview)
	if err != nil || !bytes.Contains(encoded, []byte(`"hasExisting":false`)) {
		t.Fatal(string(encoded), err)
	}
}

func TestImportKeepsConflictsAndAddsDormantBindings(t *testing.T) {
	s, data := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.changed":{"code":"KeyP","modifiers":["primary"]},"app.same":{"code":"KeyK","modifiers":["control"]}},"web:windows":{"local.keep":null}}}`)
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"app.changed":null,"app.same":{"code":"KeyK","modifiers":["primary"]},"plugin.future":{"code":"KeyJ","secondCode":"KeyK","modifiers":["alt"]}},"desktop:macos":{"other.ignore":null}}}`)
	s.SetDefinitions("native", []Definition{definition("app.absent", "KeyO")})
	preview := s.PreviewImport(source, "web:windows")
	var notifications int
	s.Subscribe(func(Snapshot) { notifications++ })
	result := s.Import(source, "web:windows", preview.Snapshot.Revision, nil)
	if result.Status != "saved" || notifications != 1 {
		t.Fatal(result, notifications)
	}
	doc, issue := ParseDocument(*data)
	if issue != "" {
		t.Fatal(issue)
	}
	target := doc.Profiles["desktop:windows"]
	if target["app.changed"].Code != "KeyP" {
		t.Fatal("default keep overwritten")
	}
	if !reflect.DeepEqual(target["app.same"].Modifiers, []string{"control"}) {
		t.Fatal("same binding rewritten")
	}
	if target["plugin.future"].SecondCode != "KeyK" {
		t.Fatal("dormant chord lost")
	}
	for _, id := range []string{"app.absent", "other.ignore"} {
		if _, ok := target[id]; ok {
			t.Fatalf("unexpected import %s", id)
		}
	}
	if _, ok := doc.Profiles["web:windows"]["local.keep"]; !ok {
		t.Fatal("unrelated profile lost")
	}
	if len(s.Effective("native")) != 1 {
		t.Fatal("import registered unknown command")
	}
	if !bytes.Contains(source, []byte(`"app.changed":null`)) {
		t.Fatal("source changed")
	}
}

func TestImportReplacesOnlyChosenConflictsIncludingNull(t *testing.T) {
	s, _ := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.one":{"code":"KeyO","modifiers":["primary"]},"app.two":null}}}`)
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":null,"app.two":{"code":"KeyT","modifiers":["primary"]}}}}`)
	preview := s.PreviewImport(source, "web:windows")
	result := s.Import(source, "web:windows", preview.Snapshot.Revision, []string{"app.one"})
	if result.Status != "saved" {
		t.Fatal(result)
	}
	target := result.Snapshot.Document.Profiles["desktop:windows"]
	if target["app.one"] != nil || target["app.two"] != nil {
		t.Fatal(target)
	}
	before := s.Get().Revision
	result = s.Import(source, "web:windows", before, nil)
	if result.Status != "saved" || result.Snapshot.Revision != before {
		t.Fatal("all-kept import must be a no-op", result)
	}
}

func TestImportRejectsStaleRevisionAndInvalidSelections(t *testing.T) {
	s, data := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.one":null}}}`)
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":{"code":"KeyO","modifiers":[]},"plugin.new":null}}}`)
	preview := s.PreviewImport(source, "web:windows")
	before := append([]byte(nil), (*data)...)
	for _, ids := range [][]string{{"missing.id"}, {"plugin.new"}, {"app.one", "app.one"}} {
		result := s.Import(source, "web:windows", preview.Snapshot.Revision, ids)
		if result.Status != "invalid-import" || result.Issue != "selection" {
			t.Fatal(result)
		}
		if !bytes.Equal(before, *data) {
			t.Fatal("invalid selection wrote")
		}
	}
	*data = []byte(`{"schemaVersion":2,"profiles":{"desktop:windows":{"external.change":null}}}`)
	result := s.Import(source, "web:windows", preview.Snapshot.Revision, []string{"app.one"})
	if result.Status != "stale" {
		t.Fatal(result)
	}
	if _, ok := result.Snapshot.Document.Profiles["desktop:windows"]["external.change"]; !ok {
		t.Fatal("external update lost")
	}
}

func TestImportRejectsActiveConflictsWithoutPartialWrite(t *testing.T) {
	s, data := memoryStore("")
	s.SetDefinitions("native", []Definition{definition("app.one", "KeyO"), definition("app.two", "KeyP"), {ID: "fixed.escape", Defaults: map[string]Binding{}, Fixed: []Binding{*binding("Escape")}}})
	for _, raw := range []string{
		`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":{"code":"KeyP","modifiers":["primary"]},"plugin.safe":null}}}`,
		`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":{"code":"KeyJ","secondCode":"KeyP","modifiers":["primary"]}}}}`,
		`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":{"code":"Escape","modifiers":[]}}}}`,
	} {
		source := []byte(raw)
		preview := s.PreviewImport(source, "web:windows")
		result := s.Import(source, "web:windows", preview.Snapshot.Revision, nil)
		if result.Status != "conflict" || len(result.Conflicts) == 0 || *data != nil {
			t.Fatalf("conflict %+v data=%s", result, *data)
		}
	}
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"app.one":{"code":"KeyP","modifiers":["primary"]},"app.two":{"code":"KeyO","modifiers":["primary"]}}}}`)
	preview := s.PreviewImport(source, "web:windows")
	if result := s.Import(source, "web:windows", preview.Snapshot.Revision, nil); result.Status != "saved" {
		t.Fatal(result)
	}
}

func TestImportStrictSourceAndUnreadableTarget(t *testing.T) {
	s, data := memoryStore("")
	for _, test := range []struct{ raw, profile, issue string }{
		{`{"schemaVersion":3,"profiles":{}}`, "web:windows", "future"},
		{`{"schemaVersion":2,"profiles":{},"extra":true}`, "web:windows", "invalid"},
		{`{"schemaVersion":2,"profiles":{}}`, "C:\\keybindings.json", "profile"},
	} {
		preview := s.PreviewImport([]byte(test.raw), test.profile)
		if preview.Status != "invalid-import" || preview.Error != test.issue {
			t.Fatal(preview)
		}
		result := s.Import([]byte(test.raw), test.profile, s.Get().Revision, nil)
		if result.Status != "invalid-import" || result.Issue != test.issue || *data != nil {
			t.Fatal(result)
		}
	}
	*data = []byte(`{"schemaVersion":99,"profiles":{}}`)
	source := []byte(`{"schemaVersion":2,"profiles":{"web:windows":{"plugin.new":null}}}`)
	preview := s.PreviewImport(source, "web:windows")
	if preview.Status != "unreadable" {
		t.Fatal(preview)
	}
	if result := s.Import(source, "web:windows", preview.Snapshot.Revision, nil); result.Status != "unreadable" {
		t.Fatal(result)
	}
}

func TestImportWriteFailureDoesNotPublishOrChangeAcceptedState(t *testing.T) {
	data := []byte(`{"schemaVersion":2,"profiles":{"desktop:windows":{"plugin.old":null}}}`)
	s := New("desktop:windows", func() ([]byte, error) { return data, nil }, func([]byte) error { return errors.New("disk full") })
	source := []byte(`{"schemaVersion":1,"profiles":{"web:windows":{"plugin.new":null}}}`)
	preview := s.PreviewImport(source, "web:windows")
	notifications := 0
	s.Subscribe(func(Snapshot) { notifications++ })
	result := s.Import(source, "web:windows", preview.Snapshot.Revision, nil)
	if result.Status != "write-failed" || result.Snapshot.Revision != preview.Snapshot.Revision || notifications != 0 {
		t.Fatal(result, notifications)
	}
	if _, ok := result.Snapshot.Document.Profiles["desktop:windows"]["plugin.new"]; ok {
		t.Fatal("failed import leaked")
	}
}
