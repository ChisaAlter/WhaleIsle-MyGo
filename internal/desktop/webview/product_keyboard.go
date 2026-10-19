package webview

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
)

func (h *Host) installProductKeyboard() {
	if h.gateway == nil || h.win == nil {
		return
	}
	gateway, child := h.gateway, h.win
	parent, closeWindow := h.parent, h.onShortcutCloseWindow
	gateway.SetProductCloseWindow(func(snapshot pluginslots.ProductKeyboardSnapshot) {
		current := func() bool {
			h.mu.Lock()
			owned := !h.stopped && h.gateway == gateway && h.win == child && h.parent == parent
			h.mu.Unlock()
			return owned && parent != nil && gateway.OwnsProductKeyboard(snapshot) && !child.IsDestroyed() && child.IsVisible() && child.IsFocused() && child.ShortcutInputEnabled() && !parent.IsDestroyed() && parent.IsVisible() && parent.ShortcutInputEnabled()
		}
		if closeWindow != nil && current() {
			closeWindow(parent, current)
		}
	})
	changed := func() {
		snapshot := gateway.ProductKeyboard()
		keys := make([]mygo.ProductKeyBinding, 0, len(snapshot.Bindings))
		for _, key := range snapshot.Bindings {
			keys = append(keys, mygo.ProductKeyBinding{Code: key.Code, Mods: key.Mods, SecondCode: key.SecondCode})
		}
		child.Page().SetProductKeyboard(snapshot.Revision, snapshot.Active, keys, func(revision, code string, mods uint8, second string) {
			go gateway.DispatchProductKey(snapshot.Page, revision, code, mods, second)
		})
	}
	gateway.SetProductKeyboardChanged(changed)
	child.OnBlur(func() { child.Page().SetProductKeyboard("", false, nil, nil) })
	child.Page().OnWillNavigate(func(*mygo.NavigateEvent) { gateway.DisconnectProductKeyboard() })
	child.OnClosed(gateway.DisconnectProductKeyboard)
	child.Page().OnRenderProcessGone(func(string) { gateway.DisconnectProductKeyboard() })
	changed()
}
