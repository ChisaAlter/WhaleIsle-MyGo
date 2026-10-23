package terminal

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Original ui-primitives regular artwork used by TerminalWorkspace's ActionBar.
var terminalSplitIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M2.2 1.6h11.6A1.6 1.6 0 0 1 15.4 3.2v9.6a1.6 1.6 0 0 1-1.6 1.6H2.2A1.6 1.6 0 0 1 0.6 12.8V3.2A1.6 1.6 0 0 1 2.2 1.6Zm0 1.35A.25.25 0 0 0 1.95 3.2v9.6c0 .138.112.25.25.25H7.25V2.95H2.2Zm6.4 0v10.1h5.2a.25.25 0 0 0 .25-.25V3.2a.25.25 0 0 0-.25-.25H8.6Z" fill="currentColor"/></svg>`))
var terminalSplitVerticalIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><g transform="rotate(90 8 8)"><path d="M2.2 1.6h11.6A1.6 1.6 0 0 1 15.4 3.2v9.6a1.6 1.6 0 0 1-1.6 1.6H2.2A1.6 1.6 0 0 1 0.6 12.8V3.2A1.6 1.6 0 0 1 2.2 1.6Zm0 1.35A.25.25 0 0 0 1.95 3.2v9.6c0 .138.112.25.25.25H7.25V2.95H2.2Zm6.4 0v10.1h5.2a.25.25 0 0 0 .25-.25V3.2a.25.25 0 0 0-.25-.25H8.6Z" fill="currentColor"/></g></svg>`))
var terminalCloseIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3.5 3.5L12.5 12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><path d="M12.5 3.5L3.5 12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>`))

var terminalPlusIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M8 2V14M2 8H14" stroke="currentColor"/></svg>`))
var terminalMaximizeIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M2.33154 9.40576V13.1685C2.3318 13.4444 2.55556 13.6685 2.83154 13.6685H6.49463V14.6685H2.83154C2.00328 14.6685 1.3318 13.9967 1.33154 13.1685V9.40576H2.33154ZM13.1685 1.33154C13.9964 1.33199 14.6683 2.00352 14.6685 2.83154V6.40576H13.6685V2.83154C13.6683 2.5558 13.4441 2.33199 13.1685 2.33154H9.49463V1.33154H13.1685Z" fill="currentColor"/><path d="M9.4292 6.57077L13.914 2.08594M6.57077 9.4292L2.08594 13.914" stroke="currentColor"/></svg>`))
var terminalTrashIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M1.28149 3.88831H14.7187M5.41602 3.88833V2.47962C5.41602 2.29282 5.52492 2.11366 5.71876 1.98157C5.9126 1.84948 6.17551 1.77527 6.44964 1.77527H9.55053C9.82466 1.77527 10.0876 1.84948 10.2814 1.98157C10.4753 2.11366 10.5842 2.29282 10.5842 2.47962V3.88833M2.57349 3.88831L3.19366 13.2943C3.21937 13.5502 3.33952 13.7872 3.53065 13.9593C3.72178 14.1313 3.97016 14.2259 4.22729 14.2246H11.7728C12.0299 14.2259 12.2783 14.1313 12.4694 13.9593C12.6605 13.7872 12.7807 13.5502 12.8064 13.2943L13.4266 3.88831M6.44946 6.98926V11.1238M9.55054 6.98926V11.1238" stroke="currentColor"/></svg>`))

func terminalActionBar(c *ui.Context, m Model, compact bool) {
	p := m.Panel
	size, glyph := float32(28), float32(14)
	if compact {
		size, glyph = 22, 12
	}
	bar := ui.Row(c).Height(size).Shrink(0).Gap(0).Border(0.5, c.Theme().Border).Radius(6).Background(c.Theme().Surface)
	bar.Children(func() {
		tool := func(caption string, icon *ui.SVG, disabled bool) bool {
			button := ui.Button(c, "").Label(caption).Tooltip(caption).Width(size).Height(size).Shrink(0).Padding(0).Radius(0).BorderWidth(0).Disabled(disabled).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center)
			button.Children(func() { ui.Icon(c, icon).Size(glyph, glyph).TextColor(c.Theme().TextMuted).PassThrough() })
			return button.Clicked()
		}
		group := p.activeGroup()
		canSplit := p.cwd != "" && group != nil && len(group.panes) < maxPanesPerGroup
		if tool(terminalLabel("左右分栏", "Split horizontal"), terminalSplitIcon, !canSplit) {
			_ = p.createPane(m.SystemDark, true, false)
		}
		if tool(terminalLabel("上下分栏", "Split vertical"), terminalSplitVerticalIcon, !canSplit) {
			_ = p.createPane(m.SystemDark, true, true)
		}
		caption := terminalLabel("最大化", "Maximize")
		if p.Maximized {
			caption = terminalLabel("还原", "Restore")
		}
		if tool(caption, terminalMaximizeIcon, false) {
			p.Maximized = !p.Maximized
		}
		if tool(terminalLabel("新终端", "New terminal"), terminalPlusIcon, p.cwd == "") {
			_ = p.createPane(m.SystemDark, false, false)
		}
		if tool(terminalLabel("关闭当前", "Close current"), terminalTrashIcon, p.active == 0) {
			requestClosePane(m, p.active)
		}
	})
}

// terminalTabs paints the pane strip above the terminal body: one 28px chip per
// pane — the active chip filled, the shell's own title when it reports one, a
// close control on the active or hovered chip — and the action bar at its end.
func terminalTabs(c *ui.Context, m Model, fg, muted ui.Color) {
	p := m.Panel
	if p == nil || len(p.panes) == 0 {
		return
	}
	ui.Row(c).FillWidth().MinWidth(0).Gap(4).Padding(2, 8, 4, 8).AlignItems(ui.Center).Children(func() {
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				for index, pane := range append([]*terminalPane(nil), p.panes...) {
					paintTerminalChip(c, m, pane, index+1, fg, muted)
				}
			})
		})
		terminalActionBar(c, m, false)
	})
}

// paintTerminalChip paints one pane chip with the same geometry as a file tab.
func paintTerminalChip(c *ui.Context, m Model, pane *terminalPane, ordinal int, fg, muted ui.Color) {
	p := m.Panel
	if p == nil || pane == nil {
		return
	}
	active := p.active == pane.id
	chip := ui.Row(c.Key(fmt.Sprintf("terminal-tab-%d", pane.id))).Gap(5).Height(28).MinWidth(80).MaxWidth(200).Shrink(0).Radius(6).Padding(0, 8, 0, 10).AlignItems(ui.Center).Clip()
	hovered := chip.Hovered()
	switch {
	case active:
		chip.Background(c.Theme().SurfaceHover)
	case hovered:
		chip.Background(ui.RGBA(muted.R, muted.G, muted.B, .16))
	}
	textColor := muted
	if active {
		textColor = fg
	}
	caption := terminalTabLabel(pane, ordinal)
	chip.Children(func() {
		title := ui.Button(c, "").Label(terminalTitle(pane)).Grow(1).MinWidth(0).Height(26).Padding(0).BorderWidth(0).Background(ui.Transparent)
		title.Children(func() {
			ui.Text(c, caption).FontSize(12).TextColor(textColor).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		})
		if title.Clicked() {
			p.activate(pane.id)
		}
		if active || hovered {
			closeLabel := terminalLabel("关闭终端", "Close terminal")
			close := ui.Button(c, "").Label(closeLabel).Size(18, 18).Shrink(0).Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(closeLabel)
			close.Children(func() { ui.Icon(c, terminalCloseIcon).Size(9, 9).TextColor(muted).PassThrough() })
			if close.Hovered() {
				close.Background(c.Theme().SurfacePressed)
			}
			if close.Clicked() {
				requestClosePane(m, pane.id)
			}
		}
	})
}

// terminalTabLabel prefers the shell's own title (OSC 0/2) and falls back to the
// pane's position in the strip, so a chip names the running program when the
// shell reports one and otherwise reads as the first, second, third terminal.
// The number is the pane's place, not its creation id, so closing a middle pane
// never leaves a gap in the strip.
func terminalTabLabel(pane *terminalPane, ordinal int) string {
	if pane == nil {
		return ""
	}
	if pane.term != nil {
		if title := friendlyTerminalTitle(strings.TrimSpace(pane.term.Title())); title != "" {
			if runes := []rune(title); len(runes) > 40 {
				return string(runes[:40]) + "…"
			}
			return title
		}
	}
	return fmt.Sprintf(terminalLabel("终端 %d", "Terminal %d"), ordinal)
}

// friendlyTerminalTitle turns a shell's OSC path title into the program name, so
// a chip reads "PowerShell" instead of a truncated C:\WINDOWS\System32 path.
func friendlyTerminalTitle(title string) string {
	if title == "" {
		return ""
	}
	normalized := strings.ReplaceAll(title, "\\", "/")
	if !strings.Contains(normalized, "/") {
		return title
	}
	base := normalized[strings.LastIndex(normalized, "/")+1:]
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	switch strings.ToLower(base) {
	case "powershell", "pwsh":
		return "PowerShell"
	case "cmd":
		return terminalLabel("命令提示符", "Command Prompt")
	case "bash":
		return "Bash"
	case "wsl":
		return "WSL"
	}
	if base != "" {
		return base
	}
	return title
}