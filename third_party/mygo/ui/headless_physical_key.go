package ui

import "github.com/egoist/mygo/internal/platform"

// PhysicalKeyDown drives the same surface/input route as a native keyboard,
// without synthesizing text or releasing the key. Code is the physical DOM code;
// key is its legacy logical key. Repeat, blocked (for example AltGr), and dead
// are independent native-event flags. Like Key, the call settles requested frames.
func (t *Tester) PhysicalKeyDown(code string, key Key, mods Modifiers, repeat, blocked, dead bool) {
	t.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Code: code, Key: platform.Key(key), Mods: platform.Modifiers(mods), Repeat: repeat, ShortcutBlocked: blocked, ShortcutDead: dead})
}

// PhysicalKeyUp releases exactly code through the real native input route. It
// does not release other held keys or modifiers. Blocked retains the native
// producer's AltGr/IME guard on a release event.
func (t *Tester) PhysicalKeyUp(code string, key Key, mods Modifiers, blocked bool) {
	t.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Code: code, Key: platform.Key(key), Mods: platform.Modifiers(mods), ShortcutBlocked: blocked})
}
