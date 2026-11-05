package pluginui

import (
	"fmt"
	"github.com/egoist/mygo/ui"
)

func (p PublicContributions) renderMenuActions(menu *ui.Menu, actions []PublicMenuAction) {
	for _, action := range actions {
		key := fmt.Sprintf("%p:%d:%s:%s:%s", action.owner, action.Page, action.Slot, action.OwnerKey, action.Handle)
		if menu.Item(action.Label).Key(key).Disabled(action.Disabled).Chosen() {
			p.Invoke(action)
		}
	}
}

// RenderMenu consumes an external-only menu directory. Live rebuilding lets
// the original source owner mount its children after the opening gesture.
func (p PublicContributions) RenderMenu(menu *ui.Menu, slot, ownerKey string) {
	if p.MenuActions == nil || p.Invoke == nil {
		return
	}
	menu.Live()
	p.renderMenuActions(menu, p.MenuActions(slot, ownerKey))
}

// RenderMenuBefore inserts only the source entries immediately preceding the
// named original builtin. An empty ID consumes the final external bucket.
func (p PublicContributions) RenderMenuBefore(menu *ui.Menu, slot, ownerKey, beforeID string) bool {
	keepNative := p.NativeWinner(slot, ownerKey, beforeID)
	if p.Entries == nil || p.MenuActions == nil || p.Invoke == nil {
		return keepNative
	}
	menu.Live()
	bucket := map[string]bool{}
	matched := beforeID == ""
	for _, entry := range p.Entries(slot, ownerKey) {
		if entry.Native {
			if entry.ID == beforeID {
				matched = true
				break
			}
			bucket = map[string]bool{}
		} else {
			bucket[entry.Handle] = true
		}
	}
	if !matched {
		return keepNative
	}
	actions := []PublicMenuAction{}
	for _, action := range p.MenuActions(slot, ownerKey) {
		if bucket[action.EntryHandle] {
			actions = append(actions, action)
		}
	}
	p.renderMenuActions(menu, actions)
	return keepNative
}
