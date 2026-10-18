package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/shortcuts"
)

func TestFunctionKeyNativeDispatch(t *testing.T) {
	for _, tc := range []struct {
		key    string
		native ui.Key
	}{{"F1", ui.KeyF1}, {"F5", ui.KeyF5}, {"F12", ui.KeyF12}} {
		mods, key := NativeCommandChord(shortcuts.Chord{Key: tc.key, Alt: true})
		if key != tc.native || mods != ui.Cmd|ui.Alt {
			t.Fatalf("%s => %v %v", tc.key, mods, key)
		}
	}
}
