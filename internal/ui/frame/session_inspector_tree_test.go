package frame

import (
	"reflect"
	"testing"
)

func TestInspectorJSONTreeExpandedAndDeterministic(t *testing.T) {
	var state inspectorJSONTreeState
	state.setRaw([]byte(`{"z":null,"a":[true,{"large":9007199254740993,"text":"hello"}],"empty":{}}`))
	rows := state.visibleRows()
	var names []string
	for _, row := range rows {
		names = append(names, row.name)
	}
	want := []string{"$", "z", "a", "0", "1", "large", "text", "empty"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %#v; want %#v", names, want)
	}
	if rows[5].label != "9007199254740993" {
		t.Fatalf("number lost precision: %q", rows[5].label)
	}
	if rows[7].label != "Object" || !rows[7].expandable {
		t.Fatalf("empty object = %+v", rows[7])
	}
	if rows[1].label != "null" {
		t.Fatalf("null = %+v", rows[1])
	}
}

func TestInspectorJSONTreeCollapseIsPathLocal(t *testing.T) {
	var state inspectorJSONTreeState
	raw := []byte(`{"a/b":{"x":1},"a":{"b":{"x":2}},"~":[3]}`)
	state.setRaw(raw)
	state.toggle("/a~1b")
	rows := state.visibleRows()
	var paths []string
	for _, row := range rows {
		paths = append(paths, row.path)
	}
	want := []string{"", "/a~1b", "/a", "/a/b", "/a/b/x", "/~0", "/~0/0"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v; want %#v", paths, want)
	}
	state.setRaw(raw)
	if !state.collapsed["/a~1b"] {
		t.Fatal("same raw value reset collapse")
	}
	state.toggle("")
	if len(state.visibleRows()) != 1 {
		t.Fatal("root collapse did not hide descendants")
	}
	state.setRaw([]byte(`{"new":[1,2]}`))
	if len(state.visibleRows()) != 1 {
		t.Fatal("live JSON refresh lost disclosure state")
	}
}

func TestInspectorJSONTreeScalarsAndInvalidJSON(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `"text\\nline"`, `1e100`, `[]`, `{}`} {
		var state inspectorJSONTreeState
		state.setRaw([]byte(raw))
		rows := state.visibleRows()
		want := raw
		if raw == "[]" {
			want = "Array(0)"
		}
		if raw == "{}" {
			want = "Object"
		}
		if state.invalid || len(rows) != 1 || rows[0].label != want {
			t.Fatalf("%q: invalid=%v rows=%+v", raw, state.invalid, rows)
		}
	}
	for _, raw := range []string{"", "{broken", "true false"} {
		var state inspectorJSONTreeState
		state.setRaw([]byte(raw))
		if !state.invalid || state.raw != raw {
			t.Fatalf("invalid input not retained: %q", raw)
		}
	}
}
