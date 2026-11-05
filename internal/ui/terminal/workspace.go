package terminal

import (
	"errors"
	"strings"
	"whaleisle/internal/ui/theme"

	myterm "github.com/egoist/mygo/plugins/terminal"
)

const maxPanesPerGroup = 4

type terminalPane struct {
	appearanceFont   theme.FontPreference
	appearanceColors theme.Colors
	id               uint64
	term             *myterm.Terminal
	err              error
}

type terminalGroup struct {
	id       uint64
	panes    []uint64
	vertical bool
}

func (p *Panel) pane(id uint64) *terminalPane {
	for _, pane := range p.panes {
		if pane.id == id {
			return pane
		}
	}
	return nil
}

func (p *Panel) activeGroup() *terminalGroup {
	for _, group := range p.groups {
		for _, id := range group.panes {
			if id == p.active {
				return group
			}
		}
	}
	return nil
}

func (p *Panel) activate(id uint64) {
	if pane := p.pane(id); pane != nil {
		if old, next := p.activeGroup(), p.groupFor(id); old != nil && old != next {
			for _, member := range old.panes {
				if hidden := p.pane(member); hidden != nil && hidden.term != nil {
					hidden.term.FlushPTYResize()
				}
			}
		}
		p.active, p.focused = id, id
		p.Term, p.Err = pane.term, pane.err
	}
}

func (p *Panel) groupFor(id uint64) *terminalGroup {
	for _, group := range p.groups {
		for _, member := range group.panes {
			if member == id {
				return group
			}
		}
	}
	return nil
}

// FlushResize preserves final PTY dimensions when the drawer is hidden.
func (p *Panel) FlushResize() {
	if p == nil {
		return
	}
	for _, pane := range p.panes {
		if pane.term != nil {
			pane.term.FlushPTYResize()
		}
	}
}

// createPane starts a shell only for an explicit New or Split action.
// New gets a separate group; Split adds to the active group, capped at four.
func (p *Panel) createPane(dark, split, vertical bool) error {
	group := p.activeGroup()
	if split && (group == nil || len(group.panes) >= maxPanesPerGroup) {
		return errors.New(terminalLabel("每组最多四个终端", "A split group supports up to four terminals"))
	}
	p.nextID++
	pane := &terminalPane{id: p.nextID}
	term, err := openNativeTerminal(p.cwd, dark, p.linkActivator(pane))
	if err != nil {
		p.notice = err.Error()
		if len(p.panes) == 0 {
			p.Err = err
		}
		return err
	}
	pane.term = term
	p.panes = append(p.panes, pane)
	if split {
		group.panes = append(group.panes, pane.id)
		group.vertical = vertical
	} else {
		p.groups = append(p.groups, &terminalGroup{id: pane.id, panes: []uint64{pane.id}})
	}
	p.notice = ""
	p.activate(pane.id)
	return nil
}

// Restart replaces only the active process, keeping other tabs and split panes.
func (p *Panel) Restart(dark bool) {
	if p == nil {
		return
	}
	pane := p.pane(p.active)
	if pane == nil {
		_ = p.createPane(dark, false, false)
		return
	}
	if pane.term != nil {
		_ = pane.term.Close()
		pane.term = nil
	}
	pane.term, pane.err = openNativeTerminal(p.cwd, dark, p.linkActivator(pane))
	p.notice = ""
	p.activate(pane.id)
}

func (p *Panel) closePane(id uint64) {
	pane := p.pane(id)
	if pane == nil {
		return
	}
	if pane.term != nil {
		_ = pane.term.Close()
	}
	next := uint64(0)
	for i := 0; i < len(p.groups); i++ {
		group := p.groups[i]
		for j, member := range group.panes {
			if member != id {
				continue
			}
			group.panes = append(group.panes[:j], group.panes[j+1:]...)
			if len(group.panes) == 0 {
				p.groups = append(p.groups[:i], p.groups[i+1:]...)
			}
			break
		}
	}
	for i, member := range p.panes {
		if member.id == id {
			p.panes = append(p.panes[:i], p.panes[i+1:]...)
			if len(p.panes) > 0 {
				next = p.panes[min(i, len(p.panes)-1)].id
			}
			break
		}
	}
	if p.active == id {
		p.active, p.focused = 0, 0
		p.Term, p.Err = nil, nil
		if next == 0 && len(p.panes) > 0 {
			next = p.panes[len(p.panes)-1].id
		}
		p.activate(next)
	}
	p.notice = ""
}

func (p *Panel) currentDirectory() string {
	if p.Term != nil {
		if dir := strings.TrimSpace(p.Term.Dir()); dir != "" {
			return dir
		}
	}
	return p.cwd
}

// Links use exactly the same workspace/Host router as the selection toolbar.
// Always consume the event: an unavailable/stale route must not fall through to
// the SDK system opener and reinterpret a path or loopback service as a URL.
func (p *Panel) linkActivator(pane *terminalPane) func(string) bool {
	return func(target string) bool {
		if p.pane(pane.id) != pane || pane.term == nil {
			return true
		}
		if p.onOpenTarget == nil {
			p.notice = "Terminal target opener is unavailable"
			return true
		}
		if err := p.onOpenTarget(target); err != nil {
			p.notice = err.Error()
		} else {
			p.notice = ""
		}
		return true
	}
}
