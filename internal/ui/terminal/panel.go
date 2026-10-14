// Package terminal embeds the MyGo Ghostty terminal plugin in the frame bottom pane.
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

// Panel holds a workspace shell session for the bottom pane.
type Panel struct {
	Term *myterm.Terminal
	Err  error
	cwd  string
}

// Open starts a shell in workspaceRoot (login shell / PowerShell on Windows).
// Returns a panel with Err set when workspaceRoot is empty or the plugin fails.
func Open(workspaceRoot string, dark bool) *Panel {
	root := strings.TrimSpace(workspaceRoot)
	p := &Panel{cwd: root}
	if root == "" {
		p.Err = errors.New("no workspace directory")
		return p
	}
	info, err := os.Stat(root)
	if err != nil {
		p.Err = fmt.Errorf("terminal workspace: %w", err)
		return p
	}
	if !info.IsDir() {
		p.Err = errors.New("terminal workspace is not a directory")
		return p
	}
	if err := prepareNativeRuntime(); err != nil {
		p.Err = err
		return p
	}
	light := myterm.LightTheme()
	darkT := myterm.DarkTheme()
	opts := myterm.Options{
		Dir:       root,
		Theme:     light,
		DarkTheme: darkT,
	}
	if dark {
		opts.Theme = darkT
	}
	term, err := myterm.New(opts)
	if err != nil {
		p.Err = err
		return p
	}
	p.Term = term
	return p
}

// Workspace returns the directory the panel was opened for.
func (p *Panel) Workspace() string {
	if p == nil {
		return ""
	}
	return p.cwd
}

// Close releases the pseudo-terminal.
func (p *Panel) Close() {
	if p == nil || p.Term == nil {
		return
	}
	p.Term.Close()
	p.Term = nil
}

// Model is input to View.
type Model struct {
	Panel      *Panel
	ThemeID    string
	SystemDark bool
	// ShellsNote is optional terminal/shells + terminal/environment summary.
	ShellsNote string
	OnRestart  func()
	OnClose    func()
}

// View paints the terminal or a short error line in the bottom split.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)

	ui.Column(c).Fill().MinHeight(0).Background(field).Children(func() {
		ui.Row(c).Padding(4, 8, 4, 8).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyTerminalTitle)).FontSize(12).Bold().TextColor(chrome.Color(colors.FG)).Grow(1)
			if m.Panel != nil {
				if cwd := strings.TrimSpace(m.Panel.Workspace()); cwd != "" {
					if ui.Button(c, i18n.T(i18n.KeyTerminalCopyCwd)).FontSize(11).Clicked() {
						mygo.Clipboard.WriteText(cwd)
					}
				}
			}
			if m.OnClose != nil && ui.Button(c, "关闭终端").FontSize(11).Clicked() {
				m.OnClose()
			}
			if m.OnRestart != nil {
				if ui.Button(c, i18n.T(i18n.KeyTerminalRestart)).FontSize(11).Clicked() {
					m.OnRestart()
				}
			}
		})
		ui.Text(c, "切换工作区保留进程；关闭或重启会终止当前工作区终端").FontSize(10).TextColor(muted).Padding(0, 8, 2, 8)
		ui.Box(c).Fill().Height(1).Background(border)
		if note := strings.TrimSpace(m.ShellsNote); note != "" {
			ui.Text(c, note).FontSize(11).TextColor(muted).Padding(4, 12, 4, 12)
		}
		if m.Panel == nil {
			ui.Text(c, i18n.T(i18n.KeyTerminalUnavailable)).FontSize(12).TextColor(muted).
				Padding(8, 12, 8, 12)
			return
		}
		if m.Panel.Err != nil {
			ui.Text(c, m.Panel.Err.Error()).FontSize(12).TextColor(muted).
				Padding(8, 12, 8, 12)
			return
		}
		if m.Panel.Term == nil {
			ui.Text(c, "终端已关闭，点击重启重新打开").FontSize(12).TextColor(muted)
			return
		}
		select {
		case <-m.Panel.Term.Done():
			ui.Text(c, fmt.Sprintf("进程已退出 (code %d)，重启可新建终端", m.Panel.Term.ExitCode())).FontSize(11).TextColor(muted)
		default:
		}
		myterm.View(c, m.Panel.Term).Fill().Grow(1).AutoFocus()
	})
}
