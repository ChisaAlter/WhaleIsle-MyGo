package pluginui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Edit commands use the original browser's native keyboard operation. Clipboard
// text is neither read nor duplicated by this bridge; original page permissions
// and handlers decide the resulting operation.
func (c *MirrorTextClient) command(name string) bool {
	type key struct {
		key, code string
		vk        int
		ctrl      bool
	}
	keys := map[string]key{"copy": {"c", "KeyC", 67, true}, "cut": {"x", "KeyX", 88, true}, "paste": {"v", "KeyV", 86, true}, "selectAll": {"a", "KeyA", 65, true}, "undo": {"z", "KeyZ", 90, true}, "redo": {"y", "KeyY", 89, true}, "delete": {"Delete", "Delete", 46, false}}
	selected, ok := keys[name]
	if !ok {
		return false
	}
	modifiers := mygo.BrowserTextModifiers(0)
	if selected.ctrl {
		modifiers = mygo.BrowserTextCtrl
	}
	down := mygo.BrowserTextEvent{Kind: mygo.BrowserTextKeyDown, Key: selected.key, Code: selected.code, WindowsVirtualKeyCode: selected.vk, Modifiers: modifiers}
	up := down
	up.Kind = mygo.BrowserTextKeyUp
	return c.enqueue(down) && c.enqueue(up)
}
func (c *MirrorTextClient) blur() {
	o := c.owner
	o.mu.Lock()
	focus, source := c.focus, c.source
	c.focus = nil
	c.marked = nil
	c.pending = 0
	if o.mirror != nil && o.mirror.text != nil {
		state := o.mirror.text
		kept := state.queue[:0]
		for _, job := range state.queue {
			if job.client != c {
				kept = append(kept, job)
			}
		}
		state.queue = kept
	}
	o.mu.Unlock()
	if focus != nil {
		go o.cancelMirrorText(source, focus)
	}
}
func (c *MirrorTextClient) handleSpecial(event ui.InputEvent) bool {
	switch event.Kind {
	case ui.InputBlur:
		c.blur()
		return false
	case ui.InputCommand:
		return c.command(event.Text)
	}
	return false
}
