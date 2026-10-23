package ui

import "github.com/egoist/mygo/internal/platform"

// ObserveInput reports pointer presses, wheel/touchpad scrolling and key presses
// anywhere in this window, before focus, shortcuts, scrollbars or controls handle
// them. Coordinates are window DIPs. It does not consume or redirect the event.
// Register while building a view; only the last build pass's observers remain
// until the next build or window close. Use it to cancel a temporary operation
// on user input, and invalidate the window if the callback changes visible state.
func (c *Context) ObserveInput(fn func(InputEvent)) {
	if raw := c.build(); raw != nil && fn != nil {
		raw.rt.inputObservers = append(raw.rt.inputObservers, fn)
	}
}

func (rt *engine) observeInput(ev platform.SurfaceEvent) {
	if len(rt.inputObservers) == 0 {
		return
	}
	input := InputEvent{X: float32(ev.X), Y: float32(ev.Y), Mods: Modifiers(ev.Mods)}
	switch ev.Kind {
	case platform.PointerDown:
		input.Kind, input.Button, input.Clicks = InputPointerDown, ev.Button, ev.Clicks
	case platform.PointerScroll:
		input.Kind, input.DX, input.DY, input.Precise = InputScroll, float32(ev.DX), float32(ev.DY), ev.Precise
	case platform.KeyPressed:
		input.Kind, input.Key, input.Repeat, input.Code, input.ShortcutBlocked = InputKeyDown, Key(ev.Key), ev.Repeat, ev.Code, ev.ShortcutBlocked
	default:
		return
	}
	for _, fn := range rt.inputObservers {
		fn(input)
	}
}
