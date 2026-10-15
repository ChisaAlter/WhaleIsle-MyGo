// Package terminal embeds native MyGo Ghostty PTYs in the workspace bottom pane.
package terminal

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	opts := myterm.Options{Dir: root, Theme: light, DarkTheme: darkTheme}
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
	field := chrome.Color(colors.Field)
	border := chrome.Color(colors.Border)
	p := m.Panel
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

	ui.Column(c).Fill().MinHeight(0).Background(field).Children(func() {
		ui.Row(c).Padding(4, 8, 4, 8).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyTerminalTitle)).FontSize(12).Bold().TextColor(fg).Grow(1)
			if p != nil {
				if ui.Button(c, i18n.T(i18n.KeyTerminalCopyCwd)).FontSize(11).Disabled(p.cwd == "").Clicked() {
					mygo.Clipboard.WriteText(p.currentDirectory())
				}
				maximizeLabel := terminalLabel("最大化", "Maximize")
				if p.Maximized {
					maximizeLabel = terminalLabel("还原", "Restore")
				}
				if ui.Button(c, maximizeLabel).FontSize(11).Clicked() {
					p.Maximized = !p.Maximized
				}
				if ui.Button(c, terminalLabel("新终端", "New terminal")).FontSize(11).Disabled(p.cwd == "").Clicked() {
					_ = p.createPane(m.SystemDark, false, false)
				}
				group := p.activeGroup()
				canSplit := group != nil && len(group.panes) < maxPanesPerGroup
				if ui.Button(c, terminalLabel("左右分栏", "Split horizontal")).FontSize(11).Disabled(!canSplit).Clicked() {
					_ = p.createPane(m.SystemDark, true, false)
				}
				if ui.Button(c, terminalLabel("上下分栏", "Split vertical")).FontSize(11).Disabled(!canSplit).Clicked() {
					_ = p.createPane(m.SystemDark, true, true)
				}
				if ui.Button(c, terminalLabel("关闭当前", "Close current")).FontSize(11).Clicked() {
					closePaneFromView(m, p.active)
				}
			}
			if m.OnRestart != nil && ui.Button(c, i18n.T(i18n.KeyTerminalRestart)).FontSize(11).Clicked() {
				m.OnRestart()
			}
		})
		ui.Text(c, terminalLabel("切换或隐藏保留进程；关闭和重启只影响当前终端。每组最多四栏。", "Switching or hiding keeps processes running. Close and restart affect only the active terminal. Up to four panes per group.")).FontSize(10).TextColor(muted).Padding(0, 8, 2, 8)
		ui.Box(c).Fill().Height(1).Background(border)
		if note := strings.TrimSpace(m.ShellsNote); note != "" {
			ui.Text(c, note).FontSize(11).TextColor(muted).Padding(4, 12, 4, 12)
		}
		if p == nil {
			ui.Text(c, i18n.T(i18n.KeyTerminalUnavailable)).FontSize(12).TextColor(muted).Padding(8, 12, 8, 12)
			return
		}
		if fragment := FormatSelection(selection); fragment != "" {
			ui.Row(c).Gap(6).Padding(4, 8, 4, 8).Children(func() {
				paintSelectionTarget(c, m, selectionKey)
				if ui.Button(c, terminalLabel("复制所选文本", "Copy selection")).FontSize(11).Clicked() {
					mygo.Clipboard.WriteText(selection)
				}
				if ui.Button(c, terminalLabel("添加到对话", "Add to chat")).FontSize(11).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Clicked() {
					if err := m.OnAddToChat(fragment); err != nil {
						p.notice = err.Error()
					} else {
						p.notice = terminalLabel("已添加到对话草稿", "Added to the chat draft")
					}
				}
			})
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
		if len(p.panes) > 1 {
			ui.Scroll(c).Height(62).FillWidth().Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for _, pane := range append([]*terminalPane(nil), p.panes...) {
						shell := pane
						ui.Row(c).Key(shell.id).Gap(4).Children(func() {
							name := terminalTitle(shell)
							if shell.id == p.active {
								name = "▸ " + name
							}
							if ui.Button(c, name).FontSize(11).Clicked() {
								p.activate(shell.id)
							}
							if ui.Button(c, "×").FontSize(11).Clicked() {
								closePaneFromView(m, shell.id)
							}
						})
					}
				})
			})
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
					if len(group.panes) > 1 {
						if ui.Button(c, terminalTitle(shell)).FontSize(10).Clicked() {
							p.activate(shell.id)
						}
					}
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
					myterm.View(c, shell.term).Fill().Grow(1).FocusBind(&p.focused, shell.id)
				})
			}
		}
		if group.vertical {
			ui.Column(c).Key(group.id).Fill().Grow(1).MinHeight(0).Gap(4).Children(paint)
		} else {
			ui.Row(c).Key(group.id).Fill().Grow(1).MinHeight(0).MinWidth(0).Gap(4).Children(paint)
		}
	})
}
