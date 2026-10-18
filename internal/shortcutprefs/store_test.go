package shortcutprefs

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
)

func binding(code string, modifiers ...string) *Binding {
	return &Binding{Code: code, Modifiers: modifiers}
}
func definition(id, code string) Definition {
	return Definition{ID: id, Defaults: map[string]Binding{"desktop:windows": *binding(code, "primary")}}
}
func memoryStore(raw string) (*Store, *[]byte) {
	var data []byte
	if raw != "" {
		data = []byte(raw)
	}
	s := New("desktop:windows", func() ([]byte, error) { return append([]byte(nil), data...), nil }, func(v []byte) error { data = append([]byte(nil), v...); return nil })
	return s, &data
}
func TestPatchPreservesUnknownProfilesAndTwoKeys(t *testing.T) {
	s, data := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"plugin.future":{"code":"KeyJ","secondCode":"KeyK","modifiers":["control","alt"]},"app.open":{"code":"KeyP","modifiers":["primary"]}},"web:linux":{"plugin.dormant":null}}}`)
	snap := s.SetDefinitions("native", []Definition{definition("app.open", "KeyO")})
	result := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.open", Binding: binding("KeyB", "primary")}})
	if result.Status != "saved" {
		t.Fatalf("save: %+v", result)
	}
	got, err := ParseDocument(*data)
	if err != "" {
		t.Fatal(err)
	}
	if got.Profiles["desktop:windows"]["plugin.future"].SecondCode != "KeyK" {
		t.Fatal("two-key plugin binding lost")
	}
	if _, ok := got.Profiles["web:linux"]["plugin.dormant"]; !ok {
		t.Fatal("other profile lost")
	}
	if result.Snapshot.Document.Profiles["desktop:windows"]["app.open"].Code != "KeyB" {
		t.Fatal("accepted snapshot stale")
	}
}
func TestResetOnlyOwnedEditableRows(t *testing.T) {
	s, _ := memoryStore(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.open":null,"plugin.unknown":{"code":"KeyU","modifiers":[]},"product.other":null,"fixed.close":null}}}`)
	snap := s.SetDefinitions("native", []Definition{definition("app.open", "KeyO"), {ID: "fixed.close", Defaults: map[string]Binding{}, Fixed: []Binding{*binding("Escape")}}})
	snap = s.SetDefinitions("product", []Definition{definition("product.other", "KeyP")})
	res := s.Apply("native", snap.Revision, []Edit{{Type: "reset-all"}})
	if res.Status != "saved" {
		t.Fatal(res.Status)
	}
	values := res.Snapshot.Document.Profiles["desktop:windows"]
	if _, ok := values["app.open"]; ok {
		t.Fatal("owned override not removed")
	}
	for _, id := range []string{"plugin.unknown", "product.other", "fixed.close"} {
		if _, ok := values[id]; !ok {
			t.Fatalf("removed %s", id)
		}
	}
	for _, id := range []string{"plugin.unknown", "product.other", "fixed.close"} {
		if got := s.Apply("native", res.Snapshot.Revision, []Edit{{Type: "set", ID: id, Binding: nil}}).Status; got != "not-ready" {
			t.Fatalf("unauthorized %s: %s", id, got)
		}
	}
}
func TestCASCatalogChangesAndExternalWrites(t *testing.T) {
	s, data := memoryStore("")
	first := s.SetDefinitions("native", []Definition{definition("app.open", "KeyO")})
	second := s.SetDefinitions("product", []Definition{definition("app.other", "KeyP")})
	if got := s.Apply("native", first.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}}).Status; got != "stale" {
		t.Fatal(got)
	}
	*data = []byte(`{"schemaVersion":2,"profiles":{"desktop:windows":{"plugin.external":null}}}`)
	result := s.Apply("native", second.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}})
	if result.Status != "stale" {
		t.Fatal(result.Status)
	}
	if _, ok := result.Snapshot.Document.Profiles["desktop:windows"]["plugin.external"]; !ok {
		t.Fatal("external data not loaded")
	}
	snap := s.Get()
	var wg sync.WaitGroup
	statuses := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}}).Status
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[string]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts["saved"] != 1 || counts["stale"] != 1 {
		t.Fatal(counts)
	}
}
func TestWriteFailureAndReadFailureRetainAcceptedDocument(t *testing.T) {
	raw := []byte(`{"schemaVersion":2,"profiles":{"desktop:windows":{"app.open":{"code":"KeyP","modifiers":["primary"]}}}}`)
	failRead := false
	s := New("desktop:windows", func() ([]byte, error) {
		if failRead {
			return nil, errors.New("read failure")
		}
		return raw, nil
	}, func([]byte) error { return errors.New("write failure") })
	snap := s.SetDefinitions("native", []Definition{definition("app.open", "KeyO")})
	result := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}})
	if result.Status != "write-failed" || result.Snapshot.Document.Profiles["desktop:windows"]["app.open"].Code != "KeyP" || result.Snapshot.Revision != snap.Revision {
		t.Fatal(result)
	}
	failRead = true
	unreadable := s.Get()
	if unreadable.Status != "unreadable" || *unreadable.Error != "read" || unreadable.Document.Profiles["desktop:windows"]["app.open"].Code != "KeyP" {
		t.Fatal(unreadable)
	}
	if got := s.Apply("native", unreadable.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}}).Status; got != "unreadable" {
		t.Fatal(got)
	}
	failRead = false
	if got := s.Get().Status; got != "ready" {
		t.Fatal(got)
	}
}
func TestPhysicalOverlapAndAtomicBatch(t *testing.T) {
	s, _ := memoryStore("")
	snap := s.SetDefinitions("native", []Definition{definition("app.one", "KeyO"), definition("app.two", "KeyP"), {ID: "fixed.escape", Defaults: map[string]Binding{}, Fixed: []Binding{*binding("Escape")}}})
	for _, candidate := range []*Binding{binding("KeyP", "primary"), {Code: "KeyJ", SecondCode: "KeyP", Modifiers: []string{"control"}}, binding("Escape")} {
		result := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.one", Binding: candidate}})
		if result.Status != "conflict" || len(result.Conflicts) == 0 {
			t.Fatalf("candidate %+v: %+v", candidate, result)
		}
	}
	result := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.one", Binding: binding("KeyP", "primary")}, {Type: "set", ID: "app.two", Binding: binding("KeyO", "primary")}})
	if result.Status != "saved" {
		t.Fatal(result)
	}
}
func TestSharedDefinitionProfilesAndDetachedSnapshots(t *testing.T) {
	s, _ := memoryStore("")
	native := definition("app.open", "KeyO")
	s.SetDefinitions("native", []Definition{native})
	product := definition("app.open", "KeyO")
	product.Defaults["web:macos"] = *binding("KeyO", "primary", "alt")
	snap := s.SetDefinitions("product", []Definition{product})
	if snap.Status != "ready" {
		t.Fatal(snap)
	}
	product.Defaults["desktop:windows"] = *binding("KeyQ")
	if got := s.Effective("product"); len(got) != 1 || got[0].Binding.Code != "KeyO" {
		t.Fatal(got)
	}
	res := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.open", Binding: binding("KeyK", "primary")}})
	res.Snapshot.Document.Profiles["desktop:windows"]["app.open"].Code = "KeyZ"
	if s.Get().Document.Profiles["desktop:windows"]["app.open"].Code != "KeyK" {
		t.Fatal("mutable snapshot escaped")
	}
	s.SetDefinitions("product", nil)
	if s.Get().Document.Profiles["desktop:windows"]["app.open"].Code != "KeyK" {
		t.Fatal("catalog revoke removed override")
	}
	var notified int
	unsubscribe := s.Subscribe(func(snapshot Snapshot) {
		notified++
		_ = s.Get()
		snapshot.Document.Profiles = map[string]map[string]*Binding{}
	})
	current := s.Get()
	s.Apply("native", current.Revision, []Edit{{Type: "reset", ID: "app.open"}})
	unsubscribe()
	if notified != 1 {
		t.Fatal(notified)
	}
}
func TestStrictOriginalSchemaAndGrammar(t *testing.T) {
	valid := []string{
		`{"schemaVersion":1,"profiles":{}}`,
		`{"schemaVersion":2,"profiles":{"desktop:windows":{"plugin.future":{"code":"F24","secondCode":"Escape","modifiers":["primary","control","meta","alt","shift"]}}}}`,
	}
	for _, raw := range valid {
		if _, err := ParseDocument([]byte(raw)); err != "" {
			t.Fatalf("valid %s: %s", raw, err)
		}
	}
	invalid := []string{`null`, `[]`, `{}`, `{"schemaVersion":2,"profiles":{},"revision":"x"}`, `{"schemaVersion":2,"profiles":{"desktop:plan9":{}}}`, `{"schemaVersion":1,"profiles":{"desktop:windows":{"app.open":{"code":"KeyJ","secondCode":"KeyK","modifiers":[]}}}}`, `{"schemaVersion":2,"profiles":{"desktop:windows":{"app.open":{"code":"KeyJ","modifiers":null}}}}`, `{"schemaVersion":2,"profiles":{"desktop:windows":{"app.open":{"code":"KeyJ","secondCode":"KeyJ","modifiers":[]}}}}`, `{"schemaVersion":2,"profiles":{"desktop:windows":{"app.open":{"code":"KeyJ","modifiers":[],"future":true}}}}`, `{"schemaVersion":2,"profiles":{"desktop:windows":{"bad_id":null}}}`}
	for _, raw := range invalid {
		if _, err := ParseDocument([]byte(raw)); err != "invalid" {
			t.Fatalf("invalid %s: %s", raw, err)
		}
	}
	if _, err := ParseDocument([]byte(`{"schemaVersion":3,"profiles":{}}`)); err != "future" {
		t.Fatal(err)
	}
	if doc, err := ParseDocument(nil); err != "" || doc.SchemaVersion != 2 {
		t.Fatal(doc, err)
	}
	for _, raw := range []string{`{"type":"set","id":"app.open","binding":null}`, `{"type":"reset","id":"app.open"}`, `{"type":"reset-all"}`} {
		if _, err := ParseEdit([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{"type":"set","id":"app.open"}`, `{"type":"reset","id":"app.open","binding":null}`, `{"type":"reset-all","id":"app.open"}`} {
		if _, err := ParseEdit([]byte(raw)); err == nil {
			t.Fatal(raw)
		}
	}
}
func TestInvalidDocumentsNeverWritten(t *testing.T) {
	for _, raw := range []string{`{"schemaVersion":99,"profiles":{}}`, `{"schemaVersion":2,"profiles":{"desktop:windows":{"plugin.future":{"future":true}}}}`} {
		s, data := memoryStore(raw)
		snap := s.SetDefinitions("native", []Definition{definition("app.open", "KeyO")})
		if snap.Status != "unreadable" {
			t.Fatal(snap)
		}
		if got := s.Apply("native", snap.Revision, []Edit{{Type: "set", ID: "app.open", Binding: nil}}).Status; got != "unreadable" {
			t.Fatal(got)
		}
		if string(*data) != raw {
			t.Fatal("invalid source overwritten")
		}
	}
}
func TestEffectiveOverridePrecedenceAndTwoKeyPairs(t *testing.T) {
	defs := []Definition{definition("app.one", "KeyO"), definition("app.two", "KeyP")}
	doc := Document{SchemaVersion: 2, Profiles: map[string]map[string]*Binding{"desktop:windows": {"app.one": binding("KeyP", "primary")}}}
	rows := Effective(defs, doc, "desktop:windows")
	if len(rows[0].Conflicts) != 0 || !reflect.DeepEqual(rows[1].Conflicts, []string{"app.one"}) {
		t.Fatal(rows)
	}
	doc.Profiles["desktop:windows"]["app.one"] = &Binding{Code: "KeyJ", SecondCode: "KeyK", Modifiers: []string{"primary"}}
	doc.Profiles["desktop:windows"]["app.two"] = &Binding{Code: "KeyJ", SecondCode: "KeyL", Modifiers: []string{"primary"}}
	rows = Effective(defs, doc, "desktop:windows")
	if len(rows[0].Conflicts) != 0 || len(rows[1].Conflicts) != 0 {
		t.Fatal(rows)
	}
	data, err := json.Marshal(rows)
	if err != nil || len(data) == 0 {
		t.Fatal(err)
	}
}
