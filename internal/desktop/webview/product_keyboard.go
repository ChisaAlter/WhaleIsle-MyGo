package webview

import "github.com/egoist/mygo"

func (h *Host) installProductKeyboard() {
	if h.gateway == nil || h.win == nil {
		return
	}
	gateway, child := h.gateway, h.win
	changed := func() {
		snapshot := gateway.ProductKeyboard()
		keys := make([]mygo.ProductKeyBinding, 0, len(snapshot.Bindings))
		for _, key := range snapshot.Bindings {
			keys = append(keys, mygo.ProductKeyBinding{Code: key.Code, Mods: key.Mods})
		}
		child.Page().SetProductKeyboard(snapshot.Revision, snapshot.Active, keys, func(revision, code string, mods uint8) {
			go gateway.DispatchProductKey(snapshot.Page, revision, code, mods)
		})
	}
	gateway.SetProductKeyboardChanged(changed)
	child.OnBlur(func() { child.Page().SetProductKeyboard("", false, nil, nil) })
	child.Page().OnWillNavigate(func(*mygo.NavigateEvent) { gateway.DisconnectProductKeyboard() })
	child.OnClosed(gateway.DisconnectProductKeyboard)
	child.Page().OnRenderProcessGone(func(string) { gateway.DisconnectProductKeyboard() })
	changed()
}
