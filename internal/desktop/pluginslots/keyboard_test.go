package pluginslots

import "testing"

func TestProductKeyboardAcceptedState(t *testing.T) {
	good := []byte(`{"type":"state","revision":"r:2","active":true,"bindings":[{"code":"KeyP","mods":5}]}`)
	s, err := decodeProductKeyboard(good)
	if err != nil || !s.Active || len(s.Bindings) != 1 {
		t.Fatal(s, err)
	}
	for _, raw := range []string{`{"type":"rpc","revision":"r","bindings":[]}`, `{"type":"state","revision":"r","bindings":[{"code":"KeyP","mods":0}]}`, `{"type":"state","revision":"r","bindings":[{"code":"KeyP","mods":5},{"code":"KeyP","mods":5}]}`, `{"type":"state","revision":"r","bindings":[{"code":"arbitrary command","mods":1}]}`} {
		if _, err := decodeProductKeyboard([]byte(raw)); err == nil {
			t.Fatal("invalid registry admitted", raw)
		}
	}
	g := &Gateway{}
	if g.DispatchProductKey(1, "r", "KeyP", 5) == nil {
		t.Fatal("key sent without authenticated product page")
	}
}
