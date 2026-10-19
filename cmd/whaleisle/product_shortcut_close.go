package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type productShortcutClose struct {
	owner   *mygo.Window
	current func() bool
}

// The factory passes the real leased product host's parent. Never replace it
// with mainState.win: another native occurrence can own a different window.
func (s *mainState) queueProductShortcutClose(owner *mygo.Window, current func() bool) {
	if owner == nil || current == nil {
		return
	}
	owner.Update(func() {
		if !current() {
			return
		}
		s.mu.Lock()
		s.productShortcutCloses = append(s.productShortcutCloses, productShortcutClose{owner, current})
		s.mu.Unlock()
	})
}

// Run after this frame's dialogs have established modal ownership. The child
// has focus, so the parent's windowFocused flag cannot stand in for its lease.
func (s *mainState) processProductShortcutClose(c *ui.Context) {
	s.mu.Lock()
	pending := s.productShortcutCloses
	s.productShortcutCloses = nil
	quitting := s.quitting
	editor := s.shortcutEditor
	s.mu.Unlock()
	if quitting || !c.MenuOverlayClear() || editor != nil && (editor.Recording != "" || editor.ResetPending) {
		return
	}
	for _, request := range pending {
		if request.owner != nil && request.current != nil && request.current() && !request.owner.IsDestroyed() && request.owner.ShortcutInputEnabled() {
			request.owner.Close()
			return
		}
	}
}
