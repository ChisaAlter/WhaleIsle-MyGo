package pluginslots

import "testing"

func TestProductKeyboardAcceptedState(t *testing.T) {
	good := []byte(`{"type":"state","revision":"r:2","active":true,"bindings":[{"code":"KeyP","mods":5}]}`)
	s, err := decodeProductKeyboard(good)
	if err != nil || !s.Active || len(s.Bindings) != 1 {
		t.Fatal(s, err)
	}
	for _, raw := range []string{`{"type":"rpc","revision":"r","bindings":[]}`, `{"type":"state","revision":"r","bindings":[{"code":"KeyP","mods":16}]}`, `{"type":"state","revision":"r","bindings":[{"code":"KeyP","mods":5},{"code":"KeyP","mods":5}]}`, `{"type":"state","revision":"r","bindings":[{"code":"arbitrary command","mods":1}]}`} {
		if _, err := decodeProductKeyboard([]byte(raw)); err == nil {
			t.Fatal("invalid registry admitted", raw)
		}
	}
	g := &Gateway{}
	if g.DispatchProductKey(1, "r", "KeyP", 5) == nil {
		t.Fatal("key sent without authenticated product page")
	}
}

func TestProductKeyboardPairValidation(t *testing.T) {
	for _, binding := range []string{`{"code":"KeyA","secondCode":"KeyB","mods":0}`, `{"code":"F24","mods":8}`} {
		if _, err := decodeProductKeyboard([]byte(`{"type":"state","revision":"r","bindings":[` + binding + `]}`)); err != nil {
			t.Fatal(err)
		}
	}
	for _, bindings := range []string{`{"code":"KeyB","secondCode":"KeyA","mods":0}`, `{"code":"KeyA","secondCode":"KeyA","mods":0}`, `{"code":"KeyA","secondCode":"unknown","mods":0}`, `{"code":"KeyA","secondCode":"KeyB","mods":0},{"code":"KeyB","mods":0}`} {
		if _, err := decodeProductKeyboard([]byte(`{"type":"state","revision":"r","bindings":[` + bindings + `]}`)); err == nil {
			t.Fatal("invalid pair admitted", bindings)
		}
	}
}
