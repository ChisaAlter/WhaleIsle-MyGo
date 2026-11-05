package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"strconv"
)

// Live opts this menu into rebuilding while it is open. It is useful when an
// asynchronous owner mounts additional items after the opening gesture. Other
// menus retain their original build-on-open behavior.
func (m *Menu) Live() {
	if m.run.collect {
		m.run.live = true
	}
}

// Key identifies an asynchronous item independently of its current list index.
// A withdrawn or replaced item cannot consume a choice for its predecessor.
func (it *MenuItem) Key(key string) *MenuItem {
	it.key = key
	if it.run.collect {
		it.run.keys[it.seq-1] = key
	}
	return it
}

func menuLiveIdentity(p *shownMenu, index int) string {
	if index < 0 || index >= len(p.menu.Items) {
		return ""
	}
	item := p.menu.Items[index]
	if item.ID > 0 && item.ID <= len(p.keys) && p.keys[item.ID-1] != "" {
		return "key:" + p.keys[item.ID-1]
	}
	return "row:" + strconv.Itoa(item.ID) + ":" + item.Label
}

func (e *node) rebuildLiveMenu(build func(*Menu)) {
	rt := e.c.rt
	p := rt.menu.overlay
	focused, submenu := menuLiveIdentity(p, rt.menu.index), menuLiveIdentity(p, rt.menu.submenu)
	run, pm := &menuRun{collect: true}, &platform.Menu{}
	build(&Menu{run: run, menu: pm, rt: rt, text: rt.states[p.text]})
	p.menu, p.labels, p.keys, p.commands = pm, run.labels, run.keys, run.commands
	p.presentation, p.closeOnPointerLeave, p.live = run.presentation, run.closeOnPointerLeave, run.live
	rt.menu.index, rt.menu.submenu = -1, -1
	for i, item := range pm.Items {
		identity := menuLiveIdentity(p, i)
		if item.Enabled && identity == focused {
			rt.menu.index = i
		}
		if item.Submenu != nil && identity == submenu {
			rt.menu.submenu = i
		}
	}
	if rt.menu.index == -1 {
		for i, item := range pm.Items {
			if item.Enabled && item.Type != platform.MenuItemSeparator {
				rt.menu.index = i
				break
			}
		}
	}
	// Headless hosts retain the same menu pointer and chosen callback contract.
	if h, ok := rt.host.(*headless); ok {
		h.menu = pm
	}
}
