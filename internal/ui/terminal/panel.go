// Package terminal embeds native MyGo Ghostty PTYs in the workspace bottom pane.
package terminal

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
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

	panes   []*terminalPane
	groups  []*terminalGroup
	active  uint64
	focused uint64
	nextID  uint64
	notice  string
	// pendingClose holds the pane whose close is waiting for confirmation.
	pendingClose uint64
}

// Open starts the first shell in a nonempty, existing workspace directory.
func Open(workspaceRoot string, dark bool) *Panel {
	p := &Panel{cwd: strings.TrimSpace(workspaceRoot)}
	_ = p.createPane(dark, false, false)
	return p
}

func openNativeTerminal(root string, dark bool) (*myterm.Terminal, error) {
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
	light, darkTheme := myterm.LightTheme(), myterm.DarkTheme()
	opts := myterm.Options{Dir: root, Theme: light, DarkTheme: darkTheme, Transparent: true, PTYResizeDelay: 150 * time.Millisecond}
	if dark {
		opts.Theme = darkTheme
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
	p.active, p.focused = 0, 0
	p.Term, p.Err = nil, nil
	p.notice = ""
}

type Model struct {
	Panel      *Panel
	ThemeID    string
	SystemDark bool
	// ShellsNote is an optional read-only environment note.
	ShellsNote string
	OnRestart  func()
	// OnClose hides/removes the workspace panel once its last PTY is closed.
	OnClose      func()
	OnAddToChat  func(string) error
	CanAddToChat bool
	// TargetBinding identifies the painted Host/session for pointer ownership.
	TargetBinding string
	OnOpenTarget  func(selection string) error
}

func terminalLabel(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
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
	if len(m.Panel.panes) == 0 && m.OnClose != nil {
		m.OnClose()
	}
}

func View(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.BackdropTerminal(chrome.Color(colors.Field))
	border := chrome.Color(colors.Border)
	p := m.Panel
	if p != nil {
		for _, pane := range append([]*terminalPane(nil), p.panes...) {
			if pane.term != nil {
				select {
				case <-pane.term.Done():
					p.closePane(pane.id)
				default:
				}
			}
		}
	}
	// FocusBind writes pointer/keyboard focus back without replacing active state
	// when a toolbar control temporarily takes focus.
	if p != nil && p.focused != 0 && p.focused != p.active {
		p.activate(p.focused)
	}
	selection := ""
	if p != nil && p.Term != nil {
		if text, ok := p.Term.SelectionText(); ok {
			selection = text
		}
	}

	selectionKey := captureSelectionKey(m, selection)

	root := ui.Column(c).Fill().MinHeight(0).MinWidth(0).Background(field)
	root.ContextMenu(func(menu *ui.Menu) {
		if p != nil && menu.Item(i18n.T(i18n.KeyTerminalCopyCwd)).Disabled(p.cwd == "").Chosen() {
			mygo.Clipboard.WriteText(p.currentDirectory())
		}
		if m.OnRestart != nil && menu.Item(i18n.T(i18n.KeyTerminalRestart)).Chosen() {
			m.OnRestart()
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
				if ui.Button(c, terminalLabel("关闭提示", "Dismiss")).FontSize(10).Clicked() {
					p.notice = ""
				}
			})
		}
		if len(p.panes) == 0 {
			message := terminalLabel("终端已关闭，点击新终端或重启重新打开", "Terminal closed. Choose New terminal or Restart to reopen.")
			if p.Err != nil {
				message = p.Err.Error()
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
				ui.Column(c).Key(shell.id).Fill().Grow(1).MinWidth(0).MinHeight(0).Children(func() {
					if shell.err != nil {
						ui.Text(c, shell.err.Error()).FontSize(12).TextColor(muted)
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
					font := theme.CurrentPreferences().Terminal
					if shell.appearanceFont != font {
						shell.term.SetFont(myterm.Font{Family: font.Family, Size: font.Size})
						shell.appearanceFont = font
					}
					colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
					if shell.appearanceColors != colors {
						native := myterm.LightTheme()
						if m.SystemDark {
							native = myterm.DarkTheme()
						}
						native.Background = chrome.Color(colors.BG)
						native.Foreground = chrome.Color(colors.FG)
						native.Cursor = chrome.Color(colors.Accent)
						native.Selection = chrome.Color(colors.Accent).Alpha(.3)
						shell.term.SetTheme(native, native)
						shell.appearanceColors = colors
					}
					surface := myterm.View(c, shell.term).Fill().Grow(1).FocusBind(&p.focused, shell.id)
					if fragment := FormatSelection(selection); shell.id == p.active && fragment != "" {
						bounds := surface.Bounds()
						ui.Overlay(c, func() {
							ui.Row(c).Absolute().Left(max(bounds.X+8, bounds.X+bounds.W-340)).Top(max(bounds.Y, bounds.Y+bounds.H-34)).Gap(0).Padding(0, 4).Height(26).Border(0.5, border).Radius(4).Background(c.Theme().Surface).Children(func() {
								if ui.Button(c, terminalLabel("复制", "Copy")).FontSize(12).Clicked() {
									mygo.Clipboard.WriteText(selection)
								}
								if ui.Button(c, terminalLabel("添加到对话", "Add to chat")).FontSize(12).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Clicked() {
									if err := m.OnAddToChat(fragment); err != nil {
										p.notice = err.Error()
									} else {
										p.notice = terminalLabel("已添加到对话草稿", "Added to the chat draft")
										shell.term.ClearSelection()
									}
								}
								paintSelectionTarget(c, m, selectionKey)
							})
						})
					}
				})
			}
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
