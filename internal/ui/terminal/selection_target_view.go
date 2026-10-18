package terminal

import (
	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// A changed Host/session, pane, process, or selected text gets a different
// pointer gesture key. Toolbar focus alone does not replace the active pane.
type terminalSelectionKey struct {
	panel         *Panel
	pane          *terminalPane
	term          *myterm.Terminal
	binding, text string
}

func captureSelectionKey(m Model, selection string) terminalSelectionKey {
	key := terminalSelectionKey{panel: m.Panel, binding: m.TargetBinding, text: selection}
	if m.Panel != nil {
		key.pane, key.term = m.Panel.pane(m.Panel.active), m.Panel.Term
	}
	return key
}

func paintSelectionTarget(c *ui.Context, m Model, key terminalSelectionKey) {
	target, ok := FirstSelectionTarget(key.text)
	if !ok {
		return
	}
	label := terminalLabel("打开路径", "Open path")
	if target.URL {
		label = terminalLabel("系统浏览器打开链接", "Open link in system browser")
		if target.Loopback {
			label = terminalLabel("Browser 打开链接", "Open link in Browser")
		}
	}
	if !ui.Button(c, label).Key(key).FontSize(11).Disabled(m.OnOpenTarget == nil).Clicked() {
		return
	}
	p := m.Panel
	if p == nil || p != key.panel || p.pane(p.active) != key.pane || p.Term == nil || p.Term != key.term {
		return
	}
	selection, selected := p.Term.SelectionText()
	if !selected || selection != key.text {
		p.notice = terminalLabel("终端选区已变化，请重新选择", "Terminal selection changed; select it again")
		return
	}
	if err := m.OnOpenTarget(key.text); err != nil {
		p.notice = err.Error()
	} else {
		p.notice = terminalLabel("已打开所选目标", "Opened selected target")
	}
}
