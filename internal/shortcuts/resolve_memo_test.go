package shortcuts

import (
	"reflect"
	"testing"
)

func TestResolveMemoMatchesAndOwnsItsResult(t *testing.T) {
	values := map[string]string{}
	first := Resolve(values)
	if !reflect.DeepEqual(first, resolve(values)) {
		t.Fatal("memoized result differs from a direct resolve")
	}
	for id := range first {
		delete(first, id)
	}
	if again := Resolve(values); len(again) == 0 || !reflect.DeepEqual(again, resolve(values)) {
		t.Fatal("a caller's edit reached the memo")
	}
	var id string
	for _, c := range Commands() {
		if c.Default != "" {
			id = c.ID
			break
		}
	}
	values[id] = ""
	if _, bound := Resolve(values)[id]; bound {
		t.Fatal("a changed binding was served from the memo")
	}
	values[id] = "Mod+Shift+F12"
	if got := Resolve(values); !reflect.DeepEqual(got, resolve(values)) {
		t.Fatal("a rebound command differs from a direct resolve")
	}
}
