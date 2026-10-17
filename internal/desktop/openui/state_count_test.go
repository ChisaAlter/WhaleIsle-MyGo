package openui

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestOpenUIStateBeyond256(t *testing.T) {
	root := t.TempDir()
	for n := 0; n < 257; n++ {
		id := Identity("host", "session", fmt.Sprint(n), 0)
		if err := SaveState(root, id, json.RawMessage(`{"kept":true}`)); err != nil {
			t.Fatalf("state %d: %v", n+1, err)
		}
	}
	for _, n := range []int{0, 256} {
		got, err := ReadState(root, Identity("host", "session", fmt.Sprint(n), 0))
		if err != nil || string(got) != `{"kept":true}` {
			t.Fatalf("state %d not preserved: %s %v", n+1, got, err)
		}
	}
}
