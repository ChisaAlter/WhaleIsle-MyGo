package webview

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestDevToolsShortcutGating(t *testing.T) {
	cases := []struct {
		packaged, openDevTools bool
		want                   bool
	}{
		{false, false, true},  // development always allows it
		{false, true, true},   // ... and the setting changes nothing there
		{true, false, false},  // packaged without the setting is blocked
		{true, true, true},    // packaged with the explicit setting is allowed
	}
	for _, c := range cases {
		if got := DevToolsShortcutAllowed(c.packaged, c.openDevTools); got != c.want {
			t.Errorf("DevToolsShortcutAllowed(packaged=%v, openDevTools=%v) = %v, want %v",
				c.packaged, c.openDevTools, got, c.want)
		}
	}
}

func TestIsDevToolsToggle(t *testing.T) {
	key := func(k string, ctrl, shift, alt, meta, down bool) mygo.KeyInput {
		return mygo.KeyInput{Key: k, Control: ctrl, Shift: shift, Alt: alt, Meta: meta, Down: down}
	}
	cases := []struct {
		name string
		in   mygo.KeyInput
		want bool
	}{
		{"ctrl+shift+i", key("i", true, true, false, false, true), true},
		{"cmd+alt+i", key("i", false, false, true, true, true), true},
		{"ctrl+shift+i key up", key("i", true, true, false, false, false), false},
		{"ctrl+i without shift", key("i", true, false, false, false, true), false},
		{"shift+i without ctrl", key("i", false, true, false, false, true), false},
		{"ctrl+shift+i with alt", key("i", true, true, true, false, true), false},
		{"cmd+shift+i", key("i", false, true, false, true, true), false},
		{"other key", key("j", true, true, false, false, true), false},
		{"empty key", key("", true, true, false, false, true), false},
	}
	for _, c := range cases {
		if got := IsDevToolsToggle(c.in); got != c.want {
			t.Errorf("%s: IsDevToolsToggle = %v, want %v", c.name, got, c.want)
		}
	}
}
