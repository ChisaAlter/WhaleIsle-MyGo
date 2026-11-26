// Package terminal embeds native MyGo Ghostty PTYs in the workspace bottom pane.
package terminal

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo"
	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Panel owns a workspace's independent PTYs and split groups. Term and Err
// mirror the active pane for existing callers. Hiding never closes a process.
type Panel struct {
	Term *myterm.Terminal
	Err  error
	cwd  string
	// Maximized uses 75% of the center column without overwriting its saved size.
	Maximized bool

	panes         []*terminalPane
	groups        []*terminalGroup
	active        uint64
	focused       uint64
	focusPending  uint64
	nextID        uint64
	notice        string
	onOpenTarget  func(string) error
	returnFocusFn func()
	// pendingClose holds the pane whose close is waiting for confirmation.
	pendingClose uint64
}

// Open starts the first shell in a nonempty, existing workspace directory.
func Open(workspaceRoot string, dark bool) *Panel {
	p := &Panel{cwd: strings.TrimSpace(workspaceRoot)}
	_ = p.createPane(dark, false, false)
	return p
}

func openNativeTerminal(root string, dark bool, onLink ...func(string) bool) (*myterm.Terminal, error) {
	if root == "" {
		return nil, errors.New("no workspace directory")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("terminal workspace: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("terminal workspace is not a directory")
	}
	if err := prepareNativeRuntime(); err != nil {
		return nil, err
	}
	// Open before the first frame has a theme id: the default palette of the
	// half; View refines it once the workspace's theme is known.
	themed := terminalTheme(theme.Default().Active(dark), dark)
	opts := myterm.Options{
		Dir: root, Theme: themed, DarkTheme: themed, Transparent: true,
		PTYResizeDelay: 150 * time.Millisecond,
		Font:           terminalFont(theme.RenderPreferences().Terminal),
	}
	opts.LinkAt = TargetAt
	if len(onLink) > 0 {
		opts.OnLinkActivate = onLink[0]
	}
	return myterm.New(opts)
}

func (p *Panel) Workspace() string {
	if p == nil {
		return ""
	}
	return p.cwd
}

// RunningCount counts live PTYs, including hidden panes, without counting
// retained output from an exited process. The caller serializes panel access.
func (p *Panel) RunningCount() int {
	if p == nil {
		return 0
	}
	running := func(term *myterm.Terminal) bool {
		if term == nil || term.Done() == nil {
			return false
		}
		select {
		case <-term.Done():
			return false
		default:
			return true
		}
	}
	if len(p.panes) == 0 {
		if running(p.Term) {
			return 1
		}
		return 0
	}
	count := 0
	for _, pane := range p.panes {
		if running(pane.term) {
			count++
		}
	}
	return count
}

// Close releases every process in this workspace, for app exit/explicit removal.
func (p *Panel) Close() {
	if p == nil {
		return
	}
	if len(p.panes) == 0 && p.Term != nil {
		_ = p.Term.Close()
	}
	for _, pane := range p.panes {
		if pane.term != nil {
			_ = pane.term.Close()
		}
	}
	p.panes, p.groups = nil, nil
	p.active, p.focused, p.focusPending = 0, 0, 0
	p.Term, p.Err = nil, nil
	p.notice = ""
}

// FocusActive arms a one-shot focus request for the active terminal pane
// when explicitly opening or switching into the terminal drawer.
func (p *Panel) FocusActive() {
	if p == nil || len(p.panes) == 0 {
		return
	}
	if p.active == 0 || p.pane(p.active) == nil {
		p.activate(p.panes[0].id)
	}
	if p.active != 0 {
		p.focusPending = p.active
	}
}

// ClearFocus cancels any pending one-shot focus request so collapsing and
// reopening the terminal drawer never steals focus from the composer, and
// returns focus if a terminal pane currently held it.
func (p *Panel) ClearFocus() {
	if p == nil {
		return
	}
	hadFocus := p.focused != 0
	p.focusPending = 0
	p.focused = 0
	if hadFocus && p.returnFocusFn != nil {
		p.returnFocusFn()
	}
}

// CloseDrawer collapses focus ownership and runs optional focus restoration when
// the terminal drawer held focus.
func (p *Panel) CloseDrawer(returnFocus ...func()) {
	if p == nil {
		return
	}
	if len(returnFocus) > 0 && returnFocus[0] != nil {
		p.returnFocusFn = returnFocus[0]
	}
	p.FlushResize()
	p.ClearFocus()
}

type Model struct {
	Panel      *Panel
	ThemeID    string
	SystemDark bool
	// ShellsNote is an optional read-only environment note.
	ShellsNote string
	OnRestart  func()
	// OnClose hides/removes the workspace panel once its last PTY is closed.
	OnClose       func()
	OnReturnFocus func()
	// OnOpenShortcuts opens the settings shortcuts page; nil keeps the menu
	// entry disabled because this build cannot navigate there.
	OnOpenShortcuts func()
	OnAddToChat     func(string) error
	CanAddToChat    bool
	// TargetBinding identifies the painted Host/session for pointer ownership.
	TargetBinding string
	OnOpenTarget  func(selection string) error
}

func terminalLabel(zh, en string) string {
	return i18n.Pick(zh, en)
}

func terminalTitle(pane *terminalPane) string {
	name := fmt.Sprintf("%s %d", i18n.T(i18n.KeyTerminalTitle), pane.id)
	if pane.term != nil {
		title := strings.TrimSpace(pane.term.Title())
		runes := []rune(title)
		if len(runes) > 40 {
			title = string(runes[:40]) + "…"
		}
		if title != "" {
			name += ": " + title
		}
	}
	return name
}

func closePaneFromView(m Model, id uint64) {
	m.Panel.closePane(id)
	if len(m.Panel.panes) == 0 {
		if m.OnReturnFocus != nil {
			m.OnReturnFocus()
		}
		if m.OnClose != nil {
			m.OnClose()
		}
	}
}

func View(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	p := m.Panel
	if p != nil {
		p.onOpenTarget = m.OnOpenTarget
		if m.OnReturnFocus != nil {
			p.returnFocusFn = m.OnReturnFocus
		}
		hadPanes := len(p.panes) > 0
		for _, pane := range append([]*terminalPane(nil), p.panes...) {
			if pane.term != nil {
				select {
				case <-pane.term.Done():
					p.closePane(pane.id)
				default:
				}
			}
		}
		if hadPanes && len(p.panes) == 0 {
			if m.OnReturnFocus != nil {
				m.OnReturnFocus()
			}
			if m.OnClose != nil {
				m.OnClose()
			}
		}
	}
	selection := ""
	if p != nil && p.Term != nil {
		if text, ok := p.Term.SelectionText(); ok {
			selection = text
		}
	}

	selectionKey := captureSelectionKey(m, selection)

	root := chrome.GlassSurface(ui.Column(c).Fill().MinHeight(0).MinWidth(0), chrome.Color(colors.BG), chrome.MaterialTerminal)
	root.ContextMenu(func(menu *ui.Menu) {
		if p != nil && menu.Item(i18n.T(i18n.KeyTerminalCopyCwd)).Disabled(p.cwd == "").Chosen() {
			mygo.Clipboard.WriteText(p.currentDirectory())
		}
		if m.OnRestart != nil && menu.Item(i18n.T(i18n.KeyTerminalRestart)).Chosen() {
			m.OnRestart()
		}
		menu.Separator()
		if m.OnOpenShortcuts == nil {
			menu.Item(i18n.T(i18n.KeySurfacesTerminalShortcuts)).Disabled(true)
		} else if menu.Item(i18n.T(i18n.KeySurfacesTerminalShortcuts)).Chosen() {
			m.OnOpenShortcuts()
		}
	})
	root.Children(func() {
		if note := strings.TrimSpace(m.ShellsNote); note != "" {
			ui.Text(c, note).FontSize(11).TextColor(muted).Padding(4, 12, 4, 12)
		}
		if p == nil {
			ui.Text(c, i18n.T(i18n.KeyTerminalUnavailable)).FontSize(12).TextColor(muted).Padding(8, 12, 8, 12)
			return
		}
		if p.notice != "" && len(p.panes) > 0 {
			ui.Row(c).Gap(6).Children(func() {
				ui.Text(c, p.notice).FontSize(11).TextColor(muted).Grow(1)
				if widgets.Button(c, colors, terminalLabel("关闭提示", "Dismiss"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					p.notice = ""
				}
			})
		}
		if len(p.panes) == 0 {
			message := terminalLabel("终端已关闭。", "Terminal closed.")
			if p.Err != nil {
				message = errcopy.Sentence(p.Err)
			}
			ui.Text(c, message).FontSize(12).TextColor(muted).Padding(8, 12, 8, 12)
			return
		}
		group := p.activeGroup()
		if group == nil {
			return
		}
		paint := func() {
			for _, id := range append([]uint64(nil), group.panes...) {
				shell := p.pane(id)
				if shell == nil {
					continue
				}
				ui.Column(c).Key(shell.id).Fill().Grow(1).MinWidth(0).MinHeight(0).Padding(4, 8, 8, 8).Children(func() {
					if shell.err != nil {
						ui.Text(c, errcopy.Sentence(shell.err)).FontSize(12).TextColor(muted)
						return
					}
					if shell.term == nil {
						return
					}
					select {
					case <-shell.term.Done():
						ui.Text(c, fmt.Sprintf(terminalLabel("进程已退出（code %d），重启可新建当前终端", "Process exited (code %d). Restart reopens this terminal."), shell.term.ExitCode())).FontSize(11).TextColor(muted)
					default:
					}
					font := theme.RenderPreferences().Terminal
					if shell.appearanceFont != font {
						shell.term.SetFont(terminalFont(font))
						shell.appearanceFont = font
					}
					colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
					if shell.appearanceColors != colors {
						native := terminalTheme(colors, m.SystemDark)
						shell.term.SetTheme(native, native)
						shell.appearanceColors = colors
					}
					surface := myterm.View(c, shell.term).Fill().Grow(1).Margin(4, 8, 4, 8)
					if p.focusPending == shell.id {
						surface.Focus()
						p.focused = shell.id
						p.focusPending = 0
					} else if surface.Focused() {
						p.focused = shell.id
						if shell.id != p.active {
							p.activate(shell.id)
							c.Invalidate()
						}
					} else if p.focused == shell.id {
						p.focused = 0
					}
					if fragment := FormatSelection(selection); shell.id == p.active && fragment != "" {
						paintSelectionActionBar(c, m, p, shell, colors, border, surface.Bounds(), selection, fragment, selectionKey)
					}
				})
			}
			p.focusPending = 0
		}
		terminalTabs(c, m, fg, muted)
		terminalClosePrompt(c, m, fg, muted)
		ui.Row(c).Fill().Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			if group.vertical {
				ui.Column(c).Key(group.id).Fill().Grow(1).MinHeight(0).Gap(0.5).Children(paint)
			} else {
				ui.Row(c).Key(group.id).Fill().Grow(1).MinHeight(0).MinWidth(0).Gap(0.5).Children(paint)
			}
		})
	})
}

func selectionActionBarLayout(bounds ui.Rect, winW, winH float32) (availLeft, availRight, pad, maxW, initialTop float32) {
	availLeft = max(float32(0), bounds.X)
	availRight = bounds.X + bounds.W
	if winW > 0 && winW < availRight {
		availRight = winW
	}
	availW := max(float32(0), availRight-availLeft)
	pad = min(design.Sp(2), availW/4)
	maxW = max(float32(0), availW-2*pad)
	maxBottom := bounds.Y + bounds.H
	if winH > 0 && winH < maxBottom {
		maxBottom = winH
	}
	initialTop = max(float32(0), max(bounds.Y, maxBottom-design.Sp(8.5)))
	return availLeft, availRight, pad, maxW, initialTop
}

func paintSelectionActionBar(c *ui.Context, m Model, p *Panel, shell *terminalPane, colors theme.Colors, border ui.Color, bounds ui.Rect, selection, fragment string, selectionKey terminalSelectionKey) {
	winW, winH := c.Size()
	availLeft, availRight, pad, maxW, initialTop := selectionActionBarLayout(bounds, winW, winH)
	ui.Overlay(c, func() {
		bar := ui.Row(c.Key("terminal-selection-bar")).Absolute().Left(availLeft+pad).Top(initialTop).MaxWidth(maxW).MinHeight(design.Sp(6.5)).Wrap().AlignItems(ui.Center).Gap(0).Padding(0, design.Sp(1)).Border(design.BorderWidth/2, border).Radius(design.RadiusMD).Background(chrome.FrostInset(colors, m.SystemDark, c.Theme().Surface)).Label(terminalLabel("终端选区操作", "Terminal selection actions"))
		bar.Children(func() {
			if widgets.Button(c, colors, terminalLabel("复制", "Copy"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
				mygo.Clipboard.WriteText(selection)
			}
			if widgets.Button(c, colors, terminalLabel("加入对话", "Add to chat"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Clicked() {
				if err := m.OnAddToChat(fragment); err != nil {
					p.notice = errcopy.Sentence(err)
				} else {
					p.notice = terminalLabel("已加入对话草稿", "Added to the chat draft")
					if shell != nil && shell.term != nil {
						shell.term.ClearSelection()
					}
				}
			}
			paintSelectionTarget(c, m, selectionKey)
		})
		bar.OffsetAfterLayout(func(r ui.Rect) ui.Point {
			desiredX := max(availLeft+pad, availRight-pad-r.W)
			if winW > 0 && desiredX+r.W > winW {
				desiredX = max(float32(0), winW-r.W)
			}
			if desiredX < availLeft {
				desiredX = availLeft
			}
			maxBottom := bounds.Y + bounds.H
			if winH > 0 && winH < maxBottom {
				maxBottom = winH
			}
			desiredY := max(bounds.Y, maxBottom-r.H-pad)
			if desiredY < 0 {
				desiredY = 0
			}
			return ui.Point{X: desiredX - r.X, Y: desiredY - r.Y}
		})
	})
}
